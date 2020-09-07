package types

// Address .
type Address struct {
	ContainerID string
	PoolID      string
	Address     string
}

// ContainerInfo .
type ContainerInfo struct {
	ID        string
	Addresses []Address
}
