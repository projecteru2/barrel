package subhandler

import (
	barrelcni "github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	"github.com/projecteru2/docker-cni/handler/cni"
)

type SuperSubhandler struct {
	Base
	cni.CNIHandler
}

func NewSuper(conf config.Config, store store.Store) *SuperSubhandler {
	return &SuperSubhandler{
		Base:       *NewBase(conf, store),
		CNIHandler: cni.CNIHandler{},
	}
}

func (h SuperSubhandler) HandleCreate(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleCreate(h.conf, &containerMeta.Meta)
}

func (h SuperSubhandler) HandleStart(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleStart(h.conf, &containerMeta.Meta)
}

func (h SuperSubhandler) HandleDelete(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.CNIHandler.HandleDelete(h.conf, &containerMeta.Meta)
}
