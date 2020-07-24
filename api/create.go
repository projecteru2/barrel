package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/juju/errors"
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

// ContainerCreateResponseBody .
type ContainerCreateResponseBody struct {
	ID       string `json:"Id"`
	Warnings []string
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
		err              error
		bodyBytes        []byte
		requestBody      map[string]interface{}
		requested        bool
		requestedAddress string
	)

	// we have to read the input first so as to allocate an ip in advance
	if bodyBytes, err = ioutil.ReadAll(request.Body); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] read request body failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "read request error",
			},
		); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] write read body error response failed %v", err)
		}
		return
	}

	if err = json.Unmarshal(bodyBytes, &requestBody); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] unmarshal request body failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "unmarshal request body error",
			},
		); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] write unmarshal error response failed %v", err)
		}
		return
	}

	if requested, requestedAddress, err = handler.checkAndRequestFixedIP(requestBody); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] checkAndRequestFixedIP failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "check and request fixed-ip error",
			},
		); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] write unmarshal error response failed %v", err)
		}
		return
	}

	if bodyBytes, err = json.Marshal(requestBody); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] marshal new request body failed %v", err)
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "marshal request body error",
			},
		); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] write marshal new request body error response failed %v", err)
		}
		return
	}

	var (
		req  http.Request = *request
		resp *http.Response
	)

	req.ContentLength = int64(len(bodyBytes))
	req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
	if resp, err = handler.sock.Request(&req); err != nil {
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

	if resp.StatusCode == http.StatusCreated { // nolint
		var content []byte
		if content, err = utils.ForwardAndRead(resp, response); err != nil {
			log.Errorf("[ContainerCreateHandler.Handle] forward and read message failed %v", err)
			return
		}
		if !requested {
			log.Debug("[ContainerCreateHandler.Handle] create container success, will try mark reserve request for ip")
			handler.tryMarkReserveRequest(bodyBytes)
			return
		}
		body := ContainerCreateResponseBody{}
		if err = json.Unmarshal(content, &body); err != nil {
			log.Error("[ContainerCreateHandler.Handle] parse container created resp body failed")
			return
		}
		if body.ID != "" {
			if err = handler.minionsClient.MarkFixedIPForContainer(body.ID, requestedAddress); err != nil {
				log.Errorf("[ContainerCreateHandler.Handle] mark fixed-ip(%s) for container(%s) failed %v", requestedAddress, body.ID, err)
			}
			return
		}
		log.Errorf("[ContainerCreateHandler.Handle] create container resp blank container id %v, related address = %s", err, requestedAddress)
	}
	if err = utils.Forward(resp, response); err != nil {
		log.Errorf("[ContainerCreateHandler.Handle] forward message failed %v", err)
	}
	log.Errorf("[ContainerCreateHandler.Handle] create container failed, status code = %d", resp.StatusCode)
	return handled
}

