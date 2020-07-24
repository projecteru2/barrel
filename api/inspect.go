package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/sock"
	log "github.com/sirupsen/logrus"
)

// ContainerInspectResult .
type ContainerInspectResult struct {
	ID string `json:"Id"`
}

// ContainerInspectHandler .
type ContainerInspectHandler struct {
	sock sock.SocketInterface
}

// GetFullContainerID .
func (handler ContainerInspectHandler) GetFullContainerID(idOrName string, version string) (fullID string, err error) {
	if idOrName == "" {
		return fullID, common.ErrNoContainerIdent
	}
	if version == "" {
		return fullID, common.ErrWrongAPIVersion
	}

	var req *http.Request
	if req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/containers/%s/json", version, idOrName), nil); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] create inspect container(%s) request error", idOrName)
		return
	}

	var resp *http.Response
	if resp, err = handler.sock.Request(req); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] send inspect container(%s) request error", idOrName)
		return
	}
	defer resp.Body.Close()

	log.Infof("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) done, response status = %s", idOrName, resp.Status)
	if resp.StatusCode == http.StatusNotFound {
		log.Infof("[ContainerInspectHandler.GetFullContainerID] container(%s) is not exists", idOrName)
		err = errors.Annotate(common.ErrContainerNotExists, idOrName)
		return
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] read inspect container(%s) response error", idOrName)
		return
	}

	if resp.StatusCode == http.StatusOK {
		container := ContainerInspectResult{}
		if err = json.Unmarshal(data, &container); err != nil {
			log.Errorf("[ContainerInspectHandler.GetFullContainerID] unmarshal inspect container(%s) response error", idOrName)
			return
		}
		if container.ID != "" {
			fullID = container.ID
			return
		}
		err = errors.Errorf("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) response error, Container.ID is empty, result: %s", idOrName, string(data))
		return
	}
	if len(data) == 0 {
		err = errors.Errorf("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) error", idOrName)
		return
	}
	return fullID, errors.Errorf("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) error, result: %s", idOrName, string(data))
}
