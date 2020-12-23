package http

import (
	"context"
	"net"
	"net/http"

	"github.com/docker/go-connections/sockets"
)

// TLSConfig .
type TLSConfig struct {
	CertFile string
	KeyFile  string
}

// Server .
type Server interface {
	ServeHTTP(string) error
	ServeHTTPS(string, TLSConfig) error
	ServeUnix(string, int) error
	Close(context.Context) error
	CloseAsync(func(error))
}

type httpServer struct {
	http.Server
}

// NewServer .
func NewServer(handler http.Handler) Server {
	return &httpServer{
		Server: http.Server{
			Handler: handler,
		},
	}
}

func (server *httpServer) ServeUnix(address string, gid int) error {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = sockets.NewUnixSocket(address, gid); err != nil {
		return err
	}
	return server.Serve(listener)
}

func (server *httpServer) ServeHTTP(address string) error {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", address); err != nil {
		return err
	}
	return server.Serve(listener)
}

func (server *httpServer) ServeHTTPS(address string, config TLSConfig) error {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", address); err != nil {
		return err
	}
	return server.ServeTLS(listener, config.CertFile, config.KeyFile)
}

func (server *httpServer) Close(ctx context.Context) error {
	return server.Shutdown(ctx)
}

func (server *httpServer) CloseAsync(cb func(error)) {
	go func() {
		cb(server.Shutdown(context.Background()))
	}()
}
