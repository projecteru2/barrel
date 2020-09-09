package utils

import (
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

var errChannelIsClosed = errors.New("channel is closed")

// WriteOnceChannel .
type WriteOnceChannel struct {
	actual chan error
	closed atomicBool
}

// NewWriteOnceChannel .
func NewWriteOnceChannel() WriteOnceChannel {
	return WriteOnceChannel{
		actual: make(chan error),
		closed: newAtomicBool(false),
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
