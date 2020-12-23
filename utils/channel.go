package utils

import (
	"sync"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

var errChannelIsClosed = errors.New("channel is closed")

// WriteOnceChannel .
type WriteOnceChannel struct {
	actual chan error
	closed AtomicBool
}

// NewWriteOnceChannel .
func NewWriteOnceChannel() WriteOnceChannel {
	return WriteOnceChannel{
		actual: make(chan error),
		closed: NewAtomicBool(false),
	}
}

// Close .
func (ch *WriteOnceChannel) Close() {
	if !ch.closed.Get() {
		if ch.closed.Cas(false, true) {
			close(ch.actual)
		}
	}
}

// Wait .
func (ch *WriteOnceChannel) Wait() (err error) {
	var ok bool
	if err, ok = <-ch.actual; !ok {
		err = errors.Wrap(err, errChannelIsClosed)
	}
	return
}

// Send .
func (ch *WriteOnceChannel) Send(err error) {
	if !ch.closed.Get() {
		if ch.closed.Cas(false, true) {
			ch.actual <- err
			close(ch.actual)
			return
		}
	}
	log.Errorf("[Signal] Error not signaled: %v", err)
}

// AutoCloseChanErr .
type AutoCloseChanErr struct {
	ch     chan error
	limit  int
	cnt    int
	mutex  sync.Mutex
	closed bool
}

// NewAutoCloseChanErr .
func NewAutoCloseChanErr(limit int) AutoCloseChanErr {
	return AutoCloseChanErr{
		ch:    make(chan error),
		limit: limit,
	}
}

// Receive .
func (ch *AutoCloseChanErr) Receive() <-chan error {
	return ch.ch
}

// Send .
func (ch *AutoCloseChanErr) Send(err error) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	if ch.closed {
		return
	}

	if err != nil {
		ch.ch <- err
		close(ch.ch)
		ch.closed = true
		return
	}
	ch.cnt++
	if ch.cnt >= ch.limit {
		close(ch.ch)
		ch.closed = true
	}
}
