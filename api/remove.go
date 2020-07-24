package api

import (
	"net/http"
	"regexp"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

var regexDeleteContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

// ContainerDeleteHandler .
type ContainerDeleteHandler struct {
	inspectHandler ContainerInspectHandler
	sock           sock.SocketInterface
	minionsClient  minions.Client
}

// ContainerDeleteRequest .
type ContainerDeleteRequest struct {
	Version  string
	IDOrName string
}

// NewContainerDeleteHandler .
func NewContainerDeleteHandler(sock sock.SocketInterface, minionsClient minions.Client) ContainerDeleteHandler {
	return ContainerDeleteHandler{
		sock:           sock,
		minionsClient:  minionsClient,
		inspectHandler: ContainerInspectHandler{sock: sock},
	}
}

// Handle .
func (handler ContainerDeleteHandler) Handle(response http.ResponseWriter, request *http.Request) (handled bool) {
	var (
		containerDeleteRequest ContainerDeleteRequest
	)
	if containerDeleteRequest, handled = handler.match(request); !handled {
		return
	}
	log.Debug("[ContainerDeleteHandler.Handle] container remove request")

	var (
		err    error
		fullID string
	)

	if fullID, err = handler.inspectHandler.GetFullContainerID(
		containerDeleteRequest.IDOrName,
		containerDeleteRequest.Version,
	); err != nil {
		log.Errorf("[ContainerDeleteHandler.Handle] get full container id failed %v", err)
		if rootErr := errors.Cause(err); rootErr == common.ErrContainerNotExists {
			writeServerResponse(response, http.StatusNotFound, err.Error())
		} else {
			writeServerResponse(response, http.StatusInternalServerError, "inspect container before remove error")
		}
		return
	}

	var resp *http.Response
	if resp, err = handler.sock.Request(request); err != nil {
		log.Errorf("[ContainerDeleteHandler.Handle] request failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container remove request to docker socket error",
			},
		); err != nil {
			log.Errorf("[ContainerDeleteHandler.Handle] write response failed %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		go handler.releaseReservedIP(containerDeleteRequest.IDOrName, fullID)
	}

	if err = utils.Forward(resp, response); err != nil {
		log.Errorf("[ContainerDeleteHandler.Handle] forward failed %v", err)
	}
	return handled
}

func (handler ContainerDeleteHandler) releaseReservedIP(idOrName string, fullID string) {
	if fullID != "" {
		log.Infof("[ContainerDeleteHandler.releaseReservedIP] release reserved IP by fullID(%s)", fullID)
		handler.releaseReservedIPByTiedContainerIDIfIdle(fullID)
		return
	}
	if idOrName != "" && len(idOrName) == 64 {
		log.Infof("[ContainerDeleteHandler.releaseReservedIP] release reserved IP by idOrName(%s) as fullID", idOrName)
		handler.releaseReservedIPByTiedContainerIDIfIdle(idOrName)
		return
	}
	log.Errorf("[ContainerDeleteHandler.releaseReservedIP] can't release container(%s) by id prefix or name", idOrName)
}

func (handler ContainerDeleteHandler) releaseReservedIPByTiedContainerIDIfIdle(fullID string) {
	if err := handler.minionsClient.ReleaseReservedIPByTiedContainerIDIfIdle(fullID); err != nil {
		log.Errorf("[ContainerDeleteHandler.releaseReservedIPByTiedContainerIDIfIdle] release reserved IP by tied container(%s) error", fullID)
		log.Errorf("[ContainerDeleteHandler.releaseReservedIPByTiedContainerIDIfIdle] release ip failed %v", err)
	}
}

func (handler ContainerDeleteHandler) match(request *http.Request) (ContainerDeleteRequest, bool) {
	req := ContainerDeleteRequest{}
	if request.Method == http.MethodDelete {
		subMatches := regexDeleteContainer.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.Version = subMatches[1]
			req.IDOrName = subMatches[2]
			log.Debugf("[ContainerDeleteHandler.match] docker api version = %s", req.Version)
			return req, true
		}
	}
	return req, false
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
		log.Errorf("[writeServerResponse] write response failed %v", err)
	}
}
