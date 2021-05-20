package subhandler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
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
	return h.specific.HandleCreate(containerMeta)
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
