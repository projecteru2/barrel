package utils

import (
	"context"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type logContext struct {
	context.Context
	entry *log.Entry
}

// WithEntry .
func WithEntry(ctx context.Context, entry *log.Entry) context.Context {
	if logCtx, ok := ctx.(logContext); ok {
		return logContext{Context: logCtx.Context, entry: entry}
	}
	return logContext{Context: ctx, entry: entry}
}

// LogEntry .
func LogEntry(ctx context.Context) *log.Entry {
	if logger, ok := ctx.(logContext); ok {
		return logger.entry
	}

	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return log.WithContext(ctx)
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return log.WithContext(ctx)
	}
	name := fn.Name()
	return log.WithField("func", name).WithField("file", file).WithField("line", line)
}

// RunFuncReturnErrorWithContext .
func RunFuncReturnErrorWithContext(ctx context.Context, run func() error) error {
	ch := make(chan error)
	go func() {
		err := run()
		ch <- err
		close(ch)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
