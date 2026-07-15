package identity

import (
	"errors"
	"time"

	"orquesta/internal/goal"
)

type AuthorizationRequestInput struct {
	RequestRef  string
	Principal   Principal
	ProjectRef  goal.ProjectRef
	Permission  Permission
	ResourceRef string
	RequestedAt time.Time
}

type AuthorizationRequest struct{ input AuthorizationRequestInput }

func NewAuthorizationRequest(input AuthorizationRequestInput) (AuthorizationRequest, error) {
	if !validCode(input.RequestRef) || ValidatePrincipal(input.Principal) != nil ||
		input.ProjectRef.String() == "" || !validCode(input.ResourceRef) || input.RequestedAt.IsZero() {
		return AuthorizationRequest{}, errors.New("identity.authorization_request_invalid")
	}
	if err := ValidatePermission(input.Permission); err != nil {
		return AuthorizationRequest{}, err
	}
	input.RequestedAt = input.RequestedAt.UTC()
	return AuthorizationRequest{input: input}, nil
}

func (request AuthorizationRequest) RequestRef() string { return request.input.RequestRef }
func (request AuthorizationRequest) Principal() Principal {
	return request.input.Principal
}
func (request AuthorizationRequest) ProjectRef() goal.ProjectRef {
	return request.input.ProjectRef
}
func (request AuthorizationRequest) Permission() Permission { return request.input.Permission }
func (request AuthorizationRequest) ResourceRef() string    { return request.input.ResourceRef }
func (request AuthorizationRequest) RequestedAt() time.Time { return request.input.RequestedAt }

type AuthorizationOutcome string

const (
	AuthorizationAllowed AuthorizationOutcome = "allowed"
	AuthorizationDenied  AuthorizationOutcome = "denied"
)

type AuthorizationDecisionInput struct {
	Request            AuthorizationRequest
	Outcome            AuthorizationOutcome
	Role               Role
	MembershipRevision MembershipRevision
	ReasonCode         string
	DecidedAt          time.Time
}

type AuthorizationDecision struct{ input AuthorizationDecisionInput }

func NewAuthorizationDecision(input AuthorizationDecisionInput) (AuthorizationDecision, error) {
	if !validCode(input.ReasonCode) || input.DecidedAt.IsZero() ||
		input.Request.RequestRef() == "" || input.DecidedAt.Before(input.Request.RequestedAt()) {
		return AuthorizationDecision{}, errors.New("identity.authorization_decision_invalid")
	}
	switch input.Outcome {
	case AuthorizationAllowed:
		if ValidateRole(input.Role) != nil || !RoleAllows(input.Role, input.Request.Permission()) ||
			(input.Role != RolePlatformAdmin && input.MembershipRevision == 0) {
			return AuthorizationDecision{}, errors.New("identity.authorization_allow_invalid")
		}
	case AuthorizationDenied:
		if input.Role != "" && ValidateRole(input.Role) != nil {
			return AuthorizationDecision{}, errors.New("identity.authorization_deny_invalid")
		}
	default:
		return AuthorizationDecision{}, errors.New("identity.authorization_outcome_invalid")
	}
	input.DecidedAt = input.DecidedAt.UTC()
	return AuthorizationDecision{input: input}, nil
}

func (decision AuthorizationDecision) Request() AuthorizationRequest {
	return decision.input.Request
}
func (decision AuthorizationDecision) Outcome() AuthorizationOutcome {
	return decision.input.Outcome
}
func (decision AuthorizationDecision) Role() Role { return decision.input.Role }
func (decision AuthorizationDecision) MembershipRevision() MembershipRevision {
	return decision.input.MembershipRevision
}
func (decision AuthorizationDecision) ReasonCode() string   { return decision.input.ReasonCode }
func (decision AuthorizationDecision) DecidedAt() time.Time { return decision.input.DecidedAt }

type AuthorizationReceiptInput struct {
	Ref        string
	Decision   AuthorizationDecision
	RecordedAt time.Time
}

type AuthorizationReceipt struct{ input AuthorizationReceiptInput }

func NewAuthorizationReceipt(input AuthorizationReceiptInput) (AuthorizationReceipt, error) {
	if !validCode(input.Ref) || input.Decision.Request().RequestRef() == "" || input.RecordedAt.IsZero() ||
		input.RecordedAt.Before(input.Decision.DecidedAt()) {
		return AuthorizationReceipt{}, errors.New("identity.authorization_receipt_invalid")
	}
	input.RecordedAt = input.RecordedAt.UTC()
	return AuthorizationReceipt{input: input}, nil
}

func (receipt AuthorizationReceipt) Ref() string { return receipt.input.Ref }
func (receipt AuthorizationReceipt) Decision() AuthorizationDecision {
	return receipt.input.Decision
}
func (receipt AuthorizationReceipt) RecordedAt() time.Time { return receipt.input.RecordedAt }
