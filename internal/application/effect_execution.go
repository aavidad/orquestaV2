package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func validateClaimedEffect(claim ActionClaim, at time.Time) error {
	if !actionUsesEffectLedger(claim.Action.Kind) {
		return nil
	}
	intent := claim.Action.EffectIntent
	if err := ValidateEffectIntent(intent); err != nil || claim.Action.EffectIntentRef != intent.Ref ||
		intent.ActionRef != claim.Action.Ref || intent.Subject.GoalRef != claim.Action.GoalRef ||
		intent.Subject.WorkItemRef != claim.Action.WorkItemRef || intent.Subject.ExecutionRef != claim.Action.ExecutionRef {
		return errors.New("application.effect_claim_intent_invalid")
	}
	approval := claim.EffectApproval
	if err := ValidateEffectApproval(intent, approval); err != nil || approval.Decision != EffectApproved ||
		approval.DecidedAt.After(at.UTC()) ||
		(approval.Source == EffectApprovalSourceExplicitDecision && !approval.ExpiresAt.After(at.UTC())) {
		return errors.New("application.effect_live_approval_required")
	}
	if claim.Action.Kind != ActionLaunchAgent {
		if claim.BudgetReservationRef != "" || claim.BudgetReservation != (governance.BudgetReservation{}) {
			return errors.New("application.local_effect_budget_reservation_forbidden")
		}
		return nil
	}
	reservation := claim.BudgetReservation
	if err := governance.ValidateBudgetReservation(reservation); err != nil ||
		claim.BudgetReservationRef != reservation.Ref || reservation.DemandRef != intent.Demand.Ref ||
		reservation.ActionRef != claim.Action.Ref || reservation.EffectIntentRef != intent.Ref ||
		reservation.ProjectRef != intent.Subject.ProjectRef.String() || reservation.GoalRef != intent.Subject.GoalRef.String() ||
		reservation.WorkItemRef != intent.Subject.WorkItemRef.String() || reservation.ExecutionRef != intent.Subject.ExecutionRef.String() ||
		reservation.PlanGeneration != uint64(intent.Subject.PlanGeneration) ||
		reservation.AppSpecGeneration != uint64(intent.Subject.AppSpecGeneration) ||
		reservation.WorkItemGeneration != uint64(claim.Action.WorkItemGeneration) || reservation.Fence > claim.Fence ||
		reservation.SpecHash != intent.Subject.SpecHash || reservation.PolicyHash != intent.PolicyHash ||
		reservation.Resources != intent.Demand.Resources {
		return errors.New("application.effect_budget_reservation_invalid")
	}
	return nil
}

func (orchestrator *Orchestrator) beginEffectAttempt(
	ctx context.Context,
	claim ActionClaim,
	at time.Time,
) (EffectAttempt, error) {
	if err := validateClaimedEffect(claim, at); err != nil {
		return EffectAttempt{}, err
	}
	intent, approval := claim.Action.EffectIntent, claim.EffectApproval
	attempt := EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, ApprovalRef: approval.Ref,
		Subject: intent.Subject, ActionRef: claim.Action.Ref, ActionFence: claim.Fence,
		WorkerRef: claim.WorkerRef, IdempotencyKey: intent.IdempotencyKey, StartedAt: at.UTC(),
	}
	persisted, _, err := orchestrator.state.RecordEffectAttempt(ctx, RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: at.UTC(),
	})
	if err != nil {
		return EffectAttempt{}, err
	}
	if err := validateEffectAttempt(claim, persisted); err != nil {
		return EffectAttempt{}, err
	}
	return persisted, nil
}

func (orchestrator *Orchestrator) validateCurrentAutomaticAuthority(
	ctx context.Context,
	claim ActionClaim,
) error {
	intent := claim.Action.EffectIntent
	if err := orchestrator.validateCurrentEffectMembership(
		ctx, intent.ProposedBy, intent.Subject.ProjectRef, intent.Permission, intent.Authority,
	); err != nil {
		return errors.New("application.effect_automatic_authority_inactive")
	}
	approval := claim.EffectApproval
	if approval.Source == EffectApprovalSourceExplicitDecision {
		if err := orchestrator.validateCurrentEffectMembership(
			ctx, approval.DecidedBy, intent.Subject.ProjectRef,
			identity.PermissionEffectsApprove, approval.AuthorizationReceipt,
		); err != nil {
			return errors.New("application.effect_explicit_authority_inactive")
		}
	}
	return nil
}

