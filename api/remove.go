package api

import (
	"net/http"
	"regexp"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/ipam"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
)

var regexDeleteContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/([a-zA-Z0-9][a-zA-Z0-9_.-]*)(\?.*)?`)

// ContainerDeleteHandler .
type ContainerDeleteHandler struct {
	inspectHandler ContainerInspectHandler
	sock           sock.SocketInterface
	ipam           ipam.IPAM
}

type containerDeleteRequest struct {
	version    string
	identifier string
}

// NewContainerDeleteHandler .
func NewContainerDeleteHandler(sock sock.SocketInterface, ipam ipam.IPAM) ContainerDeleteHandler {
	return ContainerDeleteHandler{
		sock:           sock,
		ipam:           ipam,
		inspectHandler: ContainerInspectHandler{sock: sock},
	}
}

// Handle .
func (handler ContainerDeleteHandler) Handle(ctx utils.HandleContext, response http.ResponseWriter, request *http.Request) {
	var (
		containerDeleteRequest containerDeleteRequest
		matched                bool
	)
	if containerDeleteRequest, matched = handler.match(request); !matched {
		ctx.Next()
		return
	}
	log.Debug("[ContainerDeleteHandler.Handle] container remove request")

	var (
		err           error
		containerInfo ContainerInspectResult
	)

	if containerInfo, err = handler.inspectHandler.Inspect(
		containerDeleteRequest.identifier,
		containerDeleteRequest.version,
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
		go handler.releaseReservedIP(containerDeleteRequest.identifier, containerInfo.ID)
	}

	if err = utils.Forward(resp, response); err != nil {
		log.Errorf("[ContainerDeleteHandler.Handle] forward failed %v", err)
	}
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
	if err := handler.ipam.ReleaseContainer(fullID); err != nil {
		log.Errorf("[ContainerDeleteHandler.releaseReservedIPByTiedContainerIDIfIdle] release reserved IP by tied container(%s) error", fullID)
		log.Errorf("[ContainerDeleteHandler.releaseReservedIPByTiedContainerIDIfIdle] release ip failed %v", err)
	}
}

func (handler ContainerDeleteHandler) match(request *http.Request) (containerDeleteRequest, bool) {
	req := containerDeleteRequest{}
	if request.Method == http.MethodDelete {
		subMatches := regexDeleteContainer.FindStringSubmatch(request.URL.Path)
		if len(subMatches) > 2 {
			req.version = subMatches[1]
			req.identifier = subMatches[2]
			log.Debugf("[ContainerDeleteHandler.match] docker api version = %s", req.version)
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
