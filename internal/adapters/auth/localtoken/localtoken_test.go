package localtoken

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
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
	authenticator, err := Open(filepath.Join(t.TempDir(), "auth", "token"))
	if err != nil {
		t.Fatalf("open: %v", err)
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
