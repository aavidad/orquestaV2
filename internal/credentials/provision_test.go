package credentials

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProvisionEd25519SigningCredentialCreatesAndResumesWithoutRegenerating(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x2d)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	request := validEd25519ProvisionRequest()
	firstSource := &provisionFakeSource{material: privateKey}

	first, err := ProvisionEd25519SigningCredential(
		context.Background(), store, firstSource, request,
	)
	if err != nil || first.Signing.State != CredentialProvisionCreated ||
		first.Signing.RequestedAuthority.Version != 1 || firstSource.callCount() != 1 ||
		first.SigningPublicKeyBase64 != base64.StdEncoding.EncodeToString(privateKey[ed25519.SeedSize:]) {
		t.Fatalf("first=%+v source_calls=%d err=%v", first, firstSource.callCount(), err)
	}

	retrySource := &provisionFakeSource{panicValue: "keysource-must-not-run-on-resume"}
	resumed, err := ProvisionEd25519SigningCredential(
		context.Background(), store, retrySource, request,
	)
	if err != nil || resumed.Signing.State != CredentialProvisionResumed ||
		resumed.Signing.RequestedAuthority.Version != 1 || retrySource.callCount() != 0 ||
		resumed.SigningPublicKeyBase64 != first.SigningPublicKeyBase64 {
		t.Fatalf("resumed=%+v source_calls=%d err=%v", resumed, retrySource.callCount(), err)
	}
	if got := store.createdCredentialRefs(); !reflect.DeepEqual(got, []CredentialRef{request.Signing.CredentialRef}) {
		t.Fatalf("create-only effects=%v", got)
	}
	assertProvisionProjectionDoesNotContain(t, first, privateKey)
	assertProvisionProjectionDoesNotContain(t, resumed, privateKey)
}

func TestProvisionEd25519SigningCredentialRecoversLostCreateResponse(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x3e)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	store.createCollision = ErrorAlreadyExists
	request := validEd25519ProvisionRequest()

	result, err := ProvisionEd25519SigningCredential(
		context.Background(), store, &provisionFakeSource{material: privateKey}, request,
	)
	if err != nil || result.Signing.State != CredentialProvisionResumed ||
		result.Signing.RequestedAuthority.Version != 1 || len(result.Signing.Receipts) != 1 ||
		result.SigningPublicKeyBase64 != base64.StdEncoding.EncodeToString(privateKey[ed25519.SeedSize:]) {
		t.Fatalf("lost-response recovery=%+v err=%v", result, err)
	}
	if got := store.createdCredentialRefs(); !reflect.DeepEqual(got, []CredentialRef{request.Signing.CredentialRef}) {
		t.Fatalf("lost response repeated create=%v", got)
	}
}

func TestProvisionEd25519SigningCredentialRejectsRefCollisionAndVersionDrift(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x4f)
	t.Cleanup(func() { clear(privateKey) })
	request := validEd25519ProvisionRequest()

	t.Run("invalid request ref", func(t *testing.T) {
		store := newProvisionFakeStore()
		defer store.destroy()
		candidate := request
		candidate.Signing.RequestRef = "other:continuation"
		source := &provisionFakeSource{panicValue: "invalid-ref-source"}
		result, err := ProvisionEd25519SigningCredential(context.Background(), store, source, candidate)
		if !HasErrorCode(err, ErrorInvalidRef) || source.callCount() != 0 ||
			!reflect.DeepEqual(result, ProvisionEd25519SigningCredentialResult{}) || store.effectCount() != 0 {
			t.Fatalf("invalid ref result=%+v source=%d effects=%d err=%v", result, source.callCount(), store.effectCount(), err)
		}
	})

	t.Run("existing authority collision", func(t *testing.T) {
		store := newProvisionFakeStore()
		defer store.destroy()
		conflicting := request.Signing
		conflicting.PurposeRef = "purpose:other"
		store.seed(conflicting, request.OwnerRef, privateKey, 1)
		source := &provisionFakeSource{panicValue: "collision-source"}
		result, err := ProvisionEd25519SigningCredential(context.Background(), store, source, request)
		if !HasErrorCode(err, ErrorPurposeDenied) || source.callCount() != 0 ||
			result.Signing.State != CredentialProvisionPending {
			t.Fatalf("collision result=%+v source=%d err=%v", result, source.callCount(), err)
		}
	})

	t.Run("version drift", func(t *testing.T) {
		store := &provisionVersionConflictStore{ProvisionStore: newProvisionFakeStore()}
		defer store.ProvisionStore.(*provisionFakeStore).destroy()
		store.ProvisionStore.(*provisionFakeStore).seed(request.Signing, request.OwnerRef, privateKey, 2)
		source := &provisionFakeSource{panicValue: "version-source"}
		result, err := ProvisionEd25519SigningCredential(context.Background(), store, source, request)
		if !HasErrorCode(err, ErrorVersionConflict) || source.callCount() != 0 ||
			result.Signing.State != CredentialProvisionPending {
			t.Fatalf("version drift result=%+v source=%d err=%v", result, source.callCount(), err)
		}
	})
}

type provisionVersionConflictStore struct{ ProvisionStore }

func (store *provisionVersionConflictStore) Use(
	context.Context,
	UseRequest,
	func(Secret) error,
) (Receipt, error) {
	return Receipt{}, NewError(ErrorVersionConflict, "version")
}

func TestProvisionCodexMicroVMCredentialsCreatesSigningThenAuthWithoutMaterialEgress(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x31)
	authMaterial := []byte("codex-auth-material-never-project")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	keySource := &provisionFakeSource{material: privateKey}
	authSource := &provisionFakeSource{material: authMaterial}
	request := validProvisionRequest()

	result, err := ProvisionCodexMicroVMCredentials(context.Background(), store, authSource, keySource, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Signing.State != CredentialProvisionCreated || result.Auth.State != CredentialProvisionCreated ||
		result.Signing.RequestedAuthority.Version != 1 || result.Auth.RequestedAuthority.Version != 1 ||
		len(result.Signing.Receipts) != 2 || len(result.Auth.Receipts) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	wantPublic := privateKey[ed25519.SeedSize:]
	if result.SigningKeyID != request.SigningKeyID || result.SigningPublicKeyBase64 != base64.StdEncoding.EncodeToString(wantPublic) {
		t.Fatalf("public projection mismatch: key_id=%q public=%q", result.SigningKeyID, result.SigningPublicKeyBase64)
	}
	if got := store.createdCredentialRefs(); !reflect.DeepEqual(got, []CredentialRef{request.Signing.CredentialRef, request.Auth.CredentialRef}) {
		t.Fatalf("create order=%v", got)
	}
	if got := store.useRequestsSnapshot(); len(got) != 1 || got[0].CredentialRef != request.Signing.CredentialRef ||
		got[0].Version != 1 || got[0].RequestRef != request.Signing.RequestRef+":use:v1" {
		t.Fatalf("signing use=%+v", got)
	}
	if keySource.callCount() != 1 || authSource.callCount() != 1 {
		t.Fatalf("sources key=%d auth=%d", keySource.callCount(), authSource.callCount())
	}
	assertProvisionSourceMaterialDestroyed(t, keySource)
	assertProvisionSourceMaterialDestroyed(t, authSource)
	store.assertRequestSecretsDestroyed(t)
	assertProvisionProjectionDoesNotContain(t, result, privateKey, authMaterial)
}

