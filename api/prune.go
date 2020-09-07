package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/projecteru2/barrel/handler"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
)

var regexPruneContainers *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/prune(\?.*)?`)

// ContainerPruneHandle .
type ContainerPruneHandle struct {
	sock sock.SocketInterface
	ipam types.ReservedAddressManager
}

// ContainerPruneResult .
type ContainerPruneResult struct {
	ContainersDeleted []string
}

// NewContainerPruneHandle .
func NewContainerPruneHandle(sock sock.SocketInterface, ipam types.ReservedAddressManager) ContainerPruneHandle {
	return ContainerPruneHandle{
		sock: sock,
		ipam: ipam,
	}
}

// Handle .
func (handler ContainerPruneHandle) Handle(ctx handler.Context, response http.ResponseWriter, request *http.Request) {
	if !handler.match(request) {
		ctx.Next()
		return
	}
	log.Debug("[ContainerPruneHandle.Handle] container prune request")

	var (
		resp *http.Response
		err  error
	)
	if resp, err = handler.sock.Request(request); err != nil {
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container prune request to docker socket error",
			},
		); err != nil {
			log.Errorf("[ContainerPruneHandle.Handle] write response failed %v", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var data []byte
		if data, err = utils.ReadAndForward(resp, response); err != nil {
			log.Errorf("[ContainerPruneHandle.Handle] read and forward failed %v", err)
			return
		}
		pruneResult := ContainerPruneResult{}
		if err = json.Unmarshal(data, &pruneResult); err != nil {
			log.Errorf("[ContainerPruneHandle.Handle] json unmarshal failed %v", err)
			return
		}
		size := len(pruneResult.ContainersDeleted)
		log.Infof("[ContainerPruneHandle.Handle] container prune removed %d containers", size)
		if size != 0 {
			go handler.releaseReservedIPs(pruneResult.ContainersDeleted)
		}
		return
	}

	if err = utils.Forward(resp, response); err != nil {
		log.Errorf("[ContainerPruneHandle.Handle] forward error %v", err)
	}
}

func (handler ContainerPruneHandle) match(request *http.Request) bool {
	return request.Method == http.MethodPost && regexPruneContainers.MatchString(request.URL.Path)
}

func (handler ContainerPruneHandle) releaseReservedIPs(containerIDs []string) {
	for _, fullID := range containerIDs {
		log.Debugf("[ContainerPruneHandle.releaseReservedIPs] releasing reserved IP by tied container(%s)", fullID)
		if err := handler.ipam.ReleaseContainerAddresses(fullID); err != nil {
			log.Errorf("[ContainerPruneHandle.releaseReservedIPs] release reserved IP by tied container(%s) error", fullID)
			log.Errorf("[ContainerPruneHandle.releaseReservedIPs] release IP failed %v", err)
		} else {
			log.Infof("[ContainerPruneHandle.releaseReservedIPs] release reserved IP by tied container(%s) success", fullID)
		}
	}
}
