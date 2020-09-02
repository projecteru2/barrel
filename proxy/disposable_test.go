package proxy

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestDisablableClosure(t *testing.T) {
	var disposable DisposableService = &closureWrapperDisposable{
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
		ErrServiceCalledMoreThenOnce,
		"[DisposableService::Service] must return an ErrServiceCalledMoreThenOnce error when called more then once",
	)
	assert.Nil(t, disposable.Dispose(), "dispose of given case return no error")
	assert.Equal(
		t,
		disposable.Dispose(),
		ErrDisposeCalledMoreThenOnce,
		"[DisposableService::Dispose] must return an ErrDisposeCalledMoreThenOnce error when called more then once",
	)
}

func TestClosureDisposedBeforeStart(t *testing.T) {
	err := errors.New("dispose error")
	var disposable DisposableService = &closureWrapperDisposable{
		dispose: func(serviceCalled bool) error {
			assert.False(t, serviceCalled, "dispose lambda of the given case is called with serviceCalled=false")
			return err
		},
		service: func() error {
			return nil
		},
	}
	assert.Equal(t, disposable.Dispose(), err, "dispose of given case return with given error")
	if err, ok := disposable.Service().(ErrServiceDisposabled); !ok {
		assert.NotNil(t, err, "[DisposableService::Service] given case returns an ErrServiceDisposabled error")
	} else {
		assert.Equal(t, err.Err, ErrServiceDisposedBeforeStart, "service of given case return wrapped ErrServiceDisposedBeforeStart error")
	}
}

func TestClosureServiceEndWithoutError(t *testing.T) {
	var disposable DisposableService = &closureWrapperDisposable{
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
	var disposable DisposableService = &closureWrapperDisposable{
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
	if err, ok := disposable.Service().(ErrServiceDisposabled); !ok {
		t.Error("service of given case return ErrServiceDisposabled error")
	} else {
		assert.Equal(t, err.Err, serviceError, "service of given case return wrapped specified error")
	}
}

func TestClosureServiceEndWithError(t *testing.T) {
	err := errors.New("service error")
	var disposable DisposableService = &closureWrapperDisposable{
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
