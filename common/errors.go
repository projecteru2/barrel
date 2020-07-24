package common

import "github.com/juju/errors"

var (
	// ErrCertAndKeyMissing .
	ErrCertAndKeyMissing = errors.New("can't create https host without cert and key")
	// ErrServerStop .
	ErrServerStop = errors.New("Server Stops")
	// ErrNoListener .
	ErrNoListener = errors.New("no listener is provided")
	// ErrNoContainerIdent .
	ErrNoContainerIdent = errors.New("container id or name must not be null")
	// ErrWrongAPIVersion .
	ErrWrongAPIVersion = errors.New("api version must not be null")
	// ErrContainerNotExists .
	ErrContainerNotExists = errors.New("container is not exists")
)
