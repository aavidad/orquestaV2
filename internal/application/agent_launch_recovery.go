package application

import (
	"context"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

var (
	ErrAgentLaunchRecoveryInvalid     = errors.New("application.agent_launch_recovery_invalid")
	ErrAgentLaunchRecoveryUnsupported = errors.New("application.agent_launch_recovery_unsupported")
)

const (
	agentLaunchRecoveryApprovalRequiredCode = "governance.effect_approval_required"
	agentLaunchReconciliationPendingCode    = "agent.launch_reconciliation_pending"
)

// AgentLaunchReconciler is an optional, read-only recovery capability. Launch
// adapters that do not implement it can never receive a recovery request.
type AgentLaunchReconciler interface {
	ReconcileLaunch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
}

// AgentLaunchReconcilerFrom rejects ordinary launchers structurally. Recovery
// never falls back to Launch and never infers support from an error string.
func AgentLaunchReconcilerFrom(launcher AgentLauncher) (AgentLaunchReconciler, error) {
	reconciler, ok := launcher.(AgentLaunchReconciler)
	if launcher == nil || !ok {
		return nil, ErrAgentLaunchRecoveryUnsupported
	}
	return reconciler, nil
}

func (orchestrator *Orchestrator) processAgentLaunchRecovery(
	ctx context.Context,
	claim ActionClaim,
) error {
	if ctx == nil {
		return errors.New("application.context_required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := orchestrator.state.ValidateAgentLaunchRecoveryClaim(ctx, claim); err != nil {
		return err
	}
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	attempt, err := SelectAgentLaunchRecoveryAttempt(record, claim)
	if err != nil {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	item, itemFound := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, executionFound := executionForAction(record, claim.Action)
	if !itemFound || !executionFound || execution.State != ExecutionDispatching {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	phase, phaseFound := phaseForWorkItem(record.Goal, item)
	if !phaseFound {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}

	request, err := orchestrator.reconstructAgentLaunchRecoveryRequest(record, item, execution, phase)
	if err != nil {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	request.ReferenciaColocacion = claim.ReferenciaColocacion
	request.RequierePreservacionEntorno = execution.RequierePreservacionEntorno
	request.SessionRef = execution.ExecutionSessionRef
	request, attempt, err = BuildAgentLaunchRecoveryRequest(record, claim, request)
	if err != nil {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueAgentLaunchRecovery(
			ctx, claim, execution, agentLaunchRecoveryApprovalRequiredCode,
		)
	}
	reconciler, err := AgentLaunchReconcilerFrom(orchestrator.launcher)
	if err != nil {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	request.SessionRef, request.AccessAuthority, err = orchestrator.replayAgentLaunchRecoverySession(
		ctx, record, execution,
	)
	if err != nil {
		if cancellationErr := agentLaunchRecoveryCancellation(ctx, err); cancellationErr != nil {
			return cancellationErr
		}
		if isTemporaryAgentError(err) {
			return orchestrator.requeueAgentLaunchRecovery(
				ctx, claim, execution, agentLaunchReconciliationPendingCode,
			)
		}
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	request, rebuiltAttempt, err := BuildAgentLaunchRecoveryRequest(record, claim, request)
	if err != nil || rebuiltAttempt != attempt {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	receipt, err := reconciler.ReconcileLaunch(ctx, request)
	if err != nil {
		if cancellationErr := agentLaunchRecoveryCancellation(ctx, err); cancellationErr != nil {
			return cancellationErr
		}
		if isTemporaryAgentError(err) {
			return orchestrator.requeueAgentLaunchRecovery(
				ctx, claim, execution, agentLaunchReconciliationPendingCode,
			)
		}
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil ||
		receipt.ProviderRef != orchestrator.agentCapabilities.ProviderRef ||
		receipt.ModelRef != orchestrator.agentCapabilities.ModelRef ||
		receipt.AgentRef != orchestrator.agentCapabilities.AgentRef {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	latest, latestItem, latestExecution, err := orchestrator.reloadPreparedLaunch(ctx, claim)
	if err != nil {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	latestAttempt, err := SelectAgentLaunchRecoveryAttempt(latest, claim)
	if err != nil || latestAttempt != attempt {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	if _, rebuiltAttempt, buildErr := BuildAgentLaunchRecoveryRequest(latest, claim, request); buildErr != nil || rebuiltAttempt != attempt {
		return orchestrator.failAgentLaunchRecoveryUnknownApplied(ctx, claim)
	}
	record, item, execution = latest, latestItem, latestExecution
	transitionAt := lifecycleTime(orchestrator.clock.Now(), record.Goal, item)
	return orchestrator.completeAcceptedLaunch(
		ctx, claim, record, item, execution, attempt, receipt,
		transitionAt, receipt.AcceptedAt, false,
	)
}

func (orchestrator *Orchestrator) reconstructAgentLaunchRecoveryRequest(
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	phase goal.PhaseInstance,
) (ports.AgentLaunchRequest, error) {
	var request ports.AgentLaunchRequest
	var err error
	switch {
	case isReviewerExecution(execution):
		if err := validateReviewerLaunch(record, item, execution, orchestrator.testAttestationPolicy); err != nil {
			return ports.AgentLaunchRequest{}, ErrAgentLaunchRecoveryInvalid
		}
		request, err = reviewerAgentLaunchRequest(record, item, execution, phase, orchestrator.testAttestationPolicy)
	case isCouncilExecution(execution):
		if err := validateCouncilLaunch(record, item, execution); err != nil {
			return ports.AgentLaunchRequest{}, ErrAgentLaunchRecoveryInvalid
		}
		request, err = councilAgentLaunchRequest(record, item, execution, phase)
	case execution.Purpose == ExecutionPurposeWork || execution.Purpose == ExecutionPurposeAuthor:
		bound, found := item.Execution()
		if item.State() != goal.WorkItemStateRunning || !found || bound != execution.Ref {
			return ports.AgentLaunchRequest{}, ErrAgentLaunchRecoveryInvalid
		}
		request = agentLaunchRequest(record.Goal, item, execution, phase)
	default:
		return ports.AgentLaunchRequest{}, ErrAgentLaunchRecoveryInvalid
	}
	if err != nil || bindDurableAgentLaunchEgressAuthority(record, &request) != nil {
		return ports.AgentLaunchRequest{}, ErrAgentLaunchRecoveryInvalid
	}
	return request, nil
}

func (orchestrator *Orchestrator) replayAgentLaunchRecoverySession(
	ctx context.Context,
	record GoalRecord,
	execution ExecutionRecord,
) (ports.ExecutionSessionRef, ports.AgentLaunchAccessAuthority, error) {
	if execution.ExecutionSessionRef.String() == "" {
		if orchestrator.executionSessions != nil {
			return "", ports.AgentLaunchAccessAuthority{}, ErrAgentLaunchRecoveryInvalid
		}
		return "", ports.AgentLaunchAccessAuthority{}, nil
	}
	if orchestrator.executionSessions == nil {
		return "", ports.AgentLaunchAccessAuthority{}, ErrAgentLaunchRecoveryInvalid
	}
	request := ExecutionSessionRequest(record.Goal, execution)
	receipt, err := orchestrator.executionSessions.Ensure(ctx, request)
	if err != nil {
		return "", ports.AgentLaunchAccessAuthority{}, err
	}
	method := receipt.Authority.ServicePrincipal.Method
	expected, deriveErr := DeriveExecutionSessionAuthority(request, method)
	if deriveErr != nil || receipt.EnsuredAt.IsZero() || receipt.Authority != expected ||
		receipt.Authority.Request != request ||
		receipt.Authority.SessionRef != execution.ExecutionSessionRef {
		return "", ports.AgentLaunchAccessAuthority{}, ErrAgentLaunchRecoveryInvalid
	}
	return receipt.Authority.SessionRef, ports.AgentLaunchAccessAuthority{
		ArtifactAccessRef:  receipt.Authority.ArtifactAccessRef,
		MCPAccessRef:       receipt.Authority.MCPAccessRef,
		MailboxEndpointRef: receipt.Authority.MailboxEndpointRef,
	}, nil
}

func (orchestrator *Orchestrator) failAgentLaunchRecoveryUnknownApplied(
	ctx context.Context,
	claim ActionClaim,
) error {
	if ctx == nil {
		return errors.New("application.context_required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return orchestrator.quarantineUnknownApplied(ctx, claim)
}

func (orchestrator *Orchestrator) requeueAgentLaunchRecovery(
	ctx context.Context,
	claim ActionClaim,
	execution ExecutionRecord,
	code string,
) error {
	if ctx == nil {
		return errors.New("application.context_required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	intent := claim.Action.EffectIntent
	if claim.Disposition != ActionClaimDispositionRecoverEffect ||
		claim.Action.Kind != ActionLaunchAgent ||
		!validApplicationRef(claim.RecoveryEffectAttemptRef) ||
		claim.Action.EffectIntentRef != intent.Ref || intent.QuotaRetryDelay <= 0 ||
		execution.State != ExecutionDispatching || execution.Ref != claim.Action.ExecutionRef ||
		execution.GoalRef != claim.Action.GoalRef || execution.WorkItemRef != claim.Action.WorkItemRef ||
		execution.PlanGeneration != claim.Action.PlanGeneration ||
		execution.BudgetReservationRef == "" || execution.BudgetReservationRef != claim.BudgetReservationRef ||
		execution.EffectIntentRef == "" || execution.EffectIntentRef != claim.Action.EffectIntentRef {
		return &StateError{Code: StateConflict}
	}
	now := orchestrator.clock.Now().UTC()
	return orchestrator.state.RequeueAction(ctx, ActionRequeuedState{
		Claim: claim, Execution: execution, ErrorCode: code,
		AvailableAt: now.Add(intent.QuotaRetryDelay), OperationAt: now,
		BudgetSettlement: nil, ClearEffectBinding: false,
	})
}

func agentLaunchRecoveryCancellation(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// SelectAgentLaunchRecoveryAttempt returns the sole ambiguous physical launch
// attempt bound to a newly fenced recovery claim. A receipt, exact zero-release
// or any second blocking attempt makes recovery ineligible.
func SelectAgentLaunchRecoveryAttempt(record GoalRecord, claim ActionClaim) (EffectAttempt, error) {
	if err := validateAgentLaunchRecoveryClaim(record, claim); err != nil {
		return EffectAttempt{}, err
	}
	attempt, err := PreflightAgentLaunchRecoveryAttempt(record, claim.Action)
	if err != nil || attempt.Ref != claim.RecoveryEffectAttemptRef || claim.Fence <= attempt.ActionFence {
		return EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	if err := validateRecoverableLaunchAttempt(record, claim, attempt); err != nil {
		return EffectAttempt{}, err
	}
	return attempt, nil
}

// PreflightAgentLaunchRecoveryAttempt returns the sole physical launch attempt
// that still has an ambiguous outcome. It is pure: receipts veto recovery,
// exact causal zero-releases are discarded, and malformed or non-unique
// history is never converted into authority.
func PreflightAgentLaunchRecoveryAttempt(record GoalRecord, action ActionRecord) (EffectAttempt, error) {
	blocking, err := blockingAgentLaunchAttempts(record, action)
	if err != nil || len(blocking) != 1 {
		return EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	return blocking[0], nil
}

func blockingAgentLaunchAttempts(record GoalRecord, action ActionRecord) ([]EffectAttempt, error) {
	intent := action.EffectIntent
	if action.Kind != ActionLaunchAgent || action.EffectIntentRef != intent.Ref ||
		ValidateEffectIntent(intent) != nil || intent.ActionRef != action.Ref ||
		intent.ActionKind != ActionLaunchAgent || intent.Kind != EffectKindAgentLaunch ||
		intent.Subject.GoalRef != action.GoalRef || intent.Subject.WorkItemRef != action.WorkItemRef ||
		intent.Subject.ExecutionRef != action.ExecutionRef || intent.Subject.PlanGeneration != action.PlanGeneration ||
		recoveryReceiptsRelated(record, action.Ref, intent.Ref) {
		return nil, ErrAgentLaunchRecoveryInvalid
	}
	blocking := make([]EffectAttempt, 0, 1)
	for _, attempt := range record.EffectAttempts {
		if attempt.ActionRef != action.Ref || attempt.IntentRef != intent.Ref {
			continue
		}
		if err := validateHistoricalLaunchAttemptIdentity(intent, attempt); err != nil {
			return nil, err
		}
		if !effectAttemptDefinitelyUnapplied(record, attempt) {
			blocking = append(blocking, attempt)
		}
	}
	return blocking, nil
}

// BuildAgentLaunchRecoveryRequest preserves the already-derived request and
// replaces only its effect authority with historical durable bytes.
func BuildAgentLaunchRecoveryRequest(
	record GoalRecord,
	claim ActionClaim,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchRequest, EffectAttempt, error) {
	attempt, err := SelectAgentLaunchRecoveryAttempt(record, claim)
	if err != nil {
		return ports.AgentLaunchRequest{}, EffectAttempt{}, err
	}
	execution, found := executionForAction(record, claim.Action)
	if !found || validateAgentLaunchRecoveryBindings(record, claim, execution, request) != nil {
		return ports.AgentLaunchRequest{}, EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	approval, found := exactHistoricalApproval(record.EffectApprovals, attempt.ApprovalRef)
	if !found {
		return ports.AgentLaunchRequest{}, EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	request.EffectAuthority = ports.AgentLaunchEffectAuthority{
		AuthorizationReceiptRef: claim.Action.EffectIntent.Authority.Ref(),
		EffectApprovalRef:       approval.Ref,
		EffectAttemptRef:        attempt.Ref,
		ActionFence:             attempt.ActionFence,
		StartedAt:               attempt.StartedAt,
		ClaimLeaseUntil:         attempt.ClaimLeaseUntil,
		ApprovalExpiresAt:       approval.ExpiresAt,
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchRequest{}, EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return ports.AgentLaunchRequest{}, EffectAttempt{}, ErrAgentLaunchRecoveryInvalid
	}
	return request, attempt, nil
}

func validateAgentLaunchRecoveryClaim(record GoalRecord, claim ActionClaim) error {
	intent := claim.Action.EffectIntent
	if claim.Disposition != ActionClaimDispositionRecoverEffect ||
		!validApplicationRef(claim.RecoveryEffectAttemptRef) ||
		claim.RetryBudgetExhaustion != (RetryBudgetExhaustion{}) ||
		validateClaimedRecord(claim, record, ActionLaunchAgent) != nil ||
		ValidateEffectIntent(intent) != nil ||
		claim.Action.EffectIntentRef != intent.Ref || intent.ActionRef != claim.Action.Ref ||
		intent.ActionKind != ActionLaunchAgent || intent.Kind != EffectKindAgentLaunch ||
		intent.Subject.GoalRef != claim.Action.GoalRef ||
		intent.Subject.WorkItemRef != claim.Action.WorkItemRef ||
		intent.Subject.ExecutionRef != claim.Action.ExecutionRef ||
		intent.Subject.PlanGeneration != claim.Action.PlanGeneration {
		return ErrAgentLaunchRecoveryInvalid
	}
	execution, found := executionForAction(record, claim.Action)
	if !found || execution.State != ExecutionDispatching ||
		execution.BudgetReservationRef != claim.BudgetReservationRef ||
		execution.EffectIntentRef != intent.Ref {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func validateRecoverableLaunchAttempt(record GoalRecord, claim ActionClaim, attempt EffectAttempt) error {
	intent := claim.Action.EffectIntent
	approval, found := exactHistoricalApproval(record.EffectApprovals, attempt.ApprovalRef)
	if !found || ValidateEffectApproval(intent, approval) != nil || approval.Decision != EffectApproved ||
		approval != claim.EffectApproval || attempt.StartedAt.Before(intent.CreatedAt) ||
		attempt.StartedAt.Before(approval.DecidedAt) ||
		(approval.Source == EffectApprovalSourceExplicitDecision && !approval.ExpiresAt.After(attempt.StartedAt)) {
		return ErrAgentLaunchRecoveryInvalid
	}
	return validateAgentLaunchRecoveryReservations(claim, attempt)
}

func validateAgentLaunchRecoveryReservations(claim ActionClaim, attempt EffectAttempt) error {
	intent, reservation := claim.Action.EffectIntent, claim.BudgetReservation
	capacity := claim.CapacityReservation
	if governance.ValidateBudgetReservation(reservation) != nil ||
		claim.BudgetReservationRef != reservation.Ref || reservation.DemandRef != intent.Demand.Ref ||
		reservation.ActionRef != claim.Action.Ref || reservation.EffectIntentRef != intent.Ref ||
		reservation.ProjectRef != intent.Subject.ProjectRef.String() ||
		reservation.GoalRef != intent.Subject.GoalRef.String() ||
		reservation.WorkItemRef != intent.Subject.WorkItemRef.String() ||
		reservation.ExecutionRef != intent.Subject.ExecutionRef.String() ||
		reservation.PlanGeneration != uint64(intent.Subject.PlanGeneration) ||
		reservation.AppSpecGeneration != uint64(intent.Subject.AppSpecGeneration) ||
		reservation.WorkItemGeneration != uint64(claim.Action.WorkItemGeneration) ||
		reservation.Fence > attempt.ActionFence || reservation.SpecHash != intent.Subject.SpecHash ||
		reservation.PolicyHash != intent.PolicyHash || reservation.Resources != intent.Demand.Resources ||
		reservation.ReservedAt.After(attempt.StartedAt) ||
		ValidateAgentCapacityReservation(capacity) != nil ||
		(capacity.State != AgentCapacityReserved && capacity.State != AgentCapacityQuarantined) ||
		capacity.ActionRef != claim.Action.Ref || capacity.EffectIntentRef != intent.Ref ||
		capacity.ProjectRef != intent.Subject.ProjectRef || capacity.GoalRef != intent.Subject.GoalRef ||
		capacity.WorkItemRef != intent.Subject.WorkItemRef || capacity.ExecutionRef != intent.Subject.ExecutionRef ||
		capacity.PlanGeneration != intent.Subject.PlanGeneration ||
		capacity.WorkItemGeneration != claim.Action.WorkItemGeneration || capacity.Fence > attempt.ActionFence ||
		claim.ReferenciaColocacion.String() == "" {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func validateAgentLaunchRecoveryBindings(
	record GoalRecord,
	claim ActionClaim,
	execution ExecutionRecord,
	request ports.AgentLaunchRequest,
) error {
	intent := claim.Action.EffectIntent
	durableEgress, egressErr := durableAgentLaunchEgressAuthority(record, request.WorkItemRef)
	if execution.State != ExecutionDispatching || request.ExecutionRef != execution.Ref ||
		request.GoalRef != execution.GoalRef || request.WorkItemRef != execution.WorkItemRef ||
		request.PlanGeneration != execution.PlanGeneration ||
		request.AppSpecGeneration != execution.AppSpecGeneration ||
		request.ExecutionAttempt != execution.AttemptNo || request.SpecHash != execution.SpecHash ||
		request.ActorRef != intent.Subject.ActorRef || request.ProjectRef != intent.Subject.ProjectRef ||
		request.SessionRef != execution.ExecutionSessionRef ||
		request.ExecutionWorkspaceRef != execution.ExecutionWorkspaceRef ||
		request.IdempotencyKey != execution.IdempotencyKey || request.IdempotencyKey != intent.IdempotencyKey ||
		request.ReferenciaColocacion != claim.ReferenciaColocacion ||
		request.BudgetDemand != intent.Demand || request.RequierePreservacionEntorno != execution.RequierePreservacionEntorno ||
		record.Goal.Ref() != intent.Subject.GoalRef || record.Goal.Project() != intent.Subject.ProjectRef ||
		record.Goal.Actor() != intent.Subject.ActorRef || record.Goal.SpecHash() != intent.Subject.SpecHash ||
		egressErr != nil || !ports.EqualAgentLaunchEgressAuthority(request.EgressAuthority, durableEgress) ||
		record.Goal.AppSpec().Generation() != intent.Subject.AppSpecGeneration {
		return ErrAgentLaunchRecoveryInvalid
	}
	target := authorLaunchTargetDigest(request)
	if isReviewerExecution(execution) || isCouncilExecution(execution) {
		target = reviewerLaunchTargetDigest(request)
	}
	if target != intent.TargetDigest {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func exactHistoricalApproval(approvals []EffectApproval, ref string) (EffectApproval, bool) {
	var selected EffectApproval
	matches := 0
	for _, approval := range approvals {
		if approval.Ref == ref {
			selected = approval
			matches++
		}
	}
	return selected, matches == 1
}

func validateHistoricalLaunchAttemptIdentity(intent EffectIntent, attempt EffectAttempt) error {
	if !validApplicationRef(attempt.Ref) || !validApplicationRef(attempt.ApprovalRef) ||
		attempt.IntentRef != intent.Ref || attempt.IntentDigest != intent.Digest ||
		attempt.Subject != intent.Subject || attempt.ActionRef != intent.ActionRef ||
		attempt.IdempotencyKey != intent.IdempotencyKey || attempt.WorkerRef == "" ||
		attempt.ActionFence == 0 || attempt.StartedAt.IsZero() || attempt.ClaimLeaseUntil.IsZero() ||
		!attempt.ClaimLeaseUntil.After(attempt.StartedAt) {
		return ErrAgentLaunchRecoveryInvalid
	}
	return nil
}

func recoveryReceiptsRelated(record GoalRecord, actionRef, intentRef string) bool {
	attemptRefs := make(map[string]struct{})
	for _, attempt := range record.EffectAttempts {
		if attempt.ActionRef == actionRef || attempt.IntentRef == intentRef {
			attemptRefs[attempt.Ref] = struct{}{}
		}
	}
	for _, receipt := range record.EffectReceipts {
		_, attemptRelated := attemptRefs[receipt.AttemptRef]
		if receipt.ActionRef == actionRef || receipt.IntentRef == intentRef || attemptRelated {
			return true
		}
	}
	return false
}
