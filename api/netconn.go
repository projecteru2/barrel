package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/ipam"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/utils"
	minionsTypes "github.com/projecteru2/minions/types"
	log "github.com/sirupsen/logrus"
)

// althrough netconn doesn't have a query string, we still match as if it has a query string
var regexNetworkConnect *regexp.Regexp = regexp.MustCompile(`/(.*?)/networks/([a-zA-Z0-9][a-zA-Z0-9_.-]*)/connect(\?.*)?`)

// NetworkConnectHandler .
type NetworkConnectHandler struct {
	sock           sock.SocketInterface
	inspectHandler ContainerInspectHandler
	ipam           ipam.IPAM
}

type networkConnectRequest struct {
	networkIdentifier string
	version           string
}

// NewNetworkConnectHandler .
func NewNetworkConnectHandler(sock sock.SocketInterface, ipam ipam.IPAM) NetworkConnectHandler {
	return NetworkConnectHandler{
		sock:           sock,
		inspectHandler: ContainerInspectHandler{sock},
		ipam:           ipam,
	}
}

// Handle .
func (handler NetworkConnectHandler) Handle(ctx utils.HandleContext, res http.ResponseWriter, req *http.Request) {
	var (
		networkConnectRequest networkConnectRequest
		pools                 []*minionsTypes.Pool
		matched               bool
		err                   error
	)
	if networkConnectRequest, matched = handler.match(req); !matched {
		ctx.Next()
		return
	}
	if pools, err = handler.ipam.GetIPPoolsByNetworkName(
		networkConnectRequest.networkIdentifier,
	); err != nil {
		if err == ipam.ErrUnsupervisedNetwork {
			ctx.Next()
			return
		}
		handler.writeErrorResponse(res, err, "GetIPPoolsByNetworkName")
		return
	}

	log.Debug("[NetworkConnectHandler.Handle] network connect request")
	var (
		body           []byte
		bodyObject     utils.Object
		containerInfo  ContainerInspectResult
		fixedIPAddress minionsTypes.ReservedAddress
		allocated      bool
		clientResp     *http.Response
	)
	if body, err = ioutil.ReadAll(req.Body); err != nil {
		handler.writeErrorResponse(res, err, "read server request body error")
		return
	}
	if bodyObject, err = utils.UnmarshalObject(body); err != nil {
		handler.writeErrorResponse(res, err, "unmarshal server request body")
		return
	}
	if containerInfo, err = handler.getContainerInfo(
		func(identifier string) (ContainerInspectResult, error) {
			return handler.inspectHandler.Inspect(identifier, networkConnectRequest.version)
		},
		bodyObject,
	); err != nil {
		handler.writeErrorResponse(res, err, "get container info")
		return
	}
	// it doesn't have a fixed-ip label, just ignore
	if isFixedIPLabelEnabled(containerInfo) {
		if allocated, fixedIPAddress, err = handler.checkOrRequestFixedIP(pools, bodyObject); err != nil {
			handler.writeErrorResponse(res, err, "check and request fixed-ip")
			return
		}
		if body, err = utils.Marshal(bodyObject.Any()); err != nil {
			handler.writeErrorResponse(res, err, "marshal server request")
			if allocated {
				handler.ipam.ReleaseReservedAddress(fixedIPAddress)
			}
			return
		}
	}
	if clientResp, err = handler.requestDockerd(req, body); err != nil {
		handler.writeErrorResponse(res, err, "request dockerd socket")
		if allocated {
			handler.ipam.ReleaseReservedAddress(fixedIPAddress)
		}
		return
	}
	handler.writeServerResponse(res, containerInfo.ID, allocated, fixedIPAddress, clientResp)
}

func (handler NetworkConnectHandler) match(request *http.Request) (networkConnectRequest, bool) {
	if request.Method == http.MethodDelete {
		subMatches := regexNetworkConnect.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			networkConnectRequest := networkConnectRequest{}
			networkConnectRequest.version = subMatches[1]
			networkConnectRequest.networkIdentifier = subMatches[2]
			log.Debugf("[ContainerDeleteHandler.match] docker api version = %s", networkConnectRequest.version)
			return networkConnectRequest, true
		}
	}
	return networkConnectRequest{}, false
}

