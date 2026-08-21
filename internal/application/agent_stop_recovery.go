package application

import (
	"context"
	"errors"
	"reflect"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var errAgentStopRecoveryInvalid = errors.New("application.agent_stop_recovery_invalid")

// agentStopReconciler is optional and read-only: it may only recover the
// receipt of the exact Stop request already submitted by application.
type agentStopReconciler interface {
	ReconcileStop(context.Context, ports.AgentStopRequest) (ports.AgentStopReceipt, error)
}

// agentStopRecoveryClaimValidator lets the transactional state authority
// revalidate that the newly fenced claim still names the sole unresolved Stop
// attempt. Implementations must perform this check against live durable state.
type agentStopRecoveryClaimValidator interface {
	ValidateAgentStopRecoveryClaim(context.Context, ActionClaim) error
}

func (orchestrator *Orchestrator) processAgentStopRecovery(ctx context.Context, claim ActionClaim) error {
	validator, ok := orchestrator.state.(agentStopRecoveryClaimValidator)
	if !ok || ctx == nil || ctx.Err() != nil {
		return &StateError{Code: StateConflict}
	}
	if err := validator.ValidateAgentStopRecoveryClaim(ctx, claim); err != nil {
		return err
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	item, execution, control, err := validateStopRecoveryClaim(record, claim)
	if err != nil {
		return &StateError{Code: StateConflict}
	}
	attempt, err := SelectAgentStopRecoveryAttempt(record, claim)
	if err != nil {
		return &StateError{Code: StateConflict}
	}
	reconciler, ok := orchestrator.controller.(agentStopReconciler)
	if !ok || nilStopReconciler(reconciler) {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	request := stopRequest(control, execution)
	if ports.ValidateAgentStopRequest(request) != nil ||
		request.IdempotencyKey != claim.Action.EffectIntent.IdempotencyKey ||
		stopTargetDigest(control, request) != claim.Action.EffectIntent.TargetDigest {
		return &StateError{Code: StateConflict}
	}
	reconcileCtx, cancel := orchestrator.actionCallContext(ctx, claim)
	receipt, reconcileErr := reconciler.ReconcileStop(reconcileCtx, request)
	cancel()
	if reconcileErr != nil || ports.ValidateAgentStopReceipt(request, receipt) != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	// The remote read may race a reclaim, authority rotation, or another
	// recovery worker. Revalidate the same durable claim and historical attempt
	// before constructing a receipt or terminalizing lifecycle.
	if err := validator.ValidateAgentStopRecoveryClaim(ctx, claim); err != nil {
		return err
	}
	record, err = orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	item, execution, control, err = validateStopRecoveryClaim(record, claim)
	if err != nil || stopRequest(control, execution) != request {
		return &StateError{Code: StateConflict}
	}
	revalidatedAttempt, err := SelectAgentStopRecoveryAttempt(record, claim)
	if err != nil || revalidatedAttempt != attempt {
		return &StateError{Code: StateConflict}
	}
	switch receipt.Status {
	case ports.AgentStopAlreadyCompleted, ports.AgentStopAlreadyFailed:
		external, receiptErr := effectReceipt(claim, attempt, receipt.ReceiptRef, EffectStatus(receipt.Status), unknownUsage(), orchestrator.clock.Now().UTC())
		if receiptErr != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		return orchestrator.settleStopObservedTerminal(ctx, claim, record, item, execution, control, receipt, external)
	case ports.AgentStopped, ports.AgentStopAlreadyStopped:
		external, receiptErr := effectReceipt(claim, attempt, receipt.ReceiptRef, EffectStatus(receipt.Status), unknownUsage(), orchestrator.clock.Now().UTC())
		if receiptErr != nil {
			return orchestrator.quarantineUnknownApplied(ctx, claim)
		}
		return orchestrator.settleStopped(ctx, claim, record, item, execution, control, receipt, external)
	default:
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
}

func validateStopRecoveryClaim(record GoalRecord, claim ActionClaim) (goal.WorkItem, ExecutionRecord, ControlRecord, error) {
	if claim.Disposition != ActionClaimDispositionRecoverEffect || claim.Action.Kind != ActionStopAgent ||
		!validApplicationRef(claim.RecoveryEffectAttemptRef) || claim.Fence == 0 ||
		claim.RetryBudgetExhaustion != (RetryBudgetExhaustion{}) ||
		claim.BudgetReservationRef != "" || claim.BudgetReservation != (governance.BudgetReservation{}) ||
		claim.CapacityReservation != (AgentCapacityReservation{}) ||
		claim.ReferenciaColocacion != (ports.AgentPlacementRef{}) {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errAgentStopRecoveryInvalid
	}
	item, execution, control, err := validateStopClaim(claim, record)
	if err != nil || terminalExecutionState(execution.State) || execution.State == ExecutionDispatching || execution.ExternalRef == "" {
		return goal.WorkItem{}, ExecutionRecord{}, ControlRecord{}, errAgentStopRecoveryInvalid
	}
	return item, execution, control, nil
}

// SelectAgentStopRecoveryAttempt returns the sole unresolved historical Stop
// attempt named by a newly fenced recovery claim. It is shared with durable
// repositories so claim and dispatch enforce the same fail-closed predicate.
func SelectAgentStopRecoveryAttempt(record GoalRecord, claim ActionClaim) (EffectAttempt, error) {
	if _, _, _, err := validateStopRecoveryClaim(record, claim); err != nil {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	selected, err := PreflightAgentStopRecoveryAttempt(record, claim.Action)
	if err != nil || selected.Ref != claim.RecoveryEffectAttemptRef || claim.Fence <= selected.ActionFence {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	approval, found := exactHistoricalApproval(record.EffectApprovals, selected.ApprovalRef)
	if !found || approval != claim.EffectApproval || ValidateEffectApproval(claim.Action.EffectIntent, approval) != nil {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	return selected, nil
}

// PreflightAgentStopRecoveryAttempt is the pure pre-fence predicate used by
// repositories before issuing recovery authority. Any receipt, crossed
// identity, malformed attempt, or non-unique ambiguity fails closed.
func PreflightAgentStopRecoveryAttempt(record GoalRecord, action ActionRecord) (EffectAttempt, error) {
	intent := action.EffectIntent
	if action.Kind != ActionStopAgent || action.EffectIntentRef != intent.Ref ||
		ValidateEffectIntent(intent) != nil || intent.ActionRef != action.Ref ||
		intent.ActionKind != ActionStopAgent || intent.Kind != EffectKindAgentStop ||
		intent.Subject.GoalRef != action.GoalRef || intent.Subject.WorkItemRef != action.WorkItemRef ||
		intent.Subject.ExecutionRef != action.ExecutionRef || intent.Subject.PlanGeneration != action.PlanGeneration ||
		recoveryReceiptsRelated(record, action.Ref, intent.Ref) {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	blocking := make([]EffectAttempt, 0, 1)
	for _, attempt := range record.EffectAttempts {
		actionRelated, intentRelated := attempt.ActionRef == action.Ref, attempt.IntentRef == intent.Ref
		if !actionRelated && !intentRelated {
			continue
		}
		if actionRelated != intentRelated || !validApplicationRef(attempt.Ref) ||
			!validApplicationRef(attempt.ApprovalRef) || attempt.WorkerRef == "" ||
			attempt.IntentDigest != intent.Digest || attempt.Subject != intent.Subject ||
			attempt.IdempotencyKey != intent.IdempotencyKey || attempt.ActionFence == 0 ||
			attempt.StartedAt.IsZero() || !attempt.ClaimLeaseUntil.After(attempt.StartedAt) {
			return EffectAttempt{}, errAgentStopRecoveryInvalid
		}
		if !effectAttemptDefinitelyUnapplied(record, attempt) {
			blocking = append(blocking, attempt)
		}
	}
	if len(blocking) != 1 {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	return blocking[0], nil
}

func (orchestrator *Orchestrator) reconcileAmbiguousAgentStop(
	ctx context.Context,
	claim ActionClaim,
	originalAttempt EffectAttempt,
	originalRequest ports.AgentStopRequest,
) (GoalRecord, ports.AgentStopReceipt, error) {
	if ctx == nil || ctx.Err() != nil {
		return GoalRecord{}, ports.AgentStopReceipt{}, errAgentStopRecoveryInvalid
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return GoalRecord{}, ports.AgentStopReceipt{}, err
	}
	_, execution, control, err := validateStopClaim(claim, record)
	request, intent := stopRequest(control, execution), claim.Action.EffectIntent
	if err != nil || terminalExecutionState(execution.State) || execution.State == ExecutionDispatching ||
		execution.ExternalRef == "" || request != originalRequest || ports.ValidateAgentStopRequest(request) != nil ||
		request.IdempotencyKey != intent.IdempotencyKey || stopTargetDigest(control, request) != intent.TargetDigest ||
		validateClaimedEffect(claim, originalAttempt.StartedAt) != nil ||
		recoveryReceiptsRelated(record, claim.Action.Ref, intent.Ref) {
		return GoalRecord{}, ports.AgentStopReceipt{}, errAgentStopRecoveryInvalid
	}
	attempt, err := selectAgentStopAttempt(record, claim)
	if err != nil || attempt != originalAttempt || attempt.ActionFence != claim.Fence ||
		!attempt.ClaimLeaseUntil.Equal(claim.LeaseUntil.UTC()) {
		return GoalRecord{}, ports.AgentStopReceipt{}, errAgentStopRecoveryInvalid
	}
	reconciler, ok := orchestrator.controller.(agentStopReconciler)
	if !ok || nilStopReconciler(reconciler) {
		return GoalRecord{}, ports.AgentStopReceipt{}, errAgentStopRecoveryInvalid
	}
	reconcileCtx, cancel := orchestrator.actionCallContext(ctx, claim)
	defer cancel()
	if reconcileCtx.Err() != nil {
		return GoalRecord{}, ports.AgentStopReceipt{}, errAgentStopRecoveryInvalid
	}
	receipt, err := reconciler.ReconcileStop(reconcileCtx, request)
	return record, receipt, err
}

func selectAgentStopAttempt(record GoalRecord, claim ActionClaim) (EffectAttempt, error) {
	intent := claim.Action.EffectIntent
	var selected EffectAttempt
	found := false
	for _, attempt := range record.EffectAttempts {
		actionRelated, intentRelated := attempt.ActionRef == claim.Action.Ref, attempt.IntentRef == intent.Ref
		if !actionRelated && !intentRelated {
			continue
		}
		if actionRelated != intentRelated {
			return EffectAttempt{}, errAgentStopRecoveryInvalid
		}
		if effectAttemptDefinitelyUnapplied(record, attempt) {
			continue
		}
		approval, exact := exactHistoricalApproval(record.EffectApprovals, attempt.ApprovalRef)
		if found || !exact || approval != claim.EffectApproval || validateEffectAttempt(claim, attempt) != nil ||
			ValidateEffectApproval(intent, approval) != nil {
			return EffectAttempt{}, errAgentStopRecoveryInvalid
		}
		selected, found = attempt, true
	}
	if !found {
		return EffectAttempt{}, errAgentStopRecoveryInvalid
	}
	return selected, nil
}

func nilStopReconciler(reconciler agentStopReconciler) bool {
	if reconciler == nil {
		return true
	}
	value := reflect.ValueOf(reconciler)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
