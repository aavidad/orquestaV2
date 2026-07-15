package identity

import (
	"errors"
	"time"

	"orquesta/internal/goal"
)

type MembershipRevision uint64
type MembershipStatus string

const (
	MembershipActive  MembershipStatus = "active"
	MembershipRevoked MembershipStatus = "revoked"
)

type MembershipInput struct {
	PrincipalRef PrincipalRef
	ProjectRef   goal.ProjectRef
	Role         Role
	Revision     MembershipRevision
	Status       MembershipStatus
	GrantedBy    PrincipalRef
	GrantedAt    time.Time
	RevokedBy    PrincipalRef
	RevokedAt    time.Time
}

type Membership struct {
	principalRef PrincipalRef
	projectRef   goal.ProjectRef
	role         Role
	revision     MembershipRevision
	status       MembershipStatus
	grantedBy    PrincipalRef
	grantedAt    time.Time
	revokedBy    PrincipalRef
	revokedAt    time.Time
}

func NewMembership(input MembershipInput) (Membership, error) {
	if input.PrincipalRef.String() == "" || input.ProjectRef.String() == "" ||
		input.GrantedBy.String() == "" || input.GrantedAt.IsZero() || input.Revision == 0 {
		return Membership{}, errors.New("identity.membership_required_field_missing")
	}
	if err := ValidateRole(input.Role); err != nil {
		return Membership{}, err
	}
	switch input.Status {
	case MembershipActive:
		if input.RevokedBy.String() != "" || !input.RevokedAt.IsZero() {
			return Membership{}, errors.New("identity.active_membership_has_revocation")
		}
	case MembershipRevoked:
		if input.Revision < 2 || input.RevokedBy.String() == "" || input.RevokedAt.IsZero() ||
			input.RevokedAt.Before(input.GrantedAt) {
			return Membership{}, errors.New("identity.revoked_membership_invalid")
		}
	default:
		return Membership{}, errors.New("identity.invalid_membership_status")
	}
	return Membership{
		principalRef: input.PrincipalRef, projectRef: input.ProjectRef, role: input.Role,
		revision: input.Revision, status: input.Status, grantedBy: input.GrantedBy,
		grantedAt: input.GrantedAt.UTC(), revokedBy: input.RevokedBy, revokedAt: input.RevokedAt.UTC(),
	}, nil
}

func (membership Membership) PrincipalRef() PrincipalRef  { return membership.principalRef }
func (membership Membership) ProjectRef() goal.ProjectRef { return membership.projectRef }
func (membership Membership) Role() Role                  { return membership.role }
func (membership Membership) Revision() MembershipRevision {
	return membership.revision
}
func (membership Membership) Status() MembershipStatus { return membership.status }
func (membership Membership) GrantedBy() PrincipalRef  { return membership.grantedBy }
func (membership Membership) GrantedAt() time.Time     { return membership.grantedAt }
func (membership Membership) RevokedBy() PrincipalRef  { return membership.revokedBy }
func (membership Membership) RevokedAt() time.Time     { return membership.revokedAt }
func (membership Membership) IsActive() bool           { return membership.status == MembershipActive }

func (membership Membership) Snapshot() MembershipInput {
	return MembershipInput{
		PrincipalRef: membership.principalRef, ProjectRef: membership.projectRef,
		Role: membership.role, Revision: membership.revision, Status: membership.status,
		GrantedBy: membership.grantedBy, GrantedAt: membership.grantedAt,
		RevokedBy: membership.revokedBy, RevokedAt: membership.revokedAt,
	}
}

type MembershipGrantRequestInput struct {
	RequestRef       string
	Actor            Principal
	TargetRef        PrincipalRef
	ProjectRef       goal.ProjectRef
	Role             Role
	ExpectedRevision MembershipRevision
	RequestedAt      time.Time
}

type MembershipGrantRequest struct{ input MembershipGrantRequestInput }

func NewMembershipGrantRequest(input MembershipGrantRequestInput) (MembershipGrantRequest, error) {
	if !validCode(input.RequestRef) || ValidatePrincipal(input.Actor) != nil ||
		input.TargetRef.String() == "" || input.ProjectRef.String() == "" || input.RequestedAt.IsZero() ||
		input.ExpectedRevision == maxMembershipRevision() {
		return MembershipGrantRequest{}, errors.New("identity.membership_grant_invalid")
	}
	if err := ValidateRole(input.Role); err != nil {
		return MembershipGrantRequest{}, err
	}
	input.RequestedAt = input.RequestedAt.UTC()
	return MembershipGrantRequest{input: input}, nil
}

func (request MembershipGrantRequest) RequestRef() string { return request.input.RequestRef }
func (request MembershipGrantRequest) Actor() Principal   { return request.input.Actor }
func (request MembershipGrantRequest) TargetRef() PrincipalRef {
	return request.input.TargetRef
}
func (request MembershipGrantRequest) ProjectRef() goal.ProjectRef {
	return request.input.ProjectRef
}
func (request MembershipGrantRequest) Role() Role { return request.input.Role }
func (request MembershipGrantRequest) ExpectedRevision() MembershipRevision {
	return request.input.ExpectedRevision
}
func (request MembershipGrantRequest) RequestedAt() time.Time { return request.input.RequestedAt }

