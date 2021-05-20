package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni"
)

type FSStore struct {
	root string
}

func NewStore(rootDir string) *FSStore {
	os.MkdirAll(rootDir, 0755)
	return &FSStore{
		root: rootDir,
	}
}

func (s FSStore) GetNetEndpointByID(id string) (nep *cni.NetEndpoint, err error) {
	defer func() {
		if err != nil && os.IsExist(err) {
			nep, err = nil, nil
		}
	}()

	fi, err := os.Lstat(s.NetnsPath(id))
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil, errors.Errorf("invalid data, should've been a symlink: %s", s.NetnsPath(id))
	}
	netnsPath, err := os.Readlink(s.NetnsPath(id))
	if err != nil {
		return
	}
	return &cni.NetEndpoint{
		IPv4:  filepath.Base(netnsPath),
		Netns: netnsPath,
	}, nil
}

func (s FSStore) GetNetEndpointByIP(ip string) (nep *cni.NetEndpoint, err error) {
	defer func() {
		if err != nil && os.IsExist(err) {
			nep, err = nil, nil
		}
	}()

	if _, err = os.Stat(s.NetnsPath(ip)); err != nil {
		return
	}
	return &cni.NetEndpoint{
		IPv4:  ip,
		Netns: s.NetnsPath(ip),
	}, nil
}

func (s FSStore) ConnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	if err = os.Symlink(nep.Netns, s.NetnsPath(containerID)); err != nil {
		return
	}
	return s.increaseRefcount(nep)
}

func (s FSStore) DisconnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	if err = os.Remove(s.NetnsPath(containerID)); err != nil {
		return
	}
	return s.decreaseRefcount(nep)
}

func (s FSStore) CreateNetEndpoint(netns, ipv4 string) (nep *cni.NetEndpoint, err error) {
	if err = syscall.Mount(netns, s.NetnsPath(ipv4), "none", syscall.MS_BIND, ""); err != nil {
		return
	}
	return &cni.NetEndpoint{
		IPv4:  ipv4,
		Netns: netns,
	}, nil
}

func (s FSStore) DeleteNetEndpoint(nep *cni.NetEndpoint) (err error) {
	os.Remove(s.RefcountPath(nep.IPv4))
	os.Remove(s.OccupiedPath(nep.IPv4))
	return os.Remove(s.NetnsPath(nep.IPv4))
}

func (s FSStore) OccupyNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	file, err := os.OpenFile(s.OccupiedPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return
	}
	_, err = file.Write([]byte(containerID))
	return err
}

func (s FSStore) FreeNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	bs, err := ioutil.ReadFile(s.OccupiedPath(nep.IPv4))
	if err != nil {
		return
	}
	if string(bs) != containerID {
		return errors.Errorf("invalid free request, id not match: %s", containerID)
	}
	return os.Remove(s.OccupiedPath(nep.IPv4))
}
