package docker

import (
	"context"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/juju/errors"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"
)

// althrough netconn doesn't have a query string, we still match as if it has a query string
var regexNetworkConnect *regexp.Regexp = regexp.MustCompile(`/(.*?)/networks/([a-zA-Z0-9][a-zA-Z0-9_.-]*)/connect(\?.*)?`)

type networkConnectHandler struct {
	utils.LoggerFactory
	client       barrelHttp.Client
	inspectAgent containerInspectAgent
	vessel.Helper
}

func newNetworkConnectHandler(client barrelHttp.Client, vess vessel.Helper, inspectAgent containerInspectAgent) proxy.RequestHandler {
	return networkConnectHandler{
		LoggerFactory: utils.NewObjectLogger("networkConnectHandler"),
		client:        client,
		Helper:        vess,
		inspectAgent:  inspectAgent,
	}
}

type networkConnectRequest struct {
	networkIdentifier string
	version           string
}

// Handle .
func (handler networkConnectHandler) Handle(ctx proxy.HandleContext, res http.ResponseWriter, req *http.Request) {
	logger := handler.Logger("Handle")

	var (
		networkConnectRequest networkConnectRequest
		pools                 []types.Pool
		matched               bool
		err                   error
	)
	if networkConnectRequest, matched = handler.match(req); !matched {
		ctx.Next()
		return
	}
	if pools, err = handler.CalicoIPAllocator().GetPoolsByNetworkName(
		context.Background(),
		networkConnectRequest.networkIdentifier,
	); err != nil {
		if err == types.ErrUnsupervisedNetwork {
			ctx.Next()
			return
		}
		writeErrorResponse(res, logger, err, "GetIPPoolsByNetworkName")
		return
	}

	logger.Debug("network connect request")
	var (
		body           []byte
		bodyObject     utils.Object
		containerInfo  containerInspectResult
		fixedIPAddress types.IP
		allocated      bool
		clientResp     *http.Response
	)
	if body, err = ioutil.ReadAll(req.Body); err != nil {
		writeErrorResponse(res, logger, err, "read server request body error")
		return
	}
	if bodyObject, err = utils.UnmarshalObject(body); err != nil {
		writeErrorResponse(res, logger, err, "unmarshal server request body")
		return
	}
	if containerInfo, err = handler.getContainerInfo(
		func(identifier string) (containerInspectResult, error) {
			return handler.inspectAgent.Inspect(identifier, networkConnectRequest.version)
		},
		bodyObject,
	); err != nil {
		writeErrorResponse(res, logger, err, "get container info")
		return
	}
	// it doesn't have a fixed-ip label, just ignore
	if isFixedIPLabelEnabled(containerInfo) {
		if allocated, fixedIPAddress, err = handler.checkOrRequestFixedIP(pools, bodyObject); err != nil {
			writeErrorResponse(res, logger, err, "check and request fixed-ip")
			return
		}
		if body, err = utils.Marshal(bodyObject.Any()); err != nil {
			writeErrorResponse(res, logger, err, "marshal server request")
			if allocated {
				handler.releaseReservedAddress(fixedIPAddress, "marshal body object failed")
			}
			return
		}
	}
	if clientResp, err = requestDockerd(handler.client, req, body); err != nil {
		writeErrorResponse(res, logger, err, "request dockerd socket")
		if allocated {
			handler.releaseReservedAddress(fixedIPAddress, "request dockerd failed")
		}
		return
	}
	handler.writeServerResponse(res, allocated, fixedIPAddress, clientResp)
}

func (handler networkConnectHandler) releaseReservedAddress(address types.IP, label string) {
	logger := handler.Logger("releaseReservedAddress")

	if err := handler.FixedIPAllocator().UnallocFixedIP(context.Background(), address); err != nil {
		logger.Errorf("release reserved address error when %s, cause = %v", label, err)
	}
}

