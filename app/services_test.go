package app

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"

	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

func testStarter(t *testing.T, disposeTimeout time.Duration, ss []service.Service) starter {
	return starter{
		logger:         utils.NewTestLogger(t),
		ss:             ss,
		disposeTimeout: disposeTimeout,
	}
}

type mockService struct {
	test           *testing.T
	wg             *sync.WaitGroup
	serveError     bool
	errorTimeout   time.Duration
	disposeTimeout time.Duration
	disposeError   bool
}

func (service mockService) Serve(ctx context.Context) (service.Disposable, error) {
	service.wg.Done()

	if service.serveError {
		select {
		case <-ctx.Done():
			return service, nil
		case <-time.After(service.disposeTimeout):
			service.test.Log("service error")
			return service, errors.New("service error")
		}
	}
	<-ctx.Done()
	return service, nil
}

func (service mockService) Dispose(ctx context.Context) error {
	if service.disposeError {
		select {
		case <-ctx.Done():
			return context.DeadlineExceeded
		case <-time.After(service.disposeTimeout):
			return types.ErrCannotDisposeService
		}
	}
	select {
	case <-ctx.Done():
		service.test.Log("dispose")
		return context.DeadlineExceeded
	case <-time.After(service.disposeTimeout):
		service.test.Log("dispose")
		return nil
	}
}

// Test start normal services until received an system term signal
func TestTerminateRunningServices(t *testing.T) {
	chSigs := make(chan os.Signal)
	defer close(chSigs)

	wg := sync.WaitGroup{}
	wg.Add(2)

	co := utils.Async(func() {
		// as we are cancelling services so we should receive no error
		err := testStarter(t, 0, []service.Service{
			mockService{
				wg:   &wg,
				test: t,
			},
			mockService{
				wg:   &wg,
				test: t,
			},
		}).start(chSigs)
		assert.NoError(t, err)
	})

	// wait services started
	wg.Wait()
	// terminate services
	chSigs <- syscall.SIGTERM
	co.Await()
}

// Test start normal services until received an system term signal
// however disposing service takes longer then expected
// so we shall expect an deadline exceeded error
func TestTerminateRunningServicesTimeout(t *testing.T) {
	chSigs := make(chan os.Signal)
	defer close(chSigs)

	wg := sync.WaitGroup{}
	wg.Add(2)

	co := utils.Async(func() {
		err := testStarter(t, time.Duration(1)*time.Second, []service.Service{
			mockService{
				test:           t,
				wg:             &wg,
				disposeTimeout: time.Duration(2) * time.Second,
			},
			mockService{
				test: t,
				wg:   &wg,
			},
		}).start(chSigs)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	// wait services started
	wg.Wait()
	// terminate services
	chSigs <- syscall.SIGTERM

	co.Await()
}

// Test start services among which there is an error service
// Disposing the service will be success
func TestErrorServices(t *testing.T) {
	chSigs := make(chan os.Signal)
	defer close(chSigs)

	wg := sync.WaitGroup{}
	wg.Add(2)

	co := utils.Async(func() {
		// as we are cancelling services so we should receive no error
		err := testStarter(t, time.Duration(1)*time.Second, []service.Service{
			mockService{
				test:         t,
				wg:           &wg,
				serveError:   true,
				errorTimeout: time.Duration(1) * time.Second,
			},
			mockService{
				test: t,
				wg:   &wg,
			},
		}).start(chSigs)
		assert.Error(t, err)
	})

	// wait services started
	wg.Wait()
	co.Await()
}

// Test start services among which there is an error service
// Disposing the service will timeout
func TestErrorServicesWithDisposeTimeout(t *testing.T) {
	chSigs := make(chan os.Signal)
	defer close(chSigs)

	wg := sync.WaitGroup{}
	wg.Add(2)

	co := utils.Async(func() {
		// as we are cancelling services so we should receive no error
		err := testStarter(t, time.Duration(1)*time.Second, []service.Service{
			mockService{
				test:           t,
				wg:             &wg,
				serveError:     true,
				errorTimeout:   time.Duration(1) * time.Second,
				disposeTimeout: time.Duration(2) * time.Second,
			},
			mockService{
				test: t,
				wg:   &wg,
			},
		}).start(chSigs)
		wraped, ok := err.(*errors.Err)
		assert.True(t, ok)
		assert.Equal(t, context.DeadlineExceeded, wraped.Cause())
	})

	// wait services started
	wg.Wait()
	co.Await()
}

// Test start services among which there is an error service
// Disposing the service will not be success
func TestErrorServicesWithDisposeUnsuccessful(t *testing.T) {
	chSigs := make(chan os.Signal)
	defer close(chSigs)

	wg := sync.WaitGroup{}
	wg.Add(2)

	co := utils.Async(func() {
		// as we are cancelling services so we should receive no error
		err := testStarter(t, time.Duration(1)*time.Second, []service.Service{
			mockService{
				test:         t,
				wg:           &wg,
				serveError:   true,
				errorTimeout: time.Duration(1) * time.Second,
				disposeError: true,
			},
			mockService{
				test: t,
				wg:   &wg,
			},
		}).start(chSigs)
		_, ok := err.(*errors.Err)
		assert.True(t, ok)
	})

	// wait services started
	wg.Wait()
	co.Await()
}
