package app

import (
	"context"

	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
	pluginNetwork "github.com/docker/go-plugins-helpers/network"

	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

type pluginService struct {
	ipam   pluginIpam.Ipam
	driver pluginNetwork.Driver
	server driver.PluginServer
}

func (service pluginService) Serve(ctx context.Context) (service.Disposable, error) {
	chErr := utils.NewAutoCloseChanErr(2)

	go func() {
		if err := service.server.ServeIpam(service.ipam); err != nil {
			chErr.Send(err)
			return
		}
		chErr.Send(types.ErrServiceShutdown)
	}()

	go func() {
		if err := service.server.ServeNetwork(service.driver); err != nil {
			chErr.Send(err)
			return
		}
		chErr.Send(types.ErrServiceShutdown)
	}()

	select {
	case <-ctx.Done():
		return service, nil
	case err := <-chErr.Receive():
		return service, err
	}
}

func (service pluginService) Dispose(ctx context.Context) error {
	return types.ErrCannotDisposeService
}
