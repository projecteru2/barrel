package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
)

// SpecificSubHandler covers the containers with specific IP but without fixed-ip label
type SpecificSubhandler struct {
	Base
	super SuperSubhandler
}

func NewSpecific(conf config.Config, store store.Store) *SpecificSubhandler {
	return &SpecificSubhandler{
		Base:  *NewBase(conf, store),
		super: *NewSuper(conf, store),
	}
}

func (h SpecificSubhandler) HandleCreate(containerMeta *cni.ContainerMeta) (err error) {
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
		flock.Unlock()
		return h.super.HandleCreate(containerMeta)
	}
	defer flock.Unlock()

	// borrow
	return h.BorrowNetEndpoint(containerMeta, nep)
}

func (h SpecificSubhandler) HandleStart(_ *cni.ContainerMeta) (err error) {
	// do nothing
	return
}

func (h SpecificSubhandler) HandleDelete(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// create
	if nep == nil {
		return
	}

	// borrow
	if err = h.store.FreeNetEndpoint(containerMeta.ID(), nep); err != nil {
		return
	}
	return h.DeleteDanglingNetwork(nep)
}
