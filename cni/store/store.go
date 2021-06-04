package store

import "github.com/projecteru2/barrel/cni"

type Store interface {
	GetNetEndpointByID(id string) (*cni.NetEndpoint, error)
	GetNetEndpointByIP(ip string) (*cni.NetEndpoint, error)

	ConnectNetEndpoint(containerID string, _ *cni.NetEndpoint) error
	DisconnectNetEndpoint(containerID string, _ *cni.NetEndpoint) error

	CreateNetEndpoint(netns, id, ipv4 string) (*cni.NetEndpoint, error)
	DeleteNetEndpoint(*cni.NetEndpoint) error

	OccupyNetEndpoint(containerID string, _ *cni.NetEndpoint) error
	FreeNetEndpoint(containerID string, _ *cni.NetEndpoint) error

	GetNetEndpointRefcount(*cni.NetEndpoint) (int, error)

	GetFlock(ip string) (Flock, error)
}

type Flock interface {
	Lock() error
	Unlock() error
}
