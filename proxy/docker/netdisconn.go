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

var regexNetworkDisconnect = regexp.MustCompile(`/(.*?)/networks/([a-zA-Z0-9][a-zA-Z0-9_.-]*)/disconnect(\?.*)?`)

type networkDisconnectHandler struct {
	utils.LoggerFactory
	inspectAgent containerInspectAgent
	client       barrelHttp.Client
	vessel.Helper
}

func newNetworkDisconnectHandler(client barrelHttp.Client, vess vessel.Helper, inspectAgent containerInspectAgent) proxy.RequestHandler {
	return networkDisconnectHandler{
		LoggerFactory: utils.NewObjectLogger("networkDisconnectHandler"),
		client:        client,
		Helper:        vess,
		inspectAgent:  inspectAgent,
	}
}

type networkDisconnectRequest struct {
	networkIdentifier string
	version           string
}

// Handle .
func (handler networkDisconnectHandler) Handle(ctx proxy.HandleContext, res http.ResponseWriter, req *http.Request) {
	logger := handler.Logger("Handle")

	var (
		networkDisconnectRequest networkDisconnectRequest
		matched                  bool
	)
	if networkDisconnectRequest, matched = handler.match(req); !matched {
		ctx.Next()
		return
	}
	logger.Debug("container remove request")

	var (
		pools []types.Pool
		err   error
	)
	if pools, err = handler.DockerNetworkManager().GetPoolsByNetworkName(
		context.Background(),
		networkDisconnectRequest.networkIdentifier,
	); err != nil {
		if err == types.ErrUnsupervisedNetwork {
			ctx.Next()
			return
		}
		writeErrorResponse(res, logger, err, "GetIPPoolsByNetworkName")
		return
	}

	var (
		body          []byte
		bodyObject    utils.Object
		containerInfo containerInspectResult
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
			return handler.inspectAgent.Inspect(identifier, networkDisconnectRequest.version)
		},
		bodyObject,
	); err != nil {
		writeErrorResponse(res, logger, err, "get container info")
		return
	}
	var resp *http.Response
	if resp, err = requestDockerd(handler.client, req, body); err != nil {
		writeErrorResponse(res, logger, err, "request dockerd socket")
		return
	}
	defer resp.Body.Close()

	if isFixedIPLabelEnabled(containerInfo) {
		if resp.StatusCode == http.StatusOK {
			go handler.releaseReservedAddresses(containerInfo.ID, pools)
		}
	}
	if err = utils.Forward(resp, res); err != nil {
		logger.Errorf("forward failed %v", err)
	}
}

func (handler networkDisconnectHandler) getContainerInfo(
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

func (handler networkDisconnectHandler) releaseReservedAddresses(containerID string, pools []types.Pool) {
	logger := handler.Logger("releaseReservedAddresses")

	if err := handler.ReleaseContainerAddressesByIPPools(context.Background(), containerID, pools); err != nil {
		logger.Errorf(
			"release container(%s) reserved address error, %v",
			containerID,
			err,
		)
	}
}

func (handler networkDisconnectHandler) match(request *http.Request) (networkDisconnectRequest, bool) {
	logger := handler.Logger("match")

	req := networkDisconnectRequest{}
	if request.Method == http.MethodPost {
		subMatches := regexNetworkDisconnect.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.version = subMatches[1]
			req.networkIdentifier = subMatches[2]
			logger.Debugf("docker api version = %s", req.version)
			return req, true
		}
	}
	return req, false
}
