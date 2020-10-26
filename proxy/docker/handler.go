package docker

import (
	"net/http"
	"time"

	"github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/vessel"
)

// NewHandler .
func NewHandler(dockerDaemonSocket string, dialTimeout time.Duration, vess vessel.Helper) http.Handler {
	client := newHTTPClient(dockerDaemonSocket, dialTimeout)

	inspectAgent := containerInspectAgent{client: client}
	return proxy.HTTPProxyHandler{
		Handlers: []proxy.RequestHandler{
			containerCreateHandler{
				client: client, Helper: vess,
			},
			containerDeleteHandler{
				client: client, Helper: vess, inspectAgent: inspectAgent,
			},
			containerPruneHandle{
				client: client, Helper: vess,
			},
			networkConnectHandler{
				client: client, Helper: vess, inspectAgent: inspectAgent,
			},
			networkDisconnectHandler{
				client: client, Helper: vess, inspectAgent: inspectAgent,
			},
		},
		HTTPClient: client,
	}
}
