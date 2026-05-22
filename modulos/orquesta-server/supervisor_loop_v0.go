package orquestaserver

import (
	"context"
	"time"
)

func (runtime *RuntimeV0) runSupervisorLoopV0(ctx context.Context) {
	if runtime.supervisor == nil {
		return
	}
	runtime.runSupervisorTickV0(ctx)
	ticker := time.NewTicker(runtime.config.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.runSupervisorTickV0(ctx)
		}
	}
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
		runtime.persistStateV0(ctx, runtime.tracker.MarkErrorV0(err.Error(), runtime.clock.Now()))
		return
	}
	runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorV0(result, runtime.clock.Now()))
}
