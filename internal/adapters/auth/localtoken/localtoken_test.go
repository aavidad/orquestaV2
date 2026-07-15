package localtoken

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
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

func TestIdentityProviderAuthenticatesOnlyDetachedExactCredential(t *testing.T) {
	credentialFile, err := Open(filepath.Join(t.TempDir(), "auth", "token"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	want := testPrincipal(t, "provider")
	provider, err := credentialFile.ForPrincipal(want)
	if err != nil {
		t.Fatalf("ForPrincipal: %v", err)
	}
	if provider.AuthenticationMethod() != identity.AuthenticationMethod(AuthenticationMethod) {
		t.Fatalf("AuthenticationMethod = %q", provider.AuthenticationMethod())
	}
	if _, err := credentialFile.ForPrincipal(identity.Principal{}); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("invalid principal error = %v", err)
	}
	var nilCredential *Authenticator
	if _, err := nilCredential.ForPrincipal(want); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("nil credential error = %v", err)
	}
	if _, err := provider.ForPrincipal(testPrincipal(t, "replacement")); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("principal replacement error = %v", err)
	}
	credential, err := identity.NewCredential([]byte(provider.Token()))
	if err != nil {
		t.Fatalf("NewCredential: %v", err)
	}
	got, err := provider.Authenticate(context.Background(), credential)
	if err != nil || got != want {
		t.Fatalf("Authenticate = %+v, %v", got, err)
	}
	for _, raw := range []string{"wrong", provider.Token() + "\n", "Bearer " + provider.Token()} {
		candidate, candidateErr := identity.NewCredential([]byte(raw))
		if candidateErr != nil {
			t.Fatalf("NewCredential(%q): %v", raw, candidateErr)
		}
		if _, err := provider.Authenticate(context.Background(), candidate); !IsError(err, CodeAuthenticationFailed) {
			t.Errorf("Authenticate(%q) error = %v", raw, err)
		}
	}
	if _, err := provider.Authenticate(nil, credential); !IsError(err, CodeContextInvalid) {
		t.Fatalf("nil context error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Authenticate(cancelled, credential); !IsError(err, CodeContextInvalid) {
		t.Fatalf("cancelled context error = %v", err)
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
