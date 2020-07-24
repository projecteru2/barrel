package proxy

import (
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projecteru2/barrel/api"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/sock"
	"github.com/projecteru2/barrel/sock/docker"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	minions "github.com/projecteru2/minions/lib"
	log "github.com/sirupsen/logrus"
)

// Config .
type Config struct {
	DockerdSocketPath string
	DialTimeout       time.Duration
	IPPoolNames       []string
}

// DockerdProxy .
type DockerdProxy struct {
	mux      *http.ServeMux
	handlers []types.RequestHandler
	sock     sock.SocketInterface
}

// NewProxy .
func NewProxy(config Config, etcdV3 *etcdv3.Client, calicoV3 calicov3.Interface) *DockerdProxy {
	proxy := new(DockerdProxy)
	proxy.mux = http.NewServeMux()
	proxy.mux.HandleFunc("/", proxy.dispatch)
	minionsClient := minions.NewClient(etcdV3, calicoV3)
	// TODO only docker socket support
	dockerSocket := docker.NewSocket(config.DockerdSocketPath, config.DialTimeout)
	proxy.sock = dockerSocket
	proxy.addHandler(api.NewContainerDeleteHandler(dockerSocket, minionsClient))
	proxy.addHandler(api.NewContainerPruneHandle(dockerSocket, minionsClient))
	proxy.addHandler(api.NewContainerCreateHandler(dockerSocket, minionsClient, config.IPPoolNames))
	return proxy
}

func (proxy *DockerdProxy) addHandler(handler types.RequestHandler) {
	proxy.handlers = append(proxy.handlers, handler)
}

// Start will block
func (proxy *DockerdProxy) Start(host ...types.Host) error {
	switch len(host) {
	case 0:
		return common.ErrNoListener
	case 1:
		return proxy.startOnHost(host[0])
	default:
		return proxy.startOnHosts(host)
	}
}

// StartOnListeners .
func (proxy *DockerdProxy) startOnHosts(hosts []types.Host) error {
	return startHostGroup(proxy.mux, hosts)
}

func (proxy *DockerdProxy) startOnHost(host types.Host) (err error) {
	addr := host.Listener.Addr().String()
	log.Infof("[startOnHost] Starting proxy at %s", addr)
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
	log.Infof("[dispatch] Incoming request, method = %s, url = %s", request.Method, request.URL.String())
	utils.PrintHeaders("ServerRequest", request.Header)

	for _, handler := range proxy.handlers {
		if handler.Handle(response, request) {
			return
		}
	}
	log.Info("[dispatch] handle other docker request, will forward stream")

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
