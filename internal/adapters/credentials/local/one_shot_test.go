package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"orquesta/internal/credentials"
)

func TestUseOncePersistsBeforeCallbackAndRejectsExactReplay(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	t.Cleanup(func() { _ = store.Close() })
	material := []byte("one-shot-success-secret")
	testCreate(t, store, testCredential, "request:create-one-shot-success", material)
	request := testOneShotRequest("actor:first-execution", "request:one-shot-success", 1)

	called := 0
	claimedBeforeCallback := false
	result, err := store.UseOnce(context.Background(), request, func(secret credentials.Secret) error {
		called++
		assertSecretBytes(t, secret, material)
		reopened := testOpen(t, path, nil)
		defer reopened.Close()
		nestedCalled := false
		replayed, replayErr := reopened.UseOnce(context.Background(), request, func(credentials.Secret) error {
			nestedCalled = true
			return nil
		})
		if !credentials.HasErrorCode(replayErr, credentials.ErrorAlreadyConsumed) || !replayed.Replayed || nestedCalled {
			t.Fatalf("claim was not durable before callback: result=%+v called=%v err=%v", replayed, nestedCalled, replayErr)
		}
		claimedBeforeCallback = true
		return nil
	})
	if err != nil || called != 1 || !claimedBeforeCallback || result.Replayed ||
		credentials.ValidateOneShotUseResult(request, result) != nil {
		t.Fatalf("first UseOnce result=%+v called=%d durable=%v err=%v", result, called, claimedBeforeCallback, err)
	}

	replayed, err := store.UseOnce(context.Background(), request, func(credentials.Secret) error {
		called++
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replayed.Replayed ||
		replayed.Receipt != result.Receipt || called != 1 {
		t.Fatalf("exact replay result=%+v called=%d err=%v", replayed, called, err)
	}

	independent := testOneShotRequest("actor:second-execution", "request:one-shot-independent", 1)
	second, err := store.UseOnce(context.Background(), independent, func(secret credentials.Secret) error {
		called++
		assertSecretBytes(t, secret, material)
		return nil
	})
	if err != nil || second.Replayed || called != 2 {
		t.Fatalf("independent claim was made global: result=%+v called=%d err=%v", second, called, err)
	}
}

func TestUseOnceConsumerFailureRemainsConsumedAcrossRestart(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	material := []byte("one-shot-consumer-secret")
	testCreate(t, store, testCredential, "request:create-one-shot-failure", material)
	request := testOneShotRequest("actor:failed-execution", "request:one-shot-failure", 1)

	called := 0
	result, err := store.UseOnce(context.Background(), request, func(credentials.Secret) error {
		called++
		return errors.New("safe consumer failure")
	})
	if !credentials.HasErrorCode(err, credentials.ErrorConsumerFailed) || called != 1 || result.Replayed ||
		credentials.ValidateOneShotUseResult(request, result) != nil {
		t.Fatalf("consumer failure result=%+v called=%d err=%v", result, called, err)
	}
	_ = store.Close()

	reopened := testOpen(t, path, nil)
	defer reopened.Close()
	replay, err := reopened.UseOnce(context.Background(), request, func(credentials.Secret) error {
		called++
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replay.Replayed ||
		replay.Receipt != result.Receipt || called != 1 {
		t.Fatalf("failed callback was reusable after restart: result=%+v called=%d err=%v", replay, called, err)
	}

	leaking := testOneShotRequest("actor:leaking-execution", "request:one-shot-leaking-error", 1)
	leakResult, err := reopened.UseOnce(context.Background(), leaking, func(secret credentials.Secret) error {
		return errors.New(string(secret.Bytes()))
	})
	if !credentials.HasErrorCode(err, credentials.ErrorSecretLeak) || leakResult.Replayed ||
		bytes.Contains([]byte(err.Error()), material) {
		t.Fatalf("consumer error leaked material: result=%+v err=%v", leakResult, err)
	}
	replay, err = reopened.UseOnce(context.Background(), leaking, func(credentials.Secret) error {
		t.Fatal("leaking callback claim was reused")
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replay.Replayed {
		t.Fatalf("leaking callback was not consumed: result=%+v err=%v", replay, err)
	}
}

func TestUseOnceConflictingTupleNeverInvokesCallback(t *testing.T) {
	store := testOpen(t, testStorePath(t), nil)
	defer store.Close()
	create := testCreateRequest(t, testCredential, "request:create-one-shot-conflict", []byte("conflict-secret"))
	create.ScopeRefs = []credentials.ScopeRef{testScope, "scope:second"}
	if _, err := store.Create(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	request := testOneShotRequest("actor:conflict", "request:one-shot-conflict", 1)
	if _, err := store.UseOnce(context.Background(), request, func(credentials.Secret) error { return nil }); err != nil {
		t.Fatal(err)
	}

	conflict := request
	conflict.ScopeRef = "scope:second"
	called := false
	result, err := store.UseOnce(context.Background(), conflict, func(credentials.Secret) error {
		called = true
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorIdempotencyConflict) || called || result.Receipt.CredentialRef != "" {
		t.Fatalf("tuple conflict result=%+v called=%v err=%v", result, called, err)
	}
}

func TestUseOnceRaceAcrossStoresHasOneCallback(t *testing.T) {
	path := testStorePath(t)
	first := testOpen(t, path, nil)
	testCreate(t, first, testCredential, "request:create-one-shot-race", []byte("race-secret"))
	second := testOpen(t, path, nil)
	defer first.Close()
	defer second.Close()
	request := testOneShotRequest("actor:race", "request:one-shot-race", 1)

	start := make(chan struct{})
	type outcome struct {
		result credentials.OneShotUseResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var ready sync.WaitGroup
	var callbacks atomic.Int32
	ready.Add(2)
	for _, store := range []*Store{first, second} {
		go func(store *Store) {
			ready.Done()
			<-start
			result, err := store.UseOnce(context.Background(), request, func(credentials.Secret) error {
				callbacks.Add(1)
				return nil
			})
			outcomes <- outcome{result: result, err: err}
		}(store)
	}
	ready.Wait()
	close(start)
	winners, replays := 0, 0
	for range 2 {
		result := <-outcomes
		switch {
		case result.err == nil && !result.result.Replayed:
			winners++
		case credentials.HasErrorCode(result.err, credentials.ErrorAlreadyConsumed) && result.result.Replayed:
			replays++
		default:
			t.Fatalf("unexpected race outcome: result=%+v err=%v", result.result, result.err)
		}
	}
	if winners != 1 || replays != 1 || callbacks.Load() != 1 {
		t.Fatalf("race winners=%d replays=%d callbacks=%d", winners, replays, callbacks.Load())
	}
}

func TestUseOnceAmbiguousPersistenceNeverInvokesCallbackOrRetries(t *testing.T) {
	for _, point := range []string{FailpointAfterReplacementSync, FailpointAfterStoreRename, FailpointAfterStoreDirectorySync} {
		t.Run(point, func(t *testing.T) {
			path := testStorePath(t)
			seed := testOpen(t, path, nil)
			testCreate(t, seed, testCredential, "request:create-one-shot-failpoint", []byte("failpoint-secret"))
			_ = seed.Close()
			failed := testOpen(t, path, func(current string) error {
				if current == point {
					return errors.New("stop")
				}
				return nil
			})
			request := testOneShotRequest("actor:failpoint", "request:one-shot-failpoint", 1)
			called := 0
			result, err := failed.UseOnce(context.Background(), request, func(credentials.Secret) error {
				called++
				return nil
			})
			if !credentials.HasErrorCode(err, credentials.ErrorStoreIO) || called != 0 || result.Receipt.CredentialRef != "" {
				t.Fatalf("ambiguous persist result=%+v called=%d err=%v", result, called, err)
			}
			_ = failed.Close()

			reopened := testOpen(t, path, nil)
			defer reopened.Close()
			replay, err := reopened.UseOnce(context.Background(), request, func(credentials.Secret) error {
				called++
				return nil
			})
			if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replay.Replayed || called != 0 {
				t.Fatalf("ambiguous claim retried result=%+v called=%d err=%v", replay, called, err)
			}
		})
	}
}

func TestUseOnceIsVersionExactAndRotationCreatesNewClaims(t *testing.T) {
	store := testOpen(t, testStorePath(t), nil)
	defer store.Close()
	testCreate(t, store, testCredential, "request:create-one-shot-rotate", []byte("version-one-secret"))
	versionOne := testOneShotRequest("actor:version-one", "request:one-shot-version-one", 1)
	if _, err := store.UseOnce(context.Background(), versionOne, func(credentials.Secret) error { return nil }); err != nil {
		t.Fatal(err)
	}
	rotate := credentials.RotateRequest{ActorRef: "actor:test", RequestRef: "request:rotate-one-shot", CredentialRef: testCredential,
		OwnerRef: testOwner, ExpectedVersion: 1, Material: testSecret(t, []byte("version-two-secret"))}
	defer rotate.Material.Destroy()
	if _, err := store.Rotate(context.Background(), rotate); err != nil {
		t.Fatal(err)
	}

	called := 0
	replayed, err := store.UseOnce(context.Background(), versionOne, func(credentials.Secret) error {
		called++
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replayed.Replayed || called != 0 {
		t.Fatalf("exact v1 replay after rotation result=%+v called=%d err=%v", replayed, called, err)
	}
	stale := testOneShotRequest("actor:stale-version-one", "request:stale-version-one", 1)
	if result, err := store.UseOnce(context.Background(), stale, func(credentials.Secret) error {
		called++
		return nil
	}); !credentials.HasErrorCode(err, credentials.ErrorVersionConflict) || called != 0 || result.Receipt.CredentialRef != "" {
		t.Fatalf("new stale claim result=%+v called=%d err=%v", result, called, err)
	}
	versionTwoClaims := make([]credentials.OneShotUseRequest, 0, 2)
	for index, actor := range []string{"actor:version-two-a", "actor:version-two-b"} {
		request := testOneShotRequest(actor, "request:one-shot-version-two-"+string(rune('a'+index)), 2)
		versionTwoClaims = append(versionTwoClaims, request)
		if result, err := store.UseOnce(context.Background(), request, func(secret credentials.Secret) error {
			called++
			assertSecretBytes(t, secret, []byte("version-two-secret"))
			return nil
		}); err != nil || result.Replayed {
			t.Fatalf("version two claim %d result=%+v err=%v", index, result, err)
		}
	}
	if called != 2 {
		t.Fatalf("version two callbacks=%d", called)
	}
	if _, err := store.Revoke(context.Background(), credentials.RevokeRequest{ActorRef: "actor:test", RequestRef: "request:revoke-one-shot",
		CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 2, Reason: "test"}); err != nil {
		t.Fatal(err)
	}
	replayed, err = store.UseOnce(context.Background(), versionTwoClaims[0], func(credentials.Secret) error {
		called++
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replayed.Replayed || called != 2 {
		t.Fatalf("exact replay after revoke result=%+v called=%d err=%v", replayed, called, err)
	}
	revoked := testOneShotRequest("actor:revoked-new-claim", "request:revoked-new-claim", 2)
	if result, err := store.UseOnce(context.Background(), revoked, func(credentials.Secret) error {
		called++
		return nil
	}); !credentials.HasErrorCode(err, credentials.ErrorRevoked) || result.Receipt.CredentialRef != "" || called != 2 {
		t.Fatalf("new revoked claim result=%+v called=%d err=%v", result, called, err)
	}
}

func TestUseOnceLazilyMigratesV1AndRejectsOneShotReceiptInV1(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	testCreate(t, store, testCredential, "request:create-v1-migration", []byte("migration-secret"))
	_ = store.Close()
	if schema := readDocumentSchema(t, path); schema != documentSchemaLegacy {
		t.Fatalf("legacy lifecycle created schema=%d", schema)
	}
	beforeOpen := testStoreSnapshot(t, path)

	store = testOpen(t, path, nil)
	afterOpen := testStoreSnapshot(t, path)
	if !bytes.Equal(afterOpen, beforeOpen) {
		t.Fatal("opening schema v1 rewrote the credential document")
	}
	legacyCalled := 0
	legacy := testUseRequest(testCredential, "request:legacy-use-on-v1", 1)
	if _, err := store.Use(context.Background(), legacy, func(credentials.Secret) error {
		legacyCalled++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if schema := readDocumentSchema(t, path); schema != documentSchemaLegacy {
		t.Fatalf("legacy Use migrated schema=%d", schema)
	}

	request := testOneShotRequest("actor:migration", "request:one-shot-migration", 1)
	if _, err := store.UseOnce(context.Background(), request, func(credentials.Secret) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if schema := readDocumentSchema(t, path); schema != documentSchema {
		t.Fatalf("UseOnce did not migrate schema=%d", schema)
	}
	_ = store.Close()

	reopened := testOpen(t, path, nil)
	replay, err := reopened.UseOnce(context.Background(), request, func(credentials.Secret) error {
		t.Fatal("migrated one-shot claim was reused")
		return nil
	})
	if !credentials.HasErrorCode(err, credentials.ErrorAlreadyConsumed) || !replay.Replayed || legacyCalled != 1 {
		t.Fatalf("migrated replay result=%+v legacy_called=%d err=%v", replay, legacyCalled, err)
	}
	_ = reopened.Close()

	rewriteDocumentSchema(t, path, documentSchemaLegacy)
	if rejected, err := Open(testOptions(path, nil)); !credentials.HasErrorCode(err, credentials.ErrorUnsafeFile) {
		if rejected != nil {
			_ = rejected.Close()
		}
		t.Fatalf("schema v1 accepted a use_once receipt: %v", err)
	}
}

func testOneShotRequest(actor, request string, version credentials.Version) credentials.OneShotUseRequest {
	return credentials.OneShotUseRequest{ActorRef: actor, RequestRef: request, CredentialRef: testCredential,
		OwnerRef: testOwner, ScopeRef: testScope, PurposeRef: testPurpose, Version: version}
}

func assertSecretBytes(t *testing.T, secret credentials.Secret, want []byte) {
	t.Helper()
	got := secret.Bytes()
	defer clear(got)
	if !bytes.Equal(got, want) {
		t.Fatalf("secret mismatch: got=%q", got)
	}
}

func rewriteDocumentSchema(t *testing.T, path string, schema int) {
	t.Helper()
	payload := testStoreSnapshot(t, path)
	defer clear(payload)
	var doc document
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatal(err)
	}
	defer clearDocument(&doc)
	doc.SchemaVersion = schema
	next, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	next = append(next, '\n')
	defer clear(next)
	if err := os.WriteFile(path, next, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readDocumentSchema(t *testing.T, path string) int {
	t.Helper()
	payload := testStoreSnapshot(t, path)
	defer clear(payload)
	var header struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(payload, &header); err != nil {
		t.Fatal(err)
	}
	return header.SchemaVersion
}
