package localtoken

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestOpenCreatesPrivateTokenAndReusesItAfterRestart(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "auth", "token")
	first, err := Open(tokenPath)
	if err != nil {
		t.Fatalf("open first: %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first.Token())
	if err != nil || len(decoded) != tokenByteSize {
		t.Fatalf("token is not canonical random material: length=%d err=%v", len(decoded), err)
	}
	directoryInfo, err := os.Lstat(filepath.Dir(tokenPath))
	if err != nil || !directoryInfo.IsDir() || directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %v err=%v", directoryInfo, err)
	}
	fileInfo, err := os.Lstat(tokenPath)
	if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v err=%v", fileInfo, err)
	}
	content, err := os.ReadFile(tokenPath)
	if err != nil || string(content) != first.Token() {
		t.Fatalf("persisted token mismatch: err=%v", err)
	}

	second, err := Open(tokenPath)
	if err != nil {
		t.Fatalf("open after restart: %v", err)
	}
	if second.Token() != first.Token() {
		t.Fatal("restart rotated valid token")
	}
}

func TestOpenRejectsWeakPermissionsSymlinksNonRegularAndInvalidTokens(t *testing.T) {
	validToken := base64.RawURLEncoding.EncodeToString(make([]byte, tokenByteSize))
	tests := []struct {
		name string
		want ErrorCode
		set  func(*testing.T, string) string
	}{
		{
			name: "weak directory",
			want: CodeDirectoryPermissions,
			set: func(t *testing.T, root string) string {
				directory := filepath.Join(root, "auth")
				mustMkdir(t, directory, 0o750)
				return filepath.Join(directory, "token")
			},
		},
		{
			name: "symlink directory ancestor",
			want: CodeDirectoryInvalid,
			set: func(t *testing.T, root string) string {
				realDirectory := filepath.Join(root, "real")
				mustMkdir(t, realDirectory, 0o700)
				linkedDirectory := filepath.Join(root, "linked")
				if err := os.Symlink(realDirectory, linkedDirectory); err != nil {
					t.Fatalf("symlink directory: %v", err)
				}
				return filepath.Join(linkedDirectory, "nested", "token")
			},
		},
		{
			name: "weak token file",
			want: CodeTokenFilePermissions,
			set: func(t *testing.T, root string) string {
				directory := filepath.Join(root, "auth")
				mustMkdir(t, directory, 0o700)
				path := filepath.Join(directory, "token")
				mustWrite(t, path, validToken, 0o644)
				return path
			},
		},
		{
			name: "symlink token file",
			want: CodeTokenFileInvalid,
			set: func(t *testing.T, root string) string {
				directory := filepath.Join(root, "auth")
				mustMkdir(t, directory, 0o700)
				target := filepath.Join(root, "target")
				mustWrite(t, target, validToken, 0o600)
				path := filepath.Join(directory, "token")
				if err := os.Symlink(target, path); err != nil {
					t.Fatalf("symlink token: %v", err)
				}
				return path
			},
		},
		{
			name: "non regular token file",
			want: CodeTokenFileInvalid,
			set: func(t *testing.T, root string) string {
				directory := filepath.Join(root, "auth")
				mustMkdir(t, directory, 0o700)
				path := filepath.Join(directory, "token")
				mustMkdir(t, path, 0o700)
				return path
			},
		},
		{
			name: "invalid token material",
			want: CodeTokenInvalid,
			set: func(t *testing.T, root string) string {
				directory := filepath.Join(root, "auth")
				mustMkdir(t, directory, 0o700)
				path := filepath.Join(directory, "token")
				mustWrite(t, path, strings.Repeat("x", base64.RawURLEncoding.EncodedLen(tokenByteSize)), 0o600)
				return path
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.set(t, t.TempDir())
			_, err := Open(path)
			if !IsError(err, test.want) {
				t.Fatalf("Open error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestMiddlewareRequiresOneValidBearerTokenWithoutLeakingIt(t *testing.T) {
	credential, err := Open(filepath.Join(t.TempDir(), "auth", "token"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	authenticator, err := credential.ForPrincipal(testPrincipal(t, "owner"))
	if err != nil {
		t.Fatalf("ForPrincipal: %v", err)
	}
	var accepted atomic.Int64
	handler := authenticator.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		accepted.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name    string
		headers []string
	}{
		{name: "missing"},
		{name: "wrong scheme", headers: []string{"Basic abc"}},
		{name: "missing credential", headers: []string{"Bearer "}},
		{name: "wrong credential", headers: []string{"Bearer wrong"}},
		{name: "extra field", headers: []string{"Bearer wrong token"}},
		{name: "duplicate", headers: []string{"Bearer " + authenticator.Token(), "Bearer " + authenticator.Token()}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
			for _, header := range test.headers {
				request.Header.Add("Authorization", header)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", response.Code)
			}
			if response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("challenge = %q", response.Header().Get("WWW-Authenticate"))
			}
			if strings.Contains(response.Body.String(), authenticator.Token()) {
				t.Fatal("response leaked token")
			}
		})
	}
	if accepted.Load() != 0 {
		t.Fatalf("unauthorized requests reached handler: %d", accepted.Load())
	}

	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+authenticator.Token())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || accepted.Load() != 1 {
		t.Fatalf("valid request: status=%d accepted=%d", response.Code, accepted.Load())
	}
}

func TestMiddlewareFailsClosedUntilCredentialHasExplicitValidPrincipal(t *testing.T) {
	credential, err := Open(filepath.Join(t.TempDir(), "auth", "token"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	var reached atomic.Int64
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached.Add(1) })

	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+credential.Token())
	response := httptest.NewRecorder()
	credential.Middleware(next).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || reached.Load() != 0 {
		t.Fatalf("unbound middleware status=%d reached=%d", response.Code, reached.Load())
	}

	if _, err := credential.ForPrincipal(identity.Principal{}); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("invalid principal error = %v", err)
	}
	var nilCredential *Authenticator
	if _, err := nilCredential.ForPrincipal(testPrincipal(t, "nil")); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("nil credential error = %v", err)
	}

	bound, err := credential.ForPrincipal(testPrincipal(t, "bound"))
	if err != nil {
		t.Fatalf("ForPrincipal(valid): %v", err)
	}
	if _, err := bound.ForPrincipal(testPrincipal(t, "replacement")); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("principal replacement error = %v", err)
	}
}

