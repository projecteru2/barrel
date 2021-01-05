package app

import (
	"context"
	"strings"

	"github.com/juju/errors"

	barrelHttp "github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/utils"
)

const (
	unixPrefix  = "unix://"
	httpPrefix  = "http://"
	httpsPrefix = "https://"
)

type proxyService struct {
	barrelHttp.Server
	gid       int
	tlsConfig barrelHttp.TLSConfig
	hosts     []string
}

func (service proxyService) Serve(ctx context.Context) (service.Disposable, error) {
	ch := make(chan error)
	atomicBool := utils.NewAtomicBool(true)

	for _, host := range service.hosts {
		go func(addr string) {
			err := service.serveHost(addr)
			if atomicBool.Get() && atomicBool.Cas(true, false) {
				ch <- err
				close(ch)
			}
		}(host)
	}

	select {
	case err := <-ch:
		return service, err
	case <-ctx.Done():
		return service, nil
	}
}

func (service proxyService) Dispose(ctx context.Context) error {
	return service.Close(ctx)
}

func (service proxyService) serveHost(address string) error {
	if strings.HasPrefix(address, unixPrefix) {
		return service.ServeUnix(strings.TrimPrefix(address, unixPrefix), service.gid)
	}
	if strings.HasPrefix(address, httpPrefix) {
		return service.ServeHTTP(strings.TrimPrefix(address, httpPrefix))
	}
	if strings.HasPrefix(address, httpsPrefix) {
		return service.ServeHTTPS(strings.TrimPrefix(address, httpsPrefix), service.tlsConfig)
	}

	return errors.Errorf("unsupported protocol schema %s", address)
}
