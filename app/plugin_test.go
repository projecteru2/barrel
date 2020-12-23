package app

import (
	"context"
	"sync"
	"testing"
	"time"

	pluginIpam "github.com/docker/go-plugins-helpers/ipam"
	pluginNetwork "github.com/docker/go-plugins-helpers/network"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/projecteru2/barrel/driver/mocks"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

func TestTerminatePluginService(t *testing.T) {
	ipam := mocks.IpamMock{}
	driver := mocks.DriverMock{}
	server := mocks.PluginServer{}

	wg := sync.WaitGroup{}
	wg.Add(2)

	chIpam := make(chan time.Time)
	chDriver := make(chan time.Time)
	server.On("ServeIpam", mock.Anything).Return(func(pluginIpam.Ipam) error {
		wg.Done()
		<-chIpam
		return errors.New("ipam shutdown")
	})
	server.On("ServeNetwork", mock.Anything).Return(func(pluginNetwork.Driver) error {
		wg.Done()
		<-chDriver
		return errors.New("network driver shutdown")
	})

	service := pluginService{
		ipam:   &ipam,
		driver: &driver,
		server: &server,
	}

	ctx, cancel := context.WithCancel(context.Background())

	co := utils.Async(func() {
		disposable, err := service.Serve(ctx)
		assert.NotNil(t, disposable)
		assert.NoError(t, err)

		ctx := context.Background()
		err = disposable.Dispose(ctx)
		assert.Equal(t, types.ErrCannotDisposeService, err)
	})

	wg.Wait()
	cancel()

	co.Await()
	close(chIpam)
	close(chDriver)
}

func TestPluginIpamError(t *testing.T) {
	ipam := mocks.IpamMock{}
	driver := mocks.DriverMock{}
	server := mocks.PluginServer{}

	wg := sync.WaitGroup{}
	wg.Add(2)

	chIpam := make(chan time.Time)
	chDriver := make(chan time.Time)
	server.On("ServeIpam", mock.Anything).Return(func(pluginIpam.Ipam) error {
		wg.Done()
		<-chIpam
		return errors.New("ipam shutdown")
	})
	server.On("ServeNetwork", mock.Anything).Return(func(pluginNetwork.Driver) error {
		wg.Done()
		<-chDriver
		return errors.New("network driver shutdown")
	})

	service := pluginService{
		ipam:   &ipam,
		driver: &driver,
		server: &server,
	}

	ctx, cancel := context.WithCancel(context.Background())

	co := utils.Async(func() {
		disposable, err := service.Serve(ctx)
		assert.NotNil(t, disposable)
		assert.Error(t, err)

		ctx := context.Background()
		err = disposable.Dispose(ctx)
		assert.Equal(t, types.ErrCannotDisposeService, err)
	})

	wg.Wait()
	close(chIpam)

	co.Await()
	cancel()
	close(chDriver)
}

func TestPluginNetworkDriverError(t *testing.T) {
	ipam := mocks.IpamMock{}
	driver := mocks.DriverMock{}
	server := mocks.PluginServer{}

	wg := sync.WaitGroup{}
	wg.Add(2)

	chIpam := make(chan time.Time)
	chDriver := make(chan time.Time)
	server.On("ServeIpam", mock.Anything).Return(func(pluginIpam.Ipam) error {
		wg.Done()
		<-chIpam
		return errors.New("ipam shutdown")
	})
	server.On("ServeNetwork", mock.Anything).Return(func(pluginNetwork.Driver) error {
		wg.Done()
		<-chDriver
		return errors.New("network driver shutdown")
	})

	service := pluginService{
		ipam:   &ipam,
		driver: &driver,
		server: &server,
	}

	ctx, cancel := context.WithCancel(context.Background())

	co := utils.Async(func() {
		disposable, err := service.Serve(ctx)
		assert.NotNil(t, disposable)
		assert.Error(t, err)

		ctx := context.Background()
		err = disposable.Dispose(ctx)
		assert.Equal(t, types.ErrCannotDisposeService, err)
	})

	wg.Wait()
	close(chDriver)

	co.Await()
	cancel()
	close(chIpam)
}
