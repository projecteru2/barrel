package sock

import (
	"context"
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
	}
}

func (sock DockerSocket) Request(rawRequest *http.Request) (*http.Response, error) {
	var err error

	method := rawRequest.Method
	url := processURL(rawRequest.URL.String())
	body := rawRequest.Body

	var clientRequest *http.Request
	if clientRequest, err = http.NewRequest(method, url, body); err != nil {
		log.Errorln("create HttpClientRequest error", err)
		return nil, err
	}
	clientRequest.Header = rawRequest.Header

	var clientResponse *http.Response
	clientResponse, err = sock.httpClient.Do(clientRequest)
	if err != nil {
		log.Errorln("send HttpClientRequest error", err)
	}
	return clientResponse, err
}

func processURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return "http://docker" + url
	} else if !regexpHTTPScheme.MatchString(url) {
		return "http://docker/" + url
	}
	return url
}
