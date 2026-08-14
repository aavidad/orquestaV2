package application

import (
	"errors"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var ErrAgentStopRecoveryInvalid = errors.New("application.agent_stop_recovery_invalid")

// SelectAgentStopRecoveryAttempt returns the sole ambiguous physical Stop
// attempt bound to a newly fenced recovery claim. Recovery is read-only: a
// receipt, crossed history or any second related attempt makes it ineligible.
func SelectAgentStopRecoveryAttempt(record GoalRecord, claim ActionClaim) (EffectAttempt, error) {
	if err := validateAgentStopRecoveryClaim(record, claim); err != nil {
		return EffectAttempt{}, err
	}
	attempt, err := PreflightAgentStopRecoveryAttempt(record, claim.Action)
	if err != nil || attempt.Ref != claim.RecoveryEffectAttemptRef || claim.Fence <= attempt.ActionFence {
		return EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	if err := validateRecoverableStopAttempt(record, claim, attempt); err != nil {
		return EffectAttempt{}, err
	}
	return attempt, nil
}

// PreflightAgentStopRecoveryAttempt is the pure admission probe used before a
// claim exists. It never turns malformed, settled or non-unique history into
// authority to inspect a provider.
func PreflightAgentStopRecoveryAttempt(record GoalRecord, action ActionRecord) (EffectAttempt, error) {
	intent := action.EffectIntent
	if !validAgentStopRecoveryAction(action) || !stopRecoverySubjectMatchesAction(intent.Subject, action) ||
		recoveryReceiptsRelated(record, action.Ref, intent.Ref) {
		return EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	var selected EffectAttempt
	matches := 0
	for _, attempt := range record.EffectAttempts {
		if !stopRecoveryAttemptRelated(intent, attempt) {
			continue
		}
		if !stopRecoveryAttemptMatches(intent, attempt) {
			return EffectAttempt{}, ErrAgentStopRecoveryInvalid
		}
		selected, matches = attempt, matches+1
	}
	if matches != 1 {
		return EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	return selected, nil
}

// BuildAgentStopRecoveryRequest reconstructs every provider-facing field from
// the durable control, execution and accepted launch, then restores only the
// historical Stop attempt authority selected above.
func BuildAgentStopRecoveryRequest(
	record GoalRecord,
	claim ActionClaim,
) (ports.AgentStopRequest, EffectAttempt, error) {
	attempt, err := SelectAgentStopRecoveryAttempt(record, claim)
	if err != nil {
		return ports.AgentStopRequest{}, EffectAttempt{}, err
	}
	_, execution, control, err := validateStopClaim(claim, record)
	if err != nil || !stopRecoveryExecutionEligible(execution) {
		return ports.AgentStopRequest{}, EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	request, err := agentStopRequest(record, control, execution)
	if err != nil || stopTargetDigest(control, request) != claim.Action.EffectIntent.TargetDigest {
		return ports.AgentStopRequest{}, EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	request.StopEffectAttemptRef, request.StopActionFence = attempt.Ref, attempt.ActionFence
	if err := ports.ValidateAgentStopRequest(request); err != nil {
		return ports.AgentStopRequest{}, EffectAttempt{}, ErrAgentStopRecoveryInvalid
	}
	return request, attempt, nil
}

func validateAgentStopRecoveryClaim(record GoalRecord, claim ActionClaim) error {
	if claim.Disposition != ActionClaimDispositionRecoverEffect ||
		!validApplicationRef(claim.RecoveryEffectAttemptRef) ||
		claim.RetryBudgetExhaustion != (RetryBudgetExhaustion{}) ||
		!stopRecoveryClaimHasNoResources(claim) {
		return ErrAgentStopRecoveryInvalid
	}
	if validateClaimedRecord(claim, record, ActionStopAgent) != nil ||
		!validAgentStopRecoveryAction(claim.Action) ||
		!stopRecoverySubjectMatchesAction(claim.Action.EffectIntent.Subject, claim.Action) {
		return ErrAgentStopRecoveryInvalid
	}
	if _, _, _, err := validateStopClaim(claim, record); err != nil {
		return ErrAgentStopRecoveryInvalid
	}
	return nil
}

func validateRecoverableStopAttempt(record GoalRecord, claim ActionClaim, attempt EffectAttempt) error {
	intent := claim.Action.EffectIntent
	approval, found := exactHistoricalApproval(record.EffectApprovals, attempt.ApprovalRef)
	if !found || ValidateEffectApproval(intent, approval) != nil || approval.Decision != EffectApproved ||
		approval != claim.EffectApproval {
		return ErrAgentStopRecoveryInvalid
	}
	if !stopRecoveryAttemptAuthorizedAt(intent, approval, attempt.StartedAt) {
		return ErrAgentStopRecoveryInvalid
	}
	return nil
}

func validateHistoricalStopAttemptIdentity(intent EffectIntent, attempt EffectAttempt) error {
	if !validApplicationRef(attempt.Ref) || !validApplicationRef(attempt.ApprovalRef) ||
		attempt.IntentRef != intent.Ref || attempt.IntentDigest != intent.Digest ||
		attempt.Subject != intent.Subject || attempt.ActionRef != intent.ActionRef ||
		attempt.IdempotencyKey != intent.IdempotencyKey || attempt.WorkerRef == "" {
		return ErrAgentStopRecoveryInvalid
	}
	if !validStopRecoveryAttemptWindow(attempt) {
		return ErrAgentStopRecoveryInvalid
	}
	return nil
}

func validAgentStopRecoveryAction(action ActionRecord) bool {
	intent := action.EffectIntent
	return action.Kind == ActionStopAgent && action.EffectIntentRef == intent.Ref &&
		ValidateEffectIntent(intent) == nil && intent.ActionRef == action.Ref &&
		intent.ActionKind == ActionStopAgent && intent.Kind == EffectKindAgentStop
}

func stopRecoverySubjectMatchesAction(subject EffectSubject, action ActionRecord) bool {
	return subject.GoalRef == action.GoalRef && subject.WorkItemRef == action.WorkItemRef &&
		subject.ExecutionRef == action.ExecutionRef && subject.PlanGeneration == action.PlanGeneration
}

func stopRecoveryExecutionEligible(execution ExecutionRecord) bool {
	return execution.State == ExecutionRunning && execution.ExternalRef != ""
}

func stopRecoveryClaimHasNoResources(claim ActionClaim) bool {
	return claim.BudgetReservationRef == "" && claim.BudgetReservation == (governance.BudgetReservation{}) &&
		claim.CapacityReservation == (AgentCapacityReservation{}) && claim.ReferenciaColocacion.String() == ""
}

func stopRecoveryAttemptAuthorizedAt(intent EffectIntent, approval EffectApproval, startedAt time.Time) bool {
	return !startedAt.Before(intent.CreatedAt) && !startedAt.Before(approval.DecidedAt) &&
		(approval.Source != EffectApprovalSourceExplicitDecision || approval.ExpiresAt.After(startedAt))
}

func validStopRecoveryAttemptWindow(attempt EffectAttempt) bool {
	return attempt.ActionFence != 0 && !attempt.StartedAt.IsZero() && !attempt.ClaimLeaseUntil.IsZero() &&
		attempt.ClaimLeaseUntil.After(attempt.StartedAt)
}

func stopRecoveryAttemptRelated(intent EffectIntent, attempt EffectAttempt) bool {
	return attempt.ActionRef == intent.ActionRef || attempt.IntentRef == intent.Ref
}

func stopRecoveryAttemptMatches(intent EffectIntent, attempt EffectAttempt) bool {
	return attempt.ActionRef == intent.ActionRef && attempt.IntentRef == intent.Ref &&
		validateHistoricalStopAttemptIdentity(intent, attempt) == nil
}
