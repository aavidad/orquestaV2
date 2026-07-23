package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/auth/executiontoken"
	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type bootstrapExecutionSessionAuthoritySource struct {
	authority ports.ExecutionSessionAuthority
	uses      int
}

func (source *bootstrapExecutionSessionAuthoritySource) ExecutionSessionAuthority(
	_ context.Context,
	executionRef goal.ExecutionRef,
	method string,
) (ports.ExecutionSessionAuthority, error) {
	if source == nil || source.authority.Request.ExecutionRef != executionRef ||
		method != executiontoken.AuthenticationMethod {
		return ports.ExecutionSessionAuthority{}, errors.New("bootstrap.test_execution_authority_not_found")
	}
	source.uses++
	return source.authority, nil
}

type bootstrapExecutionSessionCredentialStore struct {
	material []byte
	metadata credentials.Metadata
	uses     int
}

func (store *bootstrapExecutionSessionCredentialStore) Create(
	_ context.Context,
	request credentials.CreateRequest,
) (credentials.MutationResult, error) {
	if store.metadata.CredentialRef != "" {
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorAlreadyExists, "credential_ref")
	}
	store.material = request.Material.Bytes()
	store.metadata = credentials.Metadata{
		CredentialRef: request.CredentialRef,
		OwnerRef:      request.OwnerRef,
		ScopeRefs:     append([]credentials.ScopeRef(nil), request.ScopeRefs...),
		PurposeRef:    request.PurposeRef,
		Version:       1,
		CreatedAt:     time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC),
	}
	return credentials.MutationResult{Metadata: store.metadata}, nil
}

func (store *bootstrapExecutionSessionCredentialStore) Use(
	_ context.Context,
	request credentials.UseRequest,
	callback func(credentials.Secret) error,
) (credentials.Receipt, error) {
	if store.metadata.CredentialRef != request.CredentialRef || store.metadata.OwnerRef != request.OwnerRef ||
		store.metadata.PurposeRef != request.PurposeRef || len(store.metadata.ScopeRefs) != 1 ||
		store.metadata.ScopeRefs[0] != request.ScopeRef {
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
	store.uses++
	return credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRef: request.ScopeRef,
		PurposeRef: request.PurposeRef, Version: 1, OccurredAt: store.metadata.CreatedAt,
	}, nil
}

func (*bootstrapExecutionSessionCredentialStore) Rotate(
	context.Context,
	credentials.RotateRequest,
) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, credentials.NewError(credentials.ErrorInvalidRequest, "rotate")
}

func (*bootstrapExecutionSessionCredentialStore) Revoke(
	context.Context,
	credentials.RevokeRequest,
) (credentials.MutationResult, error) {
	return credentials.MutationResult{}, credentials.NewError(credentials.ErrorInvalidRequest, "revoke")
}

func TestCodexExecutionSessionResolverResolvesExactBoundSession(t *testing.T) {
	resolver, request, authority, store := bootstrapSessionResolver(t)

	session, err := resolver.ResolveCodexSession(context.Background(), request)
	if err != nil {
		t.Fatalf("ResolveCodexSession: %v", err)
	}
	defer session.BearerToken.Destroy()
	if session.Ref != authority.SessionRef || session.Endpoint != "http://127.0.0.1:7788/mcp" ||
		len(session.BearerToken.Bytes()) == 0 {
		t.Fatalf("resolved session is not the exact authority projection: %+v", session)
	}
	if store.uses != 1 {
		t.Fatalf("credential use count=%d want 1", store.uses)
	}
}

func TestCodexExecutionSessionResolverFailsClosedForAnyBindingMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.AgentLaunchRequest)
	}{
		{"session ref", func(request *ports.AgentLaunchRequest) {
			request.SessionRef, _ = ports.NewExecutionSessionRef("execution-session:other")
		}},
		{"project", func(request *ports.AgentLaunchRequest) { request.ProjectRef, _ = goal.NewProjectRef("project:other") }},
		{"goal", func(request *ports.AgentLaunchRequest) { request.GoalRef, _ = goal.NewGoalRef("goal:other") }},
		{"work item", func(request *ports.AgentLaunchRequest) {
			request.WorkItemRef, _ = goal.NewWorkItemRef("work-item:other")
		}},
		{"execution", func(request *ports.AgentLaunchRequest) {
			request.ExecutionRef, _ = goal.NewExecutionRef("execution:other")
		}},
		{"attempt", func(request *ports.AgentLaunchRequest) { request.ExecutionAttempt++ }},
		{"plan generation", func(request *ports.AgentLaunchRequest) { request.PlanGeneration++ }},
		{"app spec generation", func(request *ports.AgentLaunchRequest) { request.AppSpecGeneration++ }},
		{"spec hash", func(request *ports.AgentLaunchRequest) { request.SpecHash = strings.Repeat("b", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, request, _, store := bootstrapSessionResolver(t)
			test.mutate(&request)

			_, err := resolver.ResolveCodexSession(context.Background(), request)
			if err == nil || err.Error() != "bootstrap.execution_session_unavailable" {
				t.Fatalf("ResolveCodexSession mismatch error=%v", err)
			}
			if store.uses != 0 {
				t.Fatalf("mismatch materialized credential %d times", store.uses)
			}
		})
	}
}

