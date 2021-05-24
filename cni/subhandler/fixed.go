package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
)

// FixedSubhandler covers the containers with fixed-ip label but without specific IP
type FixedSubhandler struct {
	Base
	super SuperSubhandler
}

func NewFixed(conf config.Config, store store.Store) *FixedSubhandler {
	return &FixedSubhandler{
		Base:  *NewBase(conf, store),
		super: *NewSuper(conf, store),
	}
}

func (h FixedSubhandler) HandleCreate(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByID(containerMeta.ID())
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
	return h.BorrowNetEndpoint(containerMeta, nep)
}

func (h FixedSubhandler) HandleStart(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByID(containerMeta.ID())
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

func (h FixedSubhandler) HandleDelete(containerMeta *cni.ContainerMeta) (err error) {
	nep, err := h.store.GetNetEndpointByID(containerMeta.ID())
	if err != nil {
		return
	}

	return h.store.FreeNetEndpoint(containerMeta.ID(), nep)
}
