package orquestaserver

import (
	"context"
	"strings"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const idleSelfImprovementGoalLauncherUnavailableReasonV0 = "goal_launcher_unavailable"

func (runtime *RuntimeV0) maybeScheduleIdleSelfImprovementCausalV0(
	ctx context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
	tickCauses ...string,
) {
	tickCause := normalizeSupervisorTickCauseV0(tickCauses...)
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "shutdown_in_progress"})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "shutdown_in_progress", now)
		return
	}
	if runtime.config.IdleSelfImprovementDisabled {
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
		publication := runtime.idleSelfImprovementDecisionPublicationV0(decision, nil)
		if skip, skipped := runtime.tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip {
			runtime.auditIdleSelfImprovementIdenticalWatchdogV0(ctx, tickCause, publication, skipped)
			return
		} else {
			payload := idleSelfImprovementAuditPayloadWithSkippedIdenticalV0(
				decision.AuditPayload(),
				skipped,
			)
			runtime.auditEventV0(ctx, "idle_self_improvement_check", decision.AuditStatus, "", payload)
		}
		runtime.markIdleSelfImprovementDecisionCheckedV0(ctx, decision, now)
		return
	}
	request := runtime.idleSelfImprovementRequestV0(decision, now)
	requests := normalizeIdleSelfImprovementCausalRequestsV0(
		runtime.idleSelfImprovementRequestsV0(ctx, request, decision),
	)
	if len(requests) == 0 {
		payload := idleSelfImprovementAuditPayloadWithSkippedIdenticalV0(
			map[string]interface{}{"reason": "planner_empty", "request": request},
			runtime.tracker.ConsumeIdleSelfImprovementSkippedIdenticalV0(),
		)
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", payload)
		runtime.markIdleSelfImprovementCheckedV0(ctx, "planner_empty", now)
		return
	}
	publication := runtime.idleSelfImprovementDecisionPublicationV0(
		decision,
		idleSelfImprovementRequestRefsFromRequestsV0(requests),
	)
	if skip, skipped := runtime.tracker.RegisterIdleSelfImprovementPublicationV0(publication); skip {
		runtime.auditIdleSelfImprovementIdenticalWatchdogV0(ctx, tickCause, publication, skipped)
		return
	} else {
		payload := idleSelfImprovementAuditPayloadWithSkippedIdenticalV0(
			map[string]interface{}{"requests": requests},
			skipped,
		)
		runtime.auditEventV0(ctx, "idle_self_improvement_scheduled", "scheduled", "", payload)
	}
	runtime.tracker.MarkIdleSelfImprovementScheduledV0(now)
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkIdleSelfImprovementBudgetDecisionV0(decision.BudgetDecision, now),
		"idle_self_improvement_scheduled",
	)
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

func (runtime *RuntimeV0) idleSelfImprovementDecisionPublicationV0(
	decision idleSelfImprovementScheduleDecisionV0,
	requestRefs []string,
) idleSelfImprovementPublicationV0 {
	status := decision.AuditStatus
	if decision.Schedule {
		status = "scheduled"
	}
	refs := append([]string(nil), requestRefs...)
	if len(refs) == 0 {
		refs = append(refs, runtime.idleSelfImprovementDecisionRequestRefsV0(decision)...)
	}
	return newIdleSelfImprovementPublicationV0(status, decision.Reason, refs)
}

func (runtime *RuntimeV0) idleSelfImprovementDecisionRequestRefsV0(
	decision idleSelfImprovementScheduleDecisionV0,
) []string {
	refs := append([]string(nil), decision.KnownRequestRefs...)
	refs = append(refs, decision.RetryableRequestRefs...)
	for _, runRef := range append(append([]string(nil), decision.KnownRunRefs...), decision.BlockerRunRefs...) {
		refs = append(refs, idleSelfImprovementRequestRefFromRunRefV0(runRef))
	}
	state := runtime.tracker.SnapshotV0()
	if state.IdleSelfImprovementOperationalMessage != nil {
		refs = append(refs, state.IdleSelfImprovementOperationalMessage.RequestRefs...)
	}
	refs = append(refs, idleSelfImprovementRequestRefsFromReasonV0(state.IdleSelfImprovementReason)...)
	return compactConfigStringsV0(refs)
}

func idleSelfImprovementRequestRefsFromRequestsV0(
	requests []IdleSelfImprovementRequestV0,
) []string {
	refs := make([]string, 0, len(requests))
	for _, request := range requests {
		refs = append(refs, request.RequestRef)
	}
	return compactConfigStringsV0(refs)
}

func idleSelfImprovementRequestRefsFromReasonV0(reason string) []string {
	var refs []string
	for _, part := range strings.Split(reason, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || strings.TrimSpace(key) != "request_ref" {
			continue
		}
		refs = append(refs, value)
	}
	return compactConfigStringsV0(refs)
}

func (runtime *RuntimeV0) auditIdleSelfImprovementIdenticalWatchdogV0(
	ctx context.Context,
	tickCause string,
	publication idleSelfImprovementPublicationV0,
	skipped int,
) {
	if tickCause != supervisorTickCauseWatchdogV0 || skipped <= 0 {
		return
	}
	skipped = runtime.tracker.ConsumeIdleSelfImprovementSkippedIdenticalV0()
	if skipped <= 0 {
		return
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{
		"reason":            "identical_decision",
		"status":            publication.Status,
		"tick_cause":        supervisorTickCauseWatchdogV0,
		"request_refs":      append([]string(nil), publication.RequestRefs...),
		"skipped_identical": skipped,
	})
}
