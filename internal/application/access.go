package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

// ErrForbidden is the stable authorization boundary shared with transports.
// Wrapping it preserves classification without coupling callers to text.
var ErrForbidden = errors.New("application.forbidden")

var errForbidden = ErrForbidden

// Access binds one authenticated principal to one explicit project request.
// It carries no authority by itself; every use case still asks AccessRepository.
type Access struct {
	principal         identity.Principal
	projectRef        goal.ProjectRef
	authenticatedExec goal.ExecutionRef
}

func NewAccess(principal identity.Principal, projectRef goal.ProjectRef) (Access, error) {
	if err := identity.ValidatePrincipal(principal); err != nil {
		return Access{}, errors.New("application.principal_invalid")
	}
	if projectRef.String() == "" {
		return Access{}, errors.New("application.project_ref_required")
	}
	return Access{principal: principal, projectRef: projectRef}, nil
}

// NewExecutionAccess is reserved for trusted transport/runtime boundaries that
// authenticated one exact execution. Request payloads must never supply this
// binding; they may only be checked against it by application use cases.
func NewExecutionAccess(
	principal identity.Principal,
	projectRef goal.ProjectRef,
	executionRef goal.ExecutionRef,
) (Access, error) {
	access, err := NewAccess(principal, projectRef)
	if err != nil {
		return Access{}, err
	}
	if executionRef.String() == "" {
		return Access{}, errors.New("application.execution_ref_required")
	}
	access.authenticatedExec = executionRef
	return access, nil
}

func (access Access) values() (identity.Principal, goal.ProjectRef, error) {
	if err := identity.ValidatePrincipal(access.principal); err != nil || access.projectRef.String() == "" {
		return identity.Principal{}, goal.ProjectRef{}, errors.New("application.access_invalid")
	}
	return access.principal, access.projectRef, nil
}

// authenticatedExecution returns boundary-authenticated identity, never the
// caller-provided expected value. Unbound or successor executions are denied
// identically so mailbox callers cannot probe another execution's address.
func (access Access) authenticatedExecution(expected goal.ExecutionRef) (goal.ExecutionRef, error) {
	if _, _, err := access.values(); err != nil {
		return goal.ExecutionRef{}, err
	}
	if expected.String() == "" || access.authenticatedExec.String() == "" ||
		access.authenticatedExec != expected {
		return goal.ExecutionRef{}, errForbidden
	}
	return access.authenticatedExec, nil
}

type MembershipGrantState struct {
	AuthorizationReceipt identity.AuthorizationReceipt
	Request              identity.MembershipGrantRequest
	Target               identity.Principal
}

type MembershipRevokeState struct {
	AuthorizationReceipt identity.AuthorizationReceipt
	Request              identity.MembershipRevokeRequest
}

// AccessRepository is the only authority/membership port consumed by the
// application. Authorize is causally idempotent by principal and request_ref:
// an equal principal/project/permission/resource scope must return the original
// immutable receipt even when a retry supplies a later requested_at; changing
// that scope is a conflict. Adapters must apply membership CAS and audit
// atomically.
type AccessRepository interface {
	Authorize(context.Context, identity.AuthorizationRequest) (identity.AuthorizationReceipt, error)
	Membership(context.Context, identity.PrincipalRef, goal.ProjectRef) (identity.Membership, error)
	GrantMembership(context.Context, MembershipGrantState) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
	RevokeMembership(context.Context, MembershipRevokeState) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
}

// authorizationCausalFloor prevents durable adapter time from being newer
// than application facts derived from its authorization receipt.
func authorizationCausalFloor(at time.Time, receipt identity.AuthorizationReceipt) time.Time {
	at = at.UTC()
	if recordedAt := receipt.RecordedAt(); at.Before(recordedAt) {
		return recordedAt
	}
	return at
}

func (orchestrator *Orchestrator) authorize(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
	requestedAt time.Time,
) (identity.AuthorizationReceipt, error) {
	requestRef, err := orchestrator.ids.NewID(ctx, "authorization-request")
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	return orchestrator.authorizeWithRequestRef(
		ctx, access, permission, resourceRef, requestedAt, requestRef,
	)
}

func (orchestrator *Orchestrator) authorizeWithRequestRef(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
	requestedAt time.Time,
	requestRef string,
) (identity.AuthorizationReceipt, error) {
	return orchestrator.authorizeRequest(
		ctx, access, permission, resourceRef, requestedAt, requestRef, true,
	)
}

func (orchestrator *Orchestrator) authorizeIdempotentWithRequestRef(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
	requestedAt time.Time,
	requestRef string,
) (identity.AuthorizationReceipt, error) {
	return orchestrator.authorizeRequest(
		ctx, access, permission, resourceRef, requestedAt, requestRef, false,
	)
}

func (orchestrator *Orchestrator) authorizeRequest(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
	requestedAt time.Time,
	requestRef string,
	matchRequestedAt bool,
) (identity.AuthorizationReceipt, error) {
	principal, projectRef, err := access.values()
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: requestRef, Principal: principal, ProjectRef: projectRef,
		Permission: permission, ResourceRef: resourceRef, RequestedAt: requestedAt,
	})
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	receipt, err := orchestrator.access.Authorize(ctx, request)
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	if !authorizationReceiptScopeMatches(receipt, request) ||
		(matchRequestedAt && !receipt.Decision().Request().RequestedAt().Equal(request.RequestedAt())) {
		return identity.AuthorizationReceipt{}, errForbidden
	}
	decision := receipt.Decision()
	if decision.Outcome() == identity.AuthorizationDenied {
		if decision.Role() == "" &&
			(permission == identity.PermissionGoalsGet || permission == identity.PermissionArtifactsRead) {
			return identity.AuthorizationReceipt{}, &StateError{Code: StateNotFound}
		}
		return identity.AuthorizationReceipt{}, errForbidden
	}
	if decision.Outcome() != identity.AuthorizationAllowed || !identity.RoleAllows(decision.Role(), permission) {
		return identity.AuthorizationReceipt{}, errForbidden
	}
	return receipt, nil
}

func authorizationReceiptScopeMatches(
	receipt identity.AuthorizationReceipt,
	want identity.AuthorizationRequest,
) bool {
	got := receipt.Decision().Request()
	return receipt.Ref() != "" && !got.RequestedAt().IsZero() &&
		got.RequestRef() == want.RequestRef() && got.Principal() == want.Principal() &&
		got.ProjectRef() == want.ProjectRef() && got.Permission() == want.Permission() &&
		got.ResourceRef() == want.ResourceRef()
}

func (orchestrator *Orchestrator) authorizeRead(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
) (identity.AuthorizationReceipt, error) {
	return orchestrator.authorize(ctx, access, permission, resourceRef, orchestrator.clock.Now())
}

func authorizationReceiptMatches(
	receipt identity.AuthorizationReceipt,
	want identity.AuthorizationRequest,
) bool {
	return authorizationReceiptScopeMatches(receipt, want) &&
		receipt.Decision().Request().RequestedAt().Equal(want.RequestedAt())
}
