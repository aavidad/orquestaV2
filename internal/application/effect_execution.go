package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) actionCallContext(parent context.Context, claim ActionClaim) (context.Context, context.CancelFunc) {
	remaining := claim.LeaseUntil.Sub(orchestrator.clock.Now().UTC())
	if remaining <= 0 {
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, func() {}
	}
	return context.WithTimeout(parent, remaining)
}

const effectUnknownAppliedCode = "application.effect_unknown_applied"

type effectUnknownAppliedError struct {
	causeCode string
}

func (err *effectUnknownAppliedError) Error() string {
	return effectUnknownAppliedCode
}

func (err *effectUnknownAppliedError) CauseCode() string {
	if err == nil {
		return ""
	}
	return err.causeCode
}

// priorEffectAttemptBlocksDispatch reports whether a causal physical attempt
// prevents repeating the same action and intent. A terminal receipt confirms
// the effect, while only one exact causal zero-release proves it was unapplied.
// A later claim fence is recovery authority, never replay authority.
func priorEffectAttemptBlocksDispatch(record GoalRecord, action ActionRecord) bool {
	for _, attempt := range record.EffectAttempts {
		if attempt.ActionRef != action.Ref || attempt.IntentRef != action.EffectIntentRef {
			continue
		}
		if effectAttemptHasReceipt(record.EffectReceipts, attempt) {
			return true
		}
		if !effectAttemptDefinitelyUnapplied(record, attempt) {
			return true
		}
	}
	return false
}

// AgentLaunchHasBlockingEffectAttempt is the canonical admission boundary for
// a launch that already has physical history. Only attempts proven unapplied
// by one exact causal zero-release stop blocking normal admission. Malformed
// historical authority remains blocking so callers cannot turn corruption
// into permission to repeat an external effect.
func AgentLaunchHasBlockingEffectAttempt(record GoalRecord, action ActionRecord) bool {
	if action.Kind != ActionLaunchAgent || action.EffectIntentRef == "" {
		return false
	}
	blocking, err := blockingAgentLaunchAttempts(record, action)
	return err != nil || len(blocking) != 0
}

func effectAttemptHasReceipt(receipts []EffectReceipt, attempt EffectAttempt) bool {
	for _, receipt := range receipts {
		if receipt.AttemptRef == attempt.Ref && receipt.ActionRef == attempt.ActionRef &&
			receipt.IntentRef == attempt.IntentRef && receipt.ActionFence == attempt.ActionFence {
			return true
		}
	}
	return false
}

func effectAttemptDefinitelyUnapplied(record GoalRecord, attempt EffectAttempt) bool {
	matches := 0
	for _, settlement := range record.BudgetSettlements {
		if settlement.CausalAttemptRef != attempt.Ref {
			continue
		}
		reservation, found := reservationByRef(record.BudgetReservations, settlement.ReservationRef)
		if !found || reservation.ActionRef != attempt.ActionRef || reservation.EffectIntentRef != attempt.IntentRef ||
			reservation.Fence > attempt.ActionFence || settlement.Reserved != reservation.Resources ||
			!governance.IsExactZeroRelease(settlement) {
			return false
		}
		matches++
	}
	return matches == 1
}

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
) (EffectAttempt, bool, error) {
	if err := validateClaimedEffect(claim, at); err != nil {
		return EffectAttempt{}, false, err
	}
	intent, approval := claim.Action.EffectIntent, claim.EffectApproval
	attempt := EffectAttempt{
		Ref:       "effect-attempt:" + claim.Action.Ref + ":" + claim.Token,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, ApprovalRef: approval.Ref,
		Subject: intent.Subject, ActionRef: claim.Action.Ref, ActionFence: claim.Fence,
		WorkerRef: claim.WorkerRef, IdempotencyKey: intent.IdempotencyKey, StartedAt: at.UTC(),
		ClaimLeaseUntil: claim.LeaseUntil.UTC(),
	}
	persisted, created, err := orchestrator.state.RecordEffectAttempt(ctx, RecordEffectAttemptState{
		Claim: claim, Attempt: attempt, OperationAt: at.UTC(),
	})
	if err != nil {
		return EffectAttempt{}, false, err
	}
	if err := validateEffectAttempt(claim, persisted); err != nil {
		return EffectAttempt{}, false, err
	}
	return persisted, created, nil
}