func (handler NetworkConnectHandler) writeErrorResponse(res http.ResponseWriter, err error, label string) {
	log.Errorf("[NetworkConnectHandler::Handle] %s failed %v", label, err)
	if err := utils.WriteBadGateWayResponse(
		res,
		utils.HTTPSimpleMessageResponseBody{
			Message: label + " error",
		},
	); err != nil {
		log.Errorf("[NetworkConnectHandler.Handle] write %s error response failed %v", label, err)
	}
}

func (handler NetworkConnectHandler) getContainerInfo(
	inspect func(string) (ContainerInspectResult, error),
	bodyObject utils.Object,
) (ContainerInspectResult, error) {
	var (
		containerIdentifier string
		containerInfo       ContainerInspectResult
	)
	if node, ok := bodyObject.Get("Container"); !ok || node.Null() {
		return containerInfo, errors.New("Container identifier isn't provided")
	} else if containerIdentifier, ok = node.StringValue(); !ok {
		return containerInfo, errors.New("Parse container identifier error")
	}
	return inspect(containerIdentifier)
}

func (handler NetworkConnectHandler) requestDockerd(req *http.Request, body []byte) (clientResp *http.Response, err error) {
	var (
		clientReq http.Request = *req
	)
	clientReq.ContentLength = int64(len(body))
	clientReq.Body = ioutil.NopCloser(bytes.NewReader(body))
	return handler.sock.Request(&clientReq)
}

func (handler NetworkConnectHandler) checkOrRequestFixedIP(
	pools []*minionsTypes.Pool,
	body utils.Object,
) (bool, minionsTypes.ReservedAddress, error) {
	var (
		ipamConfig  utils.Object
		ipv4Address string
		ipv6Address string
		address     minionsTypes.ReservedAddress
		err         error
	)
	if ipamConfig, err = handler.getIPAMConfig(body); err != nil {
		return false, address, err
	}
	if ipv4Address, err = getStringMember(ipamConfig, "IPv4Address"); err != nil {
		return false, address, err
	}
	if ipv6Address, err = getStringMember(ipamConfig, "IPv6Address"); err != nil {
		return false, address, err
	}
	if ipv4Address == "" && ipv6Address == "" {
		var addr common.ReservedAddress
		if addr, err = handler.ipam.ReserveAddressFromPools(pools); err != nil {
			return false, address, err
		}
		if addr.Version == 4 {
			ipamConfig.Set("IPv4Address", utils.NewStringNode(addr.Address.Address))
		} else {
			ipamConfig.Set("IPv6Address", utils.NewStringNode(addr.Address.Address))
		}
		return true, address, nil
	}
	// either ipv4 or ipv6 is non blank
	return false, address, nil
}

func isFixedIPLabelEnabled(containerInfo ContainerInspectResult) bool {
	if value, ok := containerInfo.Config.Labels[FixedIPLabel]; !ok || !isFixedIPEnableByStringValue(value) {
		// don't require fixed ip
		return false
	}
	return true
}

func (handler NetworkConnectHandler) getIPAMConfig(body utils.Object) (utils.Object, error) {
	var (
		endpointConfig utils.Object
		err            error
	)
	if endpointConfig, err = ensureObjectMember(body, "EndpointConfig"); err != nil {
		return endpointConfig, err
	}
	return ensureObjectMember(endpointConfig, "IPAMConfig")
}

func (handler NetworkConnectHandler) writeServerResponse(
	res http.ResponseWriter,
	containerID string,
	allocated bool,
	fixedIPAddress minionsTypes.ReservedAddress,
	clientResp *http.Response,
) {
	defer clientResp.Body.Close()

	var err error
	if clientResp.StatusCode != http.StatusOK {
		log.Errorf("[NetworkConnectHandler.Handle] connect network failed, status code = %d", clientResp.StatusCode)
		if err = utils.Forward(clientResp, res); err != nil {
			log.Errorf("[NetworkConnectHandler.Handle] forward message failed %v", err)
		}
		if allocated {
			handler.ipam.ReleaseReservedAddress(fixedIPAddress)
		}
		return
	}
	if err = utils.Forward(clientResp, res); err != nil {
		log.Errorf("[NetworkConnectHandler.Handle] forward and read message failed %v", err)
	}
	if err := handler.ipam.AddReservedAddressForContainer(containerID, fixedIPAddress); err != nil {
		log.Errorf("[NetworkConnectHandler.Handle] add ReservedAddress(%v) for Container(%s) failed %v",
			fixedIPAddress, containerID, err)
	}
}
