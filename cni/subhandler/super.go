package subhandler

import (
	barrelcni "github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	"github.com/projecteru2/docker-cni/handler"
	"github.com/projecteru2/docker-cni/handler/cni"
)

type SuperSubhandler struct {
	Base
	super handler.Handler
}

func NewSuper(conf config.Config, store store.Store) *SuperSubhandler {
	return &SuperSubhandler{
		Base:  *NewBase(conf, store),
		super: cni.CNIHandler{},
	}
}

func (h SuperSubhandler) HandleCreate(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.super.HandleCreate(h.conf, &containerMeta.Meta)
}

func (h SuperSubhandler) HandleStart(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.super.HandleStart(h.conf, &containerMeta.Meta)
}

func (h SuperSubhandler) HandleDelete(containerMeta *barrelcni.ContainerMeta) (err error) {
	return h.super.HandleDelete(h.conf, &containerMeta.Meta)
}
