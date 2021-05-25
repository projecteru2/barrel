package filesystem

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni"
	log "github.com/sirupsen/logrus"
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
	if _, err := os.Stat(s.NetnsPath(id)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	fi, err := os.Lstat(s.NetnsPath(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil, errors.Errorf("invalid data, should've been a symlink: %s", s.NetnsPath(id))
	}
	netnsPath, err := os.Readlink(s.NetnsPath(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  filepath.Base(netnsPath),
		Netns: netnsPath,
	}, nil
}

func (s FSStore) GetNetEndpointByIP(ip string) (nep *cni.NetEndpoint, err error) {
	if _, err := os.Stat(s.NetnsPath(ip)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  ip,
		Netns: s.NetnsPath(ip),
	}, nil
}

func (s FSStore) ConnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	if err = os.Symlink(nep.Netns, s.NetnsPath(containerID)); os.IsExist(err) {
		err = nil
	}
	return errors.WithStack(err)
}

func (s FSStore) DisconnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	return errors.WithStack(os.Remove(s.NetnsPath(containerID)))
}

func (s FSStore) CreateNetEndpoint(netns, id, ipv4 string) (nep *cni.NetEndpoint, err error) {
	file, err := os.OpenFile(s.CreatedPath(ipv4), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nep, errors.WithStack(err)
	}
	defer func() {
		if err != nil {
			if e := os.Remove(s.CreatedPath(ipv4)); e != nil {
				log.Errorf("failed to remove created path: %s, %+v", ipv4)
			}
		}
	}()

	if _, err = file.Write([]byte(id)); err != nil {
		return nep, errors.WithStack(err)
	}

	if _, err := os.Create(s.NetnsPath(ipv4)); err != nil {
		return nep, errors.WithStack(err)
	}
	if err = syscall.Mount(netns, s.NetnsPath(ipv4), "none", syscall.MS_BIND, ""); err != nil {
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  ipv4,
		Netns: s.NetnsPath(ipv4),
	}, nil
}

func (s FSStore) DeleteNetEndpoint(nep *cni.NetEndpoint) (err error) {
	os.Remove(s.OccupiedPath(nep.IPv4))
	os.Remove(s.CreatedPath(nep.IPv4))
	syscall.Unmount(s.NetnsPath(nep.IPv4), syscall.MNT_DETACH)
	return errors.WithStack(os.Remove(s.NetnsPath(nep.IPv4)))
}

func (s FSStore) OccupyNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	file, err := os.OpenFile(s.OccupiedPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = file.Write([]byte(containerID))
	return errors.WithStack(err)
}

func (s FSStore) FreeNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	bs, err := ioutil.ReadFile(s.OccupiedPath(nep.IPv4))
	if err != nil {
		return errors.WithStack(err)
	}
	if string(bs) != containerID {
		return errors.Errorf("invalid free request, id not match: %s", containerID)
	}
	return errors.WithStack(os.Remove(s.OccupiedPath(nep.IPv4)))
}

func (s FSStore) GetNetEndpointRefcount(nep *cni.NetEndpoint) (rc int, err error) {
	files, err := ioutil.ReadDir(s.root)
	if err != nil {
		return
	}

	for _, file := range files {
		if file.Mode()&os.ModeSymlink != 0 {
			if destNetns, err := os.Readlink(path.Join(s.root, file.Name())); err == nil && destNetns == nep.Netns {
				rc++
			} else if err != nil {
				return 0, errors.WithStack(err)
			}
		}
	}

	if _, e := os.Stat(s.OccupiedPath(nep.IPv4)); e == nil {
		rc++
	}

	return rc, errors.WithStack(err)
}
