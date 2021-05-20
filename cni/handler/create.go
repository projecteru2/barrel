package handler

import (
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/docker-cni/config"
	"github.com/projecteru2/docker-cni/oci"
)

func (h *BarrelHandler) HandleCreate(conf config.Config, meta *oci.ContainerMeta) (err error) {
	containerMeta := &cni.ContainerMeta{*meta}
	subhandler := h.getSubhandler(conf, containerMeta)
	return subhandler.HandleCreate(containerMeta)
}
