package orquestaserver

import "time"

func (runtime *RuntimeV0) idleSelfImprovementIdleDecisionV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	decision.Trigger = idleSelfImprovementTriggerIdleV0
	if decision.IdleSince.IsZero() || now.Sub(decision.IdleSince) < runtime.config.IdleSelfImprovementAfter {
		decision.AuditStatus = "waiting"
		decision.Reason = "idle_window_waiting"
		return decision
	}
	if decision.InFlight || idleSelfImprovementAttemptBlocksV0(
		decision.IdleSince,
		decision.LastAttempt,
		now,
		runtime.config.IdleSelfImprovementAfter,
		decision.Accepted,
	) {
		decision.AuditStatus = "blocked"
		decision.Reason = "attempt_blocked"
		return decision
	}
	decision.Schedule = true
	decision.AuditStatus = "scheduled"
	decision.Reason = "scheduled"
	return decision
}

func (runtime *RuntimeV0) idleSelfImprovementCapacityDecisionV0(
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) idleSelfImprovementScheduleDecisionV0 {
	decision.Trigger = idleSelfImprovementTriggerCapacityFreeV0
	if _, ok := runtime.supervisor.(IdleSelfImprovementPlannerPortV0); !ok {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_planner_unavailable"
		return decision
	}
	if decision.InFlight {
		decision.AuditStatus = "blocked"
		decision.Reason = "capacity_attempt_in_flight"
		return decision
	}
	if decision.QueueSize <= 0 &&
		len(decision.RetryableRunRefs) == 0 &&
		len(decision.RetryableRequestRefs) == 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_queue_unknown"
		return decision
	}
	target := runtime.config.IdleSelfImprovementTargetQueue
	if target <= 0 {
		target = runtime.config.IdleSelfImprovementMaxRequests
	}
	decision.TargetQueue = target
	decision.FreeCapacity = target - decision.QueueSize
	if decision.FreeCapacity <= 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_full"
		return decision
	}
	decision.MaxRequests = minPositiveIdleSelfImprovementV0(runtime.config.IdleSelfImprovementMaxRequests, decision.FreeCapacity)
	if decision.MaxRequests <= 0 {
		decision.AuditStatus = "skipped"
		decision.Reason = "capacity_no_request_budget"
		return decision
	}
	decision.Schedule = true
	decision.AuditStatus = "scheduled"
	decision.Reason = "capacity_free"
	return decision
}

func (decision idleSelfImprovementScheduleDecisionV0) AuditPayload() map[string]interface{} {
	return map[string]interface{}{
		"reason":             decision.Reason,
		"trigger":            decision.Trigger,
		"idle_since":         formatTimeV0(decision.IdleSince),
		"last_attempt":       formatTimeV0(decision.LastAttempt),
		"in_flight":          decision.InFlight,
		"accepted":           decision.Accepted,
		"queue_size":         decision.QueueSize,
		"target_queue":       decision.TargetQueue,
		"free_capacity":      decision.FreeCapacity,
		"max_requests":       decision.MaxRequests,
		"skips":              decision.Skips,
		"known_run_refs":     append([]string(nil), decision.KnownRunRefs...),
		"known_request_refs": append([]string(nil), decision.KnownRequestRefs...),
		"retryable_run_refs": append([]string(nil), decision.RetryableRunRefs...),
		"retryable_request_refs": append(
			[]string(nil),
			decision.RetryableRequestRefs...,
		),
		"retryable_evidence": append(
			[]string(nil),
			decision.RetryableEvidenceRefs...,
		),
		"blocker_run_refs": append([]string(nil), decision.BlockerRunRefs...),
		"blocker_evidence": append([]string(nil), decision.BlockerEvidence...),
		"blocker_message":  publicAuditDiagnosticMessageV0(decision.BlockerMessage),
		"blocker_recovery_action": publicAuditDiagnosticMessageV0(
			decision.BlockerRecoveryAction,
		),
		"blocker_next_actions": append([]string(nil), decision.BlockerNextActions...),
		"external_wait_refs":   append([]string(nil), decision.ExternalWaitRefs...),
		"external_wait_evidence": append(
			[]string(nil),
			decision.ExternalWaitEvid...,
		),
	}
}

func minPositiveIdleSelfImprovementV0(left int, right int) int {
	if left <= 0 {
		return right
	}
	if right <= 0 || left < right {
		return left
	}
	return right
}
