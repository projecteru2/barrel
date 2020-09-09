package api

import (
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/handler"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
)

var regexNetworkDisconnect *regexp.Regexp = regexp.MustCompile(`/(.*?)/networks/([a-zA-Z0-9][a-zA-Z0-9_.-]*)/disconnect(\?.*)?`)

// NetworkDisconnectHandler .
type NetworkDisconnectHandler struct {
	inspectHandler ContainerInspectHandler
	sock           sock.SocketInterface
	ipam           driver.ReservedAddressManager
}

type networkDisconnectRequest struct {
	networkIdentifier string
	version           string
}

// NewNetworkDisconnectHandler .
func NewNetworkDisconnectHandler(sock sock.SocketInterface, ipam driver.ReservedAddressManager) NetworkDisconnectHandler {
	return NetworkDisconnectHandler{
		sock:           sock,
		ipam:           ipam,
		inspectHandler: ContainerInspectHandler{sock: sock},
	}
}

// Handle .
func (handler NetworkDisconnectHandler) Handle(ctx handler.Context, res http.ResponseWriter, req *http.Request) {
	var (
		networkDisconnectRequest networkDisconnectRequest
		matched                  bool
	)
	if networkDisconnectRequest, matched = handler.match(req); !matched {
		ctx.Next()
		return
	}
	log.Debug("[ContainerDeleteHandler.Handle] container remove request")

	var (
		pools []types.Pool
		err   error
	)
	if pools, err = handler.ipam.GetIPPoolsByNetworkName(
		networkDisconnectRequest.networkIdentifier,
	); err != nil {
		if err == types.ErrUnsupervisedNetwork {
			ctx.Next()
			return
		}
		handler.writeErrorResponse(res, err, "GetIPPoolsByNetworkName")
		return
	}

	var (
		body          []byte
		bodyObject    utils.Object
		containerInfo ContainerInspectResult
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
			return handler.inspectHandler.Inspect(identifier, networkDisconnectRequest.version)
		},
		bodyObject,
	); err != nil {
		handler.writeErrorResponse(res, err, "get container info")
		return
	}
	var resp *http.Response
	if resp, err = requestDockerd(handler.sock, req, body); err != nil {
		handler.writeErrorResponse(res, err, "request dockerd socket")
		return
	}
	defer resp.Body.Close()

	if isFixedIPLabelEnabled(containerInfo) {
		if resp.StatusCode == http.StatusOK {
			go handler.releaseReservedAddresses(containerInfo.ID, pools)
		}
	}
	if err = utils.Forward(resp, res); err != nil {
		log.Errorf("[ContainerDeleteHandler.Handle] forward failed %v", err)
	}
}

func (handler NetworkDisconnectHandler) writeErrorResponse(res http.ResponseWriter, err error, label string) {
	log.Errorf("[NetworkDisconnectHandler::Handle] %s failed %v", label, err)
	if err := utils.WriteBadGateWayResponse(
		res,
		utils.HTTPSimpleMessageResponseBody{
			Message: label + " error",
		},
	); err != nil {
		log.Errorf("[NetworkDisconnectHandler.Handle] write %s error response failed %v", label, err)
	}
}

func (handler NetworkDisconnectHandler) getContainerInfo(
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

func (handler NetworkDisconnectHandler) releaseReservedAddresses(containerID string, pools []types.Pool) {
	if err := handler.ipam.ReleaseContainerAddressesByIPPools(containerID, pools); err != nil {
		log.Errorf(
			"[NetworkDisconnectHandler::releaseReservedAddresses] release container(%s) reserved address error, %v",
			containerID,
			err,
		)
	}
}

func (handler NetworkDisconnectHandler) match(request *http.Request) (networkDisconnectRequest, bool) {
	req := networkDisconnectRequest{}
	if request.Method == http.MethodPost {
		subMatches := regexNetworkDisconnect.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.version = subMatches[1]
			req.networkIdentifier = subMatches[2]
			log.Debugf("[ContainerDeleteHandler.match] docker api version = %s", req.version)
			return req, true
		}
	}
	return req, false
}