func (orchestrator *Orchestrator) validateCurrentEffectMembership(
	ctx context.Context,
	principal identity.PrincipalRef,
	project goal.ProjectRef,
	permission identity.Permission,
	authorization identity.AuthorizationReceipt,
) error {
	decision := authorization.Decision()
	if decision.Role() == identity.RolePlatformAdmin && decision.MembershipRevision() == 0 {
		if identity.RoleAllows(identity.RolePlatformAdmin, permission) {
			return nil
		}
		return errors.New("application.effect_permission_inactive")
	}
	membership, err := orchestrator.access.Membership(ctx, principal, project)
	if err != nil || membership.PrincipalRef() != principal || membership.ProjectRef() != project ||
		!membership.IsActive() || membership.Role() != decision.Role() ||
		membership.Revision() != decision.MembershipRevision() ||
		!identity.RoleAllows(membership.Role(), permission) {
		return errors.New("application.effect_membership_inactive")
	}
	return nil
}

func validateEffectAttempt(claim ActionClaim, attempt EffectAttempt) error {
	intent := claim.Action.EffectIntent
	if !validApplicationRef(attempt.Ref) || attempt.IntentRef != intent.Ref || attempt.IntentDigest != intent.Digest ||
		attempt.ApprovalRef != claim.EffectApproval.Ref || attempt.Subject != intent.Subject ||
		attempt.ActionRef != claim.Action.Ref || attempt.ActionFence != claim.Fence ||
		attempt.WorkerRef != claim.WorkerRef || attempt.IdempotencyKey != intent.IdempotencyKey ||
		attempt.StartedAt.IsZero() {
		return errors.New("application.effect_attempt_invalid")
	}
	return nil
}

func effectReceipt(
	claim ActionClaim,
	attempt EffectAttempt,
	externalRef string,
	status EffectStatus,
	usage governance.ResourceUsage,
	at time.Time,
) (EffectReceipt, error) {
	intent := claim.Action.EffectIntent
	receipt := EffectReceipt{
		Ref: "effect-receipt:" + intent.Ref, IntentRef: intent.Ref, IntentDigest: intent.Digest,
		ApprovalRef: claim.EffectApproval.Ref, AttemptRef: attempt.Ref, Subject: intent.Subject,
		ActionRef: claim.Action.Ref, ActionFence: claim.Fence, IdempotencyKey: intent.IdempotencyKey,
		ExternalRef: externalRef, Status: status, Usage: usage, ConfirmedAt: at.UTC(),
	}
	if err := validateEffectReceipt(claim, attempt, receipt); err != nil {
		return EffectReceipt{}, err
	}
	return receipt, nil
}

func validateEffectReceipt(claim ActionClaim, attempt EffectAttempt, receipt EffectReceipt) error {
	intent := claim.Action.EffectIntent
	if !validApplicationRef(receipt.Ref) || !validApplicationRef(receipt.ExternalRef) ||
		!validEffectStatus(intent.Kind, receipt.Status) || receipt.IntentRef != intent.Ref ||
		receipt.IntentDigest != intent.Digest || receipt.ApprovalRef != claim.EffectApproval.Ref ||
		receipt.AttemptRef != attempt.Ref || receipt.Subject != intent.Subject ||
		receipt.ActionRef != claim.Action.Ref || receipt.ActionFence != claim.Fence ||
		receipt.IdempotencyKey != intent.IdempotencyKey || receipt.ConfirmedAt.Before(attempt.StartedAt) ||
		governance.ValidateResourceUsage(receipt.Usage) != nil {
		return errors.New("application.effect_receipt_invalid")
	}
	return nil
}

