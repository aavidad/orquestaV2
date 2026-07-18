package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

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
	RequestRef               string
	GoalRef                  goal.GoalRef
	ExpectedGoalRevision     goal.Revision
	ExpectedPlanGeneration   goal.PlanGeneration
	LeaseToken               string
	LeaseFence               uint64
	Cause                    goal.ReplanCause
	SourceWorkItemRef        goal.WorkItemRef
	ExpectedWorkItemRevision goal.Revision
	SourceExecutionRef       goal.ExecutionRef
	SourceExecutionAttempt   uint64
	Reason                   string
	Plan                     PlanSpec
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
	state, err := orchestrator.buildDirectorPlanState(
		ctx, request, fingerprint, principal, projectRef, authorization, now,
	)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	persisted, created, err := orchestrator.state.ApplyDirectorPlan(ctx, state)
	if err != nil {
		return DirectorPlanResult{}, err
	}
	if err := validateDirectorDecision(request, fingerprint, principal.Ref, projectRef, persisted); err != nil {
		return DirectorPlanResult{}, err
	}
	return DirectorPlanResult{Decision: persisted, Created: created}, nil
}

func (orchestrator *Orchestrator) buildDirectorPlanState(ctx context.Context, request ProposeDirectorPlanRequest, fingerprint string, principal identity.Principal, projectRef goal.ProjectRef, authorization identity.AuthorizationReceipt, now time.Time) (ApplyDirectorPlanState, error) {
	current, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return ApplyDirectorPlanState{}, err
	}
	if current.Goal.Project() != projectRef {
		return ApplyDirectorPlanState{}, &StateError{Code: StateNotFound}
	}
	if current.Goal.Revision() != request.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != request.ExpectedPlanGeneration {
		return ApplyDirectorPlanState{}, &StateError{Code: StateConflict}
	}
	if legacyGovernanceRecord(current) {
		return ApplyDirectorPlanState{}, errors.New("governance.legacy_reauthorization_required")
	}
	goalPolicy, policyErr := historicalEffectPolicy(current)
	if policyErr != nil {
		return ApplyDirectorPlanState{}, policyErr
	}
	if goalPolicy.PolicyHash == orchestrator.budgetPolicy.PolicyHash {
		goalPolicy.DefaultDemand = orchestrator.budgetPolicy.DefaultWorkItemDemand
	}
	plan, err := orchestrator.compilePlanExtension(ctx, current.Goal, request.Plan, goalPolicy, now)
	if err != nil {
		return ApplyDirectorPlanState{}, err
	}
	updated, updatedExecutions, retireActionRefs, proposalEvents, err := orchestrator.applyDirectorProposal(current, request, plan, now)
	if err != nil {
		return ApplyDirectorPlanState{}, err
	}
	allItems, oldItems := updated.WorkItems(), current.Goal.WorkItems()
	newAuthorities := workItemAuthorities(
		allItems[len(oldItems):], principal.Ref, identity.PermissionGoalsDirect,
		EffectApprovalSourceDirectorDecision, authorization, now,
	)
	authorities := append(append([]WorkItemAuthority(nil), current.WorkItemAuthorities...), newAuthorities...)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleReady(
		ctx, updated, current.Executions, authorities, goalPolicy, now,
	)
	if err != nil {
		return ApplyDirectorPlanState{}, err
	}
	decisionRef, err := orchestrator.ids.NewID(ctx, "director-decision")
	if err != nil {
		return ApplyDirectorPlanState{}, err
	}
	decision := DirectorDecisionRecord{
		Ref: decisionRef, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		GoalRef: request.GoalRef, PrincipalRef: principal.Ref, LeaseFence: request.LeaseFence,
		SourceGoalRevision:   request.ExpectedGoalRevision,
		SourcePlanGeneration: request.ExpectedPlanGeneration,
		Cause:                request.Cause, SourceWorkItemRef: request.SourceWorkItemRef,
		SourceWorkItemRevision: request.ExpectedWorkItemRevision,
		SourceExecutionRef:     request.SourceExecutionRef, SourceExecutionAttempt: request.SourceExecutionAttempt,
		AppliedGoalRevision: updated.Revision(), AppliedPlanGeneration: updated.PlanGeneration(),
		Reason: request.Reason, DecidedAt: now, AuthorizationReceipt: authorization,
	}
	events := append([]EventRecord{{
		Ref: "event:director-plan-applied:" + decisionRef, Kind: "director.plan_applied",
		GoalRef: request.GoalRef, OccurredAt: now,
	}}, proposalEvents...)
	events = append(events, scheduledEvents...)
	return ApplyDirectorPlanState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef,
		LeaseToken: request.LeaseToken, LeaseFence: request.LeaseFence,
		ExpectedGoalRevision:     request.ExpectedGoalRevision,
		ExpectedPlanGeneration:   request.ExpectedPlanGeneration,
		ExpectedWorkItemRevision: request.ExpectedWorkItemRevision,
		Goal:                     updated, UpdatedExecutions: updatedExecutions,
		NewExecutions: newExecutions, NewActions: newActions, RetireActionRefs: retireActionRefs,
		NewWorkItemAuthorities: newAuthorities,
		Events:                 events, Decision: decision, OperationAt: now,
	}, nil
}

