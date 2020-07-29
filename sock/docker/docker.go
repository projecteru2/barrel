package docker

import (
	"context"
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
func (sock Socket) Request(rawRequest *http.Request) (clientResponse *http.Response, err error) {
	var clientRequest *http.Request
	if clientRequest, err = http.NewRequest(rawRequest.Method, processURL(rawRequest.URL.String()), rawRequest.Body); err != nil {
		log.Errorf("[RawRequest] create HttpClientRequest failed %v", err)
		return nil, err
	}
	clientRequest.Header = rawRequest.Header
	clientRequest.ContentLength = rawRequest.ContentLength
	clientRequest.TransferEncoding = rawRequest.TransferEncoding

	if clientResponse, err = sock.httpClient.Do(clientRequest); err != nil {
		log.Errorf("[RawRequest] send HttpClientRequest failed %v", err)
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
