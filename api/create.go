package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/ipam"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/utils"
	minionsTypes "github.com/projecteru2/minions/types"
	log "github.com/sirupsen/logrus"
)

var regexCreateContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/create(\?.*)?`)

// ContainerCreateHandler .
type ContainerCreateHandler struct {
	sock sock.SocketInterface
	ipam ipam.IPAM
}

// IPAMConfig .
type IPAMConfig struct {
	IPv4Address string
	IPv6Address string
}

// ContainerCreateRequestBody .
type ContainerCreateRequestBody struct {
	NetworkingConfig struct {
		EndpointsConfig map[string]struct {
			IPAMConfig IPAMConfig
		}
	}
}

// ContainerCreateResponseBody .
type ContainerCreateResponseBody struct {
	ID       string `json:"Id"`
	Warnings []string
}

// NewContainerCreateHandler .
func NewContainerCreateHandler(sock sock.SocketInterface, ipam ipam.IPAM) ContainerCreateHandler {
	return ContainerCreateHandler{
		sock: sock,
		ipam: ipam,
	}
}

// Handle .
func (handler ContainerCreateHandler) Handle(ctx utils.HandleContext, res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost || !regexCreateContainer.MatchString(req.URL.Path) {
		ctx.Next()
		return
	}
	log.Debug("[ContainerCreateHandler.Handle] container create request")
	var (
		err            error
		body           []byte
		bodyObject     utils.Object
		fixedIPAddress []minionsTypes.ReservedAddress
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
	if fixedIPAddress, err = handler.checkAndRequestFixedIP(bodyObject); err != nil {
		handler.writeErrorResponse(res, err, "check and request fixed-ip")
		if len(fixedIPAddress) > 0 {
			for _, address := range fixedIPAddress {
				handler.ipam.ReleaseReservedAddress(address)
			}
		}
		return
	}
	if body, err = utils.Marshal(bodyObject.Any()); err != nil {
		handler.writeErrorResponse(res, err, "marshal server request")
		return
	}
	if clientResp, err = requestDockerd(handler.sock, req, body); err != nil {
		handler.writeErrorResponse(res, err, "request dockerd socket")
		return
	}
	handler.writeServerResponse(res, fixedIPAddress, clientResp)
}

func (handler ContainerCreateHandler) writeErrorResponse(res http.ResponseWriter, err error, label string) {
	log.Errorf("[ContainerCreateHandler.Handle] %s failed %v", label, err)
	if err := utils.WriteBadGateWayResponse(
		res,
		utils.HTTPSimpleMessageResponseBody{
			Message: label + " error",
		},
	); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] write %s error response failed %v", label, err)
	}
}

func isCustomNetwork(networkMode string) bool {
	return networkMode != "" &&
		networkMode != "default" &&
		networkMode != "bridge" &&
		networkMode != "host" &&
		networkMode != "none" &&
		!strings.HasPrefix(networkMode, "container:")
}

func (handler ContainerCreateHandler) checkAndRequestFixedIP(body utils.Object) ([]minionsTypes.ReservedAddress, error) {
	var (
		fixedIP     bool
		networkMode string
		addresses   []minionsTypes.ReservedAddress
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

func (handler ContainerCreateHandler) requestFixedIP(
	pools []*minionsTypes.Pool,
	ipamConfig utils.Object,
) (bool, minionsTypes.ReservedAddress, error) {
	var (
		ipv4Address string
		ipv6Address string
		err         error
	)
	if ipv4Address, err = getStringMember(ipamConfig, "IPv4Address"); err != nil {
		return false, minionsTypes.ReservedAddress{}, err
	}
	if ipv6Address, err = getStringMember(ipamConfig, "IPv6Address"); err != nil {
		return false, minionsTypes.ReservedAddress{}, err
	}
	if ipv4Address == "" && ipv6Address == "" {
		var address common.ReservedAddress
		if address, err = handler.ipam.ReserveAddressFromPools(pools); err != nil {
			return false, minionsTypes.ReservedAddress{}, err
		}
		if address.Version == 4 {
			ipamConfig.Set("IPv4Address", utils.NewStringNode(address.Address.Address))
		} else {
			ipamConfig.Set("IPv6Address", utils.NewStringNode(address.Address.Address))
		}
		return true, address.Address, err
	}
	return false, minionsTypes.ReservedAddress{}, nil
}

func (handler ContainerCreateHandler) checkFixedIPLabelAndNetworkMode(body utils.Object) (bool, string, error) {
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
	if fixedIPLabel, ok := labels.Get(FixedIPLabel); !ok || !isFixedIPEnable(fixedIPLabel) {
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

func (handler ContainerCreateHandler) visitNetworkConfigAndAllocateAddress(
	networkMode string,
	body utils.Object,
) ([]minionsTypes.ReservedAddress, error) {
	var (
		networkConfig   utils.Object
		endpointsConfig utils.Object
		err             error
		addresses       []minionsTypes.ReservedAddress
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
			pools          []*minionsTypes.Pool
			endpointConfig utils.Object
			ipamConfig     utils.Object
			address        minionsTypes.ReservedAddress
			allocated      bool
		)
		if !isCustomNetwork(networkName) {
			continue
		}
		if pools, err = handler.ipam.GetIPPoolsByNetworkName(networkName); err != nil {
			if err == ipam.ErrUnsupervisedNetwork {
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

func (handler ContainerCreateHandler) writeServerResponse(
	res http.ResponseWriter,
	fixedIPAddress []minionsTypes.ReservedAddress,
	clientResp *http.Response,
) {
	defer clientResp.Body.Close()

	var err error
	if clientResp.StatusCode != http.StatusCreated {
		log.Errorf("[ContainerCreateHandler.Handle] create container failed, status code = %d", clientResp.StatusCode)
		if err = utils.Forward(clientResp, res); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] forward message failed %v", err)
		}
		for _, address := range fixedIPAddress {
			handler.ipam.ReleaseReservedAddress(address)
		}
		return
	}
	var content []byte
	if content, err = utils.ForwardAndRead(clientResp, res); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] forward and read message failed %v", err)
		return
	}
	// unmarshal client response to get container id
	body := ContainerCreateResponseBody{}
	if err = json.Unmarshal(content, &body); err != nil {
		log.Error("[ContainerCreateHandler.Handle] parse container created resp body failed")
		return
	}
	if body.ID == "" {
		log.Errorf("[ContainerCreateHandler.Handle] create container resp blank container id %v, related address = %v", err, fixedIPAddress)
		return
	}
	if err = handler.ipam.InitContainerInfoRecord(
		minionsTypes.ContainerInfo{ID: body.ID, Addresses: fixedIPAddress},
	); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] mark fixed-ip(%s) for container(%s) failed %v", fixedIPAddress, body.ID, err)
	}
}
