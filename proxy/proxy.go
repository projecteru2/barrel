package proxy

import (
	"io"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/utils"
)

// HandleContext .
type HandleContext interface {
	Next()
}

// RequestHandler .
type RequestHandler interface {
	Handle(HandleContext, http.ResponseWriter, *http.Request)
}

type handleContextImpl struct {
	next bool
}

func (ctx *handleContextImpl) Next() {
	ctx.next = true
}

// HTTPProxyHandler .
type HTTPProxyHandler struct {
	Handlers   []RequestHandler
	HTTPClient barrelHttp.Client
}

func (ph HTTPProxyHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Infof("[ComposedHttpHandler] Incoming request, method = %s, url = %s", req.Method, req.URL.String())
	utils.PrintHeaders("ServerRequestHeaders:", req.Header)

	for _, handler := range ph.Handlers {
		ctx := &handleContextImpl{}
		handler.Handle(ctx, res, req)
		if !ctx.next {
			return
		}
	}

	ph.proxy(res, req)
}

// Handle .
func (ph HTTPProxyHandler) proxy(response http.ResponseWriter, request *http.Request) {
	log.Info("[Handle] handle other docker request, will forward stream")
	var (
		resp *http.Response
		err  error
	)
	header := request.Header
	header.Add("Host", request.Host)
	// change host of request
	request.Host = "docker"
	if resp, err = ph.HTTPClient.Request(request); err != nil {
		log.Errorf("[dispatch] send request to docker socket error %v", err)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		log.Debug("[dispatch] Forward http response")
		if err := utils.Forward(resp, response); err != nil {
			log.Errorf("[dispatch] forward docker socket response failed %v", err)
		}
		return
	}
	linkConn(response, resp)
}

func linkConn(response http.ResponseWriter, resp *http.Response) {
	log.Debug("[linkConn] Will linked upgraded connection")
	// we will hijack connection and link with dockerd connection
	// test response writer could be hijacked
	if hijacker, ok := response.(http.Hijacker); ok {
		// test resp body is writable
		if readWriteCloser, ok := resp.Body.(io.ReadWriteCloser); ok {
			doLinkConn(response, resp, hijacker, readWriteCloser)
		} else {
			log.Error("[linkConn] Can't Write To ClientRequestBody")
			if err := utils.WriteBadGateWayResponse(
				response,
				utils.HTTPSimpleMessageResponseBody{
					Message: "Can't Write To ClientRequestBody",
				},
			); err != nil {
				log.Errorf("[linkConn] link conn failed %v", err)
			}
		}
		return
	}
	log.Error("[linkConn] can't Hijack ServerResponseWriter")
	if err := utils.WriteBadGateWayResponse(
		response,
		utils.HTTPSimpleMessageResponseBody{
			Message: "Can't Hijack ServerResponseWriter",
		},
	); err != nil {
		log.Errorf("[linkConn] write bad gateway response %v", err)
	}
}

func doLinkConn(response http.ResponseWriter, resp *http.Response, hijacker http.Hijacker, readWriteCloser io.ReadWriteCloser) {
	var err error
	// first we send response to non overrided client, make sure it's ready for new protocol
	if err = utils.WriteToServerResponse(
		response,
		http.StatusSwitchingProtocols,
		resp.Header,
		nil,
	); err != nil {
		log.Errorf("[doLinkConn] write StatusSwitchingProtocols failed %v", err)
		return
	}
	var conn net.Conn
	log.Debug("[doLinkConn] Hijack server http connection")
	if conn, _, err = hijacker.Hijack(); err != nil {
		log.Errorf("[doLinkConn] Hijack ServerResponseWriter failed %v", err)
		return
	}
	defer utils.Link(conn, readWriteCloser)
	// link client conn and server conn
	log.Debug("[doLinkConn] link connection")
}