func TestProvisionCodexMicroVMCredentialsResumesExistingWithoutRegeneratingOrReadingAuth(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x44)
	authMaterial := []byte("existing-codex-auth-material")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	request := validProvisionRequest()
	store.seed(request.Signing, request.OwnerRef, privateKey, 3)
	store.seed(request.Auth, request.OwnerRef, authMaterial, 7)
	store.addScope(request.Signing.CredentialRef, "scope:stored-extra")
	store.addScope(request.Auth.CredentialRef, "scope:stored-extra")
	keySource := &provisionFakeSource{panicValue: "private-source-must-not-run"}
	authSource := &provisionFakeSource{panicValue: "auth-source-must-not-run"}

	result, err := ProvisionCodexMicroVMCredentials(context.Background(), store, authSource, keySource, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Signing.State != CredentialProvisionResumed || result.Auth.State != CredentialProvisionResumed ||
		result.Signing.RequestedAuthority.Version != 3 || result.Auth.RequestedAuthority.Version != 7 {
		t.Fatalf("resume result=%+v", result)
	}
	if !reflect.DeepEqual(result.Signing.RequestedAuthority.ScopeRefs, canonicalProvisionTestScopes(request.Signing.ScopeRefs)) ||
		!reflect.DeepEqual(result.Auth.RequestedAuthority.ScopeRefs, canonicalProvisionTestScopes(request.Auth.ScopeRefs)) {
		t.Fatalf("result claimed stored scopes beyond requested tuples: %+v", result)
	}
	if keySource.callCount() != 0 || authSource.callCount() != 0 {
		t.Fatalf("existing credentials regenerated: key=%d auth=%d", keySource.callCount(), authSource.callCount())
	}
	uses := store.useRequestsSnapshot()
	if len(uses) != 1 || uses[0].CredentialRef != request.Signing.CredentialRef || uses[0].Version != 3 {
		t.Fatalf("resume material access=%+v", uses)
	}
	if result.SigningPublicKeyBase64 != base64.StdEncoding.EncodeToString(privateKey[ed25519.SeedSize:]) {
		t.Fatalf("wrong resumed public key: %q", result.SigningPublicKeyBase64)
	}
	describes := store.describeRequestsSnapshot()
	if len(describes) != len(request.Signing.ScopeRefs)+len(request.Auth.ScopeRefs) {
		t.Fatalf("not every requested scope was described: %+v", describes)
	}
	for index, described := range describes {
		if !strings.HasSuffix(described.RequestRef, fmt.Sprintf(":describe:scope:%d", index%2)) {
			t.Fatalf("non-deterministic describe ref: %+v", described)
		}
	}
	assertProvisionProjectionDoesNotContain(t, result, privateKey, authMaterial)
}

func TestProvisionCodexMicroVMCredentialsPartialFailureIsExplicitAndRetryable(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x52)
	authMaterial := []byte("retryable-auth-material")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	request := validProvisionRequest()
	keySource := &provisionFakeSource{material: privateKey}
	failingAuth := &provisionFakeSource{returnBeforeCallback: errors.New(string(authMaterial))}

	partial, err := ProvisionCodexMicroVMCredentials(context.Background(), store, failingAuth, keySource, request)
	if !HasErrorCode(err, ErrorConsumerFailed) || bytes.Contains([]byte(err.Error()), authMaterial) {
		t.Fatalf("unsafe or wrong auth error: %v", err)
	}
	if partial.Signing.State != CredentialProvisionCreated || partial.Auth.State != CredentialProvisionPending ||
		partial.Signing.RequestedAuthority.Version != 1 || partial.Auth.RequestedAuthority.Version != 0 {
		t.Fatalf("partial result=%+v", partial)
	}
	assertProvisionProjectionDoesNotContain(t, partial, privateKey, authMaterial)

	retryKeySource := &provisionFakeSource{panicValue: "signing-source-must-not-run-on-retry"}
	retryAuth := &provisionFakeSource{material: authMaterial}
	complete, err := ProvisionCodexMicroVMCredentials(context.Background(), store, retryAuth, retryKeySource, request)
	if err != nil {
		t.Fatal(err)
	}
	if complete.Signing.State != CredentialProvisionResumed || complete.Auth.State != CredentialProvisionCreated ||
		retryKeySource.callCount() != 0 || retryAuth.callCount() != 1 {
		t.Fatalf("retry result=%+v key_calls=%d auth_calls=%d", complete, retryKeySource.callCount(), retryAuth.callCount())
	}
	if got := store.createdCredentialRefs(); !reflect.DeepEqual(got, []CredentialRef{request.Signing.CredentialRef, request.Auth.CredentialRef}) {
		t.Fatalf("retry regenerated credential: %v", got)
	}
}