func (handler networkConnectHandler) match(request *http.Request) (networkConnectRequest, bool) {
	logger := handler.Logger("match")

	if request.Method == http.MethodPost {
		subMatches := regexNetworkConnect.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			networkConnectRequest := networkConnectRequest{}
			networkConnectRequest.version = subMatches[1]
			networkConnectRequest.networkIdentifier = subMatches[2]
			logger.Debugf("docker api version = %s", networkConnectRequest.version)
			return networkConnectRequest, true
		}
	}
	return networkConnectRequest{}, false
}

func (handler networkConnectHandler) getContainerInfo(
	inspect func(string) (containerInspectResult, error),
	bodyObject utils.Object,
) (containerInspectResult, error) {
	var (
		containerIdentifier string
		containerInfo       containerInspectResult
	)
	if node, ok := bodyObject.Get("Container"); !ok || node.Null() {
		return containerInfo, errors.New("Container identifier isn't provided")
	} else if containerIdentifier, ok = node.StringValue(); !ok {
		return containerInfo, errors.New("Parse container identifier error")
	}
	return inspect(containerIdentifier)
}

func (handler networkConnectHandler) checkOrRequestFixedIP(
	pools []types.Pool,
	body utils.Object,
) (bool, types.IP, error) {
	var (
		ipamConfig  utils.Object
		ipv4Address string
		ipv6Address string
		err         error
	)
	if ipamConfig, err = handler.getIPAMConfig(body); err != nil {
		return false, types.IP{}, err
	}
	if ipv4Address, err = getStringMember(ipamConfig, "IPv4Address"); err != nil {
		return false, types.IP{}, err
	}
	if ipv6Address, err = getStringMember(ipamConfig, "IPv6Address"); err != nil {
		return false, types.IP{}, err
	}
	if ipv4Address == "" && ipv6Address == "" {
		var addr types.IPAddress
		if addr, err = handler.FixedIPAllocator().AllocFixedIPFromPools(context.Background(), pools); err != nil {
			return false, types.IP{}, err
		}
		if addr.Version == 4 {
			ipamConfig.Set("IPv4Address", utils.NewStringNode(addr.Address))
		} else {
			ipamConfig.Set("IPv6Address", utils.NewStringNode(addr.Address))
		}
		return true, addr.IP, nil
	}
	// either ipv4 or ipv6 is non blank
	return false, types.IP{}, nil
}

func isFixedIPLabelEnabled(containerInfo containerInspectResult) bool {
	if value, ok := containerInfo.Config.Labels[FixedIPLabel]; !ok || !isFixedIPEnableByStringValue(value) {
		// don't require fixed ip
		return false
	}
	return true
}

func (handler networkConnectHandler) getIPAMConfig(body utils.Object) (utils.Object, error) {
	var (
		endpointConfig utils.Object
		err            error
	)
	if endpointConfig, err = ensureObjectMember(body, "EndpointConfig"); err != nil {
		return endpointConfig, err
	}
	return ensureObjectMember(endpointConfig, "IPAMConfig")
}

func (handler networkConnectHandler) writeServerResponse(
	res http.ResponseWriter,
	allocated bool,
	fixedIPAddress types.IP,
	clientResp *http.Response,
) {
	logger := handler.Logger("writeServerResponse")

	defer clientResp.Body.Close()

	var err error
	if clientResp.StatusCode != http.StatusOK {
		logger.Errorf("connect network failed, status code = %d", clientResp.StatusCode)
		if err = utils.Forward(clientResp, res); err != nil {
			logger.Errorf("forward message failed %v", err)
		}
		if allocated {
			if err := handler.FixedIPAllocator().UnallocFixedIP(
				context.Background(),
				fixedIPAddress,
			); err != nil {
				logger.Errorf("release address error after unsuccess container create, cause = %v", err)
			}
		}
		return
	}
	if err = utils.Forward(clientResp, res); err != nil {
		logger.Errorf("forward and read message failed %v", err)
	}
}
