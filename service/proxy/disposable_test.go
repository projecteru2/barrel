package proxy

import (
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/service"
	"github.com/stretchr/testify/assert"
)

func TestDisablableClosure(t *testing.T) {
	var disposable service.DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.True(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=true")
			return nil
		},
		service: func() error {
			return nil
		},
	}
	assert.Nil(t, disposable.Service(), "service of given case return no error")
	assert.Equal(
		t,
		disposable.Service(),
		errServiceCalledMoreThenOnce,
		"[DisposableService::Service] must return an ErrServiceCalledMoreThenOnce error when called more then once",
	)
	assert.Nil(t, disposable.Dispose(), "dispose of given case return no error")
	assert.Equal(
		t,
		disposable.Dispose(),
		errDisposeCalledMoreThenOnce,
		"[DisposableService::Dispose] must return an errDisposeCalledMoreThenOnce error when called more then once",
	)
}

func TestClosureDisposedBeforeStart(t *testing.T) {
	err := errors.New("dispose error")
	var disposable service.DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.False(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=false")
			return err
		},
		service: func() error {
			return nil
		},
	}
	assert.Equal(t, disposable.Dispose(), err, "dispose of given case return with given error")
	if err, ok := disposable.Service().(errServiceDisposabled); !ok {
		assert.NotNil(t, err, "[DisposableService::Service] given case returns an errServiceDisposabled error")
	} else {
		assert.Equal(t, err.Err, errServiceDisposedBeforeStart, "service of given case return wrapped ErrServiceDisposedBeforeStart error")
	}
}

func TestClosureServiceEndWithoutError(t *testing.T) {
	var disposable service.DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.True(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=true")
			return nil
		},
		service: func() error {
			time.Sleep(time.Duration(1000) * time.Millisecond)
			return nil
		},
	}
	assert.Nil(t, disposable.Service(), "service of given case return no error")
}

func TestClosureServiceEndWithDisposedError(t *testing.T) {
	serviceError := errors.New("service error")
	var disposable service.DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.True(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=true")
			return nil
		},
		service: func() error {
			time.Sleep(time.Duration(1000) * time.Millisecond)
			return serviceError
		},
	}
	go func() {
		time.Sleep(time.Duration(100) * time.Millisecond)
		assert.Nil(t, disposable.Dispose(), "dispose of given case return with given error")
	}()
	if err, ok := disposable.Service().(errServiceDisposabled); !ok {
		t.Error("service of given case return errServiceDisposabled error")
	} else {
		assert.Equal(t, err.Err, serviceError, "service of given case return wrapped specified error")
	}
}

func TestClosureServiceEndWithError(t *testing.T) {
	err := errors.New("service error")
	var disposable service.DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.True(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=true")
			return err
		},
		service: func() error {
			return err
		},
	}
	assert.Equal(t, disposable.Service(), err, "service of given case return specified error")
	assert.Equal(t, disposable.Dispose(), err, "dispose of given case return specified error")
}
