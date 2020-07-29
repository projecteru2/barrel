package sock

import (
	"net/http"
)

// SocketInterface .
type SocketInterface interface {
	Request(rawRequest *http.Request) (*http.Response, error)
}
