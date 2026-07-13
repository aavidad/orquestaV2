package main

import (
	"context"
	"errors"
	"sync"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

type serverRequiredTestResourceCloserV0 interface {
	Close() error
}

var (
	_ serverRequiredTestResourceCloserV0 = (*serverRequiredTestRunnerV0)(nil)
	_ serverRequiredTestResourceCloserV0 = (*serverAutoprogrammingBatchTestRunnerV0)(nil)
	_ serverRequiredTestResourceCloserV0 = (*serverGoalRequiredTestAttestationWorkspaceSelectorV0)(nil)
)

// serverRequiredTestResourceShutdownHookV0 is the single owner transferred
// from stack composition to the server runtime (or to the stdio command).
// Closing in reverse construction order prevents a higher-level adapter from
// observing an executor whose admitted command descriptors are already gone.
type serverRequiredTestResourceShutdownHookV0 struct {
	once    sync.Once
	closers []serverRequiredTestResourceCloserV0
	err     error
}

func newServerRequiredTestResourceShutdownHookV0(
	values ...any,
) *serverRequiredTestResourceShutdownHookV0 {
	hook := &serverRequiredTestResourceShutdownHookV0{}
	for _, value := range values {
		closer, ok := value.(serverRequiredTestResourceCloserV0)
		if !ok || closer == nil {
			continue
		}
		hook.closers = append(hook.closers, closer)
	}
	if len(hook.closers) == 0 {
		return nil
	}
	return hook
}

func serverRequiredTestResourceShutdownHookFromStackV0(
	stack orquestaappcodexstack.StackV0,
) *serverRequiredTestResourceShutdownHookV0 {
	return newServerRequiredTestResourceShutdownHookV0(
		stack.Ports.RequiredTestRunner,
		stack.Ports.GoalRequiredTestSpecBinder,
		stack.AutoprogrammingPromotion.BatchTestRunner,
	)
}

func (hook *serverRequiredTestResourceShutdownHookV0) ShutdownV0(context.Context) error {
	if hook == nil {
		return nil
	}
	hook.once.Do(func() {
		var joined error
		for index := len(hook.closers) - 1; index >= 0; index-- {
			joined = errors.Join(joined, hook.closers[index].Close())
		}
		hook.err = joined
	})
	return hook.err
}
