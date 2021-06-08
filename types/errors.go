package types

import "github.com/juju/errors"

var (
	// ErrCIDRNotInPool .
	ErrCIDRNotInPool = errors.New("The requested subnet must match the CIDR of a configured Calico IP Pool")
	// ErrNoHosts .
	ErrNoHosts = errors.New("can't create proxy without hosts")
	// ErrServiceShutdown .
	ErrServiceShutdown = errors.New("Service shutdown")
	// ErrCannotDisposeService .
	ErrCannotDisposeService = errors.New("Can't dispose service")
	// ErrSystemTerminateSignal .
	ErrSystemTerminateSignal = errors.New("System terminal signal")
	// ErrNoListener .
	ErrNoListener = errors.New("No listener is provided")
	// ErrNoContainerIdent .
	ErrNoContainerIdent = errors.New("container id or name must not be null")
	// ErrWrongAPIVersion .
	ErrWrongAPIVersion = errors.New("api version must not be null")
	// ErrContainerNotExists .
	ErrContainerNotExists = errors.New("container is not exists")
	// ErrUnsupervisedNetwork .
	ErrUnsupervisedNetwork = errors.New("unsupervised network")
	// ErrConfiguredPoolUnfound .
	ErrConfiguredPoolUnfound = errors.New("network doesn't contains configured ip pools")
	// ErrIPInUse .
	ErrIPInUse = errors.New("ip address already in use")
	// ErrFixedIPNotAllocated .
	ErrFixedIPNotAllocated = errors.New("fixed-ip not allocated")
)