type MembershipRevokeRequestInput struct {
	RequestRef       string
	Actor            Principal
	TargetRef        PrincipalRef
	ProjectRef       goal.ProjectRef
	ExpectedRevision MembershipRevision
	RequestedAt      time.Time
}

type MembershipRevokeRequest struct{ input MembershipRevokeRequestInput }

func NewMembershipRevokeRequest(input MembershipRevokeRequestInput) (MembershipRevokeRequest, error) {
	if !validCode(input.RequestRef) || ValidatePrincipal(input.Actor) != nil ||
		input.TargetRef.String() == "" || input.ProjectRef.String() == "" ||
		input.ExpectedRevision == 0 || input.ExpectedRevision == maxMembershipRevision() || input.RequestedAt.IsZero() {
		return MembershipRevokeRequest{}, errors.New("identity.membership_revoke_invalid")
	}
	input.RequestedAt = input.RequestedAt.UTC()
	return MembershipRevokeRequest{input: input}, nil
}

func (request MembershipRevokeRequest) RequestRef() string { return request.input.RequestRef }
func (request MembershipRevokeRequest) Actor() Principal   { return request.input.Actor }
func (request MembershipRevokeRequest) TargetRef() PrincipalRef {
	return request.input.TargetRef
}
func (request MembershipRevokeRequest) ProjectRef() goal.ProjectRef {
	return request.input.ProjectRef
}
func (request MembershipRevokeRequest) ExpectedRevision() MembershipRevision {
	return request.input.ExpectedRevision
}
func (request MembershipRevokeRequest) RequestedAt() time.Time { return request.input.RequestedAt }

type MembershipAuditAction string

const (
	MembershipAuditGranted MembershipAuditAction = "membership.granted"
	MembershipAuditRevoked MembershipAuditAction = "membership.revoked"
)

type MembershipAuditReceiptInput struct {
	Ref              string
	RequestRef       string
	Action           MembershipAuditAction
	ActorRef         PrincipalRef
	TargetRef        PrincipalRef
	ProjectRef       goal.ProjectRef
	Role             Role
	PreviousRevision MembershipRevision
	Revision         MembershipRevision
	OccurredAt       time.Time
}

type MembershipAuditReceipt struct{ input MembershipAuditReceiptInput }

func NewMembershipAuditReceipt(input MembershipAuditReceiptInput) (MembershipAuditReceipt, error) {
	if !validCode(input.Ref) || !validCode(input.RequestRef) || input.ActorRef.String() == "" ||
		input.TargetRef.String() == "" || input.ProjectRef.String() == "" || input.OccurredAt.IsZero() ||
		input.PreviousRevision == maxMembershipRevision() || input.Revision != input.PreviousRevision+1 {
		return MembershipAuditReceipt{}, errors.New("identity.membership_audit_invalid")
	}
	if err := ValidateRole(input.Role); err != nil {
		return MembershipAuditReceipt{}, err
	}
	switch input.Action {
	case MembershipAuditGranted:
	case MembershipAuditRevoked:
		if input.PreviousRevision == 0 {
			return MembershipAuditReceipt{}, errors.New("identity.membership_audit_invalid")
		}
	default:
		return MembershipAuditReceipt{}, errors.New("identity.membership_audit_action_invalid")
	}
	input.OccurredAt = input.OccurredAt.UTC()
	return MembershipAuditReceipt{input: input}, nil
}

func (receipt MembershipAuditReceipt) Ref() string        { return receipt.input.Ref }
func (receipt MembershipAuditReceipt) RequestRef() string { return receipt.input.RequestRef }
func (receipt MembershipAuditReceipt) Action() MembershipAuditAction {
	return receipt.input.Action
}
func (receipt MembershipAuditReceipt) ActorRef() PrincipalRef { return receipt.input.ActorRef }
func (receipt MembershipAuditReceipt) TargetRef() PrincipalRef {
	return receipt.input.TargetRef
}
func (receipt MembershipAuditReceipt) ProjectRef() goal.ProjectRef {
	return receipt.input.ProjectRef
}
func (receipt MembershipAuditReceipt) Role() Role { return receipt.input.Role }
func (receipt MembershipAuditReceipt) PreviousRevision() MembershipRevision {
	return receipt.input.PreviousRevision
}
func (receipt MembershipAuditReceipt) Revision() MembershipRevision {
	return receipt.input.Revision
}
func (receipt MembershipAuditReceipt) OccurredAt() time.Time { return receipt.input.OccurredAt }

func maxMembershipRevision() MembershipRevision {
	return MembershipRevision(^uint64(0))
}
