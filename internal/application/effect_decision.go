package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

type DecideEffectRequest struct {
	RequestRef           string
	GoalRef              goal.GoalRef
	IntentRef            string
	ExpectedIntentDigest string
	Decision             EffectDecision
	Reason               string
}

type DecideEffectResult struct {
	Approval EffectApproval
	Created  bool
}

func (orchestrator *Orchestrator) DecideEffect(
	ctx context.Context,
	access Access,
	request DecideEffectRequest,
) (DecideEffectResult, error) {
	if orchestrator == nil {
		return DecideEffectResult{}, errors.New("application.unavailable")
	}
	if orchestrator.effectApprovalTTL <= 0 {
		return DecideEffectResult{}, errors.New("application.effect_approval_ttl_invalid")
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if err := validateDecideEffectRequest(request); err != nil {
		return DecideEffectResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return DecideEffectResult{}, err
	}
	fingerprint := decideEffectFingerprint(principal.Ref, projectRef, request)
	replayRequest := EffectReplayRequest{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: request.GoalRef,
		IntentRef: request.IntentRef, IntentDigest: request.ExpectedIntentDigest,
	}
	if replayed, found, replayErr := orchestrator.state.EffectReplay(ctx, replayRequest); replayErr != nil {
		return DecideEffectResult{}, replayErr
	} else if found {
		if err := validateEffectDecisionResult(request, fingerprint, principal.Ref, projectRef, replayed); err != nil {
			return DecideEffectResult{}, err
		}
		return DecideEffectResult{Approval: replayed}, nil
	}

	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return DecideEffectResult{}, err
	}
	intent, found := effectIntentByRef(record.EffectIntents, request.IntentRef)
	if !found {
		return DecideEffectResult{}, &StateError{Code: StateNotFound}
	}
	if err := ValidateEffectIntent(intent); err != nil {
		return DecideEffectResult{}, &StateError{Code: StateInvalid, Cause: err}
	}
	if record.Goal.Project() != projectRef || intent.Subject.ProjectRef != projectRef ||
		intent.Subject.GoalRef != request.GoalRef || intent.Digest != request.ExpectedIntentDigest {
		return DecideEffectResult{}, &StateError{Code: StateConflict}
	}
	if request.Decision == EffectApproved && intent.SecurityCriticality == governance.SecurityCriticalityCritical &&
		intent.ProposedBy == principal.Ref {
		return DecideEffectResult{}, errors.New("application.effect_critical_separation_required")
	}

	now := orchestrator.clock.Now().UTC()
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionEffectsApprove,
		effectApprovalResourceRef(request.GoalRef, request.IntentRef, request.ExpectedIntentDigest), now,
		effectApprovalAuthorizationRequestRef(request.RequestRef, fingerprint),
	)
	if err != nil {
		return DecideEffectResult{}, err
	}
	ref, err := orchestrator.ids.NewID(ctx, "effect-approval")
	if err != nil {
		return DecideEffectResult{}, err
	}
	approval := EffectApproval{
		Ref: ref, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, Subject: intent.Subject,
		ProposedBy: intent.ProposedBy, DecidedBy: principal.Ref, Decision: request.Decision,
		Source: EffectApprovalSourceExplicitDecision, SecurityCriticality: intent.SecurityCriticality,
		Reason: request.Reason, IdempotencyKey: intent.IdempotencyKey,
		AuthorizationReceipt: authorization, DecidedAt: now,
	}
	if request.Decision == EffectApproved {
		approval.ExpiresAt = now.Add(orchestrator.effectApprovalTTL)
	}
	persisted, created, err := orchestrator.state.DecideEffect(ctx, DecideEffectState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, IntentRef: intent.Ref,
		IntentDigest: intent.Digest, Approval: approval, OperationAt: now,
	})
	if err != nil {
		return orchestrator.replayEffectDecisionAfterConflict(ctx, request, replayRequest, err)
	}
	if err := validateEffectDecisionResult(request, fingerprint, principal.Ref, projectRef, persisted); err != nil {
		return DecideEffectResult{}, err
	}
	return DecideEffectResult{Approval: persisted, Created: created}, nil
}

func (orchestrator *Orchestrator) replayEffectDecisionAfterConflict(
	ctx context.Context,
	request DecideEffectRequest,
	replayRequest EffectReplayRequest,
	conflict error,
) (DecideEffectResult, error) {
	if !IsStateError(conflict, StateConflict) {
		return DecideEffectResult{}, conflict
	}
	replayed, found, err := orchestrator.state.EffectReplay(ctx, replayRequest)
	if err != nil || !found {
		return DecideEffectResult{}, conflict
	}
	if err := validateEffectDecisionResult(
		request, replayRequest.RequestFingerprint, replayRequest.PrincipalRef, replayRequest.ProjectRef, replayed,
	); err != nil {
		return DecideEffectResult{}, conflict
	}
	return DecideEffectResult{Approval: replayed}, nil
}

