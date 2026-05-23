package orquestaserver

import (
	"context"
	"sync/atomic"
	"time"
)

func (runtime *RuntimeV0) runSupervisorLoopV0(ctx context.Context) {
	if runtime.supervisor == nil {
		return
	}
	runtime.runSupervisorTickAsyncV0(ctx)
	ticker := time.NewTicker(runtime.config.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.runSupervisorTickAsyncV0(ctx)
		}
	}
}

func (runtime *RuntimeV0) runSupervisorTickAsyncV0(ctx context.Context) bool {
	if !atomic.CompareAndSwapInt32(&runtime.supervisorTickActive, 0, 1) {
		return false
	}
	go func() {
		defer atomic.StoreInt32(&runtime.supervisorTickActive, 0)
		runtime.runSupervisorTickV0(ctx)
	}()
	return true
}

func (runtime *RuntimeV0) runSupervisorTickV0(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		return
	}
	command := runtime.config.SupervisorCommand
	if command.MaxTicks <= 0 {
		command.MaxTicks = DefaultSupervisorMaxTicksV0
	}
	result, err := runtime.supervisor.RunGlobalSupervisorV0(ctx, command)
	if err != nil {
		runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorErrorV0(
			command,
			result,
			err.Error(),
			runtime.clock.Now(),
		))
		return
	}
	runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorV0(command, result, runtime.clock.Now()))
}
