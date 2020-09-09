package handler

import "net/http"

// Context .
type Context interface {
	Next()
}

// RequestHandler .
type RequestHandler interface {
	Handle(Context, http.ResponseWriter, *http.Request)
}
