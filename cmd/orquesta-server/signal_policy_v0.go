package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverSignalControllerV0 struct {
	ctx       context.Context
	cancel    context.CancelFunc
	runtime   *orquestaserver.RuntimeV0
	signals   chan os.Signal
	escalate  func(os.Signal)
	stopOnce  sync.Once
	mu        sync.Mutex
	cause     orquestaserver.ShutdownSignalCauseV0
	osSignals []os.Signal
}

func newServerSignalControllerV0(
	parent context.Context,
	runtime *orquestaserver.RuntimeV0,
) *serverSignalControllerV0 {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	controller := &serverSignalControllerV0{
		ctx:       ctx,
		cancel:    cancel,
		runtime:   runtime,
		signals:   make(chan os.Signal, 2),
		escalate:  serverEscalateSignalV0,
		osSignals: serverShutdownSignalsV0(),
	}
	signal.Notify(controller.signals, controller.osSignals...)
	go controller.runV0()
	return controller
}

func (controller *serverSignalControllerV0) runV0() {
	for {
		select {
		case sig, ok := <-controller.signals:
			if !ok {
				return
			}
			controller.observeSignalV0(sig)
		case <-controller.ctx.Done():
			return
		}
	}
}

func (controller *serverSignalControllerV0) observeSignalV0(sig os.Signal) {
	if controller == nil || sig == nil {
		return
	}
	controller.mu.Lock()
	controller.cause.SignalName = serverSignalNameV0(sig)
	controller.cause.Count++
	controller.cause.Escalated = controller.cause.Count > 1
	cause := controller.cause
	controller.mu.Unlock()
	if controller.runtime != nil {
		controller.runtime.RecordShutdownSignalV0(cause)
	}
	controller.cancel()
	if cause.Escalated && controller.escalate != nil {
		controller.stopNotificationsV0()
		controller.escalate(sig)
	}
}

func (controller *serverSignalControllerV0) contextV0() context.Context {
	if controller == nil || controller.ctx == nil {
		return context.Background()
	}
	return controller.ctx
}

func (controller *serverSignalControllerV0) shutdownCauseV0() orquestaserver.ShutdownSignalCauseV0 {
	if controller == nil {
		return orquestaserver.ShutdownSignalCauseV0{}
	}
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.cause
}

func (controller *serverSignalControllerV0) stopNotificationsV0() {
	if controller == nil {
		return
	}
	controller.stopOnce.Do(func() {
		signal.Stop(controller.signals)
		close(controller.signals)
	})
}
