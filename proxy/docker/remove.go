package docker

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/cni/subhandler"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/resources"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"
	log "github.com/sirupsen/logrus"
)

const (
	labelVolumeAutoResource = "volume-auto-res"
)

var regexDeleteContainer = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

const (
	// ResourceShared .
	ResourceShared string = "shared"
	// ResourceUnique .
	ResourceUnique string = "unique"
	// ResourceBorrowed .
	ResourceBorrowed string = "borrowed"
)

type containerDeleteHandler struct {
	utils.LoggerFactory
	inspectAgent containerInspectAgent
	client       barrelHttp.Client
	cniBase      *subhandler.Base
	vessel.Helper
}

func newContainerDeleteHandler(client barrelHttp.Client, vess vessel.Helper, inspectAgent containerInspectAgent, cniBase *subhandler.Base) proxy.RequestHandler {
	return containerDeleteHandler{
		LoggerFactory: utils.NewObjectLogger("containerDeleteHandler"),
		client:        client,
		Helper:        vess,
		cniBase:       cniBase,
		inspectAgent:  inspectAgent,
	}
}

type containerDeleteRequest struct {
	version       string
	identifier    string
	removeVolumes bool
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

	removeVolumes := containerDeleteRequest.removeVolumes && shouldRemoveVolumes(containerInfo.Config.Labels, true)
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		go handler.releaseResources(containerInfo, removeVolumes)
	}

	if err = utils.Forward(resp, response); err != nil {
		logger.Errorf("forward failed %v", err)
	}
}

func (handler containerDeleteHandler) releaseResources(containerInfo containerInspectResult, removeVolumens bool) {
	if handler.isFixedIPCNIContainer(containerInfo) {
		if err := handler.releaseCNIResources(containerInfo.ID); err != nil {
			log.Errorf("release CNI resources error: %s, %+v", containerInfo.ID, err)
		}
	} else {
		handler.releaseReservedIP(containerInfo.ID)
	}
	if removeVolumens {
		handler.releaseMounts(containerInfo)
	}
}

func (handler containerDeleteHandler) isFixedIPCNIContainer(containerInfo containerInspectResult) bool {
	if containerInfo.HostConfig.Runtime != "barrel-cni" {
		return false
	}

	for _, e := range containerInfo.Config.Env {
		if e == "fixed-ip=1" {
			return true
		}
	}
	return false
}

func (handler containerDeleteHandler) releaseCNIResources(id string) (err error) {
	return handler.cniBase.RemoveNetwork(id)
}

func (handler containerDeleteHandler) releaseMounts(containerInfo containerInspectResult) {
	var paths []string
	for _, mnt := range containerInfo.Mounts {
		if mnt.Source != "" {
			paths = append(paths, mnt.Source)
		}
	}
	resources.RecycleMounts(paths)
}

func (handler containerDeleteHandler) releaseReservedIP(id string) {
	logger := handler.Logger("releaseReservedIP")

	if id == "" {
		logger.Error("can't release container, id is empty")
	}
	logger.Infof("release reserved IP by id(%s)", id)
	handler.releaseReservedIPByTiedContainerIDIfIdle(id)
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
			req.removeVolumes = parseBoolFromQuery(request, "v", false)

			handler.Logger("match").Debugf("docker api version = %s", req.version)
			return req, true
		}
	}
	return req, false
}

func parseBoolFromQuery(request *http.Request, key string, defVal bool) bool {
	if values, ok := request.URL.Query()[key]; ok {
		if removeVolumes, err := strconv.ParseBool(values[0]); err == nil {
			return removeVolumes
		}
	}
	return defVal
}

func shouldRemoveVolumes(labels map[string]string, defVal bool) bool {
	if labels == nil {
		return defVal
	}

	val, ok := labels[labelVolumeAutoResource]
	if !ok {
		return defVal
	}

	switch val {
	case ResourceShared:
		// will introduce a reference counting later
		return false
	case ResourceUnique:
		return true
	case ResourceBorrowed:
		return false
	default:
		return defVal
	}
}
