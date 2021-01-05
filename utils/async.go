package utils

// Coroutine .
type Coroutine interface {
	Await()
}

// Async .
func Async(f func()) Coroutine {
	ch := make(chan struct{})
	go func() {
		f()
		ch <- struct{}{}
	}()
	return goroutine{ch}
}

type goroutine struct {
	ch chan struct{}
}

func (r goroutine) Await() {
	<-r.ch
	close(r.ch)
}