func TestProvisionCodexMicroVMCredentialsConvergesCreateRacesByExactAuthority(t *testing.T) {
	for _, code := range []ErrorCode{ErrorAlreadyExists, ErrorIdempotencyConflict} {
		t.Run(string(code), func(t *testing.T) {
			privateKey := testProvisionPrivateKey(0x63)
			authMaterial := []byte("racing-auth-material")
			t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
			store := newProvisionFakeStore()
			t.Cleanup(store.destroy)
			request := validProvisionRequest()
			store.createCollision = code
			keySource := &provisionFakeSource{material: privateKey}
			authSource := &provisionFakeSource{material: authMaterial}

			result, err := ProvisionCodexMicroVMCredentials(context.Background(), store, authSource, keySource, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Signing.State != CredentialProvisionResumed || result.Auth.State != CredentialProvisionResumed ||
				len(result.Signing.Receipts) != 1 || len(result.Auth.Receipts) != 0 {
				t.Fatalf("race result=%+v", result)
			}
			uses := store.useRequestsSnapshot()
			if len(uses) != 1 || uses[0].CredentialRef != request.Signing.CredentialRef {
				t.Fatalf("race read wrong material: %+v", uses)
			}
		})
	}
}

func TestProvisionCodexMicroVMCredentialsIsConcurrentAndSequentiallyIdempotent(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x71)
	authMaterial := []byte("concurrent-auth-material")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	request := validProvisionRequest()

	const workers = 24
	start := make(chan struct{})
	results := make(chan ProvisionCodexMicroVMCredentialsResult, workers)
	errs := make(chan error, workers)
	var ready sync.WaitGroup
	ready.Add(workers)
	for range workers {
		go func() {
			ready.Done()
			<-start
			result, err := ProvisionCodexMicroVMCredentials(
				context.Background(), store,
				&provisionFakeSource{material: authMaterial}, &provisionFakeSource{material: privateKey}, request,
			)
			results <- result
			errs <- err
		}()
	}
	ready.Wait()
	close(start)
	var publicKey, keyID string
	for range workers {
		result, err := <-results, <-errs
		if err != nil {
			t.Fatal(err)
		}
		if result.Signing.State == CredentialProvisionPending || result.Auth.State == CredentialProvisionPending {
			t.Fatalf("pending concurrent result=%+v", result)
		}
		if publicKey == "" {
			publicKey, keyID = result.SigningPublicKeyBase64, result.SigningKeyID
		}
		if result.SigningPublicKeyBase64 != publicKey || result.SigningKeyID != keyID {
			t.Fatalf("concurrent identity diverged: %+v", result)
		}
	}
	if got := store.createdCredentialRefs(); len(got) != 2 {
		t.Fatalf("create effects=%v", got)
	}

	keySource := &provisionFakeSource{panicValue: "sequential-key-source"}
	authSource := &provisionFakeSource{panicValue: "sequential-auth-source"}
	again, err := ProvisionCodexMicroVMCredentials(context.Background(), store, authSource, keySource, request)
	if err != nil || again.Signing.State != CredentialProvisionResumed || again.Auth.State != CredentialProvisionResumed ||
		keySource.callCount() != 0 || authSource.callCount() != 0 {
		t.Fatalf("sequential replay=%+v err=%v", again, err)
	}
}

func TestProvisionCodexMicroVMCredentialsRejectsInvalidRequestsAndTypedNilDependencies(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x22)
	t.Cleanup(func() { clear(privateKey) })
	validStore := newProvisionFakeStore()
	t.Cleanup(validStore.destroy)
	validAuth := &provisionFakeSource{material: []byte("valid-auth")}
	validKey := &provisionFakeSource{material: privateKey}
	request := validProvisionRequest()
	var nilStore *provisionFakeStore
	var nilSource *provisionFakeSource

	tests := []struct {
		name       string
		ctx        context.Context
		store      ProvisionStore
		auth       MaterialSource
		key        Ed25519PrivateKeySource
		mutate     func(*ProvisionCodexMicroVMCredentialsRequest)
		wantCancel bool
	}{
		{name: "nil context", store: validStore, auth: validAuth, key: validKey},
		{name: "typed nil store", ctx: context.Background(), store: nilStore, auth: validAuth, key: validKey},
		{name: "typed nil auth", ctx: context.Background(), store: validStore, auth: nilSource, key: validKey},
		{name: "typed nil key", ctx: context.Background(), store: validStore, auth: validAuth, key: nilSource},
		{name: "same credentials", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) {
				value.Auth.CredentialRef = value.Signing.CredentialRef
			}},
		{name: "same requests", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.Auth.RequestRef = value.Signing.RequestRef }},
		{name: "bad key id", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.SigningKeyID = "other:key" }},
		{name: "bad actor", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.ActorRef = "" }},
		{name: "bad request", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.Signing.RequestRef = "other:request" }},
		{name: "bad credential", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) {
				value.Auth.CredentialRef = "credential:../escape"
			}},
		{name: "bad owner", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.OwnerRef = "" }},
		{name: "bad scope", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.Auth.ScopeRefs = []ScopeRef{""} }},
		{name: "duplicate scope", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) {
				value.Auth.ScopeRefs = []ScopeRef{"project:first-microvm", "project:first-microvm"}
			}},
		{name: "bad purpose", ctx: context.Background(), store: validStore, auth: validAuth, key: validKey,
			mutate: func(value *ProvisionCodexMicroVMCredentialsRequest) { value.Signing.PurposeRef = "" }},
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests = append(tests, struct {
		name       string
		ctx        context.Context
		store      ProvisionStore
		auth       MaterialSource
		key        Ed25519PrivateKeySource
		mutate     func(*ProvisionCodexMicroVMCredentialsRequest)
		wantCancel bool
	}{name: "canceled", ctx: canceled, store: validStore, auth: validAuth, key: validKey, wantCancel: true})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := request
			if test.mutate != nil {
				test.mutate(&candidate)
			}
			before := validStore.effectCount()
			result, err := ProvisionCodexMicroVMCredentials(test.ctx, test.store, test.auth, test.key, candidate)
			if test.wantCancel {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err=%v", err)
				}
			} else if !HasErrorCode(err, ErrorInvalidRequest) && !HasErrorCode(err, ErrorInvalidRef) {
				t.Fatalf("err=%v", err)
			}
			if !reflect.DeepEqual(result, ProvisionCodexMicroVMCredentialsResult{}) || validStore.effectCount() != before {
				t.Fatalf("invalid request changed state: result=%+v effects=%d/%d", result, before, validStore.effectCount())
			}
		})
	}
}

