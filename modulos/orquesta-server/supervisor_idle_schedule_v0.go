package orquestaserver

import (
	"context"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

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
	BudgetDecision        orquestaautoprogramming.AutoprogrammingIdleSelfImprovementBudgetDecisionV0
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
			wakeup := runtime.residentPendingDirectorWakeupV0(ctx)
			decision.Schedule = false
			decision.AuditStatus = "blocked"
			decision.Reason = idleSelfImprovementResidentPendingBlockedReasonV0
			decision.BlockerRunRefs = activePendingRefs
			decision.BlockerEvidence = compactConfigStringsV0(append(residentPending.EvidenceRefs, wakeup.EvidenceRefs...))
			decision.BlockerMessage = wakeup.Message
			decision.BlockerRecoveryAction = wakeup.RecoveryAction
			decision.BlockerNextActions = wakeup.NextActions
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
				if runtime.config.IdleSelfImprovementIdleDisabled {
					return runtime.idleSelfImprovementCapacityDecisionV0(decision, now)
				}
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
		if runtime.config.IdleSelfImprovementIdleDisabled {
			return runtime.idleSelfImprovementCapacityDecisionV0(decision, now)
		}
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

func (runtime *RuntimeV0) residentPendingDirectorWakeupV0(
	ctx context.Context,
) IdleSelfImprovementBlockerResultV0 {
	result := IdleSelfImprovementBlockerResultV0{
		Message:        idleSelfImprovementResidentPendingBlockedReasonV0,
		RecoveryAction: "resident_director_dispatch_or_causal_blocker_required",
		NextActions:    []string{"wake_resident_director_or_reconcile_run"},
	}
	if runtime == nil || runtime.residentDirector == nil || !runtime.config.ResidentDirectorEnabled {
		result.EvidenceRefs = []string{"evidence-ref-resident-director-wakeup-unavailable"}
		result.NextActions = []string{"enable_resident_director_or_reconcile_run"}
		return result
	}
	if runtime.RequestResidentDirectorWakeupV0(idleSelfImprovementResidentPendingBlockedReasonV0) {
		result.Message = "resident_director_wakeup_requested"
		result.EvidenceRefs = []string{"evidence-ref-resident-director-wakeup-requested"}
		result.NextActions = []string{"observe_resident_director_dispatch"}
		return result
	}
	result.EvidenceRefs = []string{"evidence-ref-resident-director-wakeup-pending_or_paused"}
	result.NextActions = []string{"observe_or_resume_resident_director"}
	if runtime.residentDirectorPausedV0() {
		result.RecoveryAction = "resume_resident_director_or_reconcile_run"
	}
	runtime.auditEventV0(ctx, "resident_director_wakeup", "skipped", "", map[string]interface{}{
		"reason": idleSelfImprovementResidentPendingBlockedReasonV0,
	})
	return result
}
