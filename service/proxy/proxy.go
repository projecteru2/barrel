package proxy

import (
	"io"
	"net"
	"net/http"
	"reflect"

	"github.com/projecteru2/barrel/api"
	"github.com/projecteru2/barrel/handler"
	"github.com/projecteru2/barrel/handler/compose"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/sock/docker"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
)

// NewProxy .
func NewProxy(config types.DockerConfig, ipam types.ReservedAddressManager) (service.DisposableService, error) {
	dockerSocket := docker.NewSocket(config.DockerdSocketPath, config.DialTimeout)
	return createService(config, compose.Handlers([]handler.RequestHandler{
		api.NewContainerDeleteHandler(dockerSocket, ipam),
		api.NewContainerPruneHandle(dockerSocket, ipam),
		api.NewContainerCreateHandler(dockerSocket, ipam),
		api.NewNetworkConnectHandler(dockerSocket, ipam),
		api.NewNetworkDisconnectHandler(dockerSocket, ipam),
		&forwardProxy{dockerSocket},
	}...))
}

func createService(config types.DockerConfig, handler http.Handler) (service.DisposableService, error) {
	launcher := HostLauncher{
		dockerGid: config.DockerGid,
		certFile:  config.CertFile,
		keyFile:   config.KeyFile,
		handler:   handler,
	}
	switch len(config.Hosts) {
	case 0:
		return nil, types.ErrNoHosts
	case 1:
		return launcher.Launch(config.Hosts[0])
	default:
		return launcher.LaunchMultiple(config.Hosts...)
	}
}

type forwardProxy struct {
	sock sock.SocketInterface
}

// Handle .
func (proxy *forwardProxy) Handle(ctx handler.Context, response http.ResponseWriter, request *http.Request) {
	log.Info("[Handle] handle other docker request, will forward stream")
	var (
		resp *http.Response
		err  error
	)
	header := request.Header
	header.Add("Host", request.Host)
	// do a swallow copy of request
	newRequest := *request
	newRequest.Host = "docker"
	if resp, err = proxy.sock.Request(&newRequest); err != nil {
		log.Errorf("[dispatch] send request to docker socket error %v", err)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		log.Debugf("[dispatch] Type of body: %v", reflect.TypeOf(resp.Body)) // TODO remove reflect
		log.Debug("[dispatch] Forward http response")
		if err := utils.Forward(resp, response); err != nil {
			log.Errorf("[dispatch] forward docker socket response failed %v", err)
		}
		return
	}
	linkConn(response, resp)
}

func linkConn(response http.ResponseWriter, resp *http.Response) {
	log.Debug("[linkConn] Will linked upgraded connection")
	// we will hijack connection and link with dockerd connection
	// test response writer could be hijacked
	if hijacker, ok := response.(http.Hijacker); ok {
		// test resp body is writable
		if readWriteCloser, ok := resp.Body.(io.ReadWriteCloser); ok {
			doLinkConn(response, resp, hijacker, readWriteCloser)
		} else {
			log.Error("[linkConn] Can't Write To ClientRequestBody")
			if err := utils.WriteBadGateWayResponse(
				response,
				utils.HTTPSimpleMessageResponseBody{
					Message: "Can't Write To ClientRequestBody",
				},
			); err != nil {
				log.Errorf("[linkConn] link conn failed %v", err)
			}
		}
		return
	}
	log.Error("[linkConn] can't Hijack ServerResponseWriter")
	if err := utils.WriteBadGateWayResponse(
		response,
		utils.HTTPSimpleMessageResponseBody{
			Message: "Can't Hijack ServerResponseWriter",
		},
	); err != nil {
		log.Errorf("[linkConn] write bad gateway response %v", err)
	}
}

func doLinkConn(response http.ResponseWriter, resp *http.Response, hijacker http.Hijacker, readWriteCloser io.ReadWriteCloser) {
	var err error
	// first we send response to non overrided client, make sure it's ready for new protocol
	if err = utils.WriteToServerResponse(
		response,
		http.StatusSwitchingProtocols,
		resp.Header,
		nil,
	); err != nil {
		log.Errorf("[doLinkConn] write StatusSwitchingProtocols failed %v", err)
		return
	}
	var conn net.Conn
	log.Debug("[doLinkConn] Hijack server http connection")
	if conn, _, err = hijacker.Hijack(); err != nil {
		log.Errorf("[doLinkConn] Hijack ServerResponseWriter failed %v", err)
		return
	}
	defer utils.Link(conn, readWriteCloser)
	// link client conn and server conn
	log.Debug("[doLinkConn] link connection")
}
