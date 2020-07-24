package proxy

import (
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projecteru2/barrel/internal/api/container"
	"github.com/projecteru2/barrel/internal/sock"
	"github.com/projecteru2/barrel/internal/types"
	"github.com/projecteru2/barrel/internal/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

var (
	debug bool
)

// Initialize .
func Initialize(_debug bool) {
	debug = _debug
}

// Config .
type Config struct {
	DockerdSocketPath string
	DialTimeout       time.Duration
	IPPoolNames       []string
}

// DockerdProxy .
type DockerdProxy struct {
	mux          *http.ServeMux
	handlers     []types.RequestHandler
	dockerSocket sock.DockerSocket
}

// NewProxy .
func NewProxy(config Config, etcdV3 *etcdv3.Client, calicoV3 calicov3.Interface) *DockerdProxy {
	proxy := new(DockerdProxy)
	proxy.mux = http.NewServeMux()
	proxy.mux.HandleFunc("/", proxy.dispatch)
	minionsClient := minions.NewClient(etcdV3, calicoV3)
	dockerSocket := sock.NewDockerSocket(config.DockerdSocketPath, config.DialTimeout)
	proxy.dockerSocket = dockerSocket
	proxy.addHandler(container.NewContainerDeleteHandler(dockerSocket, minionsClient))
	proxy.addHandler(container.NewContainerPruneHandle(dockerSocket, minionsClient))
	proxy.addHandler(container.NewContainerCreateHandler(dockerSocket, minionsClient, config.IPPoolNames))
	return proxy
}

func (proxy *DockerdProxy) addHandler(handler types.RequestHandler) {
	proxy.handlers = append(proxy.handlers, handler)
}

// Start will block
func (proxy *DockerdProxy) Start(host ...Host) error {
	switch len(host) {
	case 0:
		return errors.New("no listener is provided")
	case 1:
		return proxy.startOnHost(host[0])
	default:
		return proxy.startOnHosts(host)
	}
}

// StartOnListeners .
func (proxy *DockerdProxy) startOnHosts(hosts []Host) error {
	return startHostGroup(proxy.mux, hosts)
}

func (proxy *DockerdProxy) startOnHost(host Host) (err error) {
	addr := host.Listener.Addr().String()
	log.Infof("Starting proxy at %s", addr)
	server := http.Server{
		Addr:    addr,
		Handler: proxy.mux,
	}
	if host.Cert != "" {
		return server.ServeTLS(host.Listener, host.Cert, host.Key)
	}
	return server.Serve(host.Listener)
}

func (proxy *DockerdProxy) dispatch(response http.ResponseWriter, request *http.Request) {
	log.Infof("Incoming request, method = %s, url = %s", request.Method, request.URL.String())
	if debug {
		utils.PrintHeaders("ServerRequest", request.Header)
	}

	for _, handler := range proxy.handlers {
		if handler.Handle(response, request) {
			return
		}
	}
	log.Info("handle other docker request, will forward stream")

	var (
		resp *http.Response
		err  error
	)
	header := request.Header
	header.Add("Host", request.Host)
	// do a swallow copy of request
	newRequest := *request
	newRequest.Host = "docker"
	if resp, err = proxy.dockerSocket.Request(&newRequest); err != nil {
		log.Errorln("send request to docker socket error", err)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		if debug {
			log.Info("Type of body: ", reflect.TypeOf(resp.Body))
		}
		log.Info("Forward http response")
		if err := utils.Forward(resp, response); err != nil {
			log.Errorln("forward docker socket response error", err)
		}
		return
	}
	linkConn(response, resp)
}

func linkConn(response http.ResponseWriter, resp *http.Response) {
	log.Info("Will linked upgraded connection")
	// we will hijack connection and link with dockerd connection
	// test response writer could be hijacked
	if hijacker, ok := response.(http.Hijacker); ok {
		// test resp body is writable
		if readWriteCloser, ok := resp.Body.(io.ReadWriteCloser); ok {
			doLinkConn(response, resp, hijacker, readWriteCloser)
		} else {
			log.Errorln("Can't Write To ClientRequestBody")
			if err := utils.WriteBadGateWayResponse(
				response,
				utils.HTTPSimpleMessageResponseBody{
					Message: "Can't Write To ClientRequestBody",
				},
			); err != nil {
				log.Error(err)
			}
		}
	} else {
		log.Errorln("Can't Hijack ServerResponseWriter")
		if err := utils.WriteBadGateWayResponse(
			response,
			utils.HTTPSimpleMessageResponseBody{
				Message: "Can't Hijack ServerResponseWriter",
			},
		); err != nil {
			log.Error(err)
		}
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
		log.Error("Write StatusSwitchingProtocols Error", err)
		return
	}
	var conn net.Conn
	log.Info("Hijack server http connection")
	if conn, _, err = hijacker.Hijack(); err != nil {
		log.Error("Hijack ServerResponseWriter Error", err)
		return
	}
	// link client conn and server conn
	log.Info("Link connection")
	utils.Link(conn, readWriteCloser)
}
