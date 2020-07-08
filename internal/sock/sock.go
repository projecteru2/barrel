package sock

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

// DockerSocket .
type DockerSocket struct {
	httpClient *http.Client
	// simpleHttpClient utils.SimpleHTTPClient
}

var regexpHTTPScheme *regexp.Regexp = regexp.MustCompile("((http)|(https))://.*")

func NewDockerSocket(dockerdSocketPath string, dialTimeout time.Duration) DockerSocket {
	return DockerSocket{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", dockerdSocketPath, dialTimeout)
				},
			},
		},
		// simpleHttpClient: utils.SimpleHTTPClient{
		// 	ConnFactory: func() (net.Conn, error) {
		// 		return net.DialTimeout("unix", dockerdSocketPath, dialTimeout)
		// 	},
		// },
	}
}

func (sock DockerSocket) Request(rawRequest *http.Request) (*http.Response, error) {
	return sock.DoRequest(rawRequest.Method, rawRequest.URL.String(), rawRequest.Header, rawRequest.Body)
}

func (sock DockerSocket) DoRequest(method string, url string, header http.Header, body io.Reader) (clientResponse *http.Response, err error) {
	url = processURL(url)

	var clientRequest *http.Request
	if clientRequest, err = http.NewRequest(method, url, body); err != nil {
		log.Errorln("create HttpClientRequest error", err)
		return nil, err
	}
	clientRequest.Header = header

	if clientResponse, err = sock.httpClient.Do(clientRequest); err != nil {
		log.Errorln("send HttpClientRequest error", err)
	}
	return
}

// // SimpleRequest .
// func (sock DockerSocket) SimpleRequest(simpleRequest utils.SimpleHTTPRequest) (utils.SimpleHTTPResponse, error) {
// 	return sock.simpleHttpClient.Request(simpleRequest)
// }

func processURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return "http://docker" + url
	} else if !regexpHTTPScheme.MatchString(url) {
		return "http://docker/" + url
	}
	return url
}
