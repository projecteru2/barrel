package utils

import (
	"sync"
	"testing"
	"time"

	"github.com/juju/errors"
)

func TestWriteOnceChannel(test *testing.T) {
	test.Run("TestWait", func(t *testing.T) {
		writeOnceChannel := NewWriteOnceChannel()
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			// sleeep for 50 ms then send first error
			time.Sleep(time.Duration(50) * time.Millisecond)
			writeOnceChannel.Send(errors.New("test1"))
			// wake up goroutine two
			wg.Done()
		}()

		go func() {
			// wait untile goroutine one ends, then send another error
			wg.Wait()
			writeOnceChannel.Send(errors.New("test2"))
		}()
		// wait for channel and ensure it equals to error one
		if err := writeOnceChannel.Wait(); err == nil {
			t.Error("failed to receive an error")
		} else if err.Error() != "test1" {
			t.Error("failed to receive the first error")
		}
		if err := writeOnceChannel.Wait(); errors.Cause(err) != errChannelIsClosed {
			t.Error("failed to receive an error from closed write once channel")
		}
	})
}
