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

// FSStore is an implementation of Store
type FSStore struct {
	root string
}

// NewStore .
func NewStore(rootDir string) (fs *FSStore, err error) {
	if err = os.MkdirAll(rootDir, 0755); err != nil && !os.IsExist(err) {
		return fs, errors.WithStack(err)
	}
	return &FSStore{
		root: rootDir,
	}, nil
}

// GetNetEndpointByID .
func (s FSStore) GetNetEndpointByID(id string) (*cni.NetEndpoint, error) {
	if _, err := os.Stat(s.netnsPath(id)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	fi, err := os.Lstat(s.netnsPath(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil, errors.Errorf("invalid data, should've been a symlink: %s", s.netnsPath(id))
	}
	netnsPath, err := os.Readlink(s.netnsPath(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	owner, err := ioutil.ReadFile(s.ownerPath(filepath.Base(netnsPath)))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  filepath.Base(netnsPath),
		Netns: netnsPath,
		Owner: string(owner),
	}, nil
}

// GetNetEndpointByIP .
func (s FSStore) GetNetEndpointByIP(ip string) (nep *cni.NetEndpoint, err error) {
	if _, err := os.Stat(s.netnsPath(ip)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	owner, err := ioutil.ReadFile(s.ownerPath(ip))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  ip,
		Netns: s.netnsPath(ip),
		Owner: string(owner),
	}, nil
}

// ConnectNetEndpoint .
func (s FSStore) ConnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	if err = os.Symlink(nep.Netns, s.netnsPath(containerID)); os.IsExist(err) {
		err = nil
	}
	return errors.WithStack(err)
}

// DisconnectNetEndpoint .
func (s FSStore) DisconnectNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	if err = os.Remove(s.netnsPath(containerID)); err != nil {
		log.Warnf("failed to remove netns %s: %+v", s.netnsPath(containerID), err)
	}
	return assertFileNotExists(s.netnsPath(containerID))
}

// CreateNetEndpoint .
func (s FSStore) CreateNetEndpoint(netns, id, ipv4 string) (nep *cni.NetEndpoint, err error) {
	file, err := os.OpenFile(s.ownerPath(ipv4), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nep, errors.WithStack(err)
	}
	defer func() {
		if err != nil {
			if e := os.Remove(s.ownerPath(ipv4)); e != nil {
				log.Errorf("failed to remove file: %s, %+v", ipv4, e)
			}
		}
	}()

	if _, err = file.Write([]byte(id)); err != nil {
		return nep, errors.WithStack(err)
	}

	if _, err := os.Create(s.netnsPath(ipv4)); err != nil {
		return nep, errors.WithStack(err)
	}
	defer func() {
		if err != nil {
			if e := os.Remove(s.netnsPath(nep.IPv4)); e != nil {
				log.Errorf("failed to remove file: %s, %+v", s.netnsPath(nep.IPv4), e)
			}
		}
	}()

	if err = syscall.Mount(netns, s.netnsPath(ipv4), "none", syscall.MS_BIND, ""); err != nil {
		return nil, errors.WithStack(err)
	}
	return &cni.NetEndpoint{
		IPv4:  ipv4,
		Netns: s.netnsPath(ipv4),
		Owner: id,
	}, nil
}

// DeleteNetEndpoint .
func (s FSStore) DeleteNetEndpoint(nep *cni.NetEndpoint) (err error) {
	if err = os.Remove(s.tenantPath(nep.IPv4)); err != nil {
		log.Warnf("failed to remove tenant file %s: %+v", s.tenantPath(nep.IPv4), errors.WithStack(err))
	}
	if err = os.Remove(s.ownerPath(nep.IPv4)); err != nil {
		log.Warnf("failed to remove owner file %s: %+v", s.ownerPath(nep.IPv4), errors.WithStack(err))
	}
	if err = os.Remove(s.flockPath(nep.IPv4)); err != nil {
		log.Warnf("failed to remove flock file %s: %+v", s.flockPath(nep.IPv4), errors.WithStack(err))
	}
	if err = syscall.Unmount(s.netnsPath(nep.IPv4), syscall.MNT_DETACH); err != nil {
		log.Warnf("failed to umount netns file %s: %+v", s.netnsPath(nep.IPv4), errors.WithStack(err))
	}
	if err = os.Remove(s.netnsPath(nep.IPv4)); err != nil {
		log.Warnf("failed to remove netns file %s: %+v", s.netnsPath(nep.IPv4), errors.WithStack(err))
	}
	return assertFileNotExists(s.tenantPath(nep.IPv4), s.ownerPath(nep.IPv4), s.netnsPath(nep.IPv4), s.flockPath(nep.IPv4))
}

// OccupyNetEndpoint .
func (s FSStore) OccupyNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	file, err := os.OpenFile(s.tenantPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = file.Write([]byte(containerID))
	return errors.WithStack(err)
}

// FreeNetEndpoint .
func (s FSStore) FreeNetEndpoint(containerID string, nep *cni.NetEndpoint) (err error) {
	bs, err := ioutil.ReadFile(s.tenantPath(nep.IPv4))
	if err != nil {
		return errors.WithStack(err)
	}
	if string(bs) != containerID {
		return errors.Errorf("invalid free request, id not match: %s, %s", nep.IPv4, containerID)
	}
	if err = os.Remove(s.tenantPath(nep.IPv4)); err != nil {
		log.Warnf("failed to remove tenant file %s: %+v", s.tenantPath(nep.IPv4), err)
	}
	return assertFileNotExists(s.tenantPath(nep.IPv4))
}

// GetNetEndpointRefcount .
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

	if _, e := os.Stat(s.tenantPath(nep.IPv4)); e == nil {
		rc++
	}

	return rc, errors.WithStack(err)
}
