package utils

import (
	"sync/atomic"
)

const (
	trueFlag  int32 = 1
	falseFlag int32 = 0
)

// AtomicBool .
type AtomicBool struct {
	// value must be either 1 or 0, otherwise is a UB
	value int32
}

// NewAtomicBool .
func NewAtomicBool(init bool) AtomicBool {
	if init {
		return AtomicBool{trueFlag}
	}
	return AtomicBool{falseFlag}
}

// Set .
func (b *AtomicBool) Set(value bool) {
	if value {
		atomic.StoreInt32(&b.value, trueFlag)
	} else {
		atomic.StoreInt32(&b.value, falseFlag)
	}
}

// Cas .
func (b *AtomicBool) Cas(old bool, new bool) bool {
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
func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&b.value) == trueFlag
}

// AtomicInt64 .
type AtomicInt64 struct {
	value int64
}

// Get .
func (i *AtomicInt64) Get() int64 {
	return atomic.LoadInt64(&i.value)
}

// GetInt .
func (i *AtomicInt64) GetInt() int {
	return int(atomic.LoadInt64(&i.value))
}

// Set .
func (i *AtomicInt64) Set(value int64) {
	atomic.StoreInt64(&i.value, value)
}

// Inc .
func (i *AtomicInt64) Inc() int64 {
	return atomic.AddInt64(&i.value, 1)
}

// Add .
func (i *AtomicInt64) Add(value int64) int64 {
	return atomic.AddInt64(&i.value, value)
}