func TestProvisionCodexMicroVMCredentialsSanitizesCallbacksAndRejectsAdversarialPorts(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x16)
	authMaterial := []byte("adversarial-auth-material")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	request := validProvisionRequest()

	tests := []struct {
		name  string
		setup func(*provisionFakeStore, *provisionFakeSource, *provisionFakeSource)
		code  ErrorCode
	}{
		{name: "invalid signing key", setup: func(_ *provisionFakeStore, _ *provisionFakeSource, key *provisionFakeSource) {
			key.material = []byte("not-ed25519")
		}, code: ErrorInvalidRequest},
		{name: "invalid resumed signing key", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.seed(request.Signing, request.OwnerRef, []byte("not-ed25519"), 1)
		}, code: ErrorInvalidRequest},
		{name: "key source no callback", setup: func(_ *provisionFakeStore, _ *provisionFakeSource, key *provisionFakeSource) {
			key.returnWithoutCallback = true
		}, code: ErrorConsumerFailed},
		{name: "key source twice", setup: func(_ *provisionFakeStore, _ *provisionFakeSource, key *provisionFakeSource) { key.twice = true }, code: ErrorConsumerFailed},
		{name: "invalid create projection", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.invalidCreateResult = true
		}, code: ErrorInvalidRequest},
		{name: "mismatched describe", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.seed(request.Signing, request.OwnerRef, privateKey, 1)
			store.invalidDescribeResult = true
		}, code: ErrorInvalidRequest},
		{name: "use omits callback", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.useOmitsCallback = true
		}, code: ErrorConsumerFailed},
		{name: "use calls twice", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.useCallsTwice = true
		}, code: ErrorConsumerFailed},
		{name: "invalid use receipt", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.invalidUseReceipt = true
		}, code: ErrorInvalidRequest},
		{name: "store secret error", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.createError = errors.New(string(privateKey))
		}, code: ErrorStoreIO},
		{name: "store typed secret field", setup: func(store *provisionFakeStore, _ *provisionFakeSource, _ *provisionFakeSource) {
			store.createError = &Error{Code: ErrorStoreIO, Field: string(privateKey)}
		}, code: ErrorStoreIO},
		{name: "source typed secret field", setup: func(_ *provisionFakeStore, _ *provisionFakeSource, key *provisionFakeSource) {
			key.returnBeforeCallback = &Error{Code: ErrorConsumerFailed, Field: string(privateKey)}
		}, code: ErrorConsumerFailed},
		{name: "source secret error code", setup: func(_ *provisionFakeStore, _ *provisionFakeSource, key *provisionFakeSource) {
			key.returnBeforeCallback = &Error{Code: ErrorCode(string(privateKey)), Field: "source"}
		}, code: ErrorConsumerFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newProvisionFakeStore()
			t.Cleanup(store.destroy)
			auth := &provisionFakeSource{material: authMaterial}
			key := &provisionFakeSource{material: privateKey}
			test.setup(store, auth, key)
			result, err := ProvisionCodexMicroVMCredentials(context.Background(), store, auth, key, request)
			if !HasErrorCode(err, test.code) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			assertProvisionProjectionDoesNotContain(t, result, privateKey, authMaterial)
			assertErrorDoesNotContain(t, err, privateKey, authMaterial)
		})
	}
}

func TestProvisionRejectsNonInitialCreateProjection(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x17)
	t.Cleanup(func() { clear(privateKey) })
	nonZero := provisionTestTime.Add(time.Minute)
	tests := []struct {
		name   string
		mutate func(*MutationResult)
	}{
		{name: "version seven", mutate: func(result *MutationResult) {
			result.Metadata.Version = 7
			result.Receipt.Version = 7
		}},
		{name: "rotated at", mutate: func(result *MutationResult) {
			result.Metadata.RotatedAt = nonZero
		}},
		{name: "revoked at", mutate: func(result *MutationResult) {
			result.Metadata.RevokedAt = nonZero
		}},
		{name: "revoked", mutate: func(result *MutationResult) {
			result.Metadata.Revoked = true
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newProvisionFakeStore()
			t.Cleanup(store.destroy)
			store.mutateCreateResult = test.mutate
			result, err := ProvisionCodexMicroVMCredentials(
				context.Background(), store, &provisionFakeSource{material: []byte("auth")},
				&provisionFakeSource{material: privateKey}, validProvisionRequest(),
			)
			if !HasErrorCode(err, ErrorInvalidRequest) || result.Signing.State != CredentialProvisionPending ||
				result.Auth.State != CredentialProvisionPending {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestProvisionSourcesMustSucceedExactlyOnceBeforeAnyStoreCreate(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x18)
	t.Cleanup(func() { clear(privateKey) })
	tests := []struct {
		name   string
		source *provisionFakeSource
	}{
		{name: "zero callbacks", source: &provisionFakeSource{material: privateKey, returnWithoutCallback: true}},
		{name: "two callbacks", source: &provisionFakeSource{material: privateKey, twice: true}},
		{name: "two concurrent callbacks", source: &provisionFakeSource{material: privateKey, concurrentTwice: true}},
		{name: "source error after callback", source: &provisionFakeSource{
			material: privateKey, returnAfterCallback: errors.New(string(privateKey)),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newProvisionFakeStore()
			t.Cleanup(store.destroy)
			result, err := ProvisionCodexMicroVMCredentials(
				context.Background(), store, &provisionFakeSource{material: []byte("auth")}, test.source, validProvisionRequest(),
			)
			if !HasErrorCode(err, ErrorConsumerFailed) || store.effectCount() != 0 || len(store.createdCredentialRefs()) != 0 {
				t.Fatalf("source contract produced effect: result=%+v err=%v effects=%d", result, err, store.effectCount())
			}
			assertProvisionSourceMaterialDestroyed(t, test.source)
			assertErrorDoesNotContain(t, err, privateKey)
		})
	}
}

func TestProvisionCallbackCapturedThenSourcePanicIsClosedAndInert(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x19)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	source := &provisionFakeSource{material: privateKey, captureThenPanic: string(privateKey)}
	result, err := ProvisionCodexMicroVMCredentials(
		context.Background(), store, &provisionFakeSource{material: []byte("auth")}, source, validProvisionRequest(),
	)
	if !HasErrorCode(err, ErrorConsumerFailed) || store.effectCount() != 0 {
		t.Fatalf("panic result=%+v err=%v effects=%d", result, err, store.effectCount())
	}
	retained, lateErr := source.invokeCaptured(privateKey)
	copy := retained.Bytes()
	defer clear(copy)
	if !HasErrorCode(lateErr, ErrorConsumerFailed) || len(bytes.Trim(copy, "\x00")) != 0 || store.effectCount() != 0 {
		t.Fatalf("late callback remained active: err=%v effects=%d", lateErr, store.effectCount())
	}
	assertErrorDoesNotContain(t, err, privateKey)
}

func TestProvisionUseCallbackCapturedThenStorePanicIsClosedAndInert(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x1b)
	t.Cleanup(func() { clear(privateKey) })
	request := validProvisionRequest()
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	store.seed(request.Signing, request.OwnerRef, privateKey, 1)
	store.captureUseThenPanic = string(privateKey)
	result, err := ProvisionCodexMicroVMCredentials(
		context.Background(), store, &provisionFakeSource{material: []byte("auth")},
		&provisionFakeSource{panicValue: "key-source-must-not-run"}, request,
	)
	if !HasErrorCode(err, ErrorConsumerFailed) || result.Signing.State != CredentialProvisionPending {
		t.Fatalf("use panic result=%+v err=%v", result, err)
	}
	before := store.effectCount()
	retained, lateErr := store.invokeCapturedUse(privateKey)
	copy := retained.Bytes()
	defer clear(copy)
	if !HasErrorCode(lateErr, ErrorConsumerFailed) || len(bytes.Trim(copy, "\x00")) != 0 || store.effectCount() != before {
		t.Fatalf("late use callback remained active: err=%v effects=%d/%d", lateErr, before, store.effectCount())
	}
	assertErrorDoesNotContain(t, err, privateKey)
}

func TestProvisionStoreCreateReentrantSourceCallbackRejectsWithoutDeadlock(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x1a)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	source := &provisionFakeSource{material: privateKey, captureCallback: true}
	var late Secret
	store.createHook = func() error {
		var err error
		late, err = source.invokeCaptured(privateKey)
		return err
	}
	type outcome struct {
		result ProvisionCodexMicroVMCredentialsResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := ProvisionCodexMicroVMCredentials(
			context.Background(), store, &provisionFakeSource{material: []byte("auth")}, source, validProvisionRequest(),
		)
		done <- outcome{result: result, err: err}
	}()
	select {
	case got := <-done:
		if got.err != nil || got.result.Signing.State != CredentialProvisionCreated ||
			!HasErrorCode(store.createHookError(), ErrorConsumerFailed) {
			t.Fatalf("reentrant result=%+v err=%v hook_err=%v", got.result, got.err, store.createHookError())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reentrant callback deadlocked")
	}
	copy := late.Bytes()
	defer clear(copy)
	if len(bytes.Trim(copy, "\x00")) != 0 {
		t.Fatal("reentrant callback retained material")
	}
}

func TestProvisionCodexMicroVMCredentialsPanicBecomesRedactedErrorAndCleansMaterial(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x27)
	authMaterial := []byte("panic-auth-material")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	store.panicCreate = string(privateKey)
	keySource := &provisionFakeSource{material: privateKey}
	authSource := &provisionFakeSource{material: authMaterial}

	result, err := ProvisionCodexMicroVMCredentials(context.Background(), store, authSource, keySource, validProvisionRequest())
	if !HasErrorCode(err, ErrorConsumerFailed) || err.Error() != "credentials.consumer_failed: provision" {
		t.Fatalf("panic result=%+v err=%v", result, err)
	}
	assertProvisionSourceMaterialDestroyed(t, keySource)
	store.assertRequestSecretsDestroyed(t)
	assertProvisionProjectionDoesNotContain(t, result, privateKey, authMaterial)
	assertErrorDoesNotContain(t, err, privateKey, authMaterial)
}

