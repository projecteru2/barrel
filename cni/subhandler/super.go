package subhandler

import (
	barrelcni "github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	"github.com/projecteru2/docker-cni/handler/cni"
)

// SuperSubhandler represents docker-cni framework handler
type SuperSubhandler struct {
	Base
	cni.CNIHandler
}

// NewSuper .
func NewSuper(conf config.Config, store store.Store) *SuperSubhandler {
	return &SuperSubhandler{
		Base:       *NewBase(conf, store),
		CNIHandler: cni.CNIHandler{},
	}
}

// HandleCreate .
func (h SuperSubhandler) HandleCreate(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleCreate(h.conf, &containerMeta.Meta)
}

// HandleStart .
func (h SuperSubhandler) HandleStart(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleStart(h.conf, &containerMeta.Meta)
}

// HandleDelete .
func (h SuperSubhandler) HandleDelete(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleDelete(h.conf, &containerMeta.Meta)
}
