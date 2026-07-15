package identity

import (
	"context"
	"errors"
)

type principalContextKey struct{}

type Provider interface {
	Principal(context.Context) (Principal, error)
}

// BindPrincipal is the sole request-context ingress for authenticated identity.
func BindPrincipal(ctx context.Context, principal Principal) (context.Context, error) {
	if ctx == nil {
		return nil, errors.New("identity.context_required")
	}
	if ctx.Value(principalContextKey{}) != nil {
		return nil, errors.New("identity.principal_already_bound")
	}
	if err := ValidatePrincipal(principal); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, principalContextKey{}, principal), nil
}

func PrincipalFromContext(ctx context.Context) (Principal, error) {
	if ctx == nil {
		return Principal{}, errors.New("identity.context_required")
	}
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	if !ok {
		return Principal{}, errors.New("identity.principal_missing")
	}
	if err := ValidatePrincipal(principal); err != nil {
		return Principal{}, err
	}
	return principal, nil
}

// ContextProvider lets interfaces retain the existing Provider port while the
// actual principal remains request-bound.
type ContextProvider struct{}

func (ContextProvider) Principal(ctx context.Context) (Principal, error) {
	return PrincipalFromContext(ctx)
}

var _ Provider = ContextProvider{}
