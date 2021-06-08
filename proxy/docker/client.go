package docker

import (
	"context"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	barrelHttp "github.com/projecteru2/barrel/http"
	log "github.com/sirupsen/logrus"
)

type httpClientImpl struct {
	httpClient *http.Client
}

func newHTTPClient(dockerDaemonSocket string, dialTimeout time.Duration) barrelHttp.Client {
	return httpClientImpl{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", strings.TrimPrefix(dockerDaemonSocket, "unix://"), dialTimeout)
				},
			},
		},
	}
}

func (client httpClientImpl) Request(req *http.Request) (clientResponse *http.Response, err error) {
	var clientRequest *http.Request
	if clientRequest, err = http.NewRequest(req.Method, processURL(req.URL.String()), req.Body); err != nil {
		log.Errorf("[RawRequest] create HttpClientRequest failed %v", err)
		return nil, err
	}
	clientRequest.Header = req.Header
	clientRequest.ContentLength = req.ContentLength
	clientRequest.TransferEncoding = req.TransferEncoding

	if clientResponse, err = client.httpClient.Do(clientRequest); err != nil {
		log.Errorf("[RawRequest] send HttpClientRequest failed %v", err)
	}
	return
}

var regexpHTTPScheme = regexp.MustCompile("((http)|(https))://.*")

func processURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return "http://docker" + url
	} else if !regexpHTTPScheme.MatchString(url) {
		return "http://docker/" + url
	}
	return url
}
