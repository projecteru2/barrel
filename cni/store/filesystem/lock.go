package filesystem

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni/store"
)

type LinuxFlock struct {
	pathname string
	fd       int
}

func (f *LinuxFlock) Lock() (err error) {
	return errors.WithStack(syscall.Flock(f.fd, syscall.LOCK_EX))
}

func (f *LinuxFlock) Unlock() (err error) {
	defer syscall.Close(f.fd)
	return errors.WithStack(syscall.Flock(f.fd, syscall.LOCK_UN))
}

func (s FSStore) GetFlock(ip string) (flock store.Flock, err error) {
	f, err := os.OpenFile(s.FlockPath(ip), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		if f, err = os.Open(s.FlockPath(ip)); err != nil {
			return nil, errors.WithStack(err)
		}
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return &LinuxFlock{
		pathname: s.FlockPath(ip),
		fd:       int(f.Fd()),
	}, nil
}