func TestProvisionCodexMicroVMCredentialsPreservesCancellationWithoutMaterialLeak(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x38)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	keySource := &provisionFakeSource{returnBeforeCallback: context.Canceled, material: privateKey}
	result, err := ProvisionCodexMicroVMCredentials(
		context.Background(), store, &provisionFakeSource{material: []byte("auth")}, keySource, validProvisionRequest(),
	)
	if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionPending || result.Auth.State != CredentialProvisionPending {
		t.Fatalf("cancel result=%+v err=%v", result, err)
	}
}

func TestProvisionCodexMicroVMCredentialsCancellationInsideKeyCallbackCleansMaterial(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x39)
	t.Cleanup(func() { clear(privateKey) })
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	ctx, cancel := context.WithCancel(context.Background())
	keySource := &provisionFakeSource{material: privateKey, beforeCallback: cancel}
	result, err := ProvisionCodexMicroVMCredentials(
		ctx, store, &provisionFakeSource{material: []byte("auth")}, keySource, validProvisionRequest(),
	)
	if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionPending || store.effectCount() != 0 {
		t.Fatalf("cancel result=%+v err=%v effects=%d", result, err, store.effectCount())
	}
	assertProvisionSourceMaterialDestroyed(t, keySource)
}

func TestProvisionCancellationDominatesSuccessfulFinalPorts(t *testing.T) {
	privateKey := testProvisionPrivateKey(0x3a)
	authMaterial := []byte("cancel-final-auth")
	t.Cleanup(func() { clear(privateKey); clear(authMaterial) })
	request := validProvisionRequest()

	t.Run("reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		store := newProvisionFakeStore()
		t.Cleanup(store.destroy)
		store.seed(request.Signing, request.OwnerRef, privateKey, 1)
		store.seed(request.Auth, request.OwnerRef, authMaterial, 1)
		authScopes := canonicalProvisionTestScopes(request.Auth.ScopeRefs)
		store.cancelDescribeRef = request.Auth.CredentialRef
		store.cancelDescribeScope = authScopes[len(authScopes)-1]
		store.cancelDescribe = cancel
		result, err := ProvisionCodexMicroVMCredentials(
			ctx, store, &provisionFakeSource{panicValue: "auth-source"}, &provisionFakeSource{panicValue: "key-source"}, request,
		)
		if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionResumed ||
			result.Auth.State != CredentialProvisionPending {
			t.Fatalf("reader cancel result=%+v err=%v", result, err)
		}
	})

	t.Run("source", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		store := newProvisionFakeStore()
		t.Cleanup(store.destroy)
		authSource := &provisionFakeSource{material: authMaterial, afterCallback: cancel}
		result, err := ProvisionCodexMicroVMCredentials(
			ctx, store, authSource, &provisionFakeSource{material: privateKey}, request,
		)
		created := store.createdCredentialRefs()
		if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionCreated ||
			result.Auth.State != CredentialProvisionPending || len(created) != 1 || created[0] != request.Signing.CredentialRef {
			t.Fatalf("source cancel result=%+v err=%v created=%v", result, err, created)
		}
		assertProvisionSourceMaterialDestroyed(t, authSource)
	})

	t.Run("store create", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		store := newProvisionFakeStore()
		t.Cleanup(store.destroy)
		store.cancelCreateRef = request.Auth.CredentialRef
		store.cancelCreate = cancel
		result, err := ProvisionCodexMicroVMCredentials(
			ctx, store, &provisionFakeSource{material: authMaterial}, &provisionFakeSource{material: privateKey}, request,
		)
		if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionCreated ||
			result.Auth.State != CredentialProvisionPending || len(store.createdCredentialRefs()) != 2 {
			t.Fatalf("store cancel result=%+v err=%v created=%v", result, err, store.createdCredentialRefs())
		}
	})
}

func TestProvisionStoreUseCancellationPreventsPrivateProjection(t *testing.T) {
	request := validProvisionRequest()
	ctx, cancel := context.WithCancel(context.Background())
	store := newProvisionFakeStore()
	t.Cleanup(store.destroy)
	store.seed(request.Signing, request.OwnerRef, []byte("invalid-private-material"), 1)
	store.cancelBeforeUse = cancel
	result, err := ProvisionCodexMicroVMCredentials(
		ctx, store, &provisionFakeSource{panicValue: "auth-source"}, &provisionFakeSource{panicValue: "key-source"}, request,
	)
	if !errors.Is(err, context.Canceled) || result.Signing.State != CredentialProvisionPending ||
		result.SigningPublicKeyBase64 != "" || !errors.Is(store.useConsumeError(), context.Canceled) ||
		HasErrorCode(store.useConsumeError(), ErrorInvalidRequest) {
		t.Fatalf("use cancel result=%+v err=%v consume_err=%v", result, err, store.useConsumeError())
	}
}

