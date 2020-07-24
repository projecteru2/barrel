package sock

import (
	"io"
	"net/http"
)

// SocketInterface .
type SocketInterface interface {
	Request(rawRequest *http.Request) (*http.Response, error)
	RawRequest(method string, url string, header http.Header, body io.Reader) (*http.Response, error)
}
