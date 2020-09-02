package common

import "github.com/pkg/errors"

var (
	// ErrNoHosts .
	ErrNoHosts = errors.New("can't create proxy without hosts")
	// ErrCertAndKeyMissing .
	ErrCertAndKeyMissing = errors.New("can't create https host without cert and key")
	// ErrServiceShutdown .
	ErrServiceShutdown = errors.New("Service shutdown")
	// ErrNoListener .
	ErrNoListener = errors.New("No listener is provided")
	// ErrNoContainerIdent .
	ErrNoContainerIdent = errors.New("container id or name must not be null")
	// ErrWrongAPIVersion .
	ErrWrongAPIVersion = errors.New("api version must not be null")
	// ErrContainerNotExists .
	ErrContainerNotExists = errors.New("container is not exists")
)
