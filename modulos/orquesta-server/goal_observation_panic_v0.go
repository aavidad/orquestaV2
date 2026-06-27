package orquestaserver

import (
	"context"
	"fmt"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (runtime *RuntimeV0) recoverGoalObservationTickPanicV0(ctx context.Context) {
	recovered := recover()
	if recovered == nil {
		return
	}
	message := fmt.Sprintf("panic:%v", recovered)
	runtime.auditEventV0(ctx, "goal_observer_tick_panic", "error", message, map[string]interface{}{
		"result_summary": goalObservationResultAuditSummaryV0(orquestagoal.GoalWorkObserveActiveResultV0{}),
	})
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkGoalObserverErrorV0(message, runtime.clock.Now()),
		"goal_observer_panic",
	)
}
