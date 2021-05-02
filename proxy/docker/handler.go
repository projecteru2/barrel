package docker

import (
	"net/http"
	"time"

	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/vessel"
)

// NewHandler .
func NewHandler(conf types.Config, vess vessel.Helper, res vessel.ResourceManager) http.Handler {
	client := newHTTPClient(conf.DockerDaemonUnixSocket, conf.DialTimeout)

	inspectAgent := newContainerInspectAgent(client)
	return proxy.HTTPProxyHandler{
		Handlers: []proxy.RequestHandler{
			newContainerCreateHandler(client, vess),
			newContainerDeleteHandler(client, res, conf.RecycleTimeout),
			newContainerPruneHandle(client, vess),
			newNetworkConnectHandler(client, vess, inspectAgent),
			newNetworkDisconnectHandler(client, vess, inspectAgent),
		},
		HTTPClient: client,
	}
}

// NewSimpleHandler .
func NewSimpleHandler(dockerDaemonSocket string, dialTimeout time.Duration) http.Handler {
	client := newHTTPClient(dockerDaemonSocket, dialTimeout)

	return proxy.HTTPProxyHandler{
		HTTPClient: client,
	}
}
