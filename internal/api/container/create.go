package container

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/projecteru2/barrel/internal/sock"
	"github.com/projecteru2/barrel/internal/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

var regexCreateContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/create(\?.*)?`)

type ContainerCreateHandler struct {
	dockerSocket  sock.DockerSocket
	minionsClient minions.Client
	ipPoolNames   []string
}

type IPAMConfig struct {
	IPv4Address string
	IPv6Address string
}

type ContainerCreateRequestBody struct {
	NetworkingConfig struct {
		EndpointsConfig map[string]struct {
			IPAMConfig IPAMConfig
		}
	}
}

func NewContainerCreateHandler(dockerSocket sock.DockerSocket, minionsClient minions.Client, ipPoolNames []string) ContainerCreateHandler {
	return ContainerCreateHandler{
		dockerSocket:  dockerSocket,
		minionsClient: minionsClient,
		ipPoolNames:   ipPoolNames,
	}
}

func (handler ContainerCreateHandler) Handle(response http.ResponseWriter, request *http.Request) (handled bool) {
	if handled = regexCreateContainer.MatchString(request.URL.Path); !handled {
		return
	}
	log.Info("handle container create request")

	var (
		err           error
		contentWriter = bytes.NewBuffer(nil)
	)

	var resp *http.Response
	if resp, err = handler.dockerSocket.DoRequest(
		request.Method,
		request.URL.String(),
		request.Header,
		io.TeeReader(request.Body, contentWriter),
	); err != nil {
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

	if err = utils.Forward(resp, response); err != nil {
		log.Errorln(err)
	}
	if resp.StatusCode == http.StatusCreated {
		log.Infoln("create container success, will try mark reserve request for ip")
		handler.tryMarkReserveRequest(contentWriter)
	} else {
		log.Infof("create container failed, status code = %d", resp.StatusCode)
	}
	return
}

func (handler ContainerCreateHandler) tryMarkReserveRequest(reader io.Reader) {
	var (
		requestContent []byte
		err            error
	)
	if requestContent, err = ioutil.ReadAll(reader); err != nil {
		log.Errorln("read request content error", err)
		return
	}
	log.Infof("request content is %s", string(requestContent))
	requestBody := ContainerCreateRequestBody{}
	if err := json.Unmarshal(requestContent, &requestBody); err != nil {
		log.Errorln("unmarshal request content error", err)
		return
	}
	for _, name := range handler.ipPoolNames {
		config, ok := requestBody.NetworkingConfig.EndpointsConfig[name]
		if ok {
			handler.markReserveRequest(config.IPAMConfig)
		}
	}
}

func (handler ContainerCreateHandler) markReserveRequest(config IPAMConfig) {
	if ip := parseIP(config); ip != "" {
		if err := handler.minionsClient.MarkReserveRequestForIP(ip); err != nil {
			log.Errorln("markReserveRequest error", err)
		} else {
			log.Info("markReserveRequest success")
		}
	}
}

func parseIP(config IPAMConfig) string {
	if config.IPv4Address != "" {
		log.Infof("ip is %s", config.IPv4Address)
		return config.IPv4Address
	}
	if config.IPv6Address != "" {
		log.Infof("ip is %s", config.IPv6Address)
		return config.IPv6Address
	}
	log.Info("ip is empty")
	return ""
}
