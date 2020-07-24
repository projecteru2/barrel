package proxy

import (
	"context"
	"net/http"
	"sync"

	"github.com/projecteru2/barrel/types"
)

func startHostGroup(handler http.Handler, hosts []types.Host) (err error) {
	errChan := types.NewChanWrapper(make(chan error))
	wg := sync.WaitGroup{}
	wg.Add(len(hosts))
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, host := range hosts {
		addr := host.Listener.Addr().String()
		server := types.HostServer{
			Ctx:     ctx,
			Wg:      &wg,
			ErrChan: errChan,
			Server: http.Server{
				Addr:    addr,
				Handler: handler,
			},
		}
		go server.Start(host)
	}
	return errChan.Listen()
}
