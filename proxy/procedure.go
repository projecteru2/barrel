package proxy

type Procedure interface {
	Next()
	Inspect(func() error)
}
