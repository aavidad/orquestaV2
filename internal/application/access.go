package application

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

var errForbidden = errors.New("application.forbidden")

// Access binds one authenticated principal to one explicit project request.
// It carries no authority by itself; every use case still asks AccessRepository.
type Access struct {
	principal  identity.Principal
	projectRef goal.ProjectRef
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

func (access Access) values() (identity.Principal, goal.ProjectRef, error) {
	if err := identity.ValidatePrincipal(access.principal); err != nil || access.projectRef.String() == "" {
		return identity.Principal{}, goal.ProjectRef{}, errors.New("application.access_invalid")
	}
	return access.principal, access.projectRef, nil
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
// application. Adapters must apply membership CAS and audit atomically.
type AccessRepository interface {
	Authorize(context.Context, identity.AuthorizationRequest) (identity.AuthorizationReceipt, error)
	Membership(context.Context, identity.PrincipalRef, goal.ProjectRef) (identity.Membership, error)
	GrantMembership(context.Context, MembershipGrantState) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
	RevokeMembership(context.Context, MembershipRevokeState) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
}

func (orchestrator *Orchestrator) authorize(
	ctx context.Context,
	access Access,
	permission identity.Permission,
	resourceRef string,
	requestedAt time.Time,
) (identity.AuthorizationReceipt, error) {
	principal, projectRef, err := access.values()
	if err != nil {
		return identity.AuthorizationReceipt{}, err
	}
	requestRef, err := orchestrator.ids.NewID(ctx, "authorization-request")
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
	if !authorizationReceiptMatches(receipt, request) {
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
	decision := receipt.Decision()
	got := decision.Request()
	return receipt.Ref() != "" &&
		got.RequestRef() == want.RequestRef() && got.Principal() == want.Principal() &&
		got.ProjectRef() == want.ProjectRef() && got.Permission() == want.Permission() &&
		got.ResourceRef() == want.ResourceRef() && got.RequestedAt().Equal(want.RequestedAt())
}
