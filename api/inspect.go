package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/types"
	log "github.com/sirupsen/logrus"
)

// ContainerInspectResult .
type ContainerInspectResult struct {
	ID     string `json:"Id"`
	Config struct {
		Labels map[string]string
	}
}

// ContainerInspectHandler .
type ContainerInspectHandler struct {
	sock sock.SocketInterface
}

// Inspect .
func (handler ContainerInspectHandler) Inspect(identifier string, version string) (ContainerInspectResult, error) {
	var (
		container = ContainerInspectResult{}
		req       *http.Request
		err       error
	)
	if identifier == "" {
		return container, types.ErrNoContainerIdent
	}
	if version == "" {
		return container, types.ErrWrongAPIVersion
	}
	if req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/containers/%s/json", version, identifier), nil); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] create inspect container(%s) request error", identifier)
		return container, err
	}

	var resp *http.Response
	if resp, err = handler.sock.Request(req); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] send inspect container(%s) request error", identifier)
		return container, err
	}
	defer resp.Body.Close()

	log.Infof("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) done, response status = %s", identifier, resp.Status)
	if resp.StatusCode == http.StatusNotFound {
		log.Infof("[ContainerInspectHandler.GetFullContainerID] container(%s) is not exists", identifier)
		return container, errors.Annotate(types.ErrContainerNotExists, identifier)
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Errorf("[ContainerInspectHandler.GetFullContainerID] read inspect container(%s) response error", identifier)
		return container, err
	}

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(data, &container); err != nil {
			log.Errorf("[ContainerInspectHandler.GetFullContainerID] unmarshal inspect container(%s) response error", identifier)
			return container, err
		}
		return container, nil
	}
	if len(data) == 0 {
		err = errors.Errorf("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) error", identifier)
		return container, err
	}
	return container, errors.Errorf("[ContainerInspectHandler.GetFullContainerID] inspect container(%s) error, result: %s", identifier, string(data))
}
