package handler

import (
	barrelcni "github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/barrel/cni/subhandler"
	"github.com/projecteru2/docker-cni/config"
)

// BarrelHandler is the implementation of docker-cni handler
type BarrelHandler struct {
	store store.Store
}

// NewBarrelHandler .
func NewBarrelHandler(store store.Store) *BarrelHandler {
	return &BarrelHandler{
		store: store,
	}
}

func (h *BarrelHandler) getSubhandler(conf config.Config, containerMeta *barrelcni.ContainerMeta) subhandler.Subhandler {
	if containerMeta.RequiresFixedIP() && !containerMeta.RequiresSpecificIP() {
		return subhandler.NewFixed(conf, h.store)
	}
	if containerMeta.RequiresFixedIP() && containerMeta.RequiresSpecificIP() {
		return subhandler.NewFixedSpecific(conf, h.store)
	}
	if !containerMeta.RequiresFixedIP() && containerMeta.RequiresSpecificIP() {
		return subhandler.NewSpecific(conf, h.store)
	}
	return subhandler.NewSuper(conf, h.store)
}
