package utils

import (
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func WithNetns(netnsPath string, f func() error) (err error) {
	file, err := os.Open(netnsPath)
	if err != nil {
		return errors.WithStack(err)
	}

	origin, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return errors.WithStack(err)
	}

	if err = unix.Setns(int(file.Fd()), unix.CLONE_NEWNET); err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		if e := unix.Setns(int(origin.Fd()), unix.CLONE_NEWNET); e != nil {
			log.Errorf("failed to recover netns: %+v", e)
		}
	}()
	return f()
}
