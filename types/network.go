package types

// IP .
type IP struct {
	PoolID  string
	Address string
}

// IPAddress .
type IPAddress struct {
	IP
	Version int
}

// IPInfo .
type IPInfo struct {
	ContainerID string
	PoolID      string
	Address     string
	Status      BitStatus
}

const (
	// IPStatusInUse .
	IPStatusInUse BitStatus = 1 << iota
	// IPStatusRetired .
	IPStatusRetired
)

// ContainerInfo .
type ContainerInfo struct {
	ID        string
	HostName  string
	Networks  []Network
	Addresses []IP
}

// Network .
type Network struct {
	NetworkID  string
	EndpointID string
	Address    IP
}
