package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type testExecutor struct {
	calls      int
	invocation commandcore.Invocation
	result     commandcore.Result
}

func (executor *testExecutor) Dispatch(_ context.Context, invocation commandcore.Invocation) commandcore.Result {
	executor.calls++
	executor.invocation = invocation
	result := executor.result
	result.CommandID, result.CommandVersion, result.RequestRef = invocation.CommandID, invocation.CommandVersion, invocation.RequestRef
	return result
}
func (executor *testExecutor) Definitions() []commandcore.Definition {
	return commandcore.CanonicalDefinitions()
}
func (*testExecutor) Limits() commandcore.APILimits {
	return commandcore.APILimits{MaxRequestBytes: 4096, MaxListLimit: 100}
}

type testIdentity struct {
	principal identity.Principal
	err       error
	calls     *int
}

func (provider testIdentity) Principal(context.Context) (identity.Principal, error) {
	if provider.calls != nil {
		*provider.calls++
	}
	return provider.principal, provider.err
}

func httpPrincipal(t *testing.T) identity.Principal {
	t.Helper()
	ref, _ := identity.NewPrincipalRef("principal:http")
	actor, _ := goal.NewActorRef("actor:http")
	principal, err := identity.NewPrincipal(ref, actor, identity.PrincipalKindHuman, "local_token")
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func TestHTTPCommandBoundaryBindsAuthorityOutsidePayloadAndPreservesEnvelope(t *testing.T) {
	executor := &testExecutor{result: commandcore.Result{Data: json.RawMessage(`{"ready":true}`), AuditRef: "audit:http"}}
	handler, err := New(Config{Dispatcher: executor, Identity: testIdentity{principal: httpPrincipal(t)}})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"version":"1","request_ref":"request:http","project_ref":"project:http","claimed_execution_ref":"execution:http","payload":{"goal_ref":"goal:g","message_ref":"mailbox-message:m","recipient_work_item_ref":"work-item:w"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/commands/orquesta.mailbox.get", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var result commandcore.Result
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || result.Failure != nil || executor.calls != 1 ||
		executor.invocation.Principal.Ref.String() != "principal:http" ||
		executor.invocation.ClaimedExecutionRef != "execution:http" {
		t.Fatalf("code=%d result=%+v invocation=%+v", response.Code, result, executor.invocation)
	}
}

func TestHTTPCommandBoundaryRejectsUnauthenticatedBeforeDispatcher(t *testing.T) {
	executor := &testExecutor{}
	handler, err := New(Config{Dispatcher: executor, Identity: testIdentity{err: errors.New("missing")}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/commands/orquesta.system.status", strings.NewReader(`{"version":"1","request_ref":"request:http","project_ref":"project:http","payload":{}}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || executor.calls != 0 {
		t.Fatalf("code=%d calls=%d", response.Code, executor.calls)
	}
}

func TestGeneratedBindingsExactlyMatchCanonicalDefinitions(t *testing.T) {
	handler, err := New(Config{Dispatcher: &testExecutor{}, Identity: testIdentity{principal: httpPrincipal(t)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(handler.bindings) != len(commandcore.CanonicalDefinitions()) {
		t.Fatalf("bindings=%d definitions=%d", len(handler.bindings), len(commandcore.CanonicalDefinitions()))
	}
	definitions := make(map[string]commandcore.Definition)
	for _, definition := range commandcore.CanonicalDefinitions() {
		definitions[definition.ID] = definition
	}
	for _, binding := range handler.bindings {
		definition, ok := definitions[binding.CommandID]
		if !ok || binding.Version != definition.Version || binding.Path != definition.HTTP.Path ||
			binding.ExecutionBound != definition.ExecutionBound {
			t.Fatalf("binding=%+v definition=%+v", binding, definition)
		}
		delete(definitions, binding.CommandID)
	}
	if len(definitions) != 0 {
		t.Fatalf("missing=%v", definitions)
	}
}

func TestHTTPCommandBoundaryRejectsConcatenatedJSON(t *testing.T) {
	executor := &testExecutor{}
	handler, err := New(Config{Dispatcher: executor, Identity: testIdentity{principal: httpPrincipal(t)}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/commands/orquesta.system.status", strings.NewReader(`{"version":"1","request_ref":"request:http","project_ref":"project:http","payload":{}} {}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || executor.calls != 0 {
		t.Fatalf("code=%d calls=%d", response.Code, executor.calls)
	}
}

func TestHTTPCommandBoundaryRejectsNullOrMissingPayloadBeforeAuthentication(t *testing.T) {
	for _, body := range []string{`null`, `{"version":"1","request_ref":"request:http","project_ref":"project:http"}`} {
		executor := &testExecutor{}
		handler, err := New(Config{Dispatcher: executor, Identity: testIdentity{err: errors.New("must not authenticate")}})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/commands/orquesta.system.status", strings.NewReader(body))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || executor.calls != 0 {
			t.Fatalf("body=%s code=%d calls=%d", body, response.Code, executor.calls)
		}
	}
}

func TestHTTPRejectsUnexpectedExecutionClaimBeforeIdentityOrResolver(t *testing.T) {
	executor := &testExecutor{}
	identityCalls := 0
	handler, err := New(Config{
		Dispatcher: executor,
		Identity:   testIdentity{err: errors.New("must not authenticate"), calls: &identityCalls},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"version":"1","request_ref":"request:http","project_ref":"project:http","claimed_execution_ref":"execution:http","payload":{}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/commands/orquesta.system.status", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || identityCalls != 0 || executor.calls != 0 {
		t.Fatalf("code=%d identity=%d dispatcher=%d", response.Code, identityCalls, executor.calls)
	}
}

func TestHTTPRejectsVersionDriftAndMissingExecutionClaimBeforeAuthority(t *testing.T) {
	for _, test := range []struct {
		path string
		body string
	}{
		{"/api/v1/commands/orquesta.system.status", `{"version":"2","request_ref":"request:http","project_ref":"project:http","payload":{}}`},
		{"/api/v1/commands/orquesta.mailbox.get", `{"version":"1","request_ref":"request:http","project_ref":"project:http","payload":{}}`},
	} {
		executor := &testExecutor{}
		identityCalls := 0
		handler, err := New(Config{
			Dispatcher: executor,
			Identity:   testIdentity{err: errors.New("must not authenticate"), calls: &identityCalls},
		})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)))
		if response.Code != http.StatusBadRequest || identityCalls != 0 || executor.calls != 0 {
			t.Fatalf("path=%s code=%d identity=%d dispatcher=%d", test.path, response.Code, identityCalls, executor.calls)
		}
	}
}
