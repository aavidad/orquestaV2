package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"orquesta/internal/credentials"
)

var _ credentials.UseAuthorityReader = (*Store)(nil)

func TestDescribeUseAuthorityReturnsCurrentTupleWithoutMutatingStore(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	defer store.Close()
	testCreate(t, store, testCredential, "request:create-describe", []byte("describe-secret"))

	before := testStoreSnapshot(t, path)
	defer clear(before)
	beforeDoc, err := decodeDocument(before)
	if err != nil {
		t.Fatal(err)
	}
	defer clearDocument(&beforeDoc)
	store.now = func() time.Time { panic("DescribeUseAuthority called operationTime") }

	request := testDescribeUseAuthorityRequest(testCredential)
	result, err := store.DescribeUseAuthority(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	want := credentials.DescribedUseAuthority{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: 1,
	}
	if result != want || credentials.ValidateDescribedUseAuthority(request, result) != nil {
		t.Fatalf("authority=%+v want=%+v", result, want)
	}
	after := testStoreSnapshot(t, path)
	defer clear(after)
	if !bytes.Equal(after, before) {
		t.Fatal("DescribeUseAuthority changed durable bytes")
	}
	afterDoc, err := decodeDocument(after)
	if err != nil {
		t.Fatal(err)
	}
	defer clearDocument(&afterDoc)
	if afterDoc.Revision != beforeDoc.Revision || len(afterDoc.Ledger) != len(beforeDoc.Ledger) {
		t.Fatalf("revision/ledger changed: before=%d/%d after=%d/%d",
			beforeDoc.Revision, len(beforeDoc.Ledger), afterDoc.Revision, len(afterDoc.Ledger))
	}
}

func TestDescribeUseAuthorityEnforcesAuthorizationAndRevocation(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	defer store.Close()
	testCreate(t, store, testCredential, "request:create-describe-auth", []byte("describe-auth-secret"))
	valid := testDescribeUseAuthorityRequest(testCredential)

	tests := []struct {
		name   string
		mutate func(*credentials.DescribeUseAuthorityRequest)
		code   credentials.ErrorCode
	}{
		{"invalid request", func(request *credentials.DescribeUseAuthorityRequest) { request.RequestRef = "other:test" }, credentials.ErrorInvalidRef},
		{"not found", func(request *credentials.DescribeUseAuthorityRequest) { request.CredentialRef = "credential:missing" }, credentials.ErrorNotFound},
		{"owner", func(request *credentials.DescribeUseAuthorityRequest) { request.OwnerRef = "owner:other" }, credentials.ErrorOwnerMismatch},
		{"scope", func(request *credentials.DescribeUseAuthorityRequest) { request.ScopeRef = "scope:other" }, credentials.ErrorScopeDenied},
		{"purpose", func(request *credentials.DescribeUseAuthorityRequest) { request.PurposeRef = "purpose:other" }, credentials.ErrorPurposeDenied},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			result, err := store.DescribeUseAuthority(context.Background(), request)
			if result != (credentials.DescribedUseAuthority{}) || !credentials.HasErrorCode(err, test.code) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}

	if _, err := store.Revoke(context.Background(), credentials.RevokeRequest{
		ActorRef: "actor:test", RequestRef: "request:revoke-describe",
		CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 1, Reason: "test",
	}); err != nil {
		t.Fatal(err)
	}
	result, err := store.DescribeUseAuthority(context.Background(), valid)
	if result != (credentials.DescribedUseAuthority{}) || !credentials.HasErrorCode(err, credentials.ErrorRevoked) {
		t.Fatalf("revoked result=%+v err=%v", result, err)
	}
}

func TestDescribeUseAuthorityTracksRotationAndSurvivesReopen(t *testing.T) {
	path := testStorePath(t)
	store := testOpen(t, path, nil)
	testCreate(t, store, testCredential, "request:create-describe-rotate", []byte("version-one-secret"))
	request := testDescribeUseAuthorityRequest(testCredential)
	first, err := store.DescribeUseAuthority(context.Background(), request)
	if err != nil || first.Version != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	material := testSecret(t, []byte("version-two-secret"))
	defer material.Destroy()
	if _, err := store.Rotate(context.Background(), credentials.RotateRequest{
		ActorRef: "actor:test", RequestRef: "request:rotate-describe",
		CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 1, Material: material,
	}); err != nil {
		t.Fatal(err)
	}
	second, err := store.DescribeUseAuthority(context.Background(), request)
	if err != nil || second.Version != 2 || second.CredentialRef != first.CredentialRef {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	_ = store.Close()

	reopened := testOpen(t, path, nil)
	defer reopened.Close()
	again, err := reopened.DescribeUseAuthority(context.Background(), request)
	if err != nil || again != second {
		t.Fatalf("reopened=%+v want=%+v err=%v", again, second, err)
	}
}

func TestDescribeUseAuthorityIsReadOnlyAcrossConcurrentStores(t *testing.T) {
	path := testStorePath(t)
	first := testOpen(t, path, nil)
	testCreate(t, first, testCredential, "request:create-describe-concurrent", []byte("concurrent-secret"))
	second := testOpen(t, path, nil)
	defer first.Close()
	defer second.Close()
	before := testStoreSnapshot(t, path)
	defer clear(before)
	request := testDescribeUseAuthorityRequest(testCredential)

	const readers = 32
	start := make(chan struct{})
	errorsFound := make(chan error, readers)
	var ready sync.WaitGroup
	ready.Add(readers)
	for index := range readers {
		store := first
		if index%2 == 1 {
			store = second
		}
		go func(store *Store) {
			ready.Done()
			<-start
			result, err := store.DescribeUseAuthority(context.Background(), request)
			if err == nil && result.Version != 1 {
				err = errors.New("unexpected authority version")
			}
			errorsFound <- err
		}(store)
	}
	ready.Wait()
	close(start)
	for range readers {
		if err := <-errorsFound; err != nil {
			t.Fatal(err)
		}
	}
	after := testStoreSnapshot(t, path)
	defer clear(after)
	if !bytes.Equal(after, before) {
		t.Fatal("concurrent authority reads changed durable bytes")
	}
}

func TestDescribeUseAuthorityCompletesPendingNextRecoveryWithoutNewMutation(t *testing.T) {
	path := testStorePath(t)
	seed := testOpen(t, path, nil)
	testCreate(t, seed, testCredential, "request:create-describe-recovery", []byte("recovery-v1-secret"))
	_ = seed.Close()
	failed := testOpen(t, path, func(point string) error {
		if point == FailpointAfterReplacementSync {
			return errors.New("stop")
		}
		return nil
	})
	material := testSecret(t, []byte("recovery-v2-secret"))
	defer material.Destroy()
	_, err := failed.Rotate(context.Background(), credentials.RotateRequest{
		ActorRef: "actor:test", RequestRef: "request:rotate-describe-recovery",
		CredentialRef: testCredential, OwnerRef: testOwner, ExpectedVersion: 1, Material: material,
	})
	if !credentials.HasErrorCode(err, credentials.ErrorStoreIO) {
		t.Fatalf("rotate failpoint err=%v", err)
	}
	if _, err := os.Stat(path + ".next"); err != nil {
		t.Fatalf("pending next missing: %v", err)
	}
	failed.failpoint = nil
	request := testDescribeUseAuthorityRequest(testCredential)
	result, err := failed.DescribeUseAuthority(context.Background(), request)
	if err != nil || result.Version != 2 {
		t.Fatalf("recovered authority=%+v err=%v", result, err)
	}
	if _, err := os.Stat(path + ".next"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending next not installed: %v", err)
	}
	afterRecovery := testStoreSnapshot(t, path)
	defer clear(afterRecovery)
	_ = failed.Close()
	reopened := testOpen(t, path, nil)
	defer reopened.Close()
	again, err := reopened.DescribeUseAuthority(context.Background(), request)
	if err != nil || again != result {
		t.Fatalf("reopened authority=%+v want=%+v err=%v", again, result, err)
	}
	afterReopen := testStoreSnapshot(t, path)
	defer clear(afterReopen)
	if !bytes.Equal(afterReopen, afterRecovery) {
		t.Fatal("authority read persisted beyond pending recovery")
	}
}

func testDescribeUseAuthorityRequest(ref credentials.CredentialRef) credentials.DescribeUseAuthorityRequest {
	return credentials.DescribeUseAuthorityRequest{
		ActorRef: "actor:test", RequestRef: "request:describe-use-authority",
		CredentialRef: ref, OwnerRef: testOwner, ScopeRef: testScope, PurposeRef: testPurpose,
	}
}
