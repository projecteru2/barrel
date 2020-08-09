package utils

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ErrChannelIsClosed .
var ErrChannelIsClosed = errors.New("channel is closed")

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

// Wait .
func (ch *WriteOnceChannel) Wait() (err error) {
	var ok bool
	if err, ok = <-ch.actual; !ok {
		err = ErrChannelIsClosed
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
