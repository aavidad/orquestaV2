package executiontoken

import (
	"context"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type routedProviderStub struct {
	method        identity.AuthenticationMethod
	principal     identity.Principal
	err           error
	authenticates int
}

func (provider *routedProviderStub) AuthenticationMethod() identity.AuthenticationMethod {
	return provider.method
}

func (provider *routedProviderStub) Authenticate(
	_ context.Context,
	_ identity.Credential,
) (identity.Principal, error) {
	provider.authenticates++
	return provider.principal, provider.err
}

func TestRoutedProviderRoutesHumanCredentialsWithoutExecutionFallback(t *testing.T) {
	human := &routedProviderStub{
		method:    "local_token",
		principal: routedPrincipal(t, "human", identity.PrincipalKindHuman, "local_token"),
	}
	execution := &Broker{}
	provider, err := NewRoutedProvider(human, execution)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := identity.NewCredential([]byte("human-token"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := provider.Authenticate(context.Background(), credential)
	if err != nil || got != human.principal || human.authenticates != 1 {
		t.Fatalf("Authenticate() principal=%+v err=%v human_calls=%d", got, err, human.authenticates)
	}
}

func TestRoutedProviderExecutionNamespaceFailsClosed(t *testing.T) {
	human := &routedProviderStub{
		method:    "local_token",
		principal: routedPrincipal(t, "human", identity.PrincipalKindHuman, "local_token"),
	}
	provider, err := NewRoutedProvider(human, &Broker{})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := identity.NewCredential([]byte(tokenPrefix + "invalid"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Authenticate(context.Background(), credential); err == nil {
		t.Fatal("execution namespace unexpectedly authenticated")
	}
	if human.authenticates != 0 {
		t.Fatalf("execution credential fell back to human provider: calls=%d", human.authenticates)
	}
}

func TestNewRoutedProviderRequiresBothProviders(t *testing.T) {
	human := &routedProviderStub{method: "local_token"}
	for _, test := range []struct {
		name      string
		human     identity.IdentityProvider
		execution *Broker
	}{
		{name: "human", execution: &Broker{}},
		{name: "execution", human: human},
		{name: "human method", human: &routedProviderStub{}, execution: &Broker{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if provider, err := NewRoutedProvider(test.human, test.execution); provider != nil || err == nil {
				t.Fatalf("NewRoutedProvider() provider=%v err=%v", provider, err)
			}
		})
	}
}

func routedPrincipal(
	t *testing.T,
	suffix string,
	kind identity.PrincipalKind,
	method string,
) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef("actor:" + suffix)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, kind, method)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
