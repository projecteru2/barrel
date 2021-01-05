package http

import "net/http"

// Client .
type Client interface {
	// Request .
	Request(*http.Request) (*http.Response, error)
}
