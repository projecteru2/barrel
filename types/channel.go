package types

import (
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

const (
	trueFlag  int32 = 1
	falseFalg int32 = 0
)

// ErrChanWrapper .
type ErrChanWrapper struct {
	c        chan error
	writable *atomicBool
}

// NewChanWrapper .
func NewChanWrapper(c chan error) ErrChanWrapper {
	b := new(atomicBool)
	b.set(true)
	return ErrChanWrapper{
		c:        c,
		writable: b,
	}
}

// Listen .
func (wrapper ErrChanWrapper) Listen() (err error) {
	err = <-wrapper.c
	close(wrapper.c)
	return
}

// Signal .
func (wrapper ErrChanWrapper) Signal(err error) {
	if wrapper.writable.get() {
		if wrapper.writable.cas(true, false) {
			wrapper.c <- err
			return
		}
	}
	log.Errorf("[Signal] Error not signaled: %v", err)
}

// value must be either 1 or 0, otherwise is a UB
type atomicBool struct {
	value int32
}

func (b *atomicBool) set(value bool) {
	if value {
		atomic.StoreInt32(&b.value, trueFlag)
	} else {
		atomic.StoreInt32(&b.value, falseFalg)
	}
}

func (b *atomicBool) cas(old bool, new bool) bool {
	var oldint int32
	if old {
		oldint = trueFlag
	}
	if new {
		return atomic.CompareAndSwapInt32(&b.value, oldint, trueFlag)
	}
	return atomic.CompareAndSwapInt32(&b.value, oldint, falseFalg)
}

func (b *atomicBool) get() bool {
	return atomic.LoadInt32(&b.value) == trueFlag
}
