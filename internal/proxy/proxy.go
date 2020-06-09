package proxy

import (
	"net"
	"net/http"
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

const defaultBufferSize = 256

// ProxyConfig .
type ProxyConfig struct {
	DockerdSocketPath string
	DialTimeout       time.Duration
}

// DockerdProxy .
type DockerdProxy struct {
	mux          *http.ServeMux
	handlers     []types.RequestHandler
	dockerSocket sock.DockerSocket
	netUtil      utils.NetUtil
}

// NewProxy .
func NewProxy(config ProxyConfig, etcdV3 *etcdv3.Client, calicoV3 calicov3.Interface) *DockerdProxy {
	proxy := new(DockerdProxy)
	proxy.mux = http.NewServeMux()
	proxy.mux.HandleFunc("/", proxy.dispatch)
	minionsClient := minions.NewClient(etcdV3, calicoV3)
	dockerSocket := sock.NewDockerSocket(config.DockerdSocketPath, config.DialTimeout)
	netUtil := utils.NetUtil{BufferSize: defaultBufferSize}
	proxy.dockerSocket = dockerSocket
	proxy.netUtil = netUtil
	proxy.addHandler(container.NewContainerDeleteHandler(dockerSocket, minionsClient, netUtil))
	proxy.addHandler(container.NewContainerPruneHandle(dockerSocket, minionsClient, netUtil))
	return proxy
}

func (proxy *DockerdProxy) addHandler(handler types.RequestHandler) {
	proxy.handlers = append(proxy.handlers, handler)
}

// Start .
func (proxy *DockerdProxy) Start(listener net.Listener) error {
	addr := listener.Addr().String()
	log.Infof("Starting proxy at %s", addr)
	server := http.Server{
		Addr:    addr,
		Handler: proxy.mux,
	}
	return server.Serve(listener)
}

func (proxy *DockerdProxy) dispatch(response http.ResponseWriter, request *http.Request) {
	log.Infof("Incoming request, method = %s, url = %s", request.Method, request.URL.String())

	for _, handler := range proxy.handlers {
		if handler.Handle(response, request) {
			return
		}
	}

	var (
		resp *http.Response
		err  error
	)
	if resp, err = proxy.dockerSocket.Request(request); err != nil {
		log.Errorln("send request to docker socket error", err)
		return
	}
	if err := proxy.netUtil.Forward(resp, response); err != nil {
		log.Errorln("forward docker socket response error", err)
	}
}
