package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/i18n"
)

func TestProvisionCodexMicroVMCredentialsCLIUsesCanonicalConfigAndResumesWithoutAuthSource(t *testing.T) {
	root, configPath, authPath := credentialProvisionCLIFixture(t)
	authMaterial := []byte(`{"auth":"cli-material-that-must-not-egress"}`)
	if err := os.WriteFile(authPath, authMaterial, 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := loadCredentialProvisionCatalog(t)
	arguments := credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-e2e")

	var stdout, stderr bytes.Buffer
	code := RunProvisionCodexMicroVMCredentials(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	first := decodeCredentialProvisionOutput(t, stdout.Bytes())
	if code != 0 || stderr.Len() != 0 || first.SchemaVersion != credentialProvisionOutputSchema ||
		first.Status != "complete" || first.PlacementRef != "placement:codex:microvm-runtime" ||
		first.Signing.State != string(credentials.CredentialProvisionCreated) ||
		first.Auth.State != string(credentials.CredentialProvisionCreated) {
		t.Fatalf("first provision code=%d output=%+v stderr=%q", code, first, stderr.String())
	}
	publicKey, err := base64.StdEncoding.DecodeString(first.SigningPublicKeyBase64)
	if err != nil || len(publicKey) != ed25519.PublicKeySize ||
		first.SigningKeyID != "clave-publica:runtime-isolation" {
		t.Fatalf("public signing projection=%+v decode=%v", first, err)
	}

	signingMaterial := readCredentialProvisionMaterial(
		t, root, "credential:microvm-launch-signing", "orquesta.microvm-launch-grant.v1",
		"request:credential-cli-test-read-signing",
	)
	defer clear(signingMaterial)
	authStored := readCredentialProvisionMaterial(
		t, root, "credential:codex-account-runtime", "provider:codex",
		"request:credential-cli-test-read-auth",
	)
	defer clear(authStored)
	if len(signingMaterial) != ed25519.PrivateKeySize ||
		!bytes.Equal(signingMaterial[ed25519.SeedSize:], publicKey) ||
		!bytes.Equal(authStored, authMaterial) {
		t.Fatal("provisioned credential material does not match public/auth authority")
	}
	assertCredentialProvisionOutputExcludes(t, append(stdout.Bytes(), stderr.Bytes()...), authPath,
		filepath.Join(root, "secrets", "credentials.json"), authMaterial, signingMaterial)

	if err := os.Remove(authPath); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = RunProvisionCodexMicroVMCredentials(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	resumed := decodeCredentialProvisionOutput(t, stdout.Bytes())
	if code != 0 || stderr.Len() != 0 || resumed.Status != "complete" ||
		resumed.Signing.State != string(credentials.CredentialProvisionResumed) ||
		resumed.Auth.State != string(credentials.CredentialProvisionResumed) ||
		resumed.SigningPublicKeyBase64 != first.SigningPublicKeyBase64 {
		t.Fatalf("resume code=%d output=%+v stderr=%q", code, resumed, stderr.String())
	}
	assertCredentialProvisionOutputExcludes(t, append(stdout.Bytes(), stderr.Bytes()...), authPath,
		filepath.Join(root, "secrets", "credentials.json"), authMaterial, signingMaterial)
}

func TestProvisionCodexMicroVMCredentialsCLIPartialAuthFailureIsSafeAndRetryable(t *testing.T) {
	root, configPath, authPath := credentialProvisionCLIFixture(t)
	authMaterial := []byte(`{"auth":"partial-secret-never-print"}`)
	if err := os.WriteFile(authPath, authMaterial, 0o644); err != nil {
		t.Fatal(err)
	}
	catalog := loadCredentialProvisionCatalog(t)
	arguments := credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-partial")

	var stdout, stderr bytes.Buffer
	code := RunProvisionCodexMicroVMCredentials(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	partial := decodeCredentialProvisionOutput(t, stdout.Bytes())
	if code != 1 || partial.Status != "partial" || partial.Error == nil ||
		partial.Error.Code != string(credentials.ErrorUnsafeFile) ||
		partial.Signing.State != string(credentials.CredentialProvisionCreated) ||
		partial.Auth.State != string(credentials.CredentialProvisionPending) ||
		!strings.Contains(stderr.String(), "code="+string(credentials.ErrorUnsafeFile)) {
		t.Fatalf("partial code=%d output=%+v stderr=%q", code, partial, stderr.String())
	}
	assertCredentialProvisionOutputExcludes(t, append(stdout.Bytes(), stderr.Bytes()...),
		authPath, filepath.Join(root, "secrets", "credentials.json"), authMaterial)

	if err := os.Chmod(authPath, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = RunProvisionCodexMicroVMCredentials(
		context.Background(), arguments, catalog, &stdout, &stderr,
	)
	completed := decodeCredentialProvisionOutput(t, stdout.Bytes())
	if code != 0 || stderr.Len() != 0 || completed.Status != "complete" ||
		completed.Signing.State != string(credentials.CredentialProvisionResumed) ||
		completed.Auth.State != string(credentials.CredentialProvisionCreated) ||
		completed.SigningPublicKeyBase64 != partial.SigningPublicKeyBase64 {
		t.Fatalf("retry code=%d output=%+v stderr=%q", code, completed, stderr.String())
	}
}

func TestProvisionCodexMicroVMCredentialsCLIRejectsUnsafeAuthLinksWithoutLeaking(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			root, configPath, authPath := credentialProvisionCLIFixture(t)
			material := []byte(`{"auth":"unsafe-link-secret"}`)
			target := filepath.Join(root, "auth-target.json")
			if err := os.WriteFile(target, material, 0o600); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "symlink" {
				err = os.Symlink(target, authPath)
			} else {
				err = os.Link(target, authPath)
			}
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := RunProvisionCodexMicroVMCredentials(
				context.Background(),
				credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-"+kind),
				loadCredentialProvisionCatalog(t), &stdout, &stderr,
			)
			partial := decodeCredentialProvisionOutput(t, stdout.Bytes())
			if code != 1 || partial.Error == nil ||
				partial.Error.Code != string(credentials.ErrorUnsafeFile) ||
				partial.Signing.State != string(credentials.CredentialProvisionCreated) ||
				partial.Auth.State != string(credentials.CredentialProvisionPending) {
				t.Fatalf("%s code=%d output=%+v stderr=%q", kind, code, partial, stderr.String())
			}
			assertCredentialProvisionOutputExcludes(t, append(stdout.Bytes(), stderr.Bytes()...),
				authPath, target, material)
		})
	}
}

func TestProvisionCodexMicroVMCredentialsCLIHasExactArgumentsAndCanonicalLocale(t *testing.T) {
	catalog := loadCredentialProvisionCatalog(t)
	for _, arguments := range [][]string{
		{},
		{"--config", "/tmp/orquesta.toml", "--auth-json-path", "relative/auth.json", "--request-ref", "request:x"},
		{"--config", "/tmp/orquesta.toml", "--auth-json-path", "/tmp/auth.json", "--request-ref", "invalid"},
		{"--config", "/tmp/orquesta.toml", "--auth-json-path", "/tmp/auth.json", "--request-ref", "request:x", "--locale", "en"},
	} {
		var stdout, stderr bytes.Buffer
		code := RunProvisionCodexMicroVMCredentials(
			context.Background(), arguments, catalog, &stdout, &stderr,
		)
		if code != 2 || stdout.Len() != 0 ||
			!strings.Contains(stderr.String(), "code=cli.credential_provision_arguments_invalid") {
			t.Fatalf("arguments=%q code=%d stdout=%q stderr=%q", arguments, code, stdout.String(), stderr.String())
		}
	}

	root, configPath, authPath := credentialProvisionCLIFixture(t)
	if err := os.WriteFile(authPath, []byte(`{"auth":"locale"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "secrets")); err != nil {
		t.Fatal(err)
	}
	replaceTestConfigValue(t, configPath, "[config]\n", "[api]\nlocale = \"en\"\n\n[config]\n")
	var stdout, stderr bytes.Buffer
	code := RunProvisionCodexMicroVMCredentials(
		context.Background(),
		credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-locale"),
		catalog, &stdout, &stderr,
	)
	want, err := catalog.Text("en", "error.cli.credential_provision_failed")
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), want) {
		t.Fatalf("canonical locale code=%d stdout=%q stderr=%q want=%q", code, stdout.String(), stderr.String(), want)
	}
}

func TestProvisionCodexMicroVMCredentialsCLIPreflightAndCancellationDoNotOpenStore(t *testing.T) {
	root := t.TempDir()
	configPath := writeTestConfig(t, root)
	authPath := filepath.Join(root, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"auth":"unused"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := loadCredentialProvisionCatalog(t)
	var stdout, stderr bytes.Buffer
	code := RunProvisionCodexMicroVMCredentials(
		context.Background(),
		credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-process"),
		catalog, &stdout, &stderr,
	)
	if code != 2 || stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), "code=cli.credential_provision_configuration_invalid") {
		t.Fatalf("process preflight code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(root, "secrets")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preflight touched credential store directory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	dependencies := productionCredentialProvisionDependencies()
	dependencies.loadSnapshot = func(context.Context, string) (config.Snapshot, error) {
		called = true
		return config.Snapshot{}, nil
	}
	stdout.Reset()
	stderr.Reset()
	code = runProvisionCodexMicroVMCredentials(
		ctx,
		credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-canceled"),
		catalog, &stdout, &stderr, dependencies,
	)
	if code != 1 || called || stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), "code=cli.credential_provision_canceled") {
		t.Fatalf("cancellation code=%d load=%t stdout=%q stderr=%q", code, called, stdout.String(), stderr.String())
	}
}

func TestProvisionCodexMicroVMCredentialsCLIComposesOneCanonicalAuthorityAndOwner(t *testing.T) {
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(root, "auth.json")
	const uid = 4242
	store := &credentialProvisionCloseProbe{}
	var storeOptions credentiallocal.Options
	var authOwner uint32
	var authMaximum uint64
	var received credentials.ProvisionCodexMicroVMCredentialsRequest
	dependencies := productionCredentialProvisionDependencies()
	dependencies.loadSnapshot = func(context.Context, string) (config.Snapshot, error) {
		return snapshot, nil
	}
	dependencies.effectiveUID = func() int { return uid }
	dependencies.openStore = func(options credentiallocal.Options) (credentialProvisionOwnedStore, error) {
		storeOptions = options
		return store, nil
	}
	dependencies.newAuth = func(
		_ string, owner uint32, maximum uint64,
	) (credentials.MaterialSource, error) {
		authOwner, authMaximum = owner, maximum
		return credentialProvisionMaterialSourceFunc(
			func(context.Context, func(credentials.Secret) error) error { return nil },
		), nil
	}
	dependencies.newKey = func() (credentials.Ed25519PrivateKeySource, error) {
		return credentials.Ed25519PrivateKeySourceFunc(
			func(context.Context, func(credentials.Secret) error) error { return nil },
		), nil
	}
	dependencies.provision = func(
		_ context.Context,
		gotStore credentials.ProvisionStore,
		_ credentials.MaterialSource,
		_ credentials.Ed25519PrivateKeySource,
		request credentials.ProvisionCodexMicroVMCredentialsRequest,
	) (credentials.ProvisionCodexMicroVMCredentialsResult, error) {
		if gotStore != store {
			t.Fatalf("use case received another credential authority: %T", gotStore)
		}
		received = request
		return credentialProvisionTestProjection(), nil
	}

	var stdout, stderr bytes.Buffer
	code := runProvisionCodexMicroVMCredentials(
		context.Background(),
		credentialProvisionCLIArguments(configPath, authPath, "request:canonical-authority"),
		loadCredentialProvisionCatalog(t), &stdout, &stderr, dependencies,
	)
	if code != 0 || stderr.Len() != 0 || store.closes != 1 ||
		storeOptions.Path != snapshot.CredentialsLocalPath() || storeOptions.OwnerUID != uid ||
		storeOptions.MaxStoreBytes != snapshot.CredentialsLocalMaxDocumentBytes() ||
		authOwner != uid || authMaximum != uint64(snapshot.RuntimeCodexAccountAuthMaxDocumentBytes()) {
		t.Fatalf("composition code=%d close=%d store=%+v auth_owner=%d auth_max=%d stderr=%q",
			code, store.closes, storeOptions, authOwner, authMaximum, stderr.String())
	}
	if received.ActorRef != snapshot.IdentityLocalActor() ||
		received.OwnerRef.String() != snapshot.IdentityLocalActor() ||
		received.SigningKeyID != snapshot.RuntimeMicroVMLaunchGrantKeyID() ||
		received.Signing.CredentialRef.String() != string(snapshot.RuntimeMicroVMLaunchGrantSigningCredentialRef()) ||
		received.Auth.CredentialRef.String() != string(snapshot.RuntimeCodexCredentialRef()) ||
		len(received.Signing.ScopeRefs) != 1 || len(received.Auth.ScopeRefs) != 1 ||
		received.Signing.ScopeRefs[0].String() != snapshot.ProjectDefault() ||
		received.Auth.ScopeRefs[0].String() != snapshot.ProjectDefault() ||
		received.Signing.PurposeRef.String() != "orquesta.microvm-launch-grant.v1" ||
		received.Auth.PurposeRef.String() != "provider:codex" ||
		received.Signing.RequestRef != "request:canonical-authority:signing" ||
		received.Auth.RequestRef != "request:canonical-authority:auth" {
		t.Fatalf("noncanonical provision request=%+v", received)
	}
}

func TestProvisionWithOwnedCredentialStoreClosesExactlyOnceAndSanitizesFailure(t *testing.T) {
	projection := credentialProvisionTestProjection()
	for _, test := range []struct {
		name       string
		closeError error
		closePanic any
		panicValue any
		useError   error
		wantError  error
	}{
		{name: "success"},
		{name: "use error", useError: credentials.WrapError(credentials.ErrorStoreIO, "source", errors.New("private/path"))},
		{name: "use panic", panicValue: "private panic", wantError: errCredentialProvisionPanic},
		{name: "close error", closeError: errors.New("private close"), wantError: errCredentialProvisionClose},
		{name: "close panic", closePanic: "private close panic", wantError: errCredentialProvisionClose},
		{name: "close wins over use error", closeError: errors.New("private close"), useError: errors.New("private use"), wantError: errCredentialProvisionClose},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &credentialProvisionCloseProbe{closeErr: test.closeError, closePanic: test.closePanic}
			result, err := provisionWithOwnedCredentialStore(
				context.Background(), store,
				credentialProvisionMaterialSourceFunc(func(context.Context, func(credentials.Secret) error) error { return nil }),
				credentials.Ed25519PrivateKeySourceFunc(func(context.Context, func(credentials.Secret) error) error { return nil }),
				credentials.ProvisionCodexMicroVMCredentialsRequest{},
				func(
					context.Context, credentials.ProvisionStore, credentials.MaterialSource,
					credentials.Ed25519PrivateKeySource, credentials.ProvisionCodexMicroVMCredentialsRequest,
				) (credentials.ProvisionCodexMicroVMCredentialsResult, error) {
					if test.panicValue != nil {
						panic(test.panicValue)
					}
					return projection, test.useError
				},
			)
			if store.closes != 1 {
				t.Fatalf("Close calls=%d", store.closes)
			}
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("error=%v want=%v", err, test.wantError)
				}
			} else if !errors.Is(err, test.useError) {
				t.Fatalf("error=%v want use error=%v", err, test.useError)
			}
			if test.panicValue == nil && result.SigningKeyID != projection.SigningKeyID {
				t.Fatalf("projection lost on owned close: %+v", result)
			}
			if got := projectCredentialProvisionError(err); err != nil &&
				(strings.Contains(got.Code, "private") || strings.Contains(got.Field, "private")) {
				t.Fatalf("unsafe error projection=%+v", got)
			}
		})
	}
}

func TestProvisionCodexMicroVMCredentialsCLIRejectsUnconfirmedOutputAfterClosingStore(t *testing.T) {
	root, configPath, authPath := credentialProvisionCLIFixture(t)
	secret := []byte(`{"auth":"writer-failure-secret"}`)
	if err := os.WriteFile(authPath, secret, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := loadConfigSnapshot(context.Background(), configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		provisionErr error
	}{
		{name: "success"},
		{name: "partial", provisionErr: credentials.WrapError(credentials.ErrorStoreIO, "source", errors.New("private/path"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &credentialProvisionCloseProbe{}
			dependencies := productionCredentialProvisionDependencies()
			dependencies.loadSnapshot = func(context.Context, string) (config.Snapshot, error) {
				return snapshot, nil
			}
			dependencies.openStore = func(credentiallocal.Options) (credentialProvisionOwnedStore, error) {
				return store, nil
			}
			dependencies.provision = func(
				context.Context,
				credentials.ProvisionStore,
				credentials.MaterialSource,
				credentials.Ed25519PrivateKeySource,
				credentials.ProvisionCodexMicroVMCredentialsRequest,
			) (credentials.ProvisionCodexMicroVMCredentialsResult, error) {
				return credentialProvisionTestProjection(), test.provisionErr
			}
			closedBeforeWrite := false
			var attemptedOutput []byte
			stdout := credentialProvisionWriterFunc(func(content []byte) (int, error) {
				closedBeforeWrite = store.closes == 1
				attemptedOutput = append(attemptedOutput, content...)
				return 0, errors.New("private writer failure")
			})
			var stderr bytes.Buffer
			code := runProvisionCodexMicroVMCredentials(
				context.Background(),
				credentialProvisionCLIArguments(configPath, authPath, "request:credential-cli-writer-"+test.name),
				loadCredentialProvisionCatalog(t), stdout, &stderr, dependencies,
			)
			if code != 1 || store.closes != 1 || !closedBeforeWrite || len(attemptedOutput) == 0 ||
				!strings.Contains(stderr.String(), "code=cli.output_invalid") {
				t.Fatalf("code=%d closes=%d closed_before_write=%t output=%q stderr=%q",
					code, store.closes, closedBeforeWrite, attemptedOutput, stderr.String())
			}
			assertCredentialProvisionOutputExcludes(
				t, append(attemptedOutput, stderr.Bytes()...), authPath,
				filepath.Join(root, "secrets", "credentials.json"), secret,
			)
		})
	}
}

type credentialProvisionCloseProbe struct {
	credentials.ProvisionStore
	closes     int
	closeErr   error
	closePanic any
}

func (store *credentialProvisionCloseProbe) Close() error {
	store.closes++
	if store.closePanic != nil {
		panic(store.closePanic)
	}
	return store.closeErr
}

type credentialProvisionWriterFunc func([]byte) (int, error)

func (function credentialProvisionWriterFunc) Write(content []byte) (int, error) {
	return function(content)
}

type credentialProvisionMaterialSourceFunc func(context.Context, func(credentials.Secret) error) error

func (function credentialProvisionMaterialSourceFunc) WithSecret(
	ctx context.Context,
	consume func(credentials.Secret) error,
) error {
	return function(ctx, consume)
}

func credentialProvisionCLIFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	configPath, _, _ := writeRuntimeIsolationMicroVMConfig(t, root, false)
	if err := os.Mkdir(filepath.Join(root, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, configPath, filepath.Join(root, "source-auth.json")
}

func credentialProvisionCLIArguments(configPath, authPath, requestRef string) []string {
	return []string{
		"--config", configPath,
		"--auth-json-path", authPath,
		"--request-ref", requestRef,
	}
}

func loadCredentialProvisionCatalog(t *testing.T) *i18n.Catalog {
	t.Helper()
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func decodeCredentialProvisionOutput(t *testing.T, content []byte) credentialProvisionOutput {
	t.Helper()
	var output credentialProvisionOutput
	if !json.Valid(content) {
		t.Fatalf("invalid JSON output=%q", content)
	}
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func readCredentialProvisionMaterial(
	t *testing.T,
	root string,
	credentialRef credentials.CredentialRef,
	purpose credentials.PurposeRef,
	requestRef string,
) []byte {
	t.Helper()
	store, err := credentiallocal.Open(credentiallocal.Options{
		Path: filepath.Join(root, "secrets", "credentials.json"), OwnerUID: os.Geteuid(),
		MaxStoreBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	var material []byte
	_, useErr := store.Use(context.Background(), credentials.UseRequest{
		ActorRef: "actor:local-owner", RequestRef: requestRef,
		CredentialRef: credentialRef, OwnerRef: "actor:local-owner",
		ScopeRef: "project:default", PurposeRef: purpose, Version: 1,
	}, func(secret credentials.Secret) error {
		defer secret.Destroy()
		material = secret.Bytes()
		return nil
	})
	closeErr := store.Close()
	if useErr != nil || closeErr != nil {
		clear(material)
		t.Fatalf("read credential use=%v close=%v", useErr, closeErr)
	}
	return material
}

func assertCredentialProvisionOutputExcludes(
	t *testing.T,
	projection []byte,
	pathsAndMaterial ...any,
) {
	t.Helper()
	for _, value := range pathsAndMaterial {
		var material []byte
		switch typed := value.(type) {
		case string:
			material = []byte(typed)
		case []byte:
			material = typed
		default:
			t.Fatalf("unsupported secret test value %T", value)
		}
		for _, signature := range [][]byte{
			material,
			[]byte(base64.StdEncoding.EncodeToString(material)),
			[]byte(base64.RawURLEncoding.EncodeToString(material)),
			[]byte(hex.EncodeToString(material)),
		} {
			if len(signature) > 0 && bytes.Contains(projection, signature) {
				t.Fatalf("credential projection contains forbidden value")
			}
		}
	}
}

func credentialProvisionTestProjection() credentials.ProvisionCodexMicroVMCredentialsResult {
	at := time.Unix(1_700_000_000, 0).UTC()
	return credentials.ProvisionCodexMicroVMCredentialsResult{
		Signing: credentials.CredentialProvisionStatus{
			State: credentials.CredentialProvisionCreated,
			RequestedAuthority: credentials.RequestedCredentialAuthority{
				CredentialRef: "credential:signing", OwnerRef: "actor:owner",
				ScopeRefs:  []credentials.ScopeRef{"project:default"},
				PurposeRef: "orquesta.microvm-launch-grant.v1", Version: 1,
			},
			Receipts: []credentials.Receipt{{
				CredentialRef: "credential:signing", OwnerRef: "actor:owner",
				PurposeRef: "orquesta.microvm-launch-grant.v1", Version: 1,
				RequestRef: "request:test", ActorRef: "actor:owner", Operation: "create", OccurredAt: at,
			}},
		},
		Auth: credentials.CredentialProvisionStatus{
			State: credentials.CredentialProvisionCreated,
			RequestedAuthority: credentials.RequestedCredentialAuthority{
				CredentialRef: "credential:auth", OwnerRef: "actor:owner",
				ScopeRefs:  []credentials.ScopeRef{"project:default"},
				PurposeRef: "provider:codex", Version: 1,
			},
		},
		SigningKeyID: "clave-publica:test", SigningPublicKeyBase64: "cHVibGlj",
	}
}
