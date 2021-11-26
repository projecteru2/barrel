package vessel

import (
	"context"

	dockerClient "github.com/docker/docker/client"
	"github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
)

// ContainerVessel .
type ContainerVessel interface {
	ListContainers() ([]types.ContainerInfo, error)
	// GetContainerByID(containerID string) (types.ContainerInfo, error)
	UpdateContainer(context.Context, types.ContainerInfo) error
	DeleteContainer(context.Context, types.ContainerInfo) error
}

// Vessel .
type Vessel interface {
	Hostname() string
	ContainerVessel() ContainerVessel
	CalicoIPAllocator() CalicoIPAllocator
	DockerNetworkManager() DockerNetworkManager
	FixedIPAllocator() FixedIPAllocator
}

type vessel struct {
	hostname             string
	containerVessel      ContainerVessel
	fixedIPAllocator     FixedIPAllocator
	dockerNetworkManager DockerNetworkManager
}

// NewVessel .
func NewVessel(hostname string, cliv3 clientv3.Interface, dockerCli *dockerClient.Client, driverName string, stor store.Store) Vessel {
	allocator := NewCalicoIPAllocator(cliv3, hostname)
	return vessel{
		hostname:             hostname,
		fixedIPAllocator:     NewFixedIPAllocator(allocator, stor),
		dockerNetworkManager: NewDockerNetworkManager(dockerCli, driverName, allocator),
	}
}

func (v vessel) Hostname() string {
	return v.hostname
}

func (v vessel) ContainerVessel() ContainerVessel {
	return v.containerVessel
}

func (v vessel) CalicoIPAllocator() CalicoIPAllocator {
	return v.fixedIPAllocator
}

func (v vessel) DockerNetworkManager() DockerNetworkManager {
	return v.dockerNetworkManager
}

func (v vessel) FixedIPAllocator() FixedIPAllocator {
	return v.fixedIPAllocator
}
