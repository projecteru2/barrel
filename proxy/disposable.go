package proxy

import (
	"sync"

	"github.com/pkg/errors"
)

// Disposable .
type Disposable interface {
	Dispose() error
}

// DisposableService .
type DisposableService interface {
	Disposable
	// will block until finishes or error encoutered
	Service() error
}

// ErrDisposeCalledMoreThenOnce .
var ErrDisposeCalledMoreThenOnce = errors.New("Should not call dispose more then once")

// ErrServiceCalledMoreThenOnce .
var ErrServiceCalledMoreThenOnce = errors.New("Should not call service more then once")

// ErrServiceDisposedBeforeStart .
var ErrServiceDisposedBeforeStart = errors.New("Service disposed before start")

// ErrServiceDisposabled represents a serivce is disposed
type ErrServiceDisposabled struct {
	Err error
}

func (err ErrServiceDisposabled) Error() string {
	if err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

type closureWrapperDisposable struct {
	dispose         func(bool) error
	service         func() error
	serviceIsCalled bool
	disposeIsCalled bool
	mutex           sync.Mutex
}

// NewDisposableService .
// we ensure if dispose is called before service, then service will not be called
// if dispose is called after service, then value of its args is true
func NewDisposableService(service func() error, dispose func(bool) error) DisposableService {
	return &closureWrapperDisposable{service: service, dispose: dispose}
}

func (wrapper *closureWrapperDisposable) Service() error {
	if err := wrapper.checkBeforeStart(); err != nil {
		return err
	}
	// there is a chance dispose is called before service, but handle this scene is upto the caller
	err := wrapper.service()

	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()

	if wrapper.disposeIsCalled {
		return ErrServiceDisposabled{err}
	}
	return err
}

func (wrapper *closureWrapperDisposable) checkBeforeStart() error {
	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()
	if wrapper.serviceIsCalled {
		return ErrServiceCalledMoreThenOnce
	}
	wrapper.serviceIsCalled = true
	if wrapper.disposeIsCalled {
		return ErrServiceDisposabled{ErrServiceDisposedBeforeStart}
	}
	return nil
}

func (wrapper *closureWrapperDisposable) Dispose() (err error) {
	var serviceIsCalled bool
	if serviceIsCalled, err = wrapper.checkBeforeDispose(); err != nil {
		return err
	}
	return wrapper.dispose(serviceIsCalled)
}

func (wrapper *closureWrapperDisposable) checkBeforeDispose() (bool, error) {
	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()

	if wrapper.disposeIsCalled {
		return wrapper.serviceIsCalled, ErrDisposeCalledMoreThenOnce
	}
	wrapper.disposeIsCalled = true
	return wrapper.serviceIsCalled, nil
}
