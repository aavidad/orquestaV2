package identity

import (
	"context"
	"errors"

	"orquesta/internal/goal"
)

const LocalOwnerMethod = "local_owner"

type Provider interface {
	Principal(context.Context) (Principal, error)
}

type LocalOwnerProvider struct {
	principal Principal
}

func NewLocalOwnerProvider(actorRef goal.ActorRef, projectRef goal.ProjectRef) (*LocalOwnerProvider, error) {
	if actorRef.String() == "" {
		return nil, errors.New("identity.invalid_actor_ref")
	}
	if projectRef.String() == "" {
		return nil, errors.New("identity.invalid_project_ref")
	}
	principalRef, err := NewPrincipalRef(actorRef.String())
	if err != nil {
		return nil, err
	}
	principal, err := NewPrincipalWithDefaultProject(
		principalRef,
		actorRef,
		PrincipalKindHuman,
		LocalOwnerMethod,
		projectRef,
	)
	if err != nil {
		return nil, err
	}
	return &LocalOwnerProvider{principal: principal}, nil
}

func (provider *LocalOwnerProvider) Principal(context.Context) (Principal, error) {
	if provider == nil {
		return Principal{}, errors.New("identity.provider_unavailable")
	}
	return provider.principal, nil
}
