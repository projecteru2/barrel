package store

import "github.com/projecteru2/barrel/cni"

// Store stores nep
type Store interface {
	// GetNetpointByID only returns nep for fixed-ip
	GetNetEndpointByID(id string) (*cni.NetEndpoint, error)
	// GetNetpointByIP only returns nep for fixed-ip
	GetNetEndpointByIP(ip string) (*cni.NetEndpoint, error)

	// ConnectNetEndpoint links a container to fixed-ip nep, until container is removed
	ConnectNetEndpoint(containerID string, _ *cni.NetEndpoint) error
	// DisconnectNetEndpoint unlinks a container to fixed-ip nep
	DisconnectNetEndpoint(containerID string, _ *cni.NetEndpoint) error

	// CreateNetEndpoint .
	CreateNetEndpoint(netns, owner, ipv4 string) (*cni.NetEndpoint, error)
	// DeleteNetEndpoint .
	DeleteNetEndpoint(*cni.NetEndpoint) error

	// OccupyNetEndpoint tries to borrow a nep; multiprocess safe
	OccupyNetEndpoint(containerID string, _ *cni.NetEndpoint) error
	// FreeNetEndpoint frees a occupied nep as long as the caller is the tenant
	FreeNetEndpoint(containerID string, _ *cni.NetEndpoint) error

	// GetNetEndpointRefcount calculates the ref count, not multiprocess safe
	GetNetEndpointRefcount(*cni.NetEndpoint) (int, error)

	// GetFlock news a flock
	GetFlock(ip string) (Flock, error)
}

// Flock is a multiprocess mutex
type Flock interface {
	Lock() error
	Unlock() error
}
