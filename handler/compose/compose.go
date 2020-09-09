package compose

import (
	"net/http"

	"github.com/projecteru2/barrel/handler"
	"github.com/projecteru2/barrel/utils"

	log "github.com/sirupsen/logrus"
)

// Handlers .
func Handlers(handlers ...handler.RequestHandler) http.Handler {
	return httpRequestHandler{handlers: handlers}
}

type httpRequestHandler struct {
	handlers []handler.RequestHandler
}

func (handler httpRequestHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	log.Infof("[ComposedHttpHandler] Incoming request, method = %s, url = %s", req.Method, req.URL.String())
	utils.PrintHeaders("ServerRequestHeaders:", req.Header)

	for _, handler := range handler.handlers {
		ctx := &handleContextImpl{}
		handler.Handle(ctx, res, req)
		if !ctx.next {
			return
		}
	}
}

type handleContextImpl struct {
	next bool
}

func (ctx *handleContextImpl) Next() {
	ctx.next = true
}