func validEffectStatus(kind EffectKind, status EffectStatus) bool {
	switch kind {
	case EffectKindAgentLaunch:
		return status == EffectStatusAccepted
	case EffectKindAgentStop:
		return status == EffectStatusStopped || status == EffectStatusAlreadyStopped ||
			status == EffectStatusAlreadyCompleted || status == EffectStatusAlreadyFailed
	case EffectKindPrepareWorkspace:
		return status == EffectStatusPrepared
	case EffectKindCommitChange:
		return status == EffectStatusCommitted
	case EffectKindIntegrateChange:
		return status == EffectStatusIntegrated || status == EffectStatusConflicted || status == EffectStatusStale
	default:
		return false
	}
}

func actionUsesEffectLedger(kind ActionKind) bool {
	switch kind {
	case ActionLaunchAgent, ActionStopAgent, ActionPrepareWorkspace, ActionCommitChange, ActionIntegrateChange:
		return true
	default:
		return false
	}
}

func unknownUsage() governance.ResourceUsage {
	return governance.ResourceUsage{Quality: governance.UsageQualityUnknown}
}

func settlementFor(
	record GoalRecord,
	execution ExecutionRecord,
	usage governance.ResourceUsage,
	diskBytes int64,
	at time.Time,
) (*governance.BudgetSettlement, error) {
	if execution.BudgetReservationRef == "" {
		return nil, nil
	}
	reservation, found := reservationByRef(record.BudgetReservations, execution.BudgetReservationRef)
	if !found {
		return nil, errors.New("application.budget_reservation_missing")
	}
	usage.Known |= governance.ResourceActiveTime | governance.ResourceProcessSlots | governance.ResourceDisk
	usage.Resources.ActiveTimeNS = measuredActiveTime(execution, at)
	usage.Resources.ProcessSlots = 0
	usage.Resources.DiskBytes = diskBytes
	if usage.Known != 0 && (usage.Quality == governance.UsageQualityUnknown ||
		usage.Quality == governance.UsageQualityExact) {
		usage.Quality = governance.UsageQualityMeasured
	}
	return reconcileSettlementAt(reservation, usage, at)
}

func releaseSettlement(claim ActionClaim, at time.Time) (governance.BudgetSettlement, error) {
	usage := governance.ResourceUsage{
		Resources: governance.ResourceVector{Currency: claim.BudgetReservation.Resources.Currency},
		Known:     governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	}
	settlement, err := reconcileSettlementAt(claim.BudgetReservation, usage, at)
	if err != nil {
		return governance.BudgetSettlement{}, err
	}
	return *settlement, nil
}

func settlementForExecutionAttempt(
	record GoalRecord,
	claim ActionClaim,
	execution ExecutionRecord,
	usage governance.ResourceUsage,
	diskBytes int64,
	at time.Time,
	definitelyUnapplied bool,
) (*governance.BudgetSettlement, error) {
	if definitelyUnapplied {
		settlement, err := releaseSettlement(claim, at)
		return &settlement, err
	}
	return settlementFor(record, execution, usage, diskBytes, at)
}

func claimBudgetSettlement(
	claim ActionClaim,
	at time.Time,
	definitelyUnapplied bool,
) (*governance.BudgetSettlement, error) {
	if claim.Action.Kind != ActionLaunchAgent || claim.BudgetReservationRef == "" {
		return nil, nil
	}
	if definitelyUnapplied {
		settlement, err := releaseSettlement(claim, at)
		if err != nil {
			return nil, err
		}
		return &settlement, nil
	}
	return reconcileSettlementAt(claim.BudgetReservation, unknownUsage(), at)
}

func reconcileSettlementAt(reservation governance.BudgetReservation, usage governance.ResourceUsage, at time.Time) (*governance.BudgetSettlement, error) {
	settlement, err := governance.Reconcile(reservation, usage)
	if err != nil {
		return nil, err
	}
	settlement.SettledAt = at.UTC()
	if err := governance.ValidateBudgetSettlement(settlement); err != nil {
		return nil, err
	}
	return &settlement, nil
}

func reservationByRef(records []governance.BudgetReservation, ref string) (governance.BudgetReservation, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return governance.BudgetReservation{}, false
}

func measuredActiveTime(execution ExecutionRecord, at time.Time) int64 {
	start := execution.ProviderAcceptedAt
	if start.IsZero() {
		start = execution.StartedAt
	}
	if start.IsZero() || at.Before(start) {
		return 0
	}
	return at.Sub(start).Nanoseconds()
}
