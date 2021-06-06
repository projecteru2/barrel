package subhandler

import "github.com/projecteru2/barrel/cni"

// Subhandler aims to split scenarios into specific handler
type Subhandler interface {
	HandleCreate(*cni.ContainerMeta) error
	HandleStart(*cni.ContainerMeta) error
	HandleDelete(*cni.ContainerMeta) error
}
