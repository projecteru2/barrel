package subhandler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/cni"
	"github.com/projecteru2/barrel/cni/store"
	"github.com/projecteru2/docker-cni/config"
	log "github.com/sirupsen/logrus"
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
		return h.super.HandleCreate(containerMeta)
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

	// create
	if nep == nil {
		return
	}

	// borrow
	if err = h.store.FreeNetEndpoint(containerMeta.ID(), nep); err != nil {
		return
	}
	count, err := h.store.GetNetEndpointRefcount(nep)
	if err != nil {
		return
	}
	log.Errorf("refcount: %s", count)
	if count > 0 {
		return
	}

	log.Info("refcount back to zero, cleanup")
	if err = h.store.DeleteNetEndpoint(nep); err != nil {
		return
	}
	cmd := exec.Command(os.Args[0], "cni", "--config", h.conf.Filename, "--command", "del")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"id":"%s"}`, containerMeta.ID()))
	return errors.WithStack(cmd.Run())
}