func TestProvisionCodexMicroVMCredentialsOutputContractCannotCarryPrivateMaterial(t *testing.T) {
	contract := reflect.TypeOf(ProvisionCodexMicroVMCredentialsResult{})
	assertProvisionMaterialFreeType(t, contract, map[reflect.Type]bool{})
	keySource := reflect.TypeOf((*Ed25519PrivateKeySource)(nil)).Elem()
	method, found := keySource.MethodByName("WithPrivateKey")
	if !found || keySource.NumMethod() != 1 || method.Type.NumIn() != 2 ||
		method.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		method.Type.In(1) != reflect.TypeOf((func(Secret) error)(nil)) ||
		method.Type.NumOut() != 1 || method.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		t.Fatalf("unsafe key source: %v", keySource)
	}
	for index := 0; index < method.Type.NumOut(); index++ {
		if output := method.Type.Out(index); output.Kind() == reflect.Slice && output.Elem().Kind() == reflect.Uint8 {
			t.Fatalf("key source returns bytes: %v", method.Type)
		}
	}
}

func assertProvisionMaterialFreeType(t *testing.T, value reflect.Type, visited map[reflect.Type]bool) {
	t.Helper()
	if visited[value] {
		return
	}
	visited[value] = true
	if value == reflect.TypeOf(Secret{}) || value.Kind() == reflect.Slice && value.Elem().Kind() == reflect.Uint8 {
		t.Fatalf("material-bearing output: %v", value)
	}
	if value.Kind() == reflect.Slice {
		assertProvisionMaterialFreeType(t, value.Elem(), visited)
		return
	}
	if value.Kind() != reflect.Struct || value.PkgPath() != reflect.TypeOf(Receipt{}).PkgPath() {
		return
	}
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		name := strings.ToLower(field.Name)
		for _, forbidden := range []string{"secret", "private", "material", "path", "digest"} {
			if strings.Contains(name, forbidden) {
				t.Fatalf("forbidden output field: %s.%s", value, field.Name)
			}
		}
		assertProvisionMaterialFreeType(t, field.Type, visited)
	}
}

func assertProvisionProjectionDoesNotContain(t *testing.T, result any, materials ...[]byte) {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	projections := [][]byte{payload, []byte(fmt.Sprintf("%v %+v %#v", result, result, result))}
	for _, material := range materials {
		for _, signature := range [][]byte{material, []byte(base64.StdEncoding.EncodeToString(material)), []byte(hex.EncodeToString(material))} {
			for _, projection := range projections {
				if len(signature) > 0 && bytes.Contains(projection, signature) {
					t.Fatalf("projection leaked credential material")
				}
			}
		}
	}
}

func assertErrorDoesNotContain(t *testing.T, err error, materials ...[]byte) {
	t.Helper()
	payload, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	projection := append([]byte(err.Error()), payload...)
	for _, material := range materials {
		if bytes.Contains(projection, material) || bytes.Contains(projection, []byte(base64.StdEncoding.EncodeToString(material))) ||
			bytes.Contains(projection, []byte(hex.EncodeToString(material))) {
			t.Fatal("error leaked credential material")
		}
	}
}

func assertProvisionSourceMaterialDestroyed(t *testing.T, source *provisionFakeSource) {
	t.Helper()
	for _, retained := range source.retainedSecrets() {
		copy := retained.Bytes()
		if len(bytes.Trim(copy, "\x00")) != 0 {
			clear(copy)
			t.Fatal("callback-scoped source secret survived")
		}
		clear(copy)
	}
}

func validProvisionRequest() ProvisionCodexMicroVMCredentialsRequest {
	return ProvisionCodexMicroVMCredentialsRequest{
		ActorRef: "actor:operator", OwnerRef: "actor:operator", SigningKeyID: "clave-publica:first-codex-microvm",
		Signing: ProvisionCredentialRequest{
			RequestRef: "request:provision:signing", CredentialRef: "credential:microvm-launch-signing",
			ScopeRefs: []ScopeRef{"scope:runtime", "project:first-microvm"}, PurposeRef: "orquesta.microvm-launch-grant.v1",
		},
		Auth: ProvisionCredentialRequest{
			RequestRef: "request:provision:auth", CredentialRef: "credential:codex-account-1",
			ScopeRefs: []ScopeRef{"scope:runtime", "project:first-microvm"}, PurposeRef: "purpose:codex-runtime",
		},
	}
}

func validEd25519ProvisionRequest() ProvisionEd25519SigningCredentialRequest {
	return ProvisionEd25519SigningCredentialRequest{
		ActorRef: "actor:operator",
		OwnerRef: "actor:operator",
		Signing: ProvisionCredentialRequest{
			RequestRef:    "request:provision:continuation-signing",
			CredentialRef: "credential:microvm-continuation-signing",
			ScopeRefs:     []ScopeRef{"project:first-microvm"},
			PurposeRef:    "orquesta.microvm-expired-launch-continuation-authority.v1",
		},
	}
}

func testProvisionPrivateKey(fill byte) []byte {
	seed := bytes.Repeat([]byte{fill}, ed25519.SeedSize)
	defer clear(seed)
	return ed25519.NewKeyFromSeed(seed)
}

type provisionFakeSource struct {
	mu                    sync.Mutex
	material              []byte
	calls                 int
	retained              []Secret
	returnBeforeCallback  error
	returnAfterCallback   error
	returnWithoutCallback bool
	twice                 bool
	concurrentTwice       bool
	panicValue            string
	captureThenPanic      string
	captureCallback       bool
	captured              func(Secret) error
	beforeCallback        func()
	afterCallback         func()
}

func (source *provisionFakeSource) WithSecret(ctx context.Context, consume func(Secret) error) error {
	return source.provide(ctx, consume)
}

func (source *provisionFakeSource) WithPrivateKey(ctx context.Context, consume func(Secret) error) error {
	return source.provide(ctx, consume)
}

