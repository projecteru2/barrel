package container

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/projecteru2/barrel/internal/sock"
	"github.com/projecteru2/barrel/internal/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

type ContainerPruneHandle struct {
	dockerSocket  sock.DockerSocket
	minionsClient minions.Client
	netUtil       utils.NetUtil
}

type ContainerPruneResult struct {
	ContainersDeleted []string
}

var regexPruneContainers *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/prune(\?.*)?`)

func NewContainerPruneHandle(dockerSocket sock.DockerSocket, minionsClient minions.Client, netUtil utils.NetUtil) ContainerPruneHandle {
	return ContainerPruneHandle{
		netUtil:       netUtil,
		dockerSocket:  dockerSocket,
		minionsClient: minionsClient,
	}
}

func (handler ContainerPruneHandle) Handle(response http.ResponseWriter, request *http.Request) (handled bool) {
	if !handler.match(request) {
		return
	}
	handled = true

	var (
		resp *http.Response
		err  error
	)
	if resp, err = handler.dockerSocket.Request(request); err != nil {
		if err := utils.WriteHTTPInternalServerErrorResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container prune request to docker socket error",
			},
		); err != nil {
			log.Errorln("write response error", err)
		}
	}

	if resp.StatusCode == 200 {
		var data []byte
		if data, err = handler.netUtil.ReadAndForward(resp, response); err != nil {
			log.Errorln(err)
			return
		}
		pruneResult := ContainerPruneResult{}
		if err = json.Unmarshal(data, &pruneResult); err != nil {
			log.Errorln(err)
			return
		}
		size := len(pruneResult.ContainersDeleted)
		log.Infof("container prune removed %d containers", size)
		if size != 0 {
			go handler.releaseReservedIPs(pruneResult.ContainersDeleted)
		}
		return
	}

	if err = handler.netUtil.Forward(resp, response); err != nil {
		log.Errorln(err)
	}
	return
}

func (handler ContainerPruneHandle) match(request *http.Request) bool {
	return request.Method == http.MethodPost && regexPruneContainers.MatchString(request.URL.Path)
}

func (handler ContainerPruneHandle) releaseReservedIPs(containerIDs []string) {
	for _, fullID := range containerIDs {
		log.Infof("start releasing reserved IP by tied container(%s)", fullID)
		if err := handler.minionsClient.ReleaseReservedIPByTiedContainerIDIfIdle(fullID); err != nil {
			log.Errorf("release reserved IP by tied container(%s) error", fullID)
			log.Errorln(err)
		} else {
			log.Infof("release reserved IP by tied container(%s) success", fullID)
		}
	}
}
