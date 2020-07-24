package docker

import (
	"context"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var regexpHTTPScheme *regexp.Regexp = regexp.MustCompile("((http)|(https))://.*")

// Socket .
type Socket struct {
	httpClient *http.Client
	// simpleHttpClient utils.SimpleHTTPClient
}

// NewSocket .
func NewSocket(dockerdSocketPath string, dialTimeout time.Duration) Socket {
	return Socket{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", dockerdSocketPath, dialTimeout)
				},
			},
		},
	}
}

// Request .
func (sock Socket) Request(rawRequest *http.Request) (*http.Response, error) {
	return sock.RawRequest(rawRequest.Method, rawRequest.URL.String(), rawRequest.Header, rawRequest.Body)
}

// RawRequest .
func (sock Socket) RawRequest(method string, url string, header http.Header, body io.Reader) (clientResponse *http.Response, err error) {
	url = processURL(url)

	var clientRequest *http.Request
	if clientRequest, err = http.NewRequest(method, url, body); err != nil {
		log.Errorf("[RawRequest] create HttpClientRequest error %v", err)
		return nil, err
	}
	clientRequest.Header = header

	if clientResponse, err = sock.httpClient.Do(clientRequest); err != nil {
		log.Errorf("[RawRequest] send HttpClientRequest error %v", err)
	}
	return
}

func processURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return "http://docker" + url
	} else if !regexpHTTPScheme.MatchString(url) {
		return "http://docker/" + url
	}
	return url
}
