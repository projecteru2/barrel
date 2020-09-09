package service

// Disposable .
type Disposable interface {
	Dispose() error
}

// DisposableService .
type DisposableService interface {
	Disposable
	// will block until finishes or error encoutered
	Service() error
}
