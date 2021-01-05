package service

import (
	"context"
)

// Service .
type Service interface {
	Serve(context.Context) (Disposable, error)
}

// Disposable .
type Disposable interface {
	Dispose(context.Context) error
}
