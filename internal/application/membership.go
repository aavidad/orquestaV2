package application

import (
	"context"
	"errors"

	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) GrantMembership(
	ctx context.Context,
	access Access,
	request identity.MembershipGrantRequest,
	target identity.Principal,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	if orchestrator == nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errors.New("application.unavailable")
	}
	principal, projectRef, err := access.values()
	if err != nil || request.RequestRef() == "" || request.Actor() != principal ||
		request.ProjectRef() != projectRef || identity.ValidatePrincipal(target) != nil || request.TargetRef() != target.Ref {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errors.New("application.membership_request_invalid")
	}
	receipt, err := orchestrator.authorize(
		ctx, access, identity.PermissionProjectMembershipManage,
		request.TargetRef().String(), orchestrator.clock.Now(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if !identity.CanDelegateMembershipRole(receipt.Decision().Role(), request.Role()) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errForbidden
	}
	if request.ExpectedRevision() > 0 {
		current, err := orchestrator.access.Membership(ctx, request.TargetRef(), projectRef)
		if err != nil {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
		}
		if current.PrincipalRef() != request.TargetRef() || current.ProjectRef() != projectRef ||
			(current.Revision() != request.ExpectedRevision() &&
				(current.Revision() != request.ExpectedRevision()+1 || !current.IsActive() || current.Role() != request.Role())) {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
		}
		if !identity.CanDelegateMembershipRole(receipt.Decision().Role(), current.Role()) {
			return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errForbidden
		}
	}
	membership, audit, created, err := orchestrator.access.GrantMembership(ctx, MembershipGrantState{
		AuthorizationReceipt: receipt, Request: request, Target: target,
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if !validGrantResult(request, membership, audit) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
	}
	return membership, audit, created, nil
}

func (orchestrator *Orchestrator) RevokeMembership(
	ctx context.Context,
	access Access,
	request identity.MembershipRevokeRequest,
) (identity.Membership, identity.MembershipAuditReceipt, bool, error) {
	if orchestrator == nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errors.New("application.unavailable")
	}
	principal, projectRef, err := access.values()
	if err != nil || request.RequestRef() == "" || request.Actor() != principal || request.ProjectRef() != projectRef {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errors.New("application.membership_request_invalid")
	}
	receipt, err := orchestrator.authorize(
		ctx, access, identity.PermissionProjectMembershipManage,
		request.TargetRef().String(), orchestrator.clock.Now(),
	)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	current, err := orchestrator.access.Membership(ctx, request.TargetRef(), projectRef)
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if current.PrincipalRef() != request.TargetRef() || current.ProjectRef() != projectRef ||
		!((current.Revision() == request.ExpectedRevision() && current.IsActive()) ||
			(current.Revision() == request.ExpectedRevision()+1 && current.Status() == identity.MembershipRevoked)) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
	}
	if !identity.CanDelegateMembershipRole(receipt.Decision().Role(), current.Role()) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, errForbidden
	}
	membership, audit, revoked, err := orchestrator.access.RevokeMembership(ctx, MembershipRevokeState{
		AuthorizationReceipt: receipt, Request: request,
	})
	if err != nil {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, err
	}
	if !validRevokeResult(request, current.Role(), membership, audit) {
		return identity.Membership{}, identity.MembershipAuditReceipt{}, false, &StateError{Code: StateConflict}
	}
	return membership, audit, revoked, nil
}

func validGrantResult(
	request identity.MembershipGrantRequest,
	membership identity.Membership,
	audit identity.MembershipAuditReceipt,
) bool {
	wantRevision := request.ExpectedRevision() + 1
	return membership.PrincipalRef() == request.TargetRef() && membership.ProjectRef() == request.ProjectRef() &&
		membership.Role() == request.Role() && membership.Revision() == wantRevision && membership.IsActive() &&
		membership.GrantedBy() == request.Actor().Ref && membership.GrantedAt().Equal(request.RequestedAt()) &&
		audit.RequestRef() == request.RequestRef() && audit.Action() == identity.MembershipAuditGranted &&
		audit.ActorRef() == request.Actor().Ref && audit.TargetRef() == request.TargetRef() &&
		audit.ProjectRef() == request.ProjectRef() && audit.Role() == request.Role() &&
		audit.PreviousRevision() == request.ExpectedRevision() && audit.Revision() == wantRevision &&
		audit.OccurredAt().Equal(request.RequestedAt())
}

func validRevokeResult(
	request identity.MembershipRevokeRequest,
	role identity.Role,
	membership identity.Membership,
	audit identity.MembershipAuditReceipt,
) bool {
	wantRevision := request.ExpectedRevision() + 1
	return membership.PrincipalRef() == request.TargetRef() && membership.ProjectRef() == request.ProjectRef() &&
		membership.Role() == role && membership.Revision() == wantRevision &&
		membership.Status() == identity.MembershipRevoked && membership.RevokedBy() == request.Actor().Ref &&
		membership.RevokedAt().Equal(request.RequestedAt()) &&
		audit.RequestRef() == request.RequestRef() && audit.Action() == identity.MembershipAuditRevoked &&
		audit.ActorRef() == request.Actor().Ref && audit.TargetRef() == request.TargetRef() &&
		audit.ProjectRef() == request.ProjectRef() && audit.Role() == role &&
		audit.PreviousRevision() == request.ExpectedRevision() && audit.Revision() == wantRevision &&
		audit.OccurredAt().Equal(request.RequestedAt())
}
