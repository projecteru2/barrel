package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"net/http"

	"github.com/juju/errors"
	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/vessel"
	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/utils"
)

var regexInspectContainer = regexp.MustCompile(`/[^/]+/containers/[^/]+/json`)

type containerInspectResult struct {
	ID         string `json:"Id"`
	HostConfig struct {
		Runtime string
	}
	Config struct {
		Image  string
		Labels map[string]string
		Env    []string
	}
	Mounts []struct {
		Name        string
		Source      string
		Destination string
	}
}

type containerInspectAgent struct {
	utils.LoggerFactory
	client barrelHttp.Client
}

func newContainerInspectAgent(client barrelHttp.Client) containerInspectAgent {
	return containerInspectAgent{
		LoggerFactory: utils.NewObjectLogger("containerInspectAgent"),
		client:        client,
	}
}

// Inspect .
func (handler containerInspectAgent) Inspect(identifier string, version string) (containerInspectResult, error) {
	logger := handler.Logger("Inspect")

	var (
		container = containerInspectResult{}
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
		logger.Errorf("create inspect container(%s) request error", identifier)
		return container, err
	}

	var resp *http.Response
	if resp, err = handler.client.Request(req); err != nil {
		logger.Errorf("send inspect container(%s) request error", identifier)
		return container, err
	}
	defer resp.Body.Close()

	logger.Infof("inspect container(%s) done, response status = %s", identifier, resp.Status)
	if resp.StatusCode == http.StatusNotFound {
		logger.Infof("container(%s) is not exists", identifier)
		return container, errors.Annotate(types.ErrContainerNotExists, identifier)
	}

	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		logger.Errorf("read inspect container(%s) response error", identifier)
		return container, err
	}

	if resp.StatusCode == http.StatusOK {
		if err = json.Unmarshal(data, &container); err != nil {
			logger.Errorf("unmarshal inspect container(%s) response error", identifier)
			return container, err
		}
		return container, nil
	}
	if len(data) == 0 {
		err = errors.Errorf("Container %s not found", identifier)
		return container, err
	}
	return container, errors.Errorf("Inspect container(%s) error, result: %s", identifier, string(data))
}

type containerInspectHandler struct {
	utils.LoggerFactory
	client barrelHttp.Client
	vess vessel.Helper
}

func newContainerInspectHandler(client barrelHttp.Client, vess vessel.Helper) proxy.RequestHandler {
	return containerInspectHandler{
		LoggerFactory: utils.NewObjectLogger("containerInspectHandler"),
		client: client,
		vess: vess,
	}
}

func (handler containerInspectHandler) Handle(ctx proxy.HandleContext, response http.ResponseWriter, req *http.Request) {
	logger := handler.Logger("Handle")

	if req.Method != http.MethodGet || !regexInspectContainer.MatchString(req.URL.Path) {
		ctx.Next()
		return
	}

	logger.Debug("container inspect request")
	resp, err := handler.client.Request(req)
	if err != nil {
		logger.Errorf("request failed %+v", err)
		if e := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container inspect request to docker socket error",
			},
		); e != nil {
			logger.Errorf("write response failed %+v", err)
		}
		return
	}
	defer resp.Body.Close()

	//resp = handler.injectNetworkforCNI(resp)
	if err = utils.Forward(resp, response); err != nil {
		logger.Errorf("forward failed %+v", err)
	}
}
