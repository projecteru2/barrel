package docker

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/juju/errors"

	dockerContainer "github.com/projecteru2/barrel/docker/container"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/utils/log"
	"github.com/projecteru2/barrel/vessel"
)

var regexDeleteContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

type containerDeleteHandler struct {
	client         barrelHttp.Client
	res            vessel.ResourceManager
	recycleTimeout time.Duration
}

type containerDeleteRequest struct {
	version        string
	identifier     string
	removeVolumens bool
}

func newContainerDeleteHandler(client barrelHttp.Client, res vessel.ResourceManager, recycleTimeout time.Duration) proxy.RequestHandler {
	return containerDeleteHandler{
		client:         client,
		res:            res,
		recycleTimeout: recycleTimeout,
	}
}

// Handle .
func (handler containerDeleteHandler) Handle(ctx proxy.HandleContext, response http.ResponseWriter, request *http.Request) {
	var (
		containerDeleteRequest containerDeleteRequest
		matched                bool
	)
	if containerDeleteRequest, matched = handler.match(request); !matched {
		ctx.Next()
		return
	}
	log.Debug("container remove request")

	var (
		err     error
		cont    *dockerContainer.Container
		inspect = dockerContainer.InspectContainer{
			Client:              handler.client,
			ContainerIdentifier: containerDeleteRequest.identifier,
			APIVersion:          containerDeleteRequest.version,
		}
	)

	if cont, err = inspect.Inspect(); err != nil {
		log.WithError(err).Error("Get container info error")
		logEntry := log.WithCaller()
		writeServerResponse := func(response http.ResponseWriter, statusCode int, message string) {
			if err := utils.WriteHTTPJSONResponse(
				response,
				statusCode,
				nil,
				utils.HTTPSimpleMessageResponseBody{
					Message: message,
				},
			); err != nil {
				logEntry.WithError(err).Error("Handle")
			}
		}
		if rootErr := errors.Cause(err); rootErr == types.ErrContainerNotExists {
			writeServerResponse(response, http.StatusNotFound, err.Error())
		} else {
			writeServerResponse(response, http.StatusInternalServerError, "inspect container before remove error")
		}
		return
	}

	del := containerDelete{
		containerUtil: containerUtil{
			c:        cont,
			client:   handler.client,
			servResp: response,
			servReq:  request,
			res:      handler.res,
		},
		removeVolumens: containerDeleteRequest.removeVolumens,
		recycleTimeout: handler.recycleTimeout,
	}
	if err = del.Delete(); err != nil {
		log.WithError(err).Error("Delete container error")
	}
}

func (handler containerDeleteHandler) match(request *http.Request) (containerDeleteRequest, bool) {
	req := containerDeleteRequest{}
	if request.Method == http.MethodDelete {
		subMatches := regexDeleteContainer.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.version = subMatches[1]
			req.identifier = subMatches[2]
			req.removeVolumens = parseBoolFromQuery(request, "v", false)
			log.WithCaller().WithField("DockerAPIVersion", req.version).Debug("MatchDelete")
			return req, true
		}
	}
	return req, false
}

func parseBoolFromQuery(request *http.Request, key string, defVal bool) bool {
	if values, ok := request.URL.Query()[key]; ok {
		if removeVolumens, err := strconv.ParseBool(values[0]); err == nil {
			return removeVolumens
		}
	}
	return defVal
}

type containerDelete struct {
	containerUtil
	removeVolumens bool
	recycleTimeout time.Duration
}

func (c *containerDelete) Delete() (err error) {
	return c.operate("delete", func(code int, status string) {
		if code == http.StatusNoContent {
			go c.releaseResources()
		}
	})
}

func (c *containerDelete) releaseResources() {
	ctx, cancel := context.WithTimeout(context.Background(), c.recycleTimeout)
	defer cancel()
	for _, res := range c.Resources(ctx, c.removeVolumens) {
		if err := res.Recycle(ctx); err != nil {
			log.WithError(err).Error("Recycle resource error")
		}
	}
}