func TestMiddlewareBindsOnlyValidRequestAndRejectsPreboundContext(t *testing.T) {
	credential, err := Open(filepath.Join(t.TempDir(), "auth", "token"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	want := testPrincipal(t, "owner")
	authenticator, err := credential.ForPrincipal(want)
	if err != nil {
		t.Fatalf("ForPrincipal: %v", err)
	}
	var reached atomic.Int64
	handler := authenticator.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reached.Add(1)
		got, principalErr := identity.PrincipalFromContext(request.Context())
		if principalErr != nil || got != want {
			t.Errorf("request principal = %+v, %v; want %+v", got, principalErr, want)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	for _, header := range []string{"", "Bearer wrong"} {
		request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("invalid credential status = %d", response.Code)
		}
	}
	if reached.Load() != 0 {
		t.Fatalf("invalid requests reached next: %d", reached.Load())
	}

	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+authenticator.Token())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || reached.Load() != 1 {
		t.Fatalf("valid request status=%d reached=%d", response.Code, reached.Load())
	}
	if _, err := identity.PrincipalFromContext(request.Context()); err == nil {
		t.Fatal("middleware mutated original request context")
	}

	spoof := testPrincipal(t, "spoof")
	spoofedContext, err := identity.BindPrincipal(context.Background(), spoof)
	if err != nil {
		t.Fatalf("BindPrincipal(spoof): %v", err)
	}
	request = httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil).WithContext(spoofedContext)
	request.Header.Set("Authorization", "Bearer "+authenticator.Token())
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || reached.Load() != 1 {
		t.Fatalf("prebound request status=%d reached=%d", response.Code, reached.Load())
	}
}

func TestConcurrentAuthenticatorsNeverMixRequestPrincipals(t *testing.T) {
	type fixture struct {
		authenticator *Authenticator
		principal     identity.Principal
	}
	fixtures := make([]fixture, 0, 2)
	for _, suffix := range []string{"alpha", "beta"} {
		credential, err := Open(filepath.Join(t.TempDir(), suffix, "token"))
		if err != nil {
			t.Fatalf("Open(%s): %v", suffix, err)
		}
		principal := testPrincipal(t, suffix)
		authenticator, err := credential.ForPrincipal(principal)
		if err != nil {
			t.Fatalf("ForPrincipal(%s): %v", suffix, err)
		}
		fixtures = append(fixtures, fixture{authenticator: authenticator, principal: principal})
	}

	var wait sync.WaitGroup
	var mismatches atomic.Int64
	for _, current := range fixtures {
		current := current
		handler := current.authenticator.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			principal, err := identity.PrincipalFromContext(request.Context())
			if err != nil || principal != current.principal {
				mismatches.Add(1)
			}
			writer.WriteHeader(http.StatusNoContent)
		}))
		for range 64 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
				request.Header.Set("Authorization", "Bearer "+current.authenticator.Token())
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusNoContent {
					mismatches.Add(1)
				}
			}()
		}
	}
	wait.Wait()
	if mismatches.Load() != 0 {
		t.Fatalf("mixed principal observations = %d", mismatches.Load())
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
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKindHuman, "local_token")
	if err != nil {
		t.Fatalf("NewPrincipal: %v", err)
	}
	return principal
}

func mustMkdir(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Mkdir(path, mode); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}
