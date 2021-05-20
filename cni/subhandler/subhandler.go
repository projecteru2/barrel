package subhandler

import "github.com/projecteru2/barrel/cni"

type Subhandler interface {
	HandleCreate(*cni.ContainerMeta) error
	HandleStart(*cni.ContainerMeta) error
	HandleDelete(*cni.ContainerMeta) error
}
