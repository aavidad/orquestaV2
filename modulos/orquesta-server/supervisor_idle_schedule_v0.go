package orquestaserver

import (
	"context"
	"strings"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func (runtime *RuntimeV0) maybeScheduleIdleSelfImprovementV0(
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
	port, ok := runtime.supervisor.(IdleSelfImprovementPortV0)
	if !ok || port == nil {
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
	requests := runtime.idleSelfImprovementRequestsV0(ctx, request, decision)
	if len(requests) == 0 {
		runtime.auditEventV0(ctx, "idle_self_improvement_check", "skipped", "", map[string]interface{}{"reason": "planner_empty", "request": request})
		runtime.markIdleSelfImprovementCheckedV0(ctx, "planner_empty", now)
		return
	}
	runtime.auditEventV0(ctx, "idle_self_improvement_scheduled", "scheduled", "", map[string]interface{}{"requests": requests})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkIdleSelfImprovementScheduledV0(now), "idle_self_improvement_scheduled")
	runtime.runAsyncWorkV0("idle_self_improvement_prepare", func() {
		runtime.prepareIdleSelfImprovementBatchV0(ctx, port, requests)
	})
}

type idleSelfImprovementScheduleDecisionV0 struct {
	Schedule              bool
	AuditStatus           string
	Reason                string
	Trigger               string
	IdleSince             time.Time
	LastAttempt           time.Time
	InFlight              bool
	Accepted              bool
	QueueSize             int
	TargetQueue           int
	FreeCapacity          int
	MaxRequests           int
	Skips                 int
	KnownRunRefs          []string
	KnownRequestRefs      []string
	RetryableRunRefs      []string
	RetryableRequestRefs  []string
	RetryableEvidenceRefs []string
	BlockerRunRefs        []string
	BlockerEvidence       []string
	BlockerMessage        string
	BlockerRecoveryAction string
	BlockerNextActions    []string
	ExternalWaitRefs      []string
	ExternalWaitEvid      []string
}

func (runtime *RuntimeV0) idleSelfImprovementScheduleDecisionV0(
	ctx context.Context,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	idleSince, lastAttempt, inFlight, accepted := runtime.tracker.IdleSelfImprovementWindowV0()
	metrics := collectSupervisorResultMetricsV0(result)
	knownRunRefs := idleSelfImprovementKnownRunRefsV0(result)
	knownRequestRefs := idleSelfImprovementKnownRequestRefsV0(knownRunRefs)
	freshness := runtime.idleSelfImprovementRunFreshnessV0(ctx, knownRunRefs, knownRequestRefs)
	if len(freshness.RetryableRunRefs) > 0 || len(freshness.RetryableRequestRefs) > 0 {
		knownRunRefs = idleSelfImprovementWithoutRetryableRefsV0(knownRunRefs, freshness.RetryableRunRefs, freshness.RetryableRequestRefs)
		knownRequestRefs = idleSelfImprovementWithoutRetryableRefsV0(knownRequestRefs, freshness.RetryableRunRefs, freshness.RetryableRequestRefs)
	}
	queueSize := len(knownRunRefs)
	if len(freshness.RetryableRunRefs) == 0 && len(freshness.RetryableRequestRefs) == 0 && metrics.QueueSize > queueSize {
		queueSize = metrics.QueueSize
	}
	decision := idleSelfImprovementScheduleDecisionV0{
		AuditStatus:      "skipped",
		Reason:           "supervisor_not_idle",
		Trigger:          idleSelfImprovementTriggerIdleV0,
		IdleSince:        idleSince,
		LastAttempt:      lastAttempt,
		InFlight:         inFlight,
		Accepted:         accepted,
		QueueSize:        queueSize,
		TargetQueue:      runtime.config.IdleSelfImprovementTargetQueue,
		MaxRequests:      runtime.config.IdleSelfImprovementMaxRequests,
		Skips:            metrics.Skips,
		KnownRunRefs:     knownRunRefs,
		KnownRequestRefs: knownRequestRefs,
		RetryableRunRefs: compactConfigStringsV0(freshness.RetryableRunRefs),
		RetryableRequestRefs: compactConfigStringsV0(
			freshness.RetryableRequestRefs,
		),
		RetryableEvidenceRefs: compactConfigStringsV0(
			freshness.EvidenceRefs,
		),
	}
	if blocked := runtime.idleSelfImprovementProviderBlockerV0(ctx, decision); blocked.Blocked {
		decision.Schedule = false
		decision.AuditStatus = "blocked"
		decision.Reason = firstNonEmptyIdleSelfImprovementV0(blocked.Reason, "provider_blocked")
		decision.BlockerRunRefs = compactConfigStringsV0(blocked.RunRefs)
		decision.BlockerEvidence = compactConfigStringsV0(blocked.EvidenceRefs)
		decision.BlockerMessage = strings.TrimSpace(blocked.Message)
		decision.BlockerRecoveryAction = strings.TrimSpace(blocked.RecoveryAction)
		decision.BlockerNextActions = compactConfigStringsV0(blocked.NextActions)
		return decision
	}
	if residentPending := detectResidentPendingWithoutDispatchV0(result); residentPending.Blocked {
		activePendingRefs := idleSelfImprovementWithoutRetryableRefsV0(
			residentPending.RunRefs,
			freshness.RetryableRunRefs,
			freshness.RetryableRequestRefs,
		)
		if len(activePendingRefs) > 0 {
			decision.Schedule = false
			decision.AuditStatus = "blocked"
			decision.Reason = idleSelfImprovementResidentPendingBlockedReasonV0
			decision.BlockerRunRefs = activePendingRefs
			decision.BlockerEvidence = compactConfigStringsV0(residentPending.EvidenceRefs)
			decision.BlockerMessage = idleSelfImprovementResidentPendingBlockedReasonV0
			decision.BlockerRecoveryAction = "resident_director_dispatch_or_causal_blocker_required"
			decision.BlockerNextActions = []string{"wake_resident_director_or_reconcile_run"}
			return decision
		}
	}
	if externalWait := detectIdleSelfImprovementExternalWaitBlockV0(result); externalWait.Blocked {
		activeWaitRefs := idleSelfImprovementWithoutRetryableRefsV0(
			externalWait.RunRefs,
			freshness.RetryableRunRefs,
			freshness.RetryableRequestRefs,
		)
		if len(activeWaitRefs) == 0 {
			decision.ExternalWaitRefs = compactConfigStringsV0(externalWait.RunRefs)
			decision.ExternalWaitEvid = compactConfigStringsV0(externalWait.EvidenceRefs)
			decision.RetryableRunRefs = compactConfigStringsV0(freshness.RetryableRunRefs)
			decision.RetryableRequestRefs = compactConfigStringsV0(freshness.RetryableRequestRefs)
			decision.RetryableEvidenceRefs = compactConfigStringsV0(freshness.EvidenceRefs)
			if len(decision.KnownRunRefs) == 0 {
				decision.QueueSize = 0
			}
			if supervisorResultIsIdleForSelfImprovementV0(result) {
				return runtime.idleSelfImprovementIdleDecisionV0(decision, now)
			}
			return runtime.idleSelfImprovementCapacityDecisionV0(decision, now)
		}
		decision.Schedule = false
		decision.AuditStatus = "blocked"
		decision.Reason = idleSelfImprovementExternalWaitBlockedReasonV0
		decision.ExternalWaitRefs = activeWaitRefs
		decision.ExternalWaitEvid = externalWait.EvidenceRefs
		return decision
	}
	if supervisorResultIsIdleForSelfImprovementV0(result) {
		return runtime.idleSelfImprovementIdleDecisionV0(decision, now)
	}
	return runtime.idleSelfImprovementCapacityDecisionV0(decision, now)
}

func (runtime *RuntimeV0) idleSelfImprovementProviderBlockerV0(
	ctx context.Context,
	decision idleSelfImprovementScheduleDecisionV0,
) IdleSelfImprovementBlockerResultV0 {
	blocker, ok := runtime.supervisor.(IdleSelfImprovementBlockerPortV0)
	if !ok || blocker == nil {
		return IdleSelfImprovementBlockerResultV0{}
	}
	result, err := blocker.IdleSelfImprovementBlockersV0(ctx, IdleSelfImprovementBlockerRequestV0{
		KnownRunRefs:     append([]string(nil), decision.KnownRunRefs...),
		KnownRequestRefs: append([]string(nil), decision.KnownRequestRefs...),
	})
	if err != nil {
		runtime.auditEventV0(ctx, "idle_self_improvement_blocker_error", "error", err.Error(), map[string]interface{}{
			"known_run_refs":     append([]string(nil), decision.KnownRunRefs...),
			"known_request_refs": append([]string(nil), decision.KnownRequestRefs...),
		})
		return IdleSelfImprovementBlockerResultV0{}
	}
	return result
}
