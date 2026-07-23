package executiontoken

import (
	"bytes"
	"context"
	"errors"

	"orquesta/internal/identity"
)

const RoutedAuthenticationMethod identity.AuthenticationMethod = "identity_router"

// RoutedProvider adds execution-scoped authentication to the configured human
// identity provider. The execution-token namespace is fail-closed: a token
// with the reserved prefix is never retried against the human provider.
type RoutedProvider struct {
	human     identity.IdentityProvider
	execution *Broker
}

var _ identity.IdentityProvider = (*RoutedProvider)(nil)

func NewRoutedProvider(
	human identity.IdentityProvider,
	execution *Broker,
) (*RoutedProvider, error) {
	if human == nil || human.AuthenticationMethod() == "" || execution == nil {
		return nil, errors.New("executiontoken.routed_provider_dependencies_required")
	}
	return &RoutedProvider{human: human, execution: execution}, nil
}

func (*RoutedProvider) AuthenticationMethod() identity.AuthenticationMethod {
	return RoutedAuthenticationMethod
}

func (provider *RoutedProvider) Authenticate(
	ctx context.Context,
	credential identity.Credential,
) (identity.Principal, error) {
	if provider == nil || provider.human == nil || provider.execution == nil {
		return identity.Principal{}, errors.New("executiontoken.routed_provider_unavailable")
	}
	executionCredential := false
	if err := credential.Use(func(material []byte) error {
		executionCredential = bytes.HasPrefix(material, []byte(tokenPrefix))
		return nil
	}); err != nil {
		return identity.Principal{}, err
	}
	if executionCredential {
		return provider.execution.Authenticate(ctx, credential)
	}
	return provider.human.Authenticate(ctx, credential)
}
