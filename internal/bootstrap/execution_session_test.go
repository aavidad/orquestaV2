package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/adapters/auth/executiontoken"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func v22NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type bootstrapExecutionSessionAuthoritySource struct {
	authority ports.ExecutionSessionAuthority
	uses      int
}

func (source *bootstrapExecutionSessionAuthoritySource) ExecutionSessionAuthority(_ context.Context, executionRef goal.ExecutionRef, method string) (ports.ExecutionSessionAuthority, error) {
	if source == nil || source.authority.Request.ExecutionRef != executionRef ||
		method != executiontoken.AuthenticationMethod {
		return ports.ExecutionSessionAuthority{}, errors.New("bootstrap.test_execution_authority_not_found")
	}
	source.uses++
	return source.authority, nil
}

type bootstrapExecutionSessionCredentialStore struct {
	credentials.Store
	uses int
}

func (store *bootstrapExecutionSessionCredentialStore) Use(
	ctx context.Context,
	request credentials.UseRequest,
	callback func(credentials.Secret) error,
) (credentials.Receipt, error) {
	receipt, err := store.Store.Use(ctx, request, callback)
	if err == nil {
		store.uses++
	}
	return receipt, err
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

func TestExecutionCredentialStorePhysicallyRevokesAcrossRestart(t *testing.T) {
	ctx := context.Background()
	_, _, authority, _ := bootstrapSessionResolver(t)
	source := &bootstrapExecutionSessionAuthoritySource{authority: authority}
	path := filepath.Join(t.TempDir(), "credentials.json")
	store := openBootstrapCredentialStore(t, path)
	broker, _ := executiontoken.New(store, source)
	if _, err := broker.Ensure(ctx, authority.Request); err != nil {
		t.Fatal(err)
	}
	var token []byte
	if err := broker.UseToken(ctx, authority.Request, func(value []byte) error {
		token = append([]byte(nil), value...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	credential, _ := identity.NewCredential(token)
	if principal, err := broker.Authenticate(ctx, credential); err != nil || principal != authority.ServicePrincipal {
		t.Fatalf("Authenticate principal=%+v err=%v", principal, err)
	}
	wrong := append([]byte(nil), token...)
	wrong[len(wrong)-2] ^= 1
	wrongCredential, _ := identity.NewCredential(wrong)
	if _, err := broker.Authenticate(ctx, wrongCredential); !executiontoken.IsError(err, executiontoken.CodeAuthenticationFailed) {
		t.Fatalf("wrong material err=%v", err)
	}
	_ = store.Close()
	store = openBootstrapCredentialStore(t, path)
	broker, _ = executiontoken.New(store, source)
	if err := broker.Revoke(ctx, authority.Request); err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Authenticate(ctx, credential); !executiontoken.IsError(err, executiontoken.CodeAuthenticationFailed) {
		t.Fatalf("revoked Authenticate err=%v", err)
	}
	_ = store.Close()
	content, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(content), `"material"`) {
		t.Fatalf("revoked store retained material err=%v", err)
	}
	store = openBootstrapCredentialStore(t, path)
	defer store.Close()
	broker, _ = executiontoken.New(store, source)
	if err := broker.Revoke(ctx, authority.Request); err != nil {
		t.Fatal(err)
	}
	clear(token)
	clear(wrong)
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
	client := &http.Client{Transport: transport, CheckRedirect: func(request *http.Request, _ []*http.Request) error {
		observed.Lock()
		redirectAuthorization = request.Header.Get("Authorization")
		observed.Unlock()
		return errors.New("redirect forbidden")
	}}
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
	forbiddenRequest(t, transport, server.URL+"/other")
	queryTarget := forbiddenRequest(t, transport, server.URL+"/mcp?unexpected=yes")
	if queryTarget.Header.Get("Authorization") != "" {
		t.Fatal("bearer mutated caller request on query target")
	}
	foreign := forbiddenRequest(t, transport, "http://127.0.0.1:1/mcp")
	if foreign.Header.Get("Authorization") != "" {
		t.Fatal("bearer mutated the caller request on forbidden target")
	}
	transport.destroy()
	if len(transport.material) != 0 {
		t.Fatal("transport retained bearer material after destroy")
	}
	forbiddenRequest(t, transport, foreign.URL.String())
}

func forbiddenRequest(t *testing.T, transport http.RoundTripper, target string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(request); err == nil || err.Error() != "bootstrap.execution_session_target_forbidden" {
		t.Fatalf("target %q error=%v", target, err)
	}
	return request
}

func openBootstrapCredentialStore(t *testing.T, path string) *credentiallocal.Store {
	t.Helper()
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: path, OwnerUID: os.Geteuid(), MaxStoreBytes: 1 << 20,
		Now: func() time.Time { return time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func bootstrapSessionResolver(t *testing.T) (*codexExecutionSessionResolver, ports.AgentLaunchRequest, ports.ExecutionSessionAuthority, *bootstrapExecutionSessionCredentialStore) {
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
	credentialRoot := t.TempDir()
	v22NoError(t, os.Chmod(credentialRoot, 0o700))
	localStore, err := credentiallocal.Open(credentiallocal.Options{
		Path: filepath.Join(credentialRoot, "credentials.json"), OwnerUID: os.Geteuid(),
		MaxStoreBytes: 1 << 20, Now: func() time.Time { return time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("open credential store: %v", err)
	}
	t.Cleanup(func() { _ = localStore.Close() })
	store := &bootstrapExecutionSessionCredentialStore{Store: localStore}
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