func TestExecutionBearerTransportAllowsOnlyLoopbackEndpoint(t *testing.T) {
	for _, endpoint := range []string{
		"https://127.0.0.1:7788/mcp", "http://localhost:7788/mcp", "http://127.0.0.1:7788",
		"http://127.0.0.1:7788/mcp?unexpected=yes", "http://127.0.0.1:7788/mcp#fragment",
		"http://user@127.0.0.1:7788/mcp", "http://192.0.2.1:7788/mcp",
	} {
		t.Run(endpoint, func(t *testing.T) {
			if _, err := newExecutionBearerTransport(endpoint, []byte("private-token")); err == nil || err.Error() != "bootstrap.execution_session_transport_invalid" {
				t.Fatalf("newExecutionBearerTransport(%q) error=%v", endpoint, err)
			}
		})
	}
	if _, err := newExecutionBearerTransport("http://127.0.0.1:7788/mcp", nil); err == nil {
		t.Fatal("empty bearer material was accepted")
	}
}

func TestExecutionBearerTransportBindsBearerToExactTargetAndRedirectsNeverReceiveIt(t *testing.T) {
	const token = "bootstrap-session-bearer-private"
	var observed sync.Mutex
	var initialAuthorization string
	var redirectAuthorization string
	redirected := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		observed.Lock()
		defer observed.Unlock()
		switch request.URL.Path {
		case "/mcp":
			initialAuthorization = request.Header.Get("Authorization")
			http.Redirect(writer, request, "/redirected", http.StatusFound)
		case "/redirected":
			redirected = true
			writer.WriteHeader(http.StatusNoContent)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	transport, err := newExecutionBearerTransport(server.URL+"/mcp", []byte(token))
	if err != nil {
		t.Fatalf("newExecutionBearerTransport: %v", err)
	}
	defer transport.destroy()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			observed.Lock()
			redirectAuthorization = request.Header.Get("Authorization")
			observed.Unlock()
			return errors.New("redirect forbidden")
		},
	}
	response, err := client.Get(server.URL + "/mcp")
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("redirect was followed or accepted")
	}
	observed.Lock()
	defer observed.Unlock()
	if initialAuthorization != "Bearer "+token || redirectAuthorization != "" || redirected {
		t.Fatalf("initial authorization=%q redirect authorization=%q redirected=%t", initialAuthorization, redirectAuthorization, redirected)
	}

	wrongPath, err := http.NewRequest(http.MethodGet, server.URL+"/other", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(wrongPath); err == nil || err.Error() != "bootstrap.execution_session_target_forbidden" {
		t.Fatalf("wrong path error=%v", err)
	}
	queryTarget, err := http.NewRequest(http.MethodGet, server.URL+"/mcp?unexpected=yes", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(queryTarget); err == nil || err.Error() != "bootstrap.execution_session_target_forbidden" {
		t.Fatalf("query target error=%v", err)
	}
	if queryTarget.Header.Get("Authorization") != "" {
		t.Fatal("bearer mutated caller request on query target")
	}
	foreign, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(foreign); err == nil || err.Error() != "bootstrap.execution_session_target_forbidden" {
		t.Fatalf("foreign target error=%v", err)
	}
	if foreign.Header.Get("Authorization") != "" {
		t.Fatal("bearer mutated the caller request on forbidden target")
	}

	transport.destroy()
	if len(transport.material) != 0 {
		t.Fatal("transport retained bearer material after destroy")
	}
	if _, err := transport.RoundTrip(foreign); err == nil || err.Error() != "bootstrap.execution_session_target_forbidden" {
		t.Fatalf("destroyed transport error=%v", err)
	}
}

func bootstrapSessionResolver(t *testing.T) (
	*codexExecutionSessionResolver,
	ports.AgentLaunchRequest,
	ports.ExecutionSessionAuthority,
	*bootstrapExecutionSessionCredentialStore,
) {
	t.Helper()
	project, _ := goal.NewProjectRef("project:bootstrap-session")
	goalRef, _ := goal.NewGoalRef("goal:bootstrap-session")
	workItem, _ := goal.NewWorkItemRef("work-item:bootstrap-session")
	execution, _ := goal.NewExecutionRef("execution:bootstrap-session")
	actor, _ := goal.NewActorRef("actor:bootstrap-session")
	request := ports.AgentLaunchRequest{
		ProjectRef: project, GoalRef: goalRef, WorkItemRef: workItem, ExecutionRef: execution,
		ExecutionAttempt: 1, PlanGeneration: 2, AppSpecGeneration: 3, SpecHash: strings.Repeat("a", 64), ActorRef: actor,
	}
	authority, err := application.DeriveExecutionSessionAuthority(ports.ExecutionSessionEnsureRequest{
		ProjectRef: request.ProjectRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		ExecutionRef: request.ExecutionRef, ExecutionAttempt: request.ExecutionAttempt,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration, SpecHash: request.SpecHash,
	}, executiontoken.AuthenticationMethod)
	if err != nil {
		t.Fatalf("DeriveExecutionSessionAuthority: %v", err)
	}
	request.SessionRef = authority.SessionRef
	source := &bootstrapExecutionSessionAuthoritySource{authority: authority}
	store := &bootstrapExecutionSessionCredentialStore{}
	broker, err := executiontoken.New(store, source)
	if err != nil {
		t.Fatalf("new execution token broker: %v", err)
	}
	if _, err := broker.Ensure(context.Background(), authority.Request); err != nil {
		t.Fatalf("ensure execution credential: %v", err)
	}
	resolved, err := newCodexExecutionSessionResolver(source, broker, "http://127.0.0.1:7788/mcp")
	if err != nil {
		t.Fatalf("new resolver: %v", err)
	}
	resolver, ok := resolved.(*codexExecutionSessionResolver)
	if !ok {
		t.Fatalf("resolver type=%T", resolved)
	}
	return resolver, request, authority, store
}
