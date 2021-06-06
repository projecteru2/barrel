package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni/store"
)

// LinuxFlock is an implementation of Flock
type LinuxFlock struct {
	pathname string
	fd       int
}

// Lock .
func (f *LinuxFlock) Lock() (err error) {
	return errors.WithStack(syscall.Flock(f.fd, syscall.LOCK_EX))
}

// Unlock .
func (f *LinuxFlock) Unlock() (err error) {
	defer syscall.Close(f.fd)
	return errors.WithStack(syscall.Flock(f.fd, syscall.LOCK_UN))
}

// GetFlock .
func (s FSStore) GetFlock(ip string) (flock store.Flock, err error) {
	f, err := os.OpenFile(s.flockPath(ip), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		if f, err = os.Open(s.flockPath(ip)); err != nil {
			return nil, errors.WithStack(err)
		}
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return &LinuxFlock{
		pathname: s.flockPath(ip),
		fd:       int(f.Fd()),
	}, nil
}
