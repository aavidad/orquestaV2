package orquestaserver

import (
	"context"
	"strings"
	"time"
)

func (runtime *RuntimeV0) markIdleSelfImprovementDecisionCheckedV0(
	ctx context.Context,
	decision idleSelfImprovementScheduleDecisionV0,
	now time.Time,
) {
	if decision.hasIdleSelfImprovementBlockerProjectionV0() {
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkIdleSelfImprovementBlockedV0(decision.blockerProjectionV0(), now),
			"idle_self_improvement_blocked",
		)
		return
	}
	runtime.markIdleSelfImprovementCheckedV0(ctx, decision.Reason, now)
}

func (decision idleSelfImprovementScheduleDecisionV0) hasIdleSelfImprovementBlockerProjectionV0() bool {
	return len(decision.BlockerRunRefs) > 0 ||
		len(decision.BlockerEvidence) > 0 ||
		strings.TrimSpace(decision.BlockerMessage) != "" ||
		strings.TrimSpace(decision.BlockerRecoveryAction) != ""
}

func (decision idleSelfImprovementScheduleDecisionV0) blockerProjectionV0() IdleSelfImprovementBlockerResultV0 {
	return IdleSelfImprovementBlockerResultV0{
		Blocked:        true,
		Reason:         decision.Reason,
		RunRefs:        append([]string(nil), decision.BlockerRunRefs...),
		EvidenceRefs:   append([]string(nil), decision.BlockerEvidence...),
		Message:        decision.BlockerMessage,
		RecoveryAction: decision.BlockerRecoveryAction,
		NextActions:    append([]string(nil), decision.BlockerNextActions...),
	}
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementBlockedV0(
	blocker IdleSelfImprovementBlockerResultV0,
	now time.Time,
) StateV0 {
	reason := idleSelfImprovementBlockerReasonV0(blocker)
	message := firstNonEmptyServerDiagnosticV0(blocker.Message, blocker.RecoveryAction, reason)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = reason
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "idle_self_improvement",
				ReasonCode:   idleSelfImprovementReasonCodeV0(reason),
				Status:       "blocked",
				Message:      message,
				RunRefs:      blocker.RunRefs,
				EvidenceRefs: blocker.EvidenceRefs,
				Counters: map[string]int{
					"attempts":      tracker.idleSelfImprovementAttempts,
					"accepted":      tracker.idleSelfImprovementPrepared,
					"run_refs":      len(compactConfigStringsV0(blocker.RunRefs)),
					"evidence_refs": len(compactConfigStringsV0(blocker.EvidenceRefs)),
					"next_actions":  len(compactConfigStringsV0(blocker.NextActions)),
				},
			},
		)
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	})
}

func idleSelfImprovementBlockerReasonV0(blocker IdleSelfImprovementBlockerResultV0) string {
	parts := []string{firstNonEmptyServerDiagnosticV0(blocker.Reason, "provider_blocked")}
	if len(compactConfigStringsV0(blocker.RunRefs)) > 0 {
		parts = append(parts, "run_ref="+compactConfigStringsV0(blocker.RunRefs)[0])
	}
	if len(compactConfigStringsV0(blocker.EvidenceRefs)) > 0 {
		parts = append(parts, "evidence_ref="+compactConfigStringsV0(blocker.EvidenceRefs)[0])
	}
	if strings.TrimSpace(blocker.RecoveryAction) != "" {
		parts = append(parts, projectServerOperationalReasonFieldV0("action", blocker.RecoveryAction))
	}
	return strings.Join(compactConfigStringsV0(parts), ";")
}
