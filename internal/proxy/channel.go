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

type _AtomicBool struct {
	value int32
}

func (b *_AtomicBool) set(value bool) {
	if value {
		atomic.StoreInt32(&b.value, 1)
	} else {
		atomic.StoreInt32(&b.value, 0)
	}
}

func (b *_AtomicBool) cas(old bool, new bool) bool {
	var oldint int32
	if old {
		oldint = 1
	}
	if new {
		return atomic.CompareAndSwapInt32(&b.value, oldint, 1)
	} else {
		return atomic.CompareAndSwapInt32(&b.value, oldint, 0)
	}
}

func (b *_AtomicBool) get() bool {
	return atomic.LoadInt32(&b.value) > 0
}