// beginNewEffectAttempt is the sole gate to a physical effect. An ambiguous,
// malformed, or pre-existing persistence result cannot authorize an adapter
// call and is quarantined as potentially applied.
func (orchestrator *Orchestrator) beginNewEffectAttempt(
	ctx context.Context,
	claim ActionClaim,
	at time.Time,
) (EffectAttempt, error) {
	attempt, created, err := orchestrator.beginEffectAttempt(ctx, claim, at)
	if err != nil || !created {
		return EffectAttempt{}, orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	return attempt, nil
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
	wantRef := "effect-attempt:" + claim.Action.Ref + ":" + claim.Token
	if attempt.Ref != wantRef || !validApplicationRef(attempt.Ref) ||
		attempt.IntentRef != intent.Ref || attempt.IntentDigest != intent.Digest ||
		attempt.ApprovalRef != claim.EffectApproval.Ref || attempt.Subject != intent.Subject ||
		attempt.ActionRef != claim.Action.Ref || attempt.ActionFence != claim.Fence ||
		attempt.WorkerRef != claim.WorkerRef || attempt.IdempotencyKey != intent.IdempotencyKey ||
		attempt.StartedAt.IsZero() || attempt.ClaimLeaseUntil.IsZero() ||
		!attempt.ClaimLeaseUntil.After(attempt.StartedAt) ||
		!attempt.ClaimLeaseUntil.Equal(claim.LeaseUntil.UTC()) {
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
		ActionRef: claim.Action.Ref, ActionFence: attempt.ActionFence, IdempotencyKey: intent.IdempotencyKey,
		ExternalRef: externalRef, Status: status, Usage: usage, ConfirmedAt: at.UTC(),
	}
	if err := validateEffectReceipt(claim, attempt, receipt); err != nil {
		return EffectReceipt{}, err
	}
	return receipt, nil
}

func validateEffectReceipt(claim ActionClaim, attempt EffectAttempt, receipt EffectReceipt) error {
	intent := claim.Action.EffectIntent
	confirmationClaimValid := claim.Fence == attempt.ActionFence
	recoveredStop := false
	if claim.Disposition == ActionClaimDispositionRecoverEffect {
		recoveredLaunch := claim.Action.Kind == ActionLaunchAgent && intent.Kind == EffectKindAgentLaunch
		recoveredStop = claim.Action.Kind == ActionStopAgent && intent.Kind == EffectKindAgentStop
		confirmationClaimValid = (recoveredLaunch || recoveredStop) &&
			claim.Fence > attempt.ActionFence && claim.RecoveryEffectAttemptRef == attempt.Ref
	}
	if attempt.ActionFence == 0 || !validApplicationRef(receipt.Ref) || !validApplicationRef(receipt.ExternalRef) ||
		!validEffectStatus(intent.Kind, receipt.Status) || receipt.IntentRef != intent.Ref ||
		receipt.IntentDigest != intent.Digest || receipt.ApprovalRef != claim.EffectApproval.Ref ||
		receipt.AttemptRef != attempt.Ref || receipt.Subject != intent.Subject ||
		receipt.ActionRef != claim.Action.Ref || receipt.ActionFence != attempt.ActionFence ||
		!confirmationClaimValid ||
		receipt.IdempotencyKey != intent.IdempotencyKey || receipt.ConfirmedAt.Before(attempt.StartedAt) ||
		(effectReceiptRequiresLiveLease(intent.Kind) && !recoveredStop &&
			!receipt.ConfirmedAt.Before(attempt.ClaimLeaseUntil)) ||
		(recoveredStop && !receipt.ConfirmedAt.Before(claim.LeaseUntil)) ||
		governance.ValidateResourceUsage(receipt.Usage) != nil {
		return errors.New("application.effect_receipt_invalid")
	}
	return nil
}

// Lifecycle calls may remain physically in flight after the claim lease that
// fenced their EffectAttempt expires. Their terminal receipt resolves that
// same attempt; it never authorizes a replacement physical call.
func effectReceiptRequiresLiveLease(kind EffectKind) bool {
	switch kind {
	case EffectKindAgentQuiesce, EffectKindAgentPreserve, EffectKindAgentClose:
		return false
	default:
		return true
	}
}

func validEffectStatus(kind EffectKind, status EffectStatus) bool {
	switch kind {
	case EffectKindAgentLaunch:
		return status == EffectStatusAccepted
	case EffectKindAgentQuiesce:
		return status == EffectStatusQuiesced
	case EffectKindAgentPreserve:
		return status == EffectStatusPreserved
	case EffectKindAgentClose:
		return status == EffectStatusClosed
	case EffectKindAgentStop:
		return status == EffectStatusStopped || status == EffectStatusAlreadyStopped ||
			status == EffectStatusAlreadyCompleted || status == EffectStatusAlreadyFailed
	case EffectKindPrepareWorkspace:
		return status == EffectStatusPrepared
	case EffectKindCommitChange:
		return status == EffectStatusCommitted
	case EffectKindAttestTest:
		return status == EffectStatusAttestedPassed || status == EffectStatusAttestedFailed
	case EffectKindIntegrateChange:
		return status == EffectStatusIntegrated || status == EffectStatusConflicted || status == EffectStatusStale
	default:
		return false
	}
}

func actionUsesEffectLedger(kind ActionKind) bool {
	switch kind {
	case ActionLaunchAgent, ActionQuiesceAgent, ActionPreserveAgentEnvironment, ActionCloseAgentEnvironment,
		ActionStopAgent, ActionPrepareWorkspace, ActionCommitChange, ActionAttestTest, ActionIntegrateChange:
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

func releaseSettlementForAttempt(
	claim ActionClaim,
	attemptRef string,
	at time.Time,
) (governance.BudgetSettlement, error) {
	settlement, err := releaseSettlement(claim, at)
	if err != nil {
		return governance.BudgetSettlement{}, err
	}
	settlement.CausalAttemptRef = attemptRef
	if err := governance.ValidateBudgetSettlement(settlement); err != nil {
		return governance.BudgetSettlement{}, err
	}
	return settlement, nil
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
	if definitelyUnapplied && claim.BudgetReservationRef != "" {
		settlement, err := releaseSettlementForAttempt(
			claim, "effect-attempt:"+claim.Action.Ref+":"+claim.Token, at,
		)
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