func validateDecideEffectRequest(request DecideEffectRequest) error {
	switch {
	case !validApplicationRef(request.RequestRef):
		return errors.New("application.request_ref_invalid")
	case request.GoalRef.String() == "" || !validApplicationRef(request.IntentRef):
		return errors.New("application.effect_ref_invalid")
	case !validEffectDigest(request.ExpectedIntentDigest):
		return errors.New("application.effect_digest_invalid")
	case request.Decision != EffectApproved && request.Decision != EffectDenied:
		return errors.New("application.effect_decision_invalid")
	case request.Reason == "":
		return errors.New("application.effect_reason_required")
	default:
		return nil
	}
}

func validateEffectDecisionResult(
	request DecideEffectRequest,
	fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	approval EffectApproval,
) error {
	if !validApplicationRef(approval.Ref) || approval.RequestRef != request.RequestRef ||
		approval.RequestFingerprint != fingerprint || approval.IntentRef != request.IntentRef ||
		approval.IntentDigest != request.ExpectedIntentDigest || approval.Subject.ProjectRef != projectRef ||
		approval.Subject.GoalRef != request.GoalRef || approval.DecidedBy != principal ||
		approval.Decision != request.Decision || approval.Reason != request.Reason ||
		approval.Source != EffectApprovalSourceExplicitDecision ||
		governance.ValidateSecurityCriticality(approval.SecurityCriticality) != nil ||
		!validApplicationRef(approval.IdempotencyKey) || approval.DecidedAt.IsZero() ||
		!effectApprovalAuthorizationValid(approval) {
		return &StateError{Code: StateConflict}
	}
	if approval.Decision == EffectApproved {
		if approval.SecurityCriticality == governance.SecurityCriticalityCritical && approval.DecidedBy == approval.ProposedBy {
			return &StateError{Code: StateConflict}
		}
		if !approval.ExpiresAt.After(approval.DecidedAt) {
			return &StateError{Code: StateConflict}
		}
	} else if !approval.ExpiresAt.IsZero() {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func ValidatePersistedEffectApproval(approval EffectApproval) error {
	request := DecideEffectRequest{
		RequestRef: approval.RequestRef, GoalRef: approval.Subject.GoalRef,
		IntentRef: approval.IntentRef, ExpectedIntentDigest: approval.IntentDigest,
		Decision: approval.Decision, Reason: approval.Reason,
	}
	if err := validateDecideEffectRequest(request); err != nil ||
		validateEffectDecisionResult(
			request, approval.RequestFingerprint, approval.DecidedBy, approval.Subject.ProjectRef, approval,
		) != nil {
		return errors.New("application.effect_approval_invalid")
	}
	wantFingerprint := decideEffectFingerprint(approval.DecidedBy, approval.Subject.ProjectRef, request)
	if approval.RequestFingerprint != wantFingerprint ||
		approval.AuthorizationReceipt.Decision().Request().RequestRef() !=
			effectApprovalAuthorizationRequestRef(approval.RequestRef, wantFingerprint) ||
		approval.AuthorizationReceipt.RecordedAt().After(approval.DecidedAt) {
		return errors.New("application.effect_approval_invalid")
	}
	return nil
}

func effectApprovalAuthorizationValid(approval EffectApproval) bool {
	request := approval.AuthorizationReceipt.Decision().Request()
	return approval.AuthorizationReceipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(approval.AuthorizationReceipt.Decision().Role(), identity.PermissionEffectsApprove) &&
		request.Principal().Ref == approval.DecidedBy && request.ProjectRef() == approval.Subject.ProjectRef &&
		request.Permission() == identity.PermissionEffectsApprove &&
		request.ResourceRef() == effectApprovalResourceRef(
			approval.Subject.GoalRef, approval.IntentRef, approval.IntentDigest,
		) && !approval.AuthorizationReceipt.RecordedAt().After(approval.DecidedAt)
}

func decideEffectFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request DecideEffectRequest,
) string {
	digest := sha256.New()
	for _, field := range []string{
		"orquesta.effect.decision.v1", principal.String(), projectRef.String(), request.GoalRef.String(),
		request.IntentRef, request.ExpectedIntentDigest, string(request.Decision), strings.TrimSpace(request.Reason),
	} {
		writeFingerprintField(digest, field)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func effectApprovalAuthorizationRequestRef(requestRef, fingerprint string) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.effect.approval.authorization.v1")
	writeFingerprintField(digest, requestRef)
	writeFingerprintField(digest, fingerprint)
	return "authorization-request:effect-approval:" + hex.EncodeToString(digest.Sum(nil))
}

func effectApprovalResourceRef(goalRef goal.GoalRef, intentRef, digest string) string {
	return "effect-approval:" + goalRef.String() + ":" + intentRef + ":" + digest
}

func effectIntentByRef(intents []EffectIntent, ref string) (EffectIntent, bool) {
	for _, intent := range intents {
		if intent.Ref == ref {
			return intent, true
		}
	}
	return EffectIntent{}, false
}
