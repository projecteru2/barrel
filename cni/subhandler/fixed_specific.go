package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	log "github.com/sirupsen/logrus"
)

// FixedSpecifcSubhandler covers the containers with specific IP and fixed-ip label
type FixedSpecificSubhandler struct {
	Base
	super    SuperSubhandler
	fixed    FixedSubhandler
	specific SpecificSubhandler
}

func NewFixedSpecific(conf config.Config, store store.Store) *FixedSpecificSubhandler {
	return &FixedSpecificSubhandler{
		Base:     *NewBase(conf, store),
		super:    *NewSuper(conf, store),
		fixed:    *NewFixed(conf, store),
		specific: *NewSpecific(conf, store),
	}
}

func (h FixedSpecificSubhandler) HandleCreate(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// create
	if nep == nil {
		if err = h.super.AddCNIStartHook(h.conf, &containerMeta.Meta); err != nil {
			return
		}
		return containerMeta.Save()
	}

	// borrow
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
	return h.BorrowNetEndpoint(containerMeta, nep)
}

func (h FixedSpecificSubhandler) HandleStart(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// borrow
	if nep != nil {
		return
	}

	// create
	return h.CreateNetEndpoint(containerMeta)
}

func (h FixedSpecificSubhandler) HandleDelete(containerMeta *cni.ContainerMeta) (err error) {
	return h.fixed.HandleDelete(containerMeta)
}
