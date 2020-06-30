package container

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/internal/sock"
	"github.com/projecteru2/barrel/internal/utils"
	log "github.com/sirupsen/logrus"
)

// ContainerInspectResult .
type ContainerInspectResult struct {
	ID string `json:"Id"`
}

// ContainerInspectHandler .
type ContainerInspectHandler struct {
	netUtil      utils.NetUtil
	dockerSocket sock.DockerSocket
}

// ContainerNotExistsError .
type ContainerNotExistsError struct {
	message string
}

func (err ContainerNotExistsError) Error() string {
	return err.message
}

// GetFullContainerID .
func (handler ContainerInspectHandler) GetFullContainerID(idOrName string, version string) (fullID string, err error) {
	if idOrName == "" {
		err = errors.New("container id or name must not be null")
		return
	}
	if version == "" {
		err = errors.New("api version must not be null")
		return
	}

	var req *http.Request
	if req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/containers/%s/json", version, idOrName), nil); err != nil {
		log.Errorf("create inspect container(%s) request error\n", idOrName)
		return
	}

	var resp *http.Response
	if resp, err = handler.dockerSocket.Request(req); err != nil {
		log.Errorf("send inspect container(%s) request error\n", idOrName)
		return
	}
	defer resp.Body.Close()

	log.Infof("inspect container(%s) done, response status = %s\n", idOrName, resp.Status)
	if resp.StatusCode == http.StatusNotFound {
		log.Infof("container(%s) is not exists", idOrName)
		err = ContainerNotExistsError{fmt.Sprintf("container(%s) is not exists", idOrName)}
		return
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Errorf("read inspect container(%s) response error", idOrName)
		return
	}

	if resp.StatusCode == http.StatusOK {
		container := ContainerInspectResult{}
		if err = json.Unmarshal(data, &container); err != nil {
			log.Errorf("unmarshal inspect container(%s) response error", idOrName)
			return
		}
		if container.ID != "" {
			fullID = container.ID
			return
		}
		err = errors.Errorf("inspect container(%s) response error, Container.ID is empty, result: %s", idOrName, string(data))
		return
	}
	if len(data) == 0 {
		err = errors.Errorf("inspect container(%s) error", idOrName)
		return
	}
	err = errors.Errorf("inspect container(%s) error, result: %s", idOrName, string(data))
	return
}
