package executiontoken

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type authoritySource struct {
	authority ports.ExecutionSessionAuthority
	err       error
}

func (source *authoritySource) ExecutionSessionAuthority(
	_ context.Context,
	ref goal.ExecutionRef,
	method string,
) (ports.ExecutionSessionAuthority, error) {
	if source.err != nil || source.authority.Request.ExecutionRef != ref ||
		source.authority.ServicePrincipal.Method != method {
		return ports.ExecutionSessionAuthority{}, source.err
	}
	return source.authority, nil
}

// memoryCredentialStore is an inward-port fake, not another credential
// implementation. restart clones its durable projection for recovery tests.
type memoryCredentialStore struct {
	mu       sync.Mutex
	metadata credentials.Metadata
	material []byte
}

func (store *memoryCredentialStore) Create(
	_ context.Context,
	request credentials.CreateRequest,
) (credentials.MutationResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.metadata.CredentialRef != "" {
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorAlreadyExists, "credential_ref")
	}
	store.metadata = credentials.Metadata{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRefs:  append([]credentials.ScopeRef(nil), request.ScopeRefs...),
		PurposeRef: request.PurposeRef, Version: 1,
		CreatedAt: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	}
	store.material = request.Material.Bytes()
	return credentials.MutationResult{Metadata: store.metadata}, nil
}

func (store *memoryCredentialStore) Use(
	_ context.Context,
	request credentials.UseRequest,
	callback func(credentials.Secret) error,
) (credentials.Receipt, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.metadata.CredentialRef != request.CredentialRef {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorNotFound, "credential_ref")
	}
	if store.metadata.Revoked {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	}
	if store.metadata.OwnerRef != request.OwnerRef || store.metadata.PurposeRef != request.PurposeRef ||
		len(store.metadata.ScopeRefs) != 1 || store.metadata.ScopeRefs[0] != request.ScopeRef {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorScopeDenied, "scope")
	}
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.Receipt{}, err
	}
	defer secret.Destroy()
	if err := callback(secret); err != nil {
		return credentials.Receipt{}, err
	}
	return credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: 1,
		OccurredAt: store.metadata.CreatedAt,
	}, nil
}

func (*memoryCredentialStore) Rotate(
	context.Context,
	credentials.RotateRequest,
) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, credentials.NewError(credentials.ErrorInvalidRequest, "rotate")
}

func (store *memoryCredentialStore) Revoke(
	_ context.Context,
	request credentials.RevokeRequest,
) (credentials.MutationResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.metadata.CredentialRef != request.CredentialRef ||
		store.metadata.OwnerRef != request.OwnerRef || request.ExpectedVersion != 1 {
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorNotFound, "credential_ref")
	}
	store.metadata.Revoked = true
	store.metadata.RevokedAt = store.metadata.CreatedAt.Add(time.Minute)
	clear(store.material)
	store.material = nil
	return credentials.MutationResult{Metadata: store.metadata}, nil
}

func (store *memoryCredentialStore) restart() *memoryCredentialStore {
	store.mu.Lock()
	defer store.mu.Unlock()
	result := &memoryCredentialStore{
		metadata: store.metadata,
		material: append([]byte(nil), store.material...),
	}
	result.metadata.ScopeRefs = append([]credentials.ScopeRef(nil), store.metadata.ScopeRefs...)
	return result
}

func TestBrokerAuthenticatesAcrossRestartAndRevokesWithoutPersistingToken(t *testing.T) {
	ctx := context.Background()
	request := executionSessionTestRequest(t)
	authority, err := application.DeriveExecutionSessionAuthority(request, AuthenticationMethod)
	if err != nil {
		t.Fatal(err)
	}
	source := &authoritySource{authority: authority}
	store := &memoryCredentialStore{}
	broker, err := newWithRandom(store, source, bytes.NewReader(bytes.Repeat([]byte{0x5a}, secretBytes)))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := broker.Ensure(ctx, request)
	if err != nil || receipt.Replayed || !application.SameExecutionSessionAuthority(receipt.Authority, authority) {
		t.Fatalf("Ensure receipt=%+v err=%v", receipt, err)
	}
	var token []byte
	if err := broker.UseToken(ctx, request, func(material []byte) error {
		token = append([]byte(nil), material...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	credential, err := identity.NewCredential(token)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := broker.Authenticate(ctx, credential)
	if err != nil || principal != authority.ServicePrincipal {
		t.Fatalf("Authenticate principal=%+v err=%v", principal, err)
	}
	store = store.restart()
	broker, err = newWithRandom(store, source, bytes.NewReader(bytes.Repeat([]byte{0xa5}, secretBytes)))
	if err != nil {
		t.Fatal(err)
	}
	principal, err = broker.Authenticate(ctx, credential)
	if err != nil || principal != authority.ServicePrincipal {
		t.Fatalf("restart Authenticate principal=%+v err=%v", principal, err)
	}
	source.err = errors.New("state.not_found")
	if _, err := broker.Authenticate(ctx, credential); !IsError(err, CodeAuthenticationFailed) {
		t.Fatalf("terminal authority Authenticate err=%v", err)
	}
	if bytes.Contains(store.material, token) {
		t.Fatal("composite execution token entered credential projection")
	}
	if strings.Contains(errString(broker.Authenticate(ctx, credential)), string(token)) {
		t.Fatal("authentication error leaked token")
	}
	clear(token)
}

func TestBrokerRejectsWrongMaterialAndAuthoritySubstitution(t *testing.T) {
	ctx := context.Background()
	request := executionSessionTestRequest(t)
	authority, _ := application.DeriveExecutionSessionAuthority(request, AuthenticationMethod)
	source := &authoritySource{authority: authority}
	store := &memoryCredentialStore{}
	broker, _ := newWithRandom(store, source, bytes.NewReader(bytes.Repeat([]byte{0x11}, secretBytes)))
	if _, err := broker.Ensure(ctx, request); err != nil {
		t.Fatal(err)
	}
	wrong := encodeToken(request.ExecutionRef, bytes.Repeat([]byte{0x22}, secretBytes))
	credential, _ := identity.NewCredential(wrong)
	if _, err := broker.Authenticate(ctx, credential); !IsError(err, CodeAuthenticationFailed) {
		t.Fatalf("wrong material err=%v", err)
	}
	source.authority.ServicePrincipal.Method = "other"
	if _, err := broker.Authenticate(ctx, credential); !IsError(err, CodeAuthenticationFailed) {
		t.Fatalf("substituted authority err=%v", err)
	}
	clear(wrong)
}

func executionSessionTestRequest(t *testing.T) ports.ExecutionSessionEnsureRequest {
	t.Helper()
	project, _ := goal.NewProjectRef("project:executiontoken")
	goalRef, _ := goal.NewGoalRef("goal:executiontoken")
	item, _ := goal.NewWorkItemRef("work-item:executiontoken")
	execution, _ := goal.NewExecutionRef("execution:executiontoken")
	return ports.ExecutionSessionEnsureRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution,
		ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1,
		SpecHash: strings.Repeat("a", 64),
	}
}

func errString(_ identity.Principal, err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
