package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	log "github.com/sirupsen/logrus"
)

// FixedSpecificSubhandler covers the containers with specific IP and fixed-ip label
type FixedSpecificSubhandler struct {
	Base
	super    SuperSubhandler
	fixed    FixedSubhandler
	specific SpecificSubhandler
}

// NewFixedSpecific .
func NewFixedSpecific(conf config.Config, store store.Store) *FixedSpecificSubhandler {
	return &FixedSpecificSubhandler{
		Base:     *NewBase(conf, store),
		super:    *NewSuper(conf, store),
		fixed:    *NewFixed(conf, store),
		specific: *NewSpecific(conf, store),
	}
}

// HandleCreate .
func (h FixedSpecificSubhandler) HandleCreate(containerMeta *cni.ContainerMeta) (err error) {
	flock, err := h.store.GetFlock(containerMeta.SpecificIP())
	if err != nil {
		return
	}
	if err = flock.Lock(); err != nil {
		return
	}
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// create
	if nep == nil {
		if err = flock.Unlock(); err != nil {
			return
		}
		if err = h.super.AddCNIStartHook(h.conf, &containerMeta.Meta); err != nil {
			return
		}
		return containerMeta.Save()
	}
	defer func() {
		if e := flock.Unlock(); e != nil {
			log.Errorf("failed to unlock flock %s: %+v", containerMeta.SpecificIP(), e)
		}
	}()

	// borrow
	return h.BorrowNetEndpoint(containerMeta, nep)
}

// HandleStart .
func (h FixedSpecificSubhandler) HandleStart(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// borrow
	if nep != nil {
		return h.store.ConnectNetEndpoint(containerMeta.ID(), nep)
	}

	// create
	return h.CreateNetEndpoint(containerMeta)
}

// HandleDelete .
func (h FixedSpecificSubhandler) HandleDelete(containerMeta *cni.ContainerMeta) (err error) {
	return h.fixed.HandleDelete(containerMeta)
}
