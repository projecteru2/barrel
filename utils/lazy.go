package utils

import (
	"sync"
)

// NewCachedGetter .
func NewCachedGetter(getter func() (interface{}, error)) Getter {
	return &cachedGetter{
		getter: getter,
	}
}

// NewLazyGetter .
func NewLazyGetter(getter func() (interface{}, error)) Getter {
	return &lazyGetter{
		cached: cachedGetter{
			getter: getter,
		},
	}
}

// Getter .
type Getter interface {
	Get() (interface{}, error)
}

type cachedGetter struct {
	hasCached bool
	cached    interface{}
	err       error
	getter    func() (interface{}, error)
}

func (getter *cachedGetter) Get() (interface{}, error) {
	if getter.hasCached {
		return getter.cached, getter.err
	}

	getter.cached, getter.err = getter.getter()
	return getter.cached, getter.err
}

type lazyGetter struct {
	sync.Mutex
	cached cachedGetter
}

func (getter *lazyGetter) Get() (interface{}, error) {
	if getter.cached.hasCached {
		return getter.cached.Get()
	}

	getter.Lock()
	defer getter.Unlock()

	return getter.cached.Get()
}
