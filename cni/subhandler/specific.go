package subhandler

import (
	"github.com/pkg/errors"
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
	nep, err := h.store.GetNetEndpointByIP(containerMeta.SpecificIP())
	if err != nil {
		return
	}

	// create
	if nep == nil {
		return errors.Errorf("specific ip create is not supported by CNI")
	}

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

	// create: should never reach here
	if nep == nil {
		return errors.Errorf("it's likely a bug: specific delete while nep not found")
	}

	// borrow
	if err = h.store.FreeNetEndpoint(containerMeta.ID(), nep); err != nil {
		return
	}
	return h.DeleteDanglingNetwork(nep)
}
