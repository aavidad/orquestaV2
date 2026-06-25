package orquestaserver

import (
	"context"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const idleSelfImprovementGoalLauncherUnavailableReasonV0 = "goal_launcher_unavailable"

func (runtime *RuntimeV0) maybeScheduleIdleSelfImprovementCausalV0(
	ctx context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) {
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "shutdown_in_progress"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "shutdown_in_progress", now)
		return
	}
	if runtime.config.IdleSelfImprovementAfter <= 0 {
		if runtime.blockIdleSelfImprovementDomainSessionV0(ctx, now) {
			return
		}
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "disabled"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "disabled", now)
		return
	}
	if runtime.observePendingIdleSelfImprovementGoalV0(ctx, now) {
		return
	}
	port, portOK := runtime.supervisor.(IdleSelfImprovementPortV0)
	goalLauncher, goalOK := runtime.supervisor.(IdleSelfImprovementGoalLauncherPortV0)
	goalReady := runtime.config.IdleSelfImprovementGoalFirst && goalOK && goalLauncher != nil
	if runtime.config.IdleSelfImprovementGoalFirst && !goalReady {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": idleSelfImprovementGoalLauncherUnavailableReasonV0})
		runtime.markIdleSelfImprovementCheckedV0(ctx, idleSelfImprovementGoalLauncherUnavailableReasonV0, now)
		return
	}
	if (!portOK || port == nil) && !goalReady {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "port_unavailable"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "port_unavailable", now)
		return
	}
	if runtime.blockIdleSelfImprovementDomainSessionV0(ctx, now) {
		return
	}
	decision := runtime.idleSelfImprovementScheduleDecisionV0(ctx, result, now)
	if !decision.Schedule {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", decision.AuditStatus, "", decision.AuditPayload())
		runtime.markIdleSelfImprovementDecisionCheckedV0(ctx, decision, now)
		return
	}
	request := runtime.idleSelfImprovementRequestV0(decision, now)
	requests := normalizeIdleSelfImprovementCausalRequestsV0(
		runtime.idleSelfImprovementRequestsV0(ctx, request, decision),
	)
	if len(requests) == 0 {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "planner_empty", "request": request})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "planner_empty", now)
		return
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_scheduled", "scheduled", "", map[string]interface{}{"requests": requests})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkIdleSelfImprovementScheduledV0(now), "idle_self_improvement_scheduled")
	if goalReady {
		runtime.runAsyncWorkV0("idle_self_improvement_goal_launch", func() {
			runtime.launchIdleSelfImprovementGoalsV0(ctx, goalLauncher, requests)
		})
		return
	}
	runtime.runAsyncWorkV0("idle_self_improvement_prepare", func() {
		runtime.prepareIdleSelfImprovementBatchV0(ctx, port, requests)
	})
}
