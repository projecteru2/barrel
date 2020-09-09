package types

// Address .
type Address struct {
	ContainerID string
	PoolID      string
	Address     string
}

// AddressWithVersion .
type AddressWithVersion struct {
	Version int
	Address Address
}

// ContainerInfo .
type ContainerInfo struct {
	ID        string
	Addresses []Address
}
