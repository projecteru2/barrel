package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	log "github.com/sirupsen/logrus"
)

type Base struct {
	store store.Store
	conf  config.Config
}

func NewBase(conf config.Config, store store.Store) *Base {
	return &Base{
		conf:  conf,
		store: store,
	}
}

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

func (h *Base) CreateNetEndpoint(containerMeta *cni.ContainerMeta) (err error) {
	ipv4, err := containerMeta.IPv4()
	if err != nil {
		return
	}

	nep, err := h.store.CreateNetEndpoint(containerMeta.Netns(), ipv4)
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
