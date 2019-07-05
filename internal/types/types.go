package types

import (
	"net/http"
)

// RequestHandler .
type RequestHandler interface {
	Handle(http.ResponseWriter, *http.Request) bool
}
