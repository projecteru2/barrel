package docker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/projecteru2/barrel/cni/subhandler"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"

	"github.com/juju/errors"
)

var regexCreateContainer = regexp.MustCompile(`/(.*?)/containers/create(\?.*)?`)

// IPAMConfig .
type IPAMConfig struct {
	IPv4Address string
	IPv6Address string
}

// type containerCreateRequestBody struct {
// 	NetworkingConfig struct {
// 		EndpointsConfig map[string]struct {
// 			IPAMConfig IPAMConfig
// 		}
// 	}
// }

type containerCreateResponseBody struct {
	ID       string `json:"Id"`
	Warnings []string
}

type containerCreateHandler struct {
	utils.LoggerFactory
	client  barrelHttp.Client
	vess    vessel.Helper
	cniBase *subhandler.Base
}

func newContainerCreateHandler(client barrelHttp.Client, vess vessel.Helper, cniBase *subhandler.Base) proxy.RequestHandler {
	return containerCreateHandler{
		LoggerFactory: utils.NewObjectLogger("containerCreateHandler"),
		client:        client,
		vess:          vess,
		cniBase:       cniBase,
	}
}

// Handle .
func (handler containerCreateHandler) Handle(ctx proxy.HandleContext, res http.ResponseWriter, req *http.Request) {
	logger := handler.Logger("Handle")

	if req.Method != http.MethodPost || !regexCreateContainer.MatchString(req.URL.Path) {
		ctx.Next()
		return
	}
	logger.Debug("container create request")
	var (
		err            error
		body           []byte
		bodyObject     utils.Object
		fixedIPAddress []types.IP
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

	if err = handler.adaptRequestForCNI(bodyObject); err != nil {
		writeErrorResponse(res, logger, err, "failed to adapt request for cni")
		return
	}

	if fixedIPAddress, err = handler.checkAndRequestFixedIP(bodyObject); err != nil {
		writeErrorResponse(res, logger, err, "check and request fixed-ip")
		if len(fixedIPAddress) > 0 {
			for _, address := range fixedIPAddress {
				if err := handler.vess.FixedIPAllocator().UnallocFixedIP(context.Background(), address, false); err != nil {
					logger.Errorf("release ip error after checkAndRequestFixedIP failed, cause = %v", err)
				}
			}
		}
		return
	}
	if body, err = utils.Marshal(bodyObject.Any()); err != nil {
		writeErrorResponse(res, logger, err, "marshal server request")
		return
	}
	if clientResp, err = requestDockerd(handler.client, req, body); err != nil {
		writeErrorResponse(res, logger, err, "request dockerd socket")
		return
	}
	handler.writeServerResponse(res, fixedIPAddress, clientResp)
}

func isCustomNetwork(networkMode string) bool {
	return networkMode != "" &&
		networkMode != "default" &&
		networkMode != "bridge" &&
		networkMode != "host" &&
		networkMode != "none" &&
		!strings.HasPrefix(networkMode, "container:")
}

func (handler containerCreateHandler) checkAndRequestFixedIP(body utils.Object) ([]types.IP, error) {
	var (
		fixedIP     bool
		networkMode string
		addresses   []types.IP
		err         error
	)
	if fixedIP, networkMode, err = handler.checkFixedIPLabelAndNetworkMode(body); err != nil || !fixedIP {
		return nil, err
	}
	if !isCustomNetwork(networkMode) {
		return nil, nil
	}
	if addresses, err = handler.visitNetworkConfigAndAllocateAddress(networkMode, body); err != nil {
		return addresses, err
	}
	return addresses, nil
}

func (handler containerCreateHandler) requestFixedIP(
	pools []types.Pool,
	ipamConfig utils.Object,
) (bool, types.IP, error) {
	var (
		ipv4Address string
		ipv6Address string
		err         error
	)
	if ipv4Address, err = getStringMember(ipamConfig, "IPv4Address"); err != nil {
		return false, types.IP{}, err
	}
	if ipv6Address, err = getStringMember(ipamConfig, "IPv6Address"); err != nil {
		return false, types.IP{}, err
	}
	if ipv4Address == "" && ipv6Address == "" {
		var address types.IPAddress
		if address, err = handler.vess.FixedIPAllocator().AllocFixedIPFromPools(context.Background(), pools); err != nil {
			return false, types.IP{}, err
		}
		if address.Version == 4 {
			ipamConfig.Set("IPv4Address", utils.NewStringNode(address.Address))
		} else {
			ipamConfig.Set("IPv6Address", utils.NewStringNode(address.Address))
		}
		return true, address.IP, err
	}
	return false, types.IP{}, nil
}

func (handler containerCreateHandler) checkFixedIPLabelAndNetworkMode(body utils.Object) (bool, string, error) {
	var (
		labels      utils.Object
		hostConfig  utils.Object
		networkMode string
	)
	// check labels
	if iLabels, ok := body.Get("Labels"); !ok || iLabels.Null() {
		// don't have labels so just skip check
		return false, "", nil
	} else if labels, ok = iLabels.ObjectValue(); !ok {
		return false, "", errors.Errorf("parse Labels error, labels=%s", iLabels.String())
	}
	// check fixed ip
	if fixedIPLabel, ok := labels.Get(FixedIPLabel); !ok || !flagEnabled(fixedIPLabel) {
		// don't have fixed ip label so just skip
		return false, "", nil
	}
	if iHostConfig, ok := body.Get("HostConfig"); !ok || iHostConfig.Null() {
		// should not happen, so we delete fixed-ip here
		labels.Del(FixedIPLabel)
		return false, "", nil
	} else if hostConfig, ok = iHostConfig.ObjectValue(); !ok {
		return false, "", errors.Errorf("parse HostConfig error, hostConfig=%s", iHostConfig.String())
	}
	if iNetworkMode, ok := hostConfig.Get("NetworkMode"); !ok || iNetworkMode.Null() {
		// no network mode in HostConfig, remove the label and return
		labels.Del(FixedIPLabel)
		return false, "", nil
	} else if networkMode, ok = iNetworkMode.StringValue(); !ok {
		return false, "", errors.Errorf("parse NetworkMode error, networkMode=%s", iNetworkMode.String())
	}
	return true, networkMode, nil
}

func (handler containerCreateHandler) visitNetworkConfigAndAllocateAddress(
	networkMode string,
	body utils.Object,
) ([]types.IP, error) {
	var (
		networkConfig   utils.Object
		endpointsConfig utils.Object
		err             error
		addresses       []types.IP
	)
	if networkConfig, err = ensureObjectMember(body, "NetworkingConfig"); err != nil {
		return nil, err
	}
	if endpointsConfig, err = ensureObjectMember(networkConfig, "EndpointsConfig"); err != nil {
		return nil, err
	}
	networkNames := endpointsConfig.Keys()
	if len(networkNames) == 0 {
		networkNames = []string{networkMode}
	}
	for _, networkName := range networkNames {
		var (
			pools          []types.Pool
			endpointConfig utils.Object
			ipamConfig     utils.Object
			address        types.IP
			allocated      bool
		)
		if !isCustomNetwork(networkName) {
			continue
		}
		if pools, err = handler.vess.DockerNetworkManager().GetPoolsByNetworkName(context.Background(), networkName); err != nil {
			if err == types.ErrUnsupervisedNetwork {
				continue
			}
			return addresses, err
		}
		if endpointConfig, err = ensureObjectMember(endpointsConfig, networkName); err != nil {
			return addresses, err
		} else if ipamConfig, err = ensureObjectMember(endpointConfig, "IPAMConfig"); err != nil {
			return addresses, err
		}
		if allocated, address, err = handler.requestFixedIP(pools, ipamConfig); err != nil {
			return addresses, err
		} else if allocated {
			addresses = append(addresses, address)
		}
	}
	return addresses, nil
}

func (handler containerCreateHandler) writeServerResponse(
	res http.ResponseWriter,
	fixedIPAddress []types.IP,
	clientResp *http.Response,
) {
	logger := handler.Logger("writeServerResponse")
	defer clientResp.Body.Close()

	var err error
	if clientResp.StatusCode != http.StatusCreated {
		logger.Errorf("create container failed, status code = %d", clientResp.StatusCode)
		if err = utils.Forward(clientResp, res); err != nil {
			logger.Errorf("forward message failed, cause = %v", err)
		}
		for _, address := range fixedIPAddress {
			if err := handler.vess.FixedIPAllocator().UnallocFixedIP(context.Background(), address, false); err != nil {
				logger.Errorf("release reserved address failed, cause = %v", err)
			}
		}
		return
	}
	var content []byte
	if content, err = utils.ForwardAndRead(clientResp, res); err != nil {
		logger.Errorf("forward and read message failed %v", err)
		return
	}
	// unmarshal client response to get container id
	body := containerCreateResponseBody{}
	if err = json.Unmarshal(content, &body); err != nil {
		logger.Error("parse container created resp body failed")
		return
	}
	if body.ID == "" {
		logger.Errorf("create container resp blank container id %v, related address = %v", err, fixedIPAddress)
		return
	}
	if err = handler.vess.InitContainerInfoRecord(
		context.Background(),
		types.Container{ID: body.ID, HostName: handler.vess.Hostname()},
		fixedIPAddress,
	); err != nil {
		logger.Errorf("mark fixed-ip(%s) for container(%s) failed %v", fixedIPAddress, body.ID, err)
	}
}

// steps:
// 0. condition: enable cni && custom network && specific IP not allocated by CNM
// 1. force --net none
// 2. force --runtime barrel-cni
// 3. if Labels[fixed-ip]=1 then --env fixed-ip=1
// 4. if NetworkingConfig.EndpointsConfig.IPAMConfig.IPv4Address=x then --env IPV4=x
// 5. if HostConfig.NetworkMode=x then --env IPPOOL=x
func (handler containerCreateHandler) adaptRequestForCNI(body utils.Object) (err error) {
	var (
		hostConfig  utils.Object
		labels      utils.Object
		env         utils.Array
		networkMode string
		specificIP  string
	)
	logger := handler.Logger("adaptRequestForCNI")

	// prepare hostConfig
	if iHostConfig, ok := body.Get("HostConfig"); !ok || iHostConfig.Null() {
		return nil
	} else if hostConfig, ok = iHostConfig.ObjectValue(); !ok {
		return errors.Errorf("parse HostConfig error, hostConfig=%s", iHostConfig.String())
	}

	// prepare env
	iEnv, ok := body.Get("Env")
	if !ok || iEnv.Null() {
		body.Set("Env", utils.NewArrayNode().Any())
		iEnv, _ = body.Get("Env")
	}
	env, ok = iEnv.ArrayValue()
	if !ok {
		return errors.Errorf("parse Env error, env=%s", iEnv.String())
	}

	iLabels, ok := body.Get("Labels")
	if !ok || iLabels.Null() {
		return
	}
	if labels, ok = iLabels.ObjectValue(); !ok {
		return
	}

	if !handler.cniBase.Enabled() {
		logger.Infof("cni diabled, skip cni mode")
		return
	}

	todo := []func(){}
	defer func() {
		for _, f := range todo {
			f()
		}
	}()

	if iNetworkMode, ok := hostConfig.Get("NetworkMode"); !ok || iNetworkMode.Null() {
		return
	} else if networkMode, ok = iNetworkMode.StringValue(); !ok {
		return errors.Errorf("parse NetworkMode error, networkMode=%s", iNetworkMode.String())
	}
	if !isCustomNetwork(networkMode) {
		logger.Infof("not custom network, skip cni mode")
		return
	}

	todo = append(todo,
		func() {
			logger.Infof("cni mode enabled, set network none, add env IPPOOL=%s, set runtime barrel-cni", networkMode)
			env.Add(utils.NewStringNode("IPPOOL=" + networkMode))
			hostConfig.Set("Runtime", utils.NewStringNode("barrel-cni"))
			hostConfig.Set("NetworkMode", utils.NewStringNode("none"))
		},
	)
	networkConfig, err := ensureObjectMember(body, "NetworkingConfig")
	if err != nil {
		return
	}
	endpointsConfig, err := ensureObjectMember(networkConfig, "EndpointsConfig")
	if err != nil {
		return
	}
	endpointConfig, err := ensureObjectMember(endpointsConfig, networkMode)
	if err != nil {
		return
	}
	if ipamConfig, err := ensureObjectMember(endpointConfig, "IPAMConfig"); err == nil {
		if ipv4Address, ok := ipamConfig.Get("IPv4Address"); ok && !ipv4Address.Null() {
			specificIP, _ = ipv4Address.StringValue()
		}
	}
	todo = append(todo, func() {
		logger.Info("cni mode enabled, empty EndpointConfig")
		networkConfig.Set("EndpointsConfig", utils.NewObjectNode().Any())
	})

	if fixedIPLabel, ok := labels.Get(FixedIPLabel); ok && flagEnabled(fixedIPLabel) {
		todo = append(todo, func() {
			logger.Info("cni fixed-ip mode detected, set fixed-ip env")
			env.Add(utils.NewStringNode(FixedIPLabel + "=1"))
		})
	}

	if specificIP != "" {
		todo = append(todo, func() {
			logger.Infof("cni specific-ip mode detected, set ipv4 env IPV4=%s", specificIP)
			env.Add(utils.NewStringNode("IPV4=" + specificIP))
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := handler.vess.FixedIPAllocator().GetFixedIP(ctx, types.IP{Address: specificIP, PoolID: networkMode}, nil); err != nil {
			if err == types.ErrFixedIPNotAllocated {
				return nil
			}
			return err
		}
		logger.Infof("specific IP allocated by CNM, skip cni mode: %s", specificIP)
		todo = []func(){}
		return nil
	}

	return nil

}
