package orquestaserver

import (
	"context"
	"fmt"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func (runtime *RuntimeV0) recoverSupervisorTickPanicV0(ctx context.Context) {
	recovered := recover()
	if recovered == nil {
		return
	}
	result := orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopTickErrorV0,
	}
	message := fmt.Sprintf("panic:%v", recovered)
	runtime.auditEventV0(ctx, "supervisor_tick_panic", "error", message, map[string]interface{}{
		"command": runtime.config.SupervisorCommand,
		"result":  result,
	})
	runtime.persistStateV0(ctx, runtime.tracker.MarkSupervisorErrorV0(
		runtime.config.SupervisorCommand,
		result,
		message,
		runtime.clock.Now(),
	))
}