func (handler ContainerCreateHandler) checkAndRequestFixedIP(body map[string]interface{}) (requested bool, address string, err error) {
	var (
		ipPoolName        = handler.ipPoolNames[0]
		iLabels           interface{}
		labels            map[string]interface{}
		iHostConfig       interface{}
		hostConfig        map[string]interface{}
		iNetworkMode      interface{}
		networkMode       string
		iNetworkingConfig interface{}
		networkConfig     map[string]interface{}
		iEndpointsConfig  interface{}
		endpointsConfig   map[string]interface{}
		iEndpointConfig   interface{}
		endpointConfig    map[string]interface{}
		iIPAMConfig       interface{}
		ipamConfig        map[string]interface{}
		iIPv4Address      interface{}
		iIPv6Address      interface{}
		ipv4Address       string
		ipv6Address       string
		ok                bool
	)
	// check labels
	if iLabels, ok = body["Labels"]; !ok {
		// don't have labels so just skip check
		return
	}
	if labels, ok = iLabels.(map[string]interface{}); !ok {
		err = errors.New("parse Labels error")
		return
	}
	// check fixed ip
	if _, ok = labels["fixed-ip"]; !ok {
		// don't have fixed ip label so just skip
		return
	}
	if iHostConfig, ok = body["HostConfig"]; !ok {
		// should not happen
		delete(labels, "fixed-ip")
		return
	}
	if hostConfig, ok = iHostConfig.(map[string]interface{}); !ok {
		err = errors.New("parse HostConfig error")
		return
	}
	if iNetworkMode, ok = hostConfig["NetworkMode"]; !ok {
		delete(labels, "fixed-ip")
		return
	}
	if networkMode, ok = iNetworkMode.(string); !ok {
		err = errors.New("parse NetworkMode error")
		return
	}
	// not in our pool, so remove fixed ip label
	if networkMode != ipPoolName {
		delete(labels, "fixed-ip")
		return
	}
	if iNetworkingConfig, ok = body["NetworkingConfig"]; !ok {
		networkConfig = make(map[string]interface{})
		body["NetworkingConfig"] = networkConfig
	} else if networkConfig, ok = iNetworkingConfig.(map[string]interface{}); !ok {
		err = errors.New("parse NetworkConfig error")
		return
	}
	if iEndpointsConfig, ok = networkConfig["EndpointsConfig"]; !ok {
		endpointsConfig = make(map[string]interface{})
		networkConfig["EndpointsConfig"] = endpointsConfig
	} else if endpointsConfig, ok = iEndpointsConfig.(map[string]interface{}); !ok {
		err = errors.New("parse EndpointsConfig error")
		return
	}
	if iEndpointConfig, ok = endpointsConfig[ipPoolName]; !ok {
		// the container doesn't specified an ip in our pool
		endpointConfig = make(map[string]interface{})
		endpointsConfig[ipPoolName] = endpointConfig
	} else if endpointConfig, ok = iEndpointConfig.(map[string]interface{}); !ok {
		err = errors.New("parse EndpointConfig error")
		return
	}
	if iIPAMConfig, ok = endpointConfig["IPAMConfig"]; !ok {
		// create ipamConfig
		ipamConfig = make(map[string]interface{})
		endpointConfig["IPAMConfig"] = ipamConfig
	} else if ipamConfig, ok = iIPAMConfig.(map[string]interface{}); !ok {
		err = errors.New("parse IPAMConfig error")
		return
	}

	if iIPv4Address, ok = ipamConfig["IPv4Address"]; ok {
		if ipv4Address, ok = iIPv4Address.(string); !ok {
			err = errors.New("parse IPv4Address error")
			return
		}
	}
	if iIPv6Address, ok = ipamConfig["IPv6Address"]; ok {
		if ipv6Address, ok = iIPv6Address.(string); !ok {
			err = errors.New("parse IPv6Address error")
			return
		}
	}
	if ipv4Address == "" && ipv6Address == "" {
		var ip string
		if ip, err = handler.requestIP(ipPoolName); err != nil {
			return
		}
		if isIPV4(ip) {
			address = getLegelIPv4Address(ip)
			ipamConfig["IPv4Address"] = address
			requested = true
			return
		}
		if isIPV6(ip) {
			address = getLegelIPv6Address(ip)
			ipamConfig["IPv6Address"] = address
			requested = true
			return
		}
		err = errors.Errorf("IPAddress pattern is unrecognized, %s", ip)
		return
	}
	// either ipv6 or ipv6 is non blank
	return requested, address, err
}

func (handler ContainerCreateHandler) requestIP(poolID string) (string, error) {
	return handler.minionsClient.RequestFixedIP(poolID)
}

func (handler ContainerCreateHandler) tryMarkReserveRequest(requestContent []byte) {
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
		log.Info("[ContainerCreateHandler.markReserveRequest] ip is empty, no need to reserve")
		return
	}
	if err := handler.minionsClient.MarkReserveRequestForIP(ip); err != nil {
		log.Errorf("[ContainerCreateHandler.markReserveRequest] mark failed %v", err)
	} else {
		log.Infof("[ContainerCreateHandler.markReserveRequest] marked ip(%s) success", ip)
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

func isIPV4(address string) bool {
	return strings.HasSuffix(address, "/32")
}

func isIPV6(address string) bool {
	return strings.HasSuffix(address, "/128")
}

func getLegelIPv4Address(address string) string {
	return strings.TrimSuffix(address, "/32")
}

func getLegelIPv6Address(address string) string {
	return strings.TrimSuffix(address, "/128")
}
