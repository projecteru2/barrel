package utils

import (
	"sync/atomic"
)

const (
	trueFlag  int32 = 1
	falseFlag int32 = 0
)

type atomicBool struct {
	// value must be either 1 or 0, otherwise is a UB
	value int32
}

func newAtomicBool(init bool) atomicBool {
	if init {
		return atomicBool{trueFlag}
	}
	return atomicBool{falseFlag}
}

// Set .
func (b *atomicBool) Set(value bool) {
	if value {
		atomic.StoreInt32(&b.value, trueFlag)
	} else {
		atomic.StoreInt32(&b.value, falseFlag)
	}
}

// Cas .
func (b *atomicBool) Cas(old bool, new bool) bool {
	var oldint = falseFlag
	if old {
		oldint = trueFlag
	}
	if new {
		return atomic.CompareAndSwapInt32(&b.value, oldint, trueFlag)
	}
	return atomic.CompareAndSwapInt32(&b.value, oldint, falseFlag)
}

// Get .
func (b *atomicBool) Get() bool {
	return atomic.LoadInt32(&b.value) == trueFlag
}
