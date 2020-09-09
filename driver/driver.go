package driver

import (
	pluginIPAM "github.com/docker/go-plugins-helpers/ipam"
	"github.com/projecteru2/barrel/types"
)

// ReservedAddressManager .
type ReservedAddressManager interface {
	ReserveAddressFromPools(pools []types.Pool) (types.AddressWithVersion, error)
	ReserveAddress(address types.Address) error
	InitContainerInfoRecord(containerInfo types.ContainerInfo) error
	ReserveAddressForContainer(containerID string, address types.Address) error
	ReleaseContainerAddresses(containerID string) error
	ReleaseContainerAddressesByIPPools(containerID string, pools []types.Pool) error
	ReleaseReservedAddress(address types.Address) error
	GetIPPoolsByNetworkName(name string) ([]types.Pool, error)
	IsAddressReserved(address *types.Address) (bool, error)
	AquireIfReserved(address *types.Address) (bool, error)
}

// AddressManager .
type AddressManager interface {
	pluginIPAM.Ipam
	ReservedAddressManager
}
