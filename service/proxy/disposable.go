package proxy

import (
	"sync"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/service"
)

var (
	errDisposeCalledMoreThenOnce  = errors.New("Should not call dispose more then once")
	errServiceCalledMoreThenOnce  = errors.New("Should not call service more then once")
	errServiceDisposedBeforeStart = errors.New("Service disposed before start")
)

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
func newDisposableService(service func() error, dispose func(bool) error) service.DisposableService {
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
		return errServiceDisposabled{err}
	}
	return err
}

func (wrapper *closureWrapperDisposable) checkBeforeStart() error {
	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()
	if wrapper.serviceIsCalled {
		return errServiceCalledMoreThenOnce
	}
	wrapper.serviceIsCalled = true
	if wrapper.disposeIsCalled {
		return errServiceDisposabled{errServiceDisposedBeforeStart}
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
		return wrapper.serviceIsCalled, errDisposeCalledMoreThenOnce
	}
	wrapper.disposeIsCalled = true
	return wrapper.serviceIsCalled, nil
}

type errServiceDisposabled struct {
	Err error
}

func (err errServiceDisposabled) Error() string {
	if err.Err == nil {
		return ""
	}
	return err.Err.Error()
}
