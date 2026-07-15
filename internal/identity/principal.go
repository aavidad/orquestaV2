package identity

import (
	"errors"

	"orquesta/internal/goal"
)

type PrincipalKind string

const (
	PrincipalKindHuman   PrincipalKind = "human"
	PrincipalKindService PrincipalKind = "service"
)

// Principal is the identity established by a trusted authentication adapter.
// Project scope is deliberately request-bound and is never identity metadata.
type Principal struct {
	Ref      PrincipalRef
	ActorRef goal.ActorRef
	Kind     PrincipalKind
	Method   string
}

func NewPrincipal(
	ref PrincipalRef,
	actorRef goal.ActorRef,
	kind PrincipalKind,
	method string,
) (Principal, error) {
	principal := Principal{Ref: ref, ActorRef: actorRef, Kind: kind, Method: method}
	if err := ValidatePrincipal(principal); err != nil {
		return Principal{}, err
	}
	return principal, nil
}

func ValidatePrincipal(principal Principal) error {
	if principal.Ref.String() == "" {
		return errors.New("identity.invalid_principal_ref")
	}
	if principal.ActorRef.String() == "" {
		return errors.New("identity.invalid_actor_ref")
	}
	if principal.Kind != PrincipalKindHuman && principal.Kind != PrincipalKindService {
		return errors.New("identity.invalid_principal_kind")
	}
	if !validCode(principal.Method) {
		return errors.New("identity.invalid_authentication_method")
	}
	return nil
}
