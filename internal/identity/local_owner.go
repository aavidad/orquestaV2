package identity

import (
	"context"
	"errors"

	"orquesta/internal/goal"
)

const LocalOwnerMethod = "local_owner"

type Principal struct {
	ActorRef          goal.ActorRef
	DefaultProjectRef goal.ProjectRef
	Method            string
}

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
	return &LocalOwnerProvider{principal: Principal{
		ActorRef:          actorRef,
		DefaultProjectRef: projectRef,
		Method:            LocalOwnerMethod,
	}}, nil
}

func (provider *LocalOwnerProvider) Principal(context.Context) (Principal, error) {
	if provider == nil {
		return Principal{}, errors.New("identity.provider_unavailable")
	}
	return provider.principal, nil
}