func (source *provisionFakeSource) provide(ctx context.Context, consume func(Secret) error) error {
	source.mu.Lock()
	source.calls++
	panicValue, capturePanic := source.panicValue, source.captureThenPanic
	before, after := source.returnBeforeCallback, source.returnAfterCallback
	skip, twice, concurrentTwice := source.returnWithoutCallback, source.twice, source.concurrentTwice
	captureCallback, beforeCallback, afterCallback := source.captureCallback, source.beforeCallback, source.afterCallback
	if captureCallback || capturePanic != "" {
		source.captured = consume
	}
	material := append([]byte(nil), source.material...)
	source.mu.Unlock()
	defer clear(material)
	if capturePanic != "" {
		panic(capturePanic)
	}
	if panicValue != "" {
		panic(panicValue)
	}
	if before != nil {
		return before
	}
	if skip {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	secret, err := NewSecret(material)
	if err != nil {
		return err
	}
	source.mu.Lock()
	source.retained = append(source.retained, secret)
	source.mu.Unlock()
	defer secret.Destroy()
	if beforeCallback != nil {
		beforeCallback()
	}
	if concurrentTwice {
		second, err := NewSecret(material)
		if err != nil {
			return err
		}
		source.mu.Lock()
		source.retained = append(source.retained, second)
		source.mu.Unlock()
		results := make(chan error, 2)
		go func() { results <- consume(secret) }()
		go func() {
			defer second.Destroy()
			results <- consume(second)
		}()
		firstErr, secondErr := <-results, <-results
		if firstErr != nil {
			return firstErr
		}
		return secondErr
	}
	callbackErr := consume(secret)
	if twice {
		second, err := NewSecret(material)
		if err != nil {
			return err
		}
		source.mu.Lock()
		source.retained = append(source.retained, second)
		source.mu.Unlock()
		secondErr := consume(second)
		second.Destroy()
		if secondErr != nil {
			return secondErr
		}
	}
	if afterCallback != nil {
		afterCallback()
	}
	if after != nil {
		return after
	}
	return callbackErr
}

func (source *provisionFakeSource) callCount() int {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.calls
}

func (source *provisionFakeSource) retainedSecrets() []Secret {
	source.mu.Lock()
	defer source.mu.Unlock()
	return append([]Secret(nil), source.retained...)
}

func (source *provisionFakeSource) invokeCaptured(material []byte) (Secret, error) {
	source.mu.Lock()
	callback := source.captured
	source.mu.Unlock()
	secret, err := NewSecret(material)
	if err != nil {
		return Secret{}, err
	}
	if callback == nil {
		secret.Destroy()
		return Secret{}, errors.New("callback not captured")
	}
	retained := secret
	return retained, callback(secret)
}

type provisionFakeRecord struct {
	metadata Metadata
	material []byte
}

type provisionFakeStore struct {
	mu                    sync.Mutex
	records               map[CredentialRef]provisionFakeRecord
	created               []CredentialRef
	describes             []DescribeUseAuthorityRequest
	uses                  []UseRequest
	requestSecrets        []Secret
	createCollision       ErrorCode
	invalidCreateResult   bool
	mutateCreateResult    func(*MutationResult)
	invalidDescribeResult bool
	invalidUseReceipt     bool
	useOmitsCallback      bool
	useCallsTwice         bool
	captureUseThenPanic   string
	capturedUse           func(Secret) error
	lastUseConsumeErr     error
	createError           error
	panicCreate           string
	createHook            func() error
	createHookErr         error
	cancelCreateRef       CredentialRef
	cancelCreate          func()
	cancelDescribeRef     CredentialRef
	cancelDescribeScope   ScopeRef
	cancelDescribe        func()
	cancelBeforeUse       func()
}

func newProvisionFakeStore() *provisionFakeStore {
	return &provisionFakeStore{records: make(map[CredentialRef]provisionFakeRecord)}
}

func (store *provisionFakeStore) Create(_ context.Context, request CreateRequest) (MutationResult, error) {
	if err := ValidateCreateRequest(request); err != nil {
		return MutationResult{}, err
	}
	store.mu.Lock()
	hook := store.createHook
	store.createHook = nil
	store.mu.Unlock()
	if hook != nil {
		hookErr := hook()
		store.mu.Lock()
		store.createHookErr = hookErr
		store.mu.Unlock()
	}
	store.mu.Lock()
	store.requestSecrets = append(store.requestSecrets, request.Material)
	panicValue, createErr := store.panicCreate, store.createError
	if panicValue != "" {
		store.mu.Unlock()
		panic(panicValue)
	}
	if createErr != nil {
		store.mu.Unlock()
		return MutationResult{}, createErr
	}
	if _, exists := store.records[request.CredentialRef]; exists {
		store.mu.Unlock()
		return MutationResult{}, NewError(ErrorAlreadyExists, "credential_ref")
	}
	material := request.Material.Bytes()
	metadata := Metadata{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRefs: append([]ScopeRef(nil), request.ScopeRefs...), PurposeRef: request.PurposeRef,
		Version: 1, CreatedAt: provisionTestTime,
	}
	store.records[request.CredentialRef] = provisionFakeRecord{metadata: metadata, material: material}
	store.created = append(store.created, request.CredentialRef)
	collision, invalid := store.createCollision, store.invalidCreateResult
	cancelCreate := store.cancelCreateRef == request.CredentialRef && store.cancelCreate != nil
	cancel := store.cancelCreate
	if cancelCreate {
		store.cancelCreate = nil
	}
	store.mu.Unlock()
	if collision != "" {
		return MutationResult{}, NewError(collision, "credential_ref")
	}
	receipt := provisionReceipt(request.ActorRef, request.RequestRef, "create", metadata, "")
	if invalid {
		receipt.OwnerRef = "owner:adversarial"
	}
	if cancelCreate {
		cancel()
	}
	result := MutationResult{Metadata: metadata, Receipt: receipt}
	if store.mutateCreateResult != nil {
		store.mutateCreateResult(&result)
	}
	return result, nil
}

func (store *provisionFakeStore) DescribeUseAuthority(
	_ context.Context,
	request DescribeUseAuthorityRequest,
) (DescribedUseAuthority, error) {
	if err := ValidateDescribeUseAuthorityRequest(request); err != nil {
		return DescribedUseAuthority{}, err
	}
	store.mu.Lock()
	store.describes = append(store.describes, request)
	record, found := store.records[request.CredentialRef]
	invalid := store.invalidDescribeResult
	cancelDescribe := store.cancelDescribeRef == request.CredentialRef &&
		store.cancelDescribeScope == request.ScopeRef && store.cancelDescribe != nil
	cancel := store.cancelDescribe
	if cancelDescribe {
		store.cancelDescribe = nil
	}
	store.mu.Unlock()
	if !found {
		return DescribedUseAuthority{}, NewError(ErrorNotFound, "credential_ref")
	}
	if err := authorizeProvisionFake(record.metadata, request.OwnerRef, request.ScopeRef, request.PurposeRef); err != nil {
		return DescribedUseAuthority{}, err
	}
	result := DescribedUseAuthority{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: record.metadata.Version,
	}
	if invalid {
		result.OwnerRef = "owner:adversarial"
	}
	if cancelDescribe {
		cancel()
	}
	return result, nil
}

func (store *provisionFakeStore) Use(_ context.Context, request UseRequest, consume func(Secret) error) (Receipt, error) {
	if err := ValidateUseRequest(request); err != nil {
		return Receipt{}, err
	}
	store.mu.Lock()
	record, found := store.records[request.CredentialRef]
	store.uses = append(store.uses, request)
	omit, twice, invalid := store.useOmitsCallback, store.useCallsTwice, store.invalidUseReceipt
	captureUsePanic := store.captureUseThenPanic
	if captureUsePanic != "" {
		store.capturedUse = consume
	}
	cancelBeforeUse := store.cancelBeforeUse
	store.cancelBeforeUse = nil
	store.mu.Unlock()
	if !found {
		return Receipt{}, NewError(ErrorNotFound, "credential_ref")
	}
	if request.Version != record.metadata.Version {
		return Receipt{}, NewError(ErrorVersionConflict, "version")
	}
	if err := authorizeProvisionFake(record.metadata, request.OwnerRef, request.ScopeRef, request.PurposeRef); err != nil {
		return Receipt{}, err
	}
	if captureUsePanic != "" {
		panic(captureUsePanic)
	}
	if omit {
		return provisionReceipt(request.ActorRef, request.RequestRef, "use", record.metadata, request.ScopeRef), nil
	}
	if cancelBeforeUse != nil {
		cancelBeforeUse()
	}
	invoke := func() error {
		secret, err := NewSecret(record.material)
		if err != nil {
			return err
		}
		defer secret.Destroy()
		consumeErr := consume(secret)
		store.mu.Lock()
		store.lastUseConsumeErr = consumeErr
		store.mu.Unlock()
		return consumeErr
	}
	if err := invoke(); err != nil {
		return Receipt{}, err
	}
	if twice {
		if err := invoke(); err != nil {
			return Receipt{}, err
		}
	}
	receipt := provisionReceipt(request.ActorRef, request.RequestRef, "use", record.metadata, request.ScopeRef)
	if invalid {
		receipt.Version++
	}
	return receipt, nil
}

func (*provisionFakeStore) Rotate(context.Context, RotateRequest) (MutationResult, error) {
	return MutationResult{}, errors.New("unused")
}

func (*provisionFakeStore) Revoke(context.Context, RevokeRequest) (MutationResult, error) {
	return MutationResult{}, errors.New("unused")
}

func (store *provisionFakeStore) seed(spec ProvisionCredentialRequest, owner OwnerRef, material []byte, version Version) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.records[spec.CredentialRef] = provisionFakeRecord{
		metadata: Metadata{
			CredentialRef: spec.CredentialRef, OwnerRef: owner, ScopeRefs: canonicalProvisionTestScopes(spec.ScopeRefs),
			PurposeRef: spec.PurposeRef, Version: version, CreatedAt: provisionTestTime,
		},
		material: append([]byte(nil), material...),
	}
}

