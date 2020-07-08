package container

import (
	"net/http"
	"regexp"

	"github.com/projecteru2/barrel/internal/sock"
	"github.com/projecteru2/barrel/internal/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

var regexDeleteContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

type ContainerDeleteHandler struct {
	inspectHandler ContainerInspectHandler
	dockerSocket   sock.DockerSocket
	minionsClient  minions.Client
}

type ContainerDeleteRequest struct {
	Version  string
	IDOrName string
}

func NewContainerDeleteHandler(dockerSocket sock.DockerSocket, minionsClient minions.Client) ContainerDeleteHandler {
	return ContainerDeleteHandler{
		dockerSocket:   dockerSocket,
		minionsClient:  minionsClient,
		inspectHandler: ContainerInspectHandler{dockerSocket: dockerSocket},
	}
}

func (handler ContainerDeleteHandler) Handle(response http.ResponseWriter, request *http.Request) (handled bool) {
	var (
		containerDeleteRequest ContainerDeleteRequest
	)
	if containerDeleteRequest, handled = handler.match(request); !handled {
		return
	}
	log.Info("handle container remove request")

	var (
		err    error
		fullID string
	)

	if fullID, err = handler.inspectHandler.GetFullContainerID(
		containerDeleteRequest.IDOrName,
		containerDeleteRequest.Version,
	); err != nil {
		log.Errorln(err)
		_, containerNotExists := err.(ContainerNotExistsError)
		if containerNotExists {
			writeServerResponse(response, http.StatusNotFound, err.Error())
		} else {
			writeServerResponse(response, http.StatusInternalServerError, "inspect container before remove error")
		}
		return
	}

	var resp *http.Response
	if resp, err = handler.dockerSocket.Request(request); err != nil {
		log.Errorln(err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container remove request to docker socket error",
			},
		); err != nil {
			log.Errorln("write response error", err)
		}
		return
	}

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		go handler.releaseReservedIP(containerDeleteRequest.IDOrName, fullID)
	}

	if err = utils.Forward(resp, response); err != nil {
		log.Errorln(err)
	}
	return
}

func writeServerResponse(response http.ResponseWriter, statusCode int, message string) {
	if err := utils.WriteHTTPJSONResponse(
		response,
		statusCode,
		nil,
		utils.HTTPSimpleMessageResponseBody{
			Message: message,
		},
	); err != nil {
		log.Errorln("write response error", err)
	}
}

func (handler ContainerDeleteHandler) releaseReservedIP(idOrName string, fullID string) {
	if fullID != "" {
		log.Infof("release reserved IP by fullID(%s)", fullID)
		handler.releaseReservedIPByTiedContainerIDIfIdle(fullID)
		return
	}
	if idOrName != "" && len(idOrName) == 64 {
		log.Infof("release reserved IP by idOrName(%s) as fullID", idOrName)
		handler.releaseReservedIPByTiedContainerIDIfIdle(idOrName)
		return
	}
	log.Errorf("can't release container(%s) by id prefix or name", idOrName)
}

func (handler ContainerDeleteHandler) releaseReservedIPByTiedContainerIDIfIdle(fullID string) {
	if err := handler.minionsClient.ReleaseReservedIPByTiedContainerIDIfIdle(fullID); err != nil {
		log.Errorf("release reserved IP by tied container(%s) error\n", fullID)
		log.Errorln(err)
	}
}

func (handler ContainerDeleteHandler) match(request *http.Request) (ContainerDeleteRequest, bool) {
	req := ContainerDeleteRequest{}
	if request.Method == http.MethodDelete {
		subMatches := regexDeleteContainer.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.Version = subMatches[1]
			req.IDOrName = subMatches[2]
			log.Printf("docker api version = %s\n", req.Version)
			return req, true
		}
	}
	return req, false
}
