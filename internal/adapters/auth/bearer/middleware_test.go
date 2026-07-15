package bearer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type provider struct {
	want      string
	principal identity.Principal
	calls     atomic.Int64
}

func (*provider) AuthenticationMethod() identity.AuthenticationMethod { return "test" }

func (source *provider) Authenticate(_ context.Context, credential identity.Credential) (identity.Principal, error) {
	source.calls.Add(1)
	valid := false
	err := credential.Use(func(material []byte) error {
		valid = string(material) == source.want
		return nil
	})
	if err != nil || !valid {
		return identity.Principal{}, errors.New("test.authentication_failed")
	}
	return source.principal, nil
}

func TestMiddlewareAuthenticatesOnceAndBindsOnlyReturnedPrincipal(t *testing.T) {
	want := testPrincipal(t, "owner")
	source := &provider{want: "V11_BEARER_SECRET", principal: want}
	middleware, err := New(source)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var reached atomic.Int64
	handler := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reached.Add(1)
		principal, err := identity.PrincipalFromContext(request.Context())
		if err != nil || principal != want {
			t.Errorf("principal = %+v, %v", principal, err)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+source.want)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || source.calls.Load() != 1 || reached.Load() != 1 {
		t.Fatalf("status=%d calls=%d reached=%d", response.Code, source.calls.Load(), reached.Load())
	}
	if _, err := identity.PrincipalFromContext(request.Context()); err == nil {
		t.Fatal("middleware mutated original request")
	}
}

func TestMiddlewareRejectsMalformedWrongDuplicateAndPreboundRequestsWithoutLeak(t *testing.T) {
	source := &provider{want: "V11_BEARER_SECRET", principal: testPrincipal(t, "owner")}
	middleware, err := New(source)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var reached atomic.Int64
	handler := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached.Add(1) }))
	tests := []struct {
		name    string
		headers []string
	}{
		{name: "missing"},
		{name: "scheme", headers: []string{"Basic x"}},
		{name: "empty", headers: []string{"Bearer "}},
		{name: "space", headers: []string{"Bearer wrong value"}},
		{name: "wrong", headers: []string{"Bearer wrong"}},
		{name: "duplicate", headers: []string{"Bearer V11_BEARER_SECRET", "Bearer V11_BEARER_SECRET"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
			for _, header := range test.headers {
				request.Header.Add("Authorization", header)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("status=%d challenge=%q", response.Code, response.Header().Get("WWW-Authenticate"))
			}
			if strings.Contains(response.Body.String(), source.want) {
				t.Fatal("response leaked credential")
			}
		})
	}
	prebound, err := identity.BindPrincipal(context.Background(), testPrincipal(t, "spoof"))
	if err != nil {
		t.Fatalf("BindPrincipal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil).WithContext(prebound)
	request.Header.Set("Authorization", "Bearer "+source.want)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || reached.Load() != 0 {
		t.Fatalf("prebound status=%d reached=%d", response.Code, reached.Load())
	}
}

func TestMiddlewareRequiresProviderAndHandler(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("nil provider accepted")
	}
	var middleware *Middleware
	request := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	response := httptest.NewRecorder()
	middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil middleware status=%d", response.Code)
	}
	source := &provider{want: "secret", principal: testPrincipal(t, "owner")}
	valid, err := New(source)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	request = httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response = httptest.NewRecorder()
	valid.Wrap(nil).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil next status=%d", response.Code)
	}
}

func testPrincipal(t *testing.T, suffix string) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:" + suffix)
	if err != nil {
		t.Fatalf("NewPrincipalRef: %v", err)
	}
	actorRef, err := goal.NewActorRef("actor:" + suffix)
	if err != nil {
		t.Fatalf("NewActorRef: %v", err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "test")
	if err != nil {
		t.Fatalf("NewPrincipal: %v", err)
	}
	return principal
}
