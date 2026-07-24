package executiontoken

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type providerAuthoritySource struct {
	authority ports.ExecutionSessionAuthority
}

func (source *providerAuthoritySource) ExecutionSessionAuthority(_ context.Context, ref goal.ExecutionRef, method string) (ports.ExecutionSessionAuthority, error) {
	if source.authority.Request.ExecutionRef != ref || source.authority.ServicePrincipal.Method != method {
		return ports.ExecutionSessionAuthority{}, nil
	}
	return source.authority, nil
}

type providerCredentialStore struct {
	credentials.Store
	material []byte
}

func (store *providerCredentialStore) Create(_ context.Context, request credentials.CreateRequest) (credentials.MutationResult, error) {
	store.material = request.Material.Bytes()
	return credentials.MutationResult{Metadata: credentials.Metadata{
		CreatedAt: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}}, nil
}

func (store *providerCredentialStore) Use(_ context.Context, request credentials.UseRequest, callback func(credentials.Secret) error) (credentials.Receipt, error) {
	secret, _ := credentials.NewSecret(store.material)
	defer secret.Destroy()
	if err := callback(secret); err != nil {
		return credentials.Receipt{}, err
	}
	return credentials.Receipt{OccurredAt: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)}, nil
}

func TestBrokerRejectsAuthoritySubstitution(t *testing.T) {
	ctx := context.Background()
	request := providerSessionRequest()
	authority, _ := application.DeriveExecutionSessionAuthority(request, AuthenticationMethod)
	source := &providerAuthoritySource{authority: authority}
	broker, _ := New(&providerCredentialStore{}, source)
	broker.random = bytes.NewReader(bytes.Repeat([]byte{0x11}, secretBytes))
	if _, err := broker.Ensure(ctx, request); err != nil {
		t.Fatal(err)
	}
	var token []byte
	_ = broker.UseToken(ctx, request, func(value []byte) error { token = append([]byte(nil), value...); return nil })
	credential, _ := identity.NewCredential(token)
	source.authority.ServicePrincipal.Method = "other"
	if _, err := broker.Authenticate(ctx, credential); !IsError(err, CodeAuthenticationFailed) {
		t.Fatalf("substituted authority err=%v", err)
	}
}

func providerSessionRequest() ports.ExecutionSessionEnsureRequest {
	project, _ := goal.NewProjectRef("project:executiontoken")
	goalRef, _ := goal.NewGoalRef("goal:executiontoken")
	item, _ := goal.NewWorkItemRef("work-item:executiontoken")
	execution, _ := goal.NewExecutionRef("execution:executiontoken")
	return ports.ExecutionSessionEnsureRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution,
		ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1, SpecHash: strings.Repeat("a", 64),
	}
}
