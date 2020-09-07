package types

// AddressWithVersion .
type AddressWithVersion struct {
	Version int
	Address Address
}

// Pool .
type Pool struct {
	CIDR    string
	Name    string
	Gateway string
}

// ReservedAddressManager .
type ReservedAddressManager interface {
	ReserveAddressFromPools(pools []Pool) (AddressWithVersion, error)
	ReserveAddress(address Address) error
	InitContainerInfoRecord(containerInfo ContainerInfo) error
	ReserveAddressForContainer(containerID string, address Address) error
	ReleaseContainerAddresses(containerID string) error
	ReleaseContainerAddressesByIPPools(containerID string, pools []Pool) error
	ReleaseReservedAddress(address Address) error
	GetIPPoolsByNetworkName(name string) ([]Pool, error)
	IsAddressReserved(address *Address) (bool, error)
	AquireIfReserved(address *Address) (bool, error)
}
