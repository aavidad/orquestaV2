package localtoken

import (
	"context"
	"encoding/base64"
	"fmt"
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

func TestManifestAuthenticatesDistinctExplicitHumanPrincipalsAndCombinesWithLegacy(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "principals.json")
	for _, name := range []string{"alice.token", "bob.token"} {
		if _, err := Open(filepath.Join(root, name)); err != nil {
			t.Fatalf("provision %s: %v", name, err)
		}
	}
	mustWrite(t, manifestPath, manifestJSON(
		manifestEntry{PrincipalRef: "actor:alice", ActorRef: "actor:alice", TokenPath: "alice.token"},
		manifestEntry{PrincipalRef: "actor:bob", ActorRef: "actor:bob", TokenPath: "bob.token"},
	), 0o600)
	secondaries, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
	})
	if err != nil {
		t.Fatalf("OpenManifest: %v", err)
	}
	ownerCredential, err := Open(filepath.Join(root, "owner.token"))
	if err != nil {
		t.Fatal(err)
	}
	ownerProvider, err := ownerCredential.ForPrincipal(testPrincipal(t, "owner"))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := Combine(ownerProvider, secondaries)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ tokenPath, principalRef, actorRef string }{
		{"owner.token", "principal:owner", "actor:owner"},
		{"alice.token", "actor:alice", "actor:alice"},
		{"bob.token", "actor:bob", "actor:bob"},
	} {
		token, readErr := os.ReadFile(filepath.Join(root, test.tokenPath))
		if readErr != nil {
			t.Fatal(readErr)
		}
		credential, credentialErr := identity.NewCredential(token)
		if credentialErr != nil {
			t.Fatal(credentialErr)
		}
		principal, authenticateErr := provider.Authenticate(context.Background(), credential)
		if authenticateErr != nil || principal.Ref.String() != test.principalRef ||
			principal.ActorRef.String() != test.actorRef || principal.Kind != identity.PrincipalKindHuman {
			t.Fatalf("%s authenticated as %+v err=%v", test.tokenPath, principal, authenticateErr)
		}
	}
	alice, _ := os.ReadFile(filepath.Join(root, "alice.token"))
	bob, _ := os.ReadFile(filepath.Join(root, "bob.token"))
	if string(alice) == string(bob) || string(alice) == ownerCredential.Token() ||
		string(bob) == ownerCredential.Token() {
		t.Fatal("local principals share credential material")
	}
	for _, raw := range []string{"unknown", "orqex1.ZXhlY3V0aW9uOmZha2U.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		credential, _ := identity.NewCredential([]byte(raw))
		if _, err := provider.Authenticate(context.Background(), credential); !IsError(err, CodeAuthenticationFailed) {
			t.Fatalf("reserved/unknown credential escaped default deny: %v", err)
		}
	}
}

func TestManifestRejectsUnsafeFilesPathsDuplicatesAndUnknownShape(t *testing.T) {
	valid := manifestEntry{PrincipalRef: "actor:alice", ActorRef: "actor:alice", TokenPath: "alice.token"}
	tests := []struct {
		name  string
		setup func(*testing.T, string) (string, int)
	}{
		{name: "symlink manifest", setup: func(t *testing.T, root string) (string, int) {
			target := filepath.Join(root, "target.json")
			mustWrite(t, target, manifestJSON(valid), 0o600)
			path := filepath.Join(root, "principals.json")
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			return path, os.Geteuid()
		}},
		{name: "hardlink manifest", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			mustWrite(t, path, manifestJSON(valid), 0o600)
			if err := os.Link(path, filepath.Join(root, "other.json")); err != nil {
				t.Fatal(err)
			}
			return path, os.Geteuid()
		}},
		{name: "wrong owner contract", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			mustWrite(t, path, manifestJSON(valid), 0o600)
			return path, os.Geteuid() + 1
		}},
		{name: "parent traversal", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			entry := valid
			entry.TokenPath = "../alice.token"
			mustWrite(t, path, manifestJSON(entry), 0o600)
			return path, os.Geteuid()
		}},
		{name: "duplicate principal", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			other := valid
			other.ActorRef, other.TokenPath = "actor:bob", "bob.token"
			mustWrite(t, path, manifestJSON(valid, other), 0o600)
			return path, os.Geteuid()
		}},
		{name: "principal actor mismatch", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			entry := valid
			entry.PrincipalRef = "principal:alice"
			mustWrite(t, path, manifestJSON(entry), 0o600)
			return path, os.Geteuid()
		}},
		{name: "unknown field", setup: func(t *testing.T, root string) (string, int) {
			path := filepath.Join(root, "principals.json")
			mustWrite(t, path, `{"schema_version":1,"document_type":"orquesta.local_token_principals","principals":[],"role":"project_owner"}`, 0o600)
			return path, os.Geteuid()
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			path, owner := test.setup(t, root)
			if _, err := OpenManifest(ManifestOptions{
				Path: path, OwnerUID: owner, MaxDocumentBytes: 4096, MaxEntries: 8,
			}); err == nil {
				t.Fatal("unsafe manifest accepted")
			}
		})
	}
}

