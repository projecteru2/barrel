package docker

import (
	"context"
	"net/http"
	"regexp"

	"github.com/juju/errors"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"
)

var regexDeleteContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

type containerDeleteHandler struct {
	utils.LoggerFactory
	inspectAgent containerInspectAgent
	client       barrelHttp.Client
	vessel.Helper
}

func newContainerDeleteHandler(client barrelHttp.Client, vess vessel.Helper, inspectAgent containerInspectAgent) proxy.RequestHandler {
	return containerDeleteHandler{
		LoggerFactory: utils.NewObjectLogger("containerDeleteHandler"),
		client:        client,
		Helper:        vess,
		inspectAgent:  inspectAgent,
	}
}

type containerDeleteRequest struct {
	version    string
	identifier string
}

// Handle .
func (handler containerDeleteHandler) Handle(ctx proxy.HandleContext, response http.ResponseWriter, request *http.Request) {
	logger := handler.Logger("Handle")

	var (
		containerDeleteRequest containerDeleteRequest
		matched                bool
	)
	if containerDeleteRequest, matched = handler.match(request); !matched {
		ctx.Next()
		return
	}
	logger.Debug("container remove request")

	var (
		err           error
		containerInfo containerInspectResult
	)

	if containerInfo, err = handler.inspectAgent.Inspect(
		containerDeleteRequest.identifier,
		containerDeleteRequest.version,
	); err != nil {
		logger.Errorf("get full container id failed %v", err)
		if rootErr := errors.Cause(err); rootErr == types.ErrContainerNotExists {
			writeServerResponse(response, logger, http.StatusNotFound, err.Error())
		} else {
			writeServerResponse(response, logger, http.StatusInternalServerError, "inspect container before remove error")
		}
		return
	}

	var resp *http.Response
	if resp, err = handler.client.Request(request); err != nil {
		logger.Errorf("request failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container remove request to docker socket error",
			},
		); err != nil {
			logger.Errorf("write response failed %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		go handler.releaseReservedIP(containerDeleteRequest.identifier, containerInfo.ID)
	}

	if err = utils.Forward(resp, response); err != nil {
		logger.Errorf("forward failed %v", err)
	}
}

func (handler containerDeleteHandler) releaseReservedIP(idOrName string, fullID string) {
	logger := handler.Logger("releaseReservedIP")

	if fullID != "" {
		logger.Infof("release reserved IP by fullID(%s)", fullID)
		handler.releaseReservedIPByTiedContainerIDIfIdle(fullID)
		return
	}
	if idOrName != "" && len(idOrName) == 64 {
		logger.Infof("release reserved IP by idOrName(%s) as fullID", idOrName)
		handler.releaseReservedIPByTiedContainerIDIfIdle(idOrName)
		return
	}
	logger.Errorf("can't release container(%s) by id prefix or name", idOrName)
}

func (handler containerDeleteHandler) releaseReservedIPByTiedContainerIDIfIdle(fullID string) {
	logger := handler.Logger("releaseReservedIPByTiedContainerIDIfIdle")

	if err := handler.ReleaseContainerAddresses(context.Background(), fullID); err != nil {
		logger.Errorf("release reserved IP by tied container(%s) error", fullID)
		logger.Errorf("release ip failed %v", err)
	}
}

func (handler containerDeleteHandler) match(request *http.Request) (containerDeleteRequest, bool) {
	req := containerDeleteRequest{}
	if request.Method == http.MethodDelete {
		subMatches := regexDeleteContainer.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.version = subMatches[1]
			req.identifier = subMatches[2]
			handler.Logger("match").Debugf("docker api version = %s", req.version)
			return req, true
		}
	}
	return req, false
}
