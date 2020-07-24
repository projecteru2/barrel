package proxy

import (
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

type _ErrChanWrapper struct {
	c        chan error
	writable *_AtomicBool
}

func _NewChanWrapper(c chan error) _ErrChanWrapper {
	b := new(_AtomicBool)
	b.set(true)
	return _ErrChanWrapper{
		c:        c,
		writable: b,
	}
}

func (wrapper _ErrChanWrapper) signal(err error) {
	if wrapper.writable.get() {
		if wrapper.writable.cas(true, false) {
			wrapper.c <- err
			return
		}
	}
	log.Error("Error not signaled: ", err)
}

func (wrapper _ErrChanWrapper) listen() (err error) {
	err = <-wrapper.c
	close(wrapper.c)
	return
}

const (
	_True  int32 = 1
	_False int32 = 0
)

// value must be either 1 or 0, otherwise is a UB
type _AtomicBool struct {
	value int32
}

func _NewAtomicBool(value bool) _AtomicBool {
	if value {
		return _AtomicBool{_True}
	}
	return _AtomicBool{_False}
}

func _NewSharedAtomicBool(value bool) (b *_AtomicBool) {
	b = new(_AtomicBool)
	b.set(value)
	return
}

func (b *_AtomicBool) set(value bool) {
	if value {
		atomic.StoreInt32(&b.value, _True)
	} else {
		atomic.StoreInt32(&b.value, _False)
	}
}

func (b *_AtomicBool) cas(old bool, new bool) bool {
	var oldint int32
	if old {
		oldint = _True
	}
	if new {
		return atomic.CompareAndSwapInt32(&b.value, oldint, _True)
	} else {
		return atomic.CompareAndSwapInt32(&b.value, oldint, _False)
	}
}

func (b *_AtomicBool) get() bool {
	return atomic.LoadInt32(&b.value) == _True
}
