package api

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

var regexCreateContainer *regexp.Regexp = regexp.MustCompile(`/(.*?)/containers/create(\?.*)?`)

// ContainerCreateHandler .
type ContainerCreateHandler struct {
	sock          sock.SocketInterface
	minionsClient minions.Client
	ipPoolNames   []string
}

// IPAMConfig .
type IPAMConfig struct {
	IPv4Address string
	IPv6Address string
}

// ContainerCreateRequestBody .
type ContainerCreateRequestBody struct {
	NetworkingConfig struct {
		EndpointsConfig map[string]struct {
			IPAMConfig IPAMConfig
		}
	}
}

// NewContainerCreateHandler .
func NewContainerCreateHandler(sock sock.SocketInterface, minionsClient minions.Client, ipPoolNames []string) ContainerCreateHandler {
	return ContainerCreateHandler{
		sock:          sock,
		minionsClient: minionsClient,
		ipPoolNames:   ipPoolNames,
	}
}

// Handle .
func (handler ContainerCreateHandler) Handle(response http.ResponseWriter, request *http.Request) (handled bool) {
	if handled = regexCreateContainer.MatchString(request.URL.Path); !handled {
		return
	}
	log.Debug("[ContainerCreateHandler.Handle] container create request")

	var (
		err           error
		contentWriter = bytes.NewBuffer(nil)
	)

	var resp *http.Response
	if resp, err = handler.sock.RawRequest(
		request.Method,
		request.URL.String(),
		request.Header,
		io.TeeReader(request.Body, contentWriter),
	); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] request failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "send container remove request to docker socket error",
			},
		); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] write response failed %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if err = utils.Forward(resp, response); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] forward message failed %v", err)
	}
	if resp.StatusCode == http.StatusCreated {
		log.Debug("[ContainerCreateHandler.Handle] create container success, will try mark reserve request for ip")
		handler.tryMarkReserveRequest(contentWriter)
	} else {
		log.Errorf("[ContainerCreateHandler.Handle] create container failed, status code = %d", resp.StatusCode)
	}
	return handled
}

func (handler ContainerCreateHandler) tryMarkReserveRequest(reader io.Reader) {
	var (
		requestContent []byte
		err            error
	)
	if requestContent, err = ioutil.ReadAll(reader); err != nil {
		log.Errorf("[ContainerCreateHandler.tryMarkReserveRequest] read request failed %v", err)
		return
	}
	log.Debugf("request content is %s", string(requestContent))
	requestBody := ContainerCreateRequestBody{}
	if err := json.Unmarshal(requestContent, &requestBody); err != nil {
		log.Errorf("[ContainerCreateHandler.tryMarkReserveRequest] unmarshal request failed %v", err)
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
	ip := parseIP(config)
	if ip == "" {
		return
	}
	if err := handler.minionsClient.MarkReserveRequestForIP(ip); err != nil {
		log.Errorf("[ContainerCreateHandler.markReserveRequest] mark failed %v", err)
	} else {
		log.Debug("[ContainerCreateHandler.markReserveRequest] success")
	}
}

func parseIP(config IPAMConfig) string {
	if config.IPv4Address != "" {
		log.Infof("[parseIP] ip is %s", config.IPv4Address)
		return config.IPv4Address
	}
	if config.IPv6Address != "" {
		log.Infof("[parseIP] ip is %s", config.IPv6Address)
		return config.IPv6Address
	}
	log.Info("[parseIP] ip is empty")
	return ""
}
