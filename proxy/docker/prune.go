package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel"
)

var regexPruneContainers *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/prune(\?.*)?`)

type containerPruneHandle struct {
	client barrelHttp.Client
	vessel.Helper
}

type containerPruneResult struct {
	utils.LoggerFactory
	ContainersDeleted []string
}

// Handle .
func (handler containerPruneHandle) Handle(ctx proxy.HandleContext, response http.ResponseWriter, request *http.Request) {
	logger := handler.Logger("Handle")

	if !handler.match(request) {
		ctx.Next()
		return
	}
	logger.Debug("container prune request")

	var (
		resp *http.Response
		err  error
	)
	if resp, err = handler.client.Request(request); err != nil {
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container prune request to docker socket error",
			},
		); err != nil {
			logger.Errorf("write response failed %v", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var data []byte
		if data, err = utils.ReadAndForward(resp, response); err != nil {
			logger.Errorf("read and forward failed %v", err)
			return
		}
		pruneResult := containerPruneResult{}
		if err = json.Unmarshal(data, &pruneResult); err != nil {
			logger.Errorf("json unmarshal failed %v", err)
			return
		}
		size := len(pruneResult.ContainersDeleted)
		logger.Infof("container prune removed %d containers", size)
		if size != 0 {
			go handler.releaseReservedIPs(pruneResult.ContainersDeleted)
		}
		return
	}

	if err = utils.Forward(resp, response); err != nil {
		logger.Errorf("forward error %v", err)
	}
}

func (handler containerPruneHandle) match(request *http.Request) bool {
	return request.Method == http.MethodPost && regexPruneContainers.MatchString(request.URL.Path)
}

func (handler containerPruneHandle) releaseReservedIPs(containerIDs []string) {
	logger := handler.Logger("releaseReservedIPs")

	for _, fullID := range containerIDs {
		logger.Debugf("releasing reserved IP by tied container(%s)", fullID)
		if err := handler.ReleaseContainerAddresses(context.Background(), fullID); err != nil {
			logger.Errorf("release reserved IP by tied container(%s) error", fullID)
			logger.Errorf("release IP failed %v", err)
		} else {
			logger.Infof("release reserved IP by tied container(%s) success", fullID)
		}
	}
}
