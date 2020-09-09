package proxy

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/sockets"
	"github.com/juju/errors"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
)

const (
	unixPrefix  = "unix://"
	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

// HostLauncher .
type HostLauncher struct {
	dockerGid int64
	certFile  string
	keyFile   string
	handler   http.Handler
}

// Launch .
func (launcher *HostLauncher) Launch(address string) (service.DisposableService, error) {
	var (
		server http.Server = http.Server{
			Handler: launcher.handler,
		}
		service func() error
		err     error
		dispose = func(_ bool) error {
			return server.Shutdown(context.Background())
		}
	)
	if service, err = launcher.newHost(&server, address); err != nil {
		return nil, err
	}
	return newDisposableService(service, dispose), nil
}

// LaunchMultiple .
func (launcher *HostLauncher) LaunchMultiple(addresses ...string) (service.DisposableService, error) {
	var (
		server http.Server = http.Server{
			Handler: launcher.handler,
		}
		services []func() error
		err      error
		dispose  = func(_ bool) error {
			return server.Shutdown(context.Background())
		}
	)
	for _, address := range addresses {
		var service func() error
		if service, err = launcher.newHost(&server, address); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return newDisposableService(func() error {
		ch := utils.NewWriteOnceChannel()
		for _, serv := range services {
			go func(service func() error) {
				var err error
				if err = service(); err != nil {
					log.Errorf("[HostLauncher::LaunchMultiple] service encountered error, %v", err)
				}
				ch.Send(err)
			}(serv)
		}
		err := ch.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("[HostLauncher::LaunchMultiple] shutdown server error, %v", err)
		}
		return err
	}, dispose), nil
}

func (launcher *HostLauncher) newHost(server *http.Server, address string) (service func() error, err error) {
	switch {
	case strings.HasPrefix(address, unixPrefix):
		return launcher.newUnixHost(server, strings.TrimPrefix(address, unixPrefix))
	case strings.HasPrefix(address, httpPrefix):
		return launcher.newHTTPHost(server, strings.TrimPrefix(address, httpPrefix))
	case strings.HasPrefix(address, httpsPrefix):
		return launcher.newHTTPSHost(server, strings.TrimPrefix(address, httpsPrefix))
	}
	return nil, errors.Errorf("unsupported protocol schema %s", address)
}

func (launcher *HostLauncher) newUnixHost(server *http.Server, address string) (func() error, error) {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = sockets.NewUnixSocket(address, int(launcher.dockerGid)); err != nil {
		return nil, err
	}
	return func() error {
		return server.Serve(listener)
	}, nil
}

func (launcher *HostLauncher) newHTTPHost(server *http.Server, address string) (func() error, error) {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", address); err != nil {
		return nil, err
	}
	return func() error {
		return server.Serve(listener)
	}, nil
}

func (launcher *HostLauncher) newHTTPSHost(server *http.Server, address string) (func() error, error) {
	if launcher.certFile == "" || launcher.keyFile == "" {
		return nil, types.ErrCertAndKeyMissing
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", address); err != nil {
		return nil, err
	}
	return func() error {
		return server.ServeTLS(listener, launcher.certFile, launcher.keyFile)
	}, nil
}
