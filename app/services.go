package app

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/juju/errors"

	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/utils"
)

type starter struct {
	logger         utils.Logger
	ss             []service.Service
	disposeTimeout time.Duration
}

func (starter starter) start(sigs <-chan os.Signal) error {
	chErr := utils.NewAutoCloseChanErr(len(starter.ss))

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(len(starter.ss))

	for _, elm := range starter.ss {
		go func(serv service.Service) {
			chErr.Send(starter.startService(ctx, serv))
			wg.Done()
		}(elm)
	}

	select {
	case <-sigs:
		starter.logger.Info("received terminate sigs")
		cancel()
		err := <-chErr.Receive()
		wg.Wait()
		return err
	case err := <-chErr.Receive():
		starter.logger.Errorf("error received, cause=%v", err)
		cancel()
		wg.Wait()
		starter.logger.Info("canceled")
		return err
	}
}

func (starter starter) startServiceWithoutDisposeTimeout(ctx context.Context, serv service.Service) error {
	return serveContext{ctx: ctx, timeoutCtx: context.Background()}.serveAndDispose(serv)
}

func (starter starter) startService(ctx context.Context, serv service.Service) error {
	if starter.disposeTimeout == time.Duration(0) {
		return starter.startServiceWithoutDisposeTimeout(ctx, serv)
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), starter.disposeTimeout)
	defer timeoutCancel()

	return serveContext{ctx: ctx, timeoutCtx: timeoutCtx}.serveAndDispose(serv)
}

type serveContext struct {
	ctx        context.Context
	timeoutCtx context.Context
}

func (ctx serveContext) serveAndDispose(serv service.Service) error {
	disposable, serveErr := serv.Serve(ctx.ctx)
	if err := disposable.Dispose(ctx.timeoutCtx); err != nil {
		if serveErr != nil {
			return errors.Wrap(serveErr, err)
		}
		return err
	}
	return serveErr
}
