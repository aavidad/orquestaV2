package local

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orquesta/internal/credentials"
)

func TestProvisionCodexMicroVMCredentialsRecoversLostCreateResponseAfterReopen(t *testing.T) {
	seed := bytes.Repeat([]byte{0x46}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	authMaterial := []byte("local-reopen-auth-material")
	t.Cleanup(func() { clear(seed); clear(privateKey); clear(authMaterial) })
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "credentials.json")
	open := func() *Store {
		store, err := Open(Options{
			Path: path, OwnerUID: os.Geteuid(), MaxStoreBytes: 1 << 20,
			Now: func() time.Time { return time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC) },
		})
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	request := provisionIntegrationRequest()
	firstStore := open()
	lostResponse := &provisionLostCreateResponseStore{
		ProvisionStore: firstStore,
		credentialRef:  request.Signing.CredentialRef,
	}
	firstKeySource := &provisionIntegrationSource{material: privateKey}
	firstAuthSource := &provisionIntegrationSource{panicIfCalled: true}
	partial, err := credentials.ProvisionCodexMicroVMCredentials(
		context.Background(), lostResponse, firstAuthSource, firstKeySource, request,
	)
	if !credentials.HasErrorCode(err, credentials.ErrorStoreIO) ||
		partial.Signing.State != credentials.CredentialProvisionPending ||
		partial.Auth.State != credentials.CredentialProvisionPending ||
		firstKeySource.calls != 1 || firstAuthSource.calls != 0 || !lostResponse.lost {
		t.Fatalf("lost response result=%+v err=%v key_calls=%d auth_calls=%d lost=%t",
			partial, err, firstKeySource.calls, firstAuthSource.calls, lostResponse.lost)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := open()
	t.Cleanup(func() { _ = reopened.Close() })
	retryKeySource := &provisionIntegrationSource{panicIfCalled: true}
	retryAuthSource := &provisionIntegrationSource{material: authMaterial}
	complete, err := credentials.ProvisionCodexMicroVMCredentials(
		context.Background(), reopened, retryAuthSource, retryKeySource, request,
	)
	if err != nil || complete.Signing.State != credentials.CredentialProvisionResumed ||
		complete.Auth.State != credentials.CredentialProvisionCreated || retryKeySource.calls != 0 ||
		retryAuthSource.calls != 1 || complete.SigningKeyID != request.SigningKeyID ||
		complete.SigningPublicKeyBase64 != base64.StdEncoding.EncodeToString(privateKey[ed25519.SeedSize:]) {
		t.Fatalf("reopened retry result=%+v err=%v key_calls=%d auth_calls=%d",
			complete, err, retryKeySource.calls, retryAuthSource.calls)
	}

	finalKeySource := &provisionIntegrationSource{panicIfCalled: true}
	finalAuthSource := &provisionIntegrationSource{panicIfCalled: true}
	resumed, err := credentials.ProvisionCodexMicroVMCredentials(
		context.Background(), reopened, finalAuthSource, finalKeySource, request,
	)
	if err != nil || resumed.Signing.State != credentials.CredentialProvisionResumed ||
		resumed.Auth.State != credentials.CredentialProvisionResumed || finalKeySource.calls != 0 ||
		finalAuthSource.calls != 0 || resumed.SigningPublicKeyBase64 != complete.SigningPublicKeyBase64 {
		t.Fatalf("final resume result=%+v err=%v key_calls=%d auth_calls=%d",
			resumed, err, finalKeySource.calls, finalAuthSource.calls)
	}
}

type provisionLostCreateResponseStore struct {
	credentials.ProvisionStore
	credentialRef credentials.CredentialRef
	lost          bool
}

func (store *provisionLostCreateResponseStore) Create(
	ctx context.Context,
	request credentials.CreateRequest,
) (credentials.MutationResult, error) {
	result, err := store.ProvisionStore.Create(ctx, request)
	if err == nil && request.CredentialRef == store.credentialRef && !store.lost {
		store.lost = true
		// The local adapter has already synced and installed the durable document;
		// only the successful response is lost at this wrapper boundary.
		return credentials.MutationResult{}, credentials.NewError(credentials.ErrorStoreIO, "lost_create_response")
	}
	return result, err
}

type provisionIntegrationSource struct {
	material      []byte
	calls         int
	panicIfCalled bool
}

func (source *provisionIntegrationSource) WithSecret(
	ctx context.Context,
	consume func(credentials.Secret) error,
) error {
	return source.provide(ctx, consume)
}

func (source *provisionIntegrationSource) WithPrivateKey(
	ctx context.Context,
	consume func(credentials.Secret) error,
) error {
	return source.provide(ctx, consume)
}

func (source *provisionIntegrationSource) provide(
	ctx context.Context,
	consume func(credentials.Secret) error,
) error {
	source.calls++
	if source.panicIfCalled {
		panic("provision source must not run")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	secret, err := credentials.NewSecret(source.material)
	if err != nil {
		return err
	}
	defer secret.Destroy()
	return consume(secret)
}

func provisionIntegrationRequest() credentials.ProvisionCodexMicroVMCredentialsRequest {
	return credentials.ProvisionCodexMicroVMCredentialsRequest{
		ActorRef: "actor:provision-local", OwnerRef: "owner:microvm-1",
		SigningKeyID: "clave-publica:microvm-launch-1",
		Signing: credentials.ProvisionCredentialRequest{
			RequestRef: "request:provision:signing", CredentialRef: "credential:microvm-launch-signing",
			ScopeRefs:  []credentials.ScopeRef{"scope:microvm-launch", "scope:microvm-runtime"},
			PurposeRef: "purpose:launch-grant-signing",
		},
		Auth: credentials.ProvisionCredentialRequest{
			RequestRef: "request:provision:auth", CredentialRef: "credential:codex-account-1",
			ScopeRefs:  []credentials.ScopeRef{"scope:provider-api", "scope:microvm-runtime"},
			PurposeRef: "purpose:codex-auth",
		},
	}
}