// requireCurrentDirectorAccess prevents revoked principals from replaying results.
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
	if request.Cause == "" {
		if request.SourceWorkItemRef.String() != "" || request.ExpectedWorkItemRevision != 0 ||
			request.SourceExecutionRef.String() != "" || request.SourceExecutionAttempt != 0 {
			return errors.New("application.director_plan_replan_fence_unexpected")
		}
		return nil
	}
	if request.SourceWorkItemRef.String() == "" || request.ExpectedWorkItemRevision == 0 ||
		request.SourceExecutionRef.String() == "" || request.SourceExecutionAttempt == 0 ||
		(request.Cause != goal.ReplanCauseSplitPending && request.Cause != goal.ReplanCauseExecutionStopped &&
			request.Cause != goal.ReplanCauseExecutionFailed) || len(request.Plan.Phases) != 0 {
		return errors.New("application.director_plan_replan_fence_invalid")
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
		decision.Cause != request.Cause || decision.SourceWorkItemRef != request.SourceWorkItemRef ||
		decision.SourceWorkItemRevision != request.ExpectedWorkItemRevision ||
		decision.SourceExecutionRef != request.SourceExecutionRef ||
		decision.SourceExecutionAttempt != request.SourceExecutionAttempt ||
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
	return fingerprintFields(
		"orquesta.director.claim.v1", principal.String(), projectRef.String(), request.GoalRef.String(),
	)
}

func directorRenewFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request RenewDirectorRequest,
) string {
	return fingerprintFields(
		"orquesta.director.renew.v1", principal.String(), projectRef.String(), request.GoalRef.String(),
		request.Token, strconv.FormatUint(request.Fence, 10),
	)
}

func directorPlanFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request ProposeDirectorPlanRequest,
) string {
	digest := fingerprintDigest(
		"orquesta.director.plan.v1", principal.String(), projectRef.String(), request.GoalRef.String(),
		strconv.FormatUint(uint64(request.ExpectedGoalRevision), 10),
		strconv.FormatUint(uint64(request.ExpectedPlanGeneration), 10), request.LeaseToken,
		strconv.FormatUint(request.LeaseFence, 10), string(request.Cause), request.SourceWorkItemRef.String(),
		strconv.FormatUint(uint64(request.ExpectedWorkItemRevision), 10), request.SourceExecutionRef.String(),
		strconv.FormatUint(request.SourceExecutionAttempt, 10), request.Reason,
	)
	writePlanFingerprint(digest, &request.Plan)
	return fingerprintHex(digest)
}

func directorAuthorizationRequestRef(kind DirectorMutationKind, requestRef, fingerprint string) string {
	return "authorization-request:director:" + fingerprintFields(
		"orquesta.director.authorization.v1", string(kind), requestRef, fingerprint,
	)
}
