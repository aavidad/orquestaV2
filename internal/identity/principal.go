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
// DefaultProjectRef is temporary local compatibility metadata and never grants
// V10 project authority; application requests carry their project explicitly.
type Principal struct {
	Ref               PrincipalRef
	ActorRef          goal.ActorRef
	Kind              PrincipalKind
	Method            string
	DefaultProjectRef goal.ProjectRef
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

func NewPrincipalWithDefaultProject(
	ref PrincipalRef,
	actorRef goal.ActorRef,
	kind PrincipalKind,
	method string,
	defaultProjectRef goal.ProjectRef,
) (Principal, error) {
	principal := Principal{
		Ref: ref, ActorRef: actorRef, Kind: kind, Method: method,
		DefaultProjectRef: defaultProjectRef,
	}
	if err := ValidatePrincipal(principal); err != nil {
		return Principal{}, err
	}
	if principal.DefaultProjectRef.String() == "" {
		return Principal{}, errors.New("identity.invalid_default_project_ref")
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
