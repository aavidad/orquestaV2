package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type ClaimDirectorRequest struct {
	RequestRef string
	GoalRef    goal.GoalRef
}

type RenewDirectorRequest struct {
	RequestRef string
	GoalRef    goal.GoalRef
	Token      string
	Fence      uint64
}

type ProposeDirectorPlanRequest struct {
	RequestRef             string
	GoalRef                goal.GoalRef
	ExpectedGoalRevision   goal.Revision
	ExpectedPlanGeneration goal.PlanGeneration
	LeaseToken             string
	LeaseFence             uint64
	Reason                 string
	Plan                   PlanSpec
}

type DirectorLeaseResult struct {
	Lease   DirectorLeaseRecord
	Changed bool
}

type DirectorPlanResult struct {
	Decision DirectorDecisionRecord
	Created  bool
}

func (orchestrator *Orchestrator) ClaimDirector(
	ctx context.Context,
	access Access,
	request ClaimDirectorRequest,
) (DirectorLeaseResult, error) {
	if orchestrator == nil {
		return DirectorLeaseResult{}, errors.New("application.unavailable")
	}
	if err := validateClaimDirectorRequest(request); err != nil {
		return DirectorLeaseResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	fingerprint := directorClaimFingerprint(principal.Ref, projectRef, request)
	if err := orchestrator.requireCurrentDirectorAccess(ctx, principal.Ref, projectRef); err != nil {
		return DirectorLeaseResult{}, err
	}
	replay, found, err := orchestrator.state.DirectorReplay(ctx, DirectorReplayRequest{
		Kind: DirectorMutationClaim, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	if found {
		if err := validateDirectorLeaseResult(replay.Lease, principal.Ref, request.GoalRef, "", 0, false); err != nil {
			return DirectorLeaseResult{}, err
		}
		return DirectorLeaseResult{Lease: replay.Lease}, nil
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(), orchestrator.clock.Now().UTC(),
		directorAuthorizationRequestRef(DirectorMutationClaim, request.RequestRef, fingerprint),
	)
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	token, err := orchestrator.ids.NewID(ctx, "director-lease-token")
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	operationAt := orchestrator.clock.Now().UTC()
	lease, changed, err := orchestrator.state.ClaimDirector(ctx, ClaimDirectorState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, Token: token,
		LeaseDuration: orchestrator.directorLeaseDuration, RequestedAt: operationAt,
	})
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	if err := validateDirectorLeaseResult(lease, principal.Ref, request.GoalRef, token, 0, changed); err != nil {
		return DirectorLeaseResult{}, err
	}
	return DirectorLeaseResult{Lease: lease, Changed: changed}, nil
}

func (orchestrator *Orchestrator) RenewDirector(
	ctx context.Context,
	access Access,
	request RenewDirectorRequest,
) (DirectorLeaseResult, error) {
	if orchestrator == nil {
		return DirectorLeaseResult{}, errors.New("application.unavailable")
	}
	if err := validateRenewDirectorRequest(request); err != nil {
		return DirectorLeaseResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	fingerprint := directorRenewFingerprint(principal.Ref, projectRef, request)
	if err := orchestrator.requireCurrentDirectorAccess(ctx, principal.Ref, projectRef); err != nil {
		return DirectorLeaseResult{}, err
	}
	replay, found, err := orchestrator.state.DirectorReplay(ctx, DirectorReplayRequest{
		Kind: DirectorMutationRenew, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	if found {
		if err := validateDirectorLeaseResult(
			replay.Lease, principal.Ref, request.GoalRef, request.Token, request.Fence, true,
		); err != nil {
			return DirectorLeaseResult{}, err
		}
		return DirectorLeaseResult{Lease: replay.Lease}, nil
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(), orchestrator.clock.Now().UTC(),
		directorAuthorizationRequestRef(DirectorMutationRenew, request.RequestRef, fingerprint),
	)
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	operationAt := orchestrator.clock.Now().UTC()
	lease, changed, err := orchestrator.state.RenewDirector(ctx, RenewDirectorState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, Token: request.Token,
		Fence: request.Fence, LeaseDuration: orchestrator.directorLeaseDuration, RequestedAt: operationAt,
	})
	if err != nil {
		return DirectorLeaseResult{}, err
	}
	if err := validateDirectorLeaseResult(
		lease, principal.Ref, request.GoalRef, request.Token, request.Fence, true,
	); err != nil {
		return DirectorLeaseResult{}, err
	}
	return DirectorLeaseResult{Lease: lease, Changed: changed}, nil
}

func (orchestrator *Orchestrator) ProposeDirectorPlan(
	ctx context.Context,
	access Access,
	request ProposeDirectorPlanRequest,
) (DirectorPlanResult, error) {
	if orchestrator == nil {
		return DirectorPlanResult{}, errors.New("application.unavailable")
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if err := validateProposeDirectorPlanRequest(request); err != nil {
		return DirectorPlanResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return DirectorPlanResult{}, err
	}
	fingerprint := directorPlanFingerprint(principal.Ref, projectRef, request)
	if err := orchestrator.requireCurrentDirectorAccess(ctx, principal.Ref, projectRef); err != nil {
		return DirectorPlanResult{}, err
	}
	replay, found, err := orchestrator.state.DirectorReplay(ctx, DirectorReplayRequest{
		Kind: DirectorMutationPlan, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil {
		return DirectorPlanResult{}, err
	}
	if found {
		if err := validateDirectorDecision(
			request, fingerprint, principal.Ref, projectRef, replay.Decision,
		); err != nil {
			return DirectorPlanResult{}, err
		}
		return DirectorPlanResult{Decision: replay.Decision}, nil
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(), orchestrator.clock.Now().UTC(),
		directorAuthorizationRequestRef(DirectorMutationPlan, request.RequestRef, fingerprint),
	)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	now := orchestrator.clock.Now().UTC()

	current, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	if current.Goal.Project() != projectRef {
		return DirectorPlanResult{}, &StateError{Code: StateNotFound}
	}
	if current.Goal.Revision() != request.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != request.ExpectedPlanGeneration {
		return DirectorPlanResult{}, &StateError{Code: StateConflict}
	}
	plan, err := orchestrator.compilePlanExtension(ctx, current.Goal, request.Plan, now)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	updated, err := current.Goal.ApplyPlan(request.ExpectedGoalRevision, plan)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleReady(
		ctx, updated, current.Executions, now,
	)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	decisionRef, err := orchestrator.ids.NewID(ctx, "director-decision")
	if err != nil {
		return DirectorPlanResult{}, err
	}
	decision := DirectorDecisionRecord{
		Ref: decisionRef, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		GoalRef: request.GoalRef, PrincipalRef: principal.Ref, LeaseFence: request.LeaseFence,
		SourceGoalRevision:   request.ExpectedGoalRevision,
		SourcePlanGeneration: request.ExpectedPlanGeneration,
		AppliedGoalRevision:  updated.Revision(), AppliedPlanGeneration: updated.PlanGeneration(),
		Reason: request.Reason, DecidedAt: now, AuthorizationReceipt: authorization,
	}
	events := append([]EventRecord{{
		Ref: "event:director-plan-applied:" + decisionRef, Kind: "director.plan_applied",
		GoalRef: request.GoalRef, OccurredAt: now,
	}}, scheduledEvents...)
	persistedDecision, created, err := orchestrator.state.ApplyDirectorPlan(ctx, ApplyDirectorPlanState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef,
		LeaseToken: request.LeaseToken, LeaseFence: request.LeaseFence,
		ExpectedGoalRevision:   request.ExpectedGoalRevision,
		ExpectedPlanGeneration: request.ExpectedPlanGeneration,
		Goal:                   updated, NewExecutions: newExecutions, NewActions: newActions,
		Events: events, Decision: decision, OperationAt: now,
	})
	if err != nil {
		return DirectorPlanResult{}, err
	}
	if err := validateDirectorDecision(
		request, fingerprint, principal.Ref, current.Goal.Project(), persistedDecision,
	); err != nil {
		return DirectorPlanResult{}, err
	}
	return DirectorPlanResult{Decision: persistedDecision, Created: created}, nil
}

// requireCurrentDirectorAccess authorizes the read-only replay lookup against
// the live membership projection. It creates no second authorization receipt,
// but a revoked or demoted principal can never recover a lease token or result.
func (orchestrator *Orchestrator) requireCurrentDirectorAccess(
	ctx context.Context,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
) error {
	membership, err := orchestrator.access.Membership(ctx, principalRef, projectRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			return errForbidden
		}
		return err
	}
	if membership.PrincipalRef() != principalRef || membership.ProjectRef() != projectRef ||
		!membership.IsActive() || !identity.RoleAllows(membership.Role(), identity.PermissionGoalsDirect) {
		return errForbidden
	}
	return nil
}

func validateClaimDirectorRequest(request ClaimDirectorRequest) error {
	if !validApplicationRef(request.RequestRef) {
		return errors.New("application.request_ref_invalid")
	}
	if request.GoalRef.String() == "" {
		return errors.New("application.goal_ref_required")
	}
	return nil
}

func validateRenewDirectorRequest(request RenewDirectorRequest) error {
	if err := validateClaimDirectorRequest(ClaimDirectorRequest{
		RequestRef: request.RequestRef, GoalRef: request.GoalRef,
	}); err != nil {
		return err
	}
	if !validApplicationRef(request.Token) {
		return errors.New("application.director_lease_token_invalid")
	}
	if request.Fence == 0 {
		return errors.New("application.director_lease_fence_invalid")
	}
	return nil
}

func validateProposeDirectorPlanRequest(request ProposeDirectorPlanRequest) error {
	if err := validateRenewDirectorRequest(RenewDirectorRequest{
		RequestRef: request.RequestRef, GoalRef: request.GoalRef,
		Token: request.LeaseToken, Fence: request.LeaseFence,
	}); err != nil {
		return err
	}
	if request.ExpectedGoalRevision == 0 || request.ExpectedPlanGeneration == 0 {
		return errors.New("application.director_plan_revision_required")
	}
	if request.Reason == "" {
		return errors.New("application.director_plan_reason_required")
	}
	if len(request.Plan.WorkItems) == 0 {
		return errors.New("application.director_plan_work_items_required")
	}
	return nil
}

func validApplicationRef(value string) bool {
	return strings.TrimSpace(value) == value && value != ""
}

func validateDirectorLeaseResult(
	lease DirectorLeaseRecord,
	principal identity.PrincipalRef,
	goalRef goal.GoalRef,
	token string,
	fence uint64,
	bindToken bool,
) error {
	if lease.GoalRef != goalRef || lease.PrincipalRef != principal ||
		!validApplicationRef(lease.Token) || lease.Fence == 0 || lease.LeaseUntil.IsZero() {
		return &StateError{Code: StateConflict}
	}
	if bindToken && (lease.Token != token || (fence != 0 && lease.Fence != fence)) {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateDirectorDecision(
	request ProposeDirectorPlanRequest,
	fingerprint string,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	decision DirectorDecisionRecord,
) error {
	if decision.Ref == "" || decision.RequestRef != request.RequestRef ||
		decision.RequestFingerprint != fingerprint || decision.GoalRef != request.GoalRef ||
		decision.PrincipalRef != principal || decision.LeaseFence != request.LeaseFence ||
		decision.SourceGoalRevision != request.ExpectedGoalRevision ||
		decision.SourcePlanGeneration != request.ExpectedPlanGeneration ||
		decision.AppliedGoalRevision != request.ExpectedGoalRevision+1 ||
		decision.AppliedPlanGeneration != request.ExpectedPlanGeneration+1 ||
		decision.Reason != request.Reason || decision.DecidedAt.IsZero() ||
		!directorAuthorizationValid(decision.AuthorizationReceipt, principal, projectRef, request.GoalRef) {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func directorAuthorizationValid(
	receipt identity.AuthorizationReceipt,
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
) bool {
	request := receipt.Decision().Request()
	return receipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(receipt.Decision().Role(), identity.PermissionGoalsDirect) &&
		request.Principal().Ref == principal && request.ProjectRef() == projectRef &&
		request.Permission() == identity.PermissionGoalsDirect && request.ResourceRef() == goalRef.String()
}

func directorClaimFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request ClaimDirectorRequest,
) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.director.claim.v1")
	writeFingerprintField(digest, principal.String())
	writeFingerprintField(digest, projectRef.String())
	writeFingerprintField(digest, request.GoalRef.String())
	return hex.EncodeToString(digest.Sum(nil))
}

func directorRenewFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request RenewDirectorRequest,
) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.director.renew.v1")
	writeFingerprintField(digest, principal.String())
	writeFingerprintField(digest, projectRef.String())
	writeFingerprintField(digest, request.GoalRef.String())
	writeFingerprintField(digest, request.Token)
	writeFingerprintField(digest, strconv.FormatUint(request.Fence, 10))
	return hex.EncodeToString(digest.Sum(nil))
}

func directorPlanFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request ProposeDirectorPlanRequest,
) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.director.plan.v1")
	writeFingerprintField(digest, principal.String())
	writeFingerprintField(digest, projectRef.String())
	writeFingerprintField(digest, request.GoalRef.String())
	writeFingerprintField(digest, strconv.FormatUint(uint64(request.ExpectedGoalRevision), 10))
	writeFingerprintField(digest, strconv.FormatUint(uint64(request.ExpectedPlanGeneration), 10))
	writeFingerprintField(digest, request.LeaseToken)
	writeFingerprintField(digest, strconv.FormatUint(request.LeaseFence, 10))
	writeFingerprintField(digest, request.Reason)
	writePlanFingerprint(digest, &request.Plan)
	return hex.EncodeToString(digest.Sum(nil))
}

func directorAuthorizationRequestRef(kind DirectorMutationKind, requestRef, fingerprint string) string {
	digest := sha256.New()
	writeFingerprintField(digest, "orquesta.director.authorization.v1")
	writeFingerprintField(digest, string(kind))
	writeFingerprintField(digest, requestRef)
	writeFingerprintField(digest, fingerprint)
	return "authorization-request:director:" + hex.EncodeToString(digest.Sum(nil))
}
