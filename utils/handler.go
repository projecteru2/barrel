package utils

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// HandleContext .
type HandleContext interface {
	Next()
}

// RequestHandler .
type RequestHandler interface {
	Handle(HandleContext, http.ResponseWriter, *http.Request)
}

type httpRequestHandler struct {
	handlers []RequestHandler
}

func (handler httpRequestHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Infof("[ComposedHttpHandler] Incoming request, method = %s, url = %s", req.Method, req.URL.String())
	PrintHeaders("ServerRequestHeaders:", req.Header)

	for _, handler := range handler.handlers {
		ctx := handleContextImpl{}
		handler.Handle(&ctx, res, req)
		if !ctx.next {
			return
		}
	}
}

// ComposeHandlers .
func ComposeHandlers(handlers ...RequestHandler) http.Handler {
	return httpRequestHandler{handlers: handlers}
}

type handleContextImpl struct {
	next bool
}

func (ctx *handleContextImpl) Next() {
	ctx.next = true
}
