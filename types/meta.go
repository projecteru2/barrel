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
	Attrs       *IPAttributes
}

// IPAttributes .
type IPAttributes struct {
	Borrowers []Container
}

const (
	// IPStatusInUse .
	IPStatusInUse BitStatus = 1 << iota
	// IPStatusRetired .
	IPStatusRetired
)

// Container .
type Container struct {
	ID       string
	HostName string
}

// ContainerInfo .
type ContainerInfo struct {
	Container
	Networks  []Network
	Addresses []IP
}

// Network .
type Network struct {
	NetworkID  string
	EndpointID string
	Address    IP
}