func (store *provisionFakeStore) addScope(ref CredentialRef, scope ScopeRef) {
	store.mu.Lock()
	defer store.mu.Unlock()
	record := store.records[ref]
	record.metadata.ScopeRefs = append(record.metadata.ScopeRefs, scope)
	store.records[ref] = record
}

func (store *provisionFakeStore) createdCredentialRefs() []CredentialRef {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]CredentialRef(nil), store.created...)
}

func (store *provisionFakeStore) useRequestsSnapshot() []UseRequest {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]UseRequest(nil), store.uses...)
}

func (store *provisionFakeStore) describeRequestsSnapshot() []DescribeUseAuthorityRequest {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]DescribeUseAuthorityRequest(nil), store.describes...)
}

func (store *provisionFakeStore) effectCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.created) + len(store.uses)
}

func (store *provisionFakeStore) createHookError() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.createHookErr
}

func (store *provisionFakeStore) useConsumeError() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.lastUseConsumeErr
}

func (store *provisionFakeStore) invokeCapturedUse(material []byte) (Secret, error) {
	store.mu.Lock()
	callback := store.capturedUse
	store.mu.Unlock()
	secret, err := NewSecret(material)
	if err != nil {
		return Secret{}, err
	}
	if callback == nil {
		secret.Destroy()
		return Secret{}, errors.New("use callback not captured")
	}
	retained := secret
	return retained, callback(secret)
}

func (store *provisionFakeStore) assertRequestSecretsDestroyed(t *testing.T) {
	t.Helper()
	store.mu.Lock()
	retained := append([]Secret(nil), store.requestSecrets...)
	store.mu.Unlock()
	for _, secret := range retained {
		copy := secret.Bytes()
		if len(bytes.Trim(copy, "\x00")) != 0 {
			clear(copy)
			t.Fatal("Store.Create request secret survived callback")
		}
		clear(copy)
	}
}

func (store *provisionFakeStore) destroy() {
	store.mu.Lock()
	defer store.mu.Unlock()
	for ref, record := range store.records {
		clear(record.material)
		record.material = nil
		store.records[ref] = record
	}
}

func authorizeProvisionFake(metadata Metadata, owner OwnerRef, scope ScopeRef, purpose PurposeRef) error {
	if metadata.OwnerRef != owner {
		return NewError(ErrorOwnerMismatch, "owner_ref")
	}
	if metadata.PurposeRef != purpose {
		return NewError(ErrorPurposeDenied, "purpose_ref")
	}
	found := false
	for _, candidate := range metadata.ScopeRefs {
		if candidate == scope {
			found = true
			break
		}
	}
	if !found {
		return NewError(ErrorScopeDenied, "scope_ref")
	}
	if metadata.Revoked {
		return NewError(ErrorRevoked, "credential_ref")
	}
	return nil
}

func provisionReceipt(actor, request, operation string, metadata Metadata, scope ScopeRef) Receipt {
	return Receipt{
		CredentialRef: metadata.CredentialRef, OwnerRef: metadata.OwnerRef,
		ScopeRef: scope, PurposeRef: metadata.PurposeRef, Version: metadata.Version,
		RequestRef: request, ActorRef: actor, Operation: operation, OccurredAt: provisionTestTime,
	}
}

func canonicalProvisionTestScopes(values []ScopeRef) []ScopeRef {
	result := append([]ScopeRef(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

var provisionTestTime = time.Unix(1_800_000_000, 0).UTC()

var _ Store = (*provisionFakeStore)(nil)
var _ UseAuthorityReader = (*provisionFakeStore)(nil)
var _ MaterialSource = (*provisionFakeSource)(nil)
var _ Ed25519PrivateKeySource = (*provisionFakeSource)(nil)
