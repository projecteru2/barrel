package subhandler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	log "github.com/sirupsen/logrus"
)

// Base in fact isn't a subhandler, just provides some common functions
type Base struct {
	store store.Store
	conf  config.Config
}

// NewBase .
func NewBase(conf config.Config, store store.Store) *Base {
	return &Base{
		conf:  conf,
		store: store,
	}
}

// Enabled .
func (h *Base) Enabled() bool {
	return h.conf != config.Config{}
}

// BorrowNetEndpoint will snatch the nep
func (h *Base) BorrowNetEndpoint(containerMeta *cni.ContainerMeta, nep *cni.NetEndpoint) (err error) {
	if err = h.store.OccupyNetEndpoint(containerMeta.ID(), nep); err != nil {
		return
	}
	defer func() {
		if err != nil {
			if e := h.store.FreeNetEndpoint(containerMeta.ID(), nep); e != nil {
				log.Errorf("failed to free occupied nep: %+v", err)
			}
		}
	}()

	containerMeta.SetNetns(nep.Netns)
	return containerMeta.Save()
}

// CreateNetEndpoint .
func (h *Base) CreateNetEndpoint(containerMeta *cni.ContainerMeta) (err error) {
	ipv4, err := containerMeta.IPv4()
	if err != nil {
		return
	}

	nep, err := h.store.CreateNetEndpoint(containerMeta.Netns(), containerMeta.ID(), ipv4)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			if e := h.store.DeleteNetEndpoint(nep); e != nil {
				log.Errorf("failed to delete nep: %s, %+v", nep.Netns, e)
			}
		}
	}()

	if err = h.store.ConnectNetEndpoint(containerMeta.ID(), nep); err != nil {
		return
	}
	defer func() {
		if err != nil {
			if e := h.store.DisconnectNetEndpoint(containerMeta.ID(), nep); e != nil {
				log.Errorf("failed to disconnect nep: %s, %+v", containerMeta.ID(), e)
			}
		}
	}()

	return h.store.OccupyNetEndpoint(containerMeta.ID(), nep)
}

// DeleteDanglingNetwork .
func (h *Base) DeleteDanglingNetwork(nep *cni.NetEndpoint) (err error) {
	return h.withFlock(nep.IPv4, func() (err error) {
		count, err := h.store.GetNetEndpointRefcount(nep)
		if err != nil {
			return
		}
		if count > 0 {
			return
		}

		log.Infof("refcount back to zero, cleanup: %+v", nep)
		if err = h.store.DeleteNetEndpoint(nep); err != nil {
			return errors.WithStack(err)
		}

		cmd := exec.Command(os.Args[0], "cni", "--config", h.conf.Filename, "--command", "del") // nolint
		cmd.Args[0] = "barrel-cni"
		cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"id":"%s"}`, nep.Owner))
		return errors.WithStack(cmd.Run())
	})
}

// RemoveNetwork will be exposed to docker proxy in delete phase
func (h *Base) RemoveNetwork(id string) (err error) {
	nep, err := h.store.GetNetEndpointByID(id)
	if err != nil {
		return
	}

	if nep == nil {
		log.Warnf("nep not found: %s", id)
		return
	}

	if err = h.store.DisconnectNetEndpoint(id, nep); err != nil {
		return
	}
	return h.DeleteDanglingNetwork(nep)
}

func (h *Base) withFlock(ip string, f func() error) (err error) {
	flock, err := h.store.GetFlock(ip)
	if err != nil {
		return
	}
	if err = flock.Lock(); err != nil {
		return
	}
	defer func() {
		if e := flock.Unlock(); e != nil {
			log.Errorf("failed to unlock %s: %+v", ip, e)
		}
	}()
	return f()
}
