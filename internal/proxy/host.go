package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Host .
type Host struct {
	Listener net.Listener
	Cert     string
	Key      string
}

type _HostServer struct {
	server  http.Server
	ctx     context.Context
	wg      *sync.WaitGroup
	errChan _ErrChanWrapper
}

func startHostGroup(mux *http.ServeMux, hosts []Host) (err error) {
	errChan := _NewChanWrapper(make(chan error))
	wg := sync.WaitGroup{}
	wg.Add(len(hosts))
	ctx, cancel := context.WithCancel(context.Background())
	for _, host := range hosts {
		addr := host.Listener.Addr().String()
		log.Infof("Starting proxy at %s", addr)
		server := _HostServer{
			ctx:     ctx,
			wg:      &wg,
			errChan: errChan,
			server: http.Server{
				Addr:    addr,
				Handler: mux,
			},
		}
		go server.start(host)
	}
	err = errChan.listen()
	log.Error(err)
	cancel()
	wg.Wait()
	return
}

func (server *_HostServer) start(host Host) {
	go server.waitDone()
	if err := server.startOnHost(host); err != nil {
		server.errChan.signal(err)
	} else {
		// we don't expect one of our host to stop
		server.errChan.signal(errors.New("Server Stops"))
	}
	server.wg.Done()
}

func (server *_HostServer) startOnHost(host Host) error {
	if host.Cert != "" {
		return server.server.ServeTLS(host.Listener, host.Cert, host.Key)
	}
	return server.server.Serve(host.Listener)
}

func (server *_HostServer) waitDone() {
	<-server.ctx.Done()
	if err := server.server.Shutdown(context.Background()); err != nil {
		log.Error(err)
	}
}
