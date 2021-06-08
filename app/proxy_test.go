package app

import (
	"context"
	"io/fs"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/projecteru2/barrel/http"
	"github.com/projecteru2/barrel/http/mocks"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/utils/os"
	osMocks "github.com/projecteru2/barrel/utils/os/mocks"
)

func TestTerminateProxyService(t *testing.T) {
	mockOS := &osMocks.OS{}
	cancel := os.Mock(mockOS)
	defer cancel()

	mockOS.On("Stat", mock.Anything).Return(func(file string) fs.FileInfo {
		return os.FileInfo{}
	}, func(file string) error {
		return nil
	})

	server := mocks.Server{}
	servicesLaunched := sync.WaitGroup{}
	servicesLaunched.Add(3)

	chHTTP := make(chan time.Time)
	chHTTPS := make(chan time.Time)
	chUnix := make(chan time.Time)
	defer func() {
		close(chHTTP)
		close(chHTTPS)
		close(chUnix)
	}()
	notifyEnd := func() {
		chHTTP <- time.Now()
		chHTTPS <- time.Now()
		chUnix <- time.Now()
	}
	server.On("ServeHTTP", mock.Anything).Return(func(string) error {
		servicesLaunched.Done()
		<-chHTTP
		return types.ErrServiceShutdown
	})
	server.On("ServeHTTPS", mock.Anything, mock.Anything).Return(func(string, http.TLSConfig) error {
		servicesLaunched.Done()
		<-chHTTPS
		return types.ErrServiceShutdown
	})
	server.On("ServeUnix", mock.Anything, mock.Anything).Return(func(string, int) error {
		servicesLaunched.Done()
		<-chUnix
		return types.ErrServiceShutdown
	})
	server.On("Close", mock.Anything).Run(func(mock.Arguments) {
		notifyEnd()
	}).Return(nil)
	server.On("CloseAsync", mock.Anything).Run(func(args mock.Arguments) {
		notifyEnd()
		callback, ok := args.Get(0).(func(error))
		assert.True(t, ok)
		if ok {
			callback(nil)
		}
	})

	service := proxyService{
		Server: &server,
		gid:    1,
		tlsConfig: http.TLSConfig{
			CertFile: "/etc/eru/barrel/cert.ca",
			KeyFile:  "/etc/eru/barrel/key.ca",
		},
		hosts: []string{
			"unix:///var/run/barrel.sock",
			"http://127.0.0.1:80",
			"https://127.0.0.1:443",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	co := utils.Async(func() {
		disposable, err := service.Serve(ctx)
		assert.NotNil(t, disposable)
		assert.NoError(t, err)

		ctx := context.Background()
		err = disposable.Dispose(ctx)
		assert.NoError(t, err)
	})

	servicesLaunched.Wait()
	cancel()

	co.Await()
}

// func TestWrongProxyHost(t *testing.T) {
// 	// TODO
// }

// func TestProxyServiceError(t *testing.T) {
// 	// TODO
// }

func doTestProxyServiceError(t *testing.T) {
	server := mocks.Server{}
	servicesLaunched := sync.WaitGroup{}
	servicesLaunched.Add(3)

	chHTTP := make(chan time.Time)
	chHTTPS := make(chan time.Time)
	chUnix := make(chan time.Time)
	defer func() {
		close(chHTTP)
		close(chHTTPS)
		close(chUnix)
	}()
	notifyEnd := func() {
		chHTTP <- time.Now()
		chHTTPS <- time.Now()
		chUnix <- time.Now()
	}
	server.On("ServeHTTP", mock.Anything).Return(func(string) error {
		servicesLaunched.Done()
		<-chHTTP
		return types.ErrServiceShutdown
	})
	server.On("ServeHTTPS", mock.Anything, mock.Anything).Return(func(string, http.TLSConfig) error {
		servicesLaunched.Done()
		<-chHTTPS
		return types.ErrServiceShutdown
	})
	server.On("ServeUnix", mock.Anything, mock.Anything).Return(func(string, int) error {
		servicesLaunched.Done()
		<-chUnix
		return types.ErrServiceShutdown
	})
	server.On("Close", mock.Anything).Run(func(mock.Arguments) {
		notifyEnd()
	}).Return(nil)
	server.On("CloseAsync", mock.Anything).Run(func(args mock.Arguments) {
		notifyEnd()
		callback, ok := args.Get(0).(func(error))
		assert.True(t, ok)
		if ok {
			callback(nil)
		}
	})

	service := proxyService{
		Server:    &server,
		gid:       1,
		tlsConfig: http.TLSConfig{},
		hosts: []string{
			"unix:///var/run/barrel.sock",
			"http://127.0.0.1:80",
			"https://127.0.0.1:443",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	co := utils.Async(func() {
		disposable, err := service.Serve(ctx)
		assert.NotNil(t, disposable)
		assert.NoError(t, err)

		ctx := context.Background()
		err = disposable.Dispose(ctx)
		assert.NoError(t, err)
	})

	servicesLaunched.Wait()
	cancel()

	co.Await()
}
