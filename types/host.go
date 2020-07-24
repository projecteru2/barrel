package types

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/projecteru2/barrel/common"
	log "github.com/sirupsen/logrus"
)

// Host .
type Host struct {
	Listener net.Listener
	Cert     string
	Key      string
}

// HostServer .
type HostServer struct {
	Server  http.Server
	Ctx     context.Context
	Wg      *sync.WaitGroup
	ErrChan ErrChanWrapper
}

// Start .
func (server *HostServer) Start(host Host) {
	go server.waitDone()
	defer server.Wg.Done()
	if err := server.startOnHost(host); err != nil {
		server.ErrChan.Signal(err)
	} else {
		// we don't expect one of our host to stop
		server.ErrChan.Signal(common.ErrServerStop)
	}
}

func (server *HostServer) startOnHost(host Host) error {
	if host.Cert != "" {
		return server.Server.ServeTLS(host.Listener, host.Cert, host.Key)
	}
	return server.Server.Serve(host.Listener)
}

func (server *HostServer) waitDone() {
	<-server.Ctx.Done()
	if err := server.Server.Shutdown(context.Background()); err != nil { // TODO timeout ctx
		log.Errorf("[waitDone] wait done failed %v", err)
	}
}