func TestManifestRejectsHardlinkedOrSharedTokensAndPrimaryIdentityDuplicates(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString(make([]byte, tokenByteSize))
	firstPath, secondPath := filepath.Join(root, "alice.token"), filepath.Join(root, "bob.token")
	mustWrite(t, firstPath, token, 0o600)
	if err := os.Link(firstPath, secondPath); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "principals.json")
	mustWrite(t, manifestPath, manifestJSON(
		manifestEntry{PrincipalRef: "actor:alice", ActorRef: "actor:alice", TokenPath: "alice.token"},
		manifestEntry{PrincipalRef: "actor:bob", ActorRef: "actor:bob", TokenPath: "bob.token"},
	), 0o600)
	if _, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
	}); !IsError(err, CodeTokenFileInvalid) {
		t.Fatalf("hardlinked token error=%v", err)
	}

	if err := os.Remove(secondPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(firstPath); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{firstPath, secondPath} {
		if _, err := Open(path); err != nil {
			t.Fatalf("provision %s: %v", path, err)
		}
	}
	secondaries, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	primaryFile, err := Open(filepath.Join(root, "primary.token"))
	if err != nil {
		t.Fatal(err)
	}
	primary, err := primaryFile.ForPrincipal(testPrincipal(t, "alice"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Combine(primary, secondaries); !IsError(err, CodePrincipalInvalid) {
		t.Fatalf("duplicate primary identity accepted: %v", err)
	}
}

func TestManifestRejectsReservedPrimaryIdentityBeforeReadingSecondaryTokens(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "principals.json")
	mustWrite(t, manifestPath, manifestJSON(
		manifestEntry{PrincipalRef: "actor:owner", ActorRef: "actor:owner", TokenPath: "owner-copy.token"},
		manifestEntry{PrincipalRef: "actor:bob", ActorRef: "actor:bob", TokenPath: "bob.token"},
	), 0o600)
	primary := testPrincipal(t, "owner")
	if _, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
		Reserved: []identity.Principal{primary},
	}); !IsError(err, CodeManifestInvalid) {
		t.Fatalf("reserved primary identity accepted: %v", err)
	}
	for _, name := range []string{"owner-copy.token", "bob.token"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("reserved identity read or created %s: %v", name, err)
		}
	}
}

func TestManifestCapacityAndLateInvalidEntryFailBeforeCreatingAnyToken(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	first := manifestEntry{PrincipalRef: "actor:first", ActorRef: "actor:first", TokenPath: "first.token"}
	second := manifestEntry{PrincipalRef: "actor:second", ActorRef: "actor:second", TokenPath: "second.token"}
	manifestPath := filepath.Join(root, "principals.json")
	mustWrite(t, manifestPath, manifestJSON(first, second), 0o600)
	if _, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 1,
	}); !IsError(err, CodeManifestInvalid) {
		t.Fatalf("entry limit error=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "first.token")); !os.IsNotExist(err) {
		t.Fatalf("capacity failure created token: %v", err)
	}

	second.PrincipalRef = "principal:second"
	mustWrite(t, manifestPath, manifestJSON(first, second), 0o600)
	if _, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
	}); !IsError(err, CodeManifestInvalid) {
		t.Fatalf("late invalid entry error=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "first.token")); !os.IsNotExist(err) {
		t.Fatalf("late validation failure created earlier token: %v", err)
	}
}

func TestManifestRequiresPreprovisionedTokenWithoutCreatingIt(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(root, "missing.token")
	manifestPath := filepath.Join(root, "principals.json")
	mustWrite(t, manifestPath, manifestJSON(
		manifestEntry{PrincipalRef: "actor:missing", ActorRef: "actor:missing", TokenPath: "missing.token"},
	), 0o600)
	if _, err := OpenManifest(ManifestOptions{
		Path: manifestPath, OwnerUID: os.Geteuid(), MaxDocumentBytes: 4096, MaxEntries: 8,
	}); !IsError(err, CodeTokenFileInvalid) {
		t.Fatalf("missing secondary token error=%v", err)
	}
	if _, err := os.Lstat(tokenPath); !os.IsNotExist(err) {
		t.Fatalf("startup created missing secondary token: %v", err)
	}
}

func manifestJSON(entries ...manifestEntry) string {
	var principals strings.Builder
	for index, entry := range entries {
		if index > 0 {
			principals.WriteByte(',')
		}
		fmt.Fprintf(&principals, `{"principal_ref":%q,"actor_ref":%q,"token_path":%q}`,
			entry.PrincipalRef, entry.ActorRef, entry.TokenPath)
	}
	return fmt.Sprintf(`{"schema_version":1,"document_type":"orquesta.local_token_principals","principals":[%s]}`,
		principals.String())
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
