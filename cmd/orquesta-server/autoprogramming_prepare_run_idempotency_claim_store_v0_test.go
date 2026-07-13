package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestServerAutoprogrammingPrepareRunIdempotencyClaimStoreV0CreateOnceReplayConflictAndHashedNameV0(t *testing.T) {
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	claim := serverAutoprogrammingPrepareRunIdempotencyClaimForTestV0(t, "idem-server-claim-secret-shaped-value-001")
	first, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim)
	if err != nil || !orquestaautoprogramming.EqualAutoprogrammingPrepareRunIdempotencyClaimV0(first, replay) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	claimDir := filepath.Join(store.RootDir, "prepare-run-idempotency-claims")
	entries, err := os.ReadDir(claimDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey) || entries[0].Name() == claim.IdempotencyKey+".json" {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	claim.CommandSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim); !errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0) {
		t.Fatalf("conflict err=%v", err)
	}
}

func TestServerAutoprogrammingIntentManifestStoreV0MissingIsDistinctFromUnavailableV0(t *testing.T) {
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	if err := os.MkdirAll(store.RootDir, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := store.LoadAutoprogrammingIntentManifestV0(context.Background(), "request-ref-server-manifest-missing-001")
	if !errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingIntentManifestNotFoundV0) || errors.Is(err, errAutoprogrammingIntentManifestUnavailableV0) {
		t.Fatalf("missing err=%v", err)
	}
}

func TestServerAutoprogrammingPrepareRunIdempotencyClaimStoreV0VerifiesBodyKeyAgainstHashedNameV0(t *testing.T) {
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	claim := serverAutoprogrammingPrepareRunIdempotencyClaimForTestV0(t, "idem-server-claim-body-key-001")
	forged := claim
	forged.IdempotencyKey = "idem-server-claim-forged-001"
	raw, err := json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	claimDir := filepath.Join(store.RootDir, "prepare-run-idempotency-claims")
	if err := os.MkdirAll(claimDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(claimDir, autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey))
	if err := os.WriteFile(path, raw, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim); !errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimConflictV0) {
		t.Fatalf("forged body accepted: %v", err)
	}
}

func TestServerAutoprogrammingPrepareRunIdempotencyClaimStoreV0ConcurrentCreateLeavesOneImmutableNameV0(t *testing.T) {
	store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
	claim := serverAutoprogrammingPrepareRunIdempotencyClaimForTestV0(t, "idem-server-claim-race-001")
	const creators = 64
	start := make(chan struct{})
	errs := make(chan error, creators)
	var wait sync.WaitGroup
	for index := 0; index < creators; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim)
			errs <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	claimDir := filepath.Join(store.RootDir, "prepare-run-idempotency-claims")
	entries, err := os.ReadDir(claimDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey) {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	info, err := os.Stat(filepath.Join(claimDir, entries[0].Name()))
	if err != nil || info.Mode().Perm() != 0o400 || info.Sys().(*syscall.Stat_t).Nlink != 1 {
		t.Fatalf("info=%v err=%v", info, err)
	}
}

func TestServerAutoprogrammingPrepareRunIdempotencyClaimStoreV0RejectsUnsafePhysicalNamesV0(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(t *testing.T, path string, raw []byte)
	}{
		{name: "expanded mode", prepare: func(t *testing.T, path string, raw []byte) {
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "symlink", prepare: func(t *testing.T, path string, raw []byte) {
			target := filepath.Join(t.TempDir(), "target.json")
			if err := os.WriteFile(target, raw, 0o400); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "hardlink", prepare: func(t *testing.T, path string, raw []byte) {
			target := filepath.Join(filepath.Dir(path), "other.json")
			if err := os.WriteFile(target, raw, 0o400); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(target, path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "fifo", prepare: func(t *testing.T, path string, _ []byte) {
			if err := syscall.Mkfifo(path, 0o400); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := serverAutoprogrammingIntentManifestStoreV0{RootDir: newIntentManifestStoreRootForTestV0(t)}
			claim := serverAutoprogrammingPrepareRunIdempotencyClaimForTestV0(t, "idem-server-claim-unsafe-001")
			raw, err := json.Marshal(claim)
			if err != nil {
				t.Fatal(err)
			}
			claimDir := filepath.Join(store.RootDir, "prepare-run-idempotency-claims")
			if err := os.MkdirAll(claimDir, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(claimDir, autoprogrammingPrepareRunIdempotencyClaimNameV0(claim.IdempotencyKey))
			tc.prepare(t, path, raw)
			if _, err := store.CreateAutoprogrammingPrepareRunIdempotencyClaimIfAbsentV0(context.Background(), claim); !errors.Is(err, orquestaautoprogramming.ErrAutoprogrammingPrepareRunIdempotencyClaimUnavailableV0) {
				t.Fatalf("unsafe name accepted: %v", err)
			}
		})
	}
}

func serverAutoprogrammingPrepareRunIdempotencyClaimForTestV0(
	t *testing.T,
	idempotencyKey string,
) orquestaautoprogramming.AutoprogrammingPrepareRunIdempotencyClaimV0 {
	t.Helper()
	request := serverPrepareRunBatchWorkspaceInputV0().AutoprogrammingRequest
	request.RequestRef = "request-ref-server-claim-001"
	command, issues := orquestaautoprogramming.BuildAutoprogrammingIntentManifestFromPrepareRunEnvelopeV0(
		orquestaautoprogramming.AutoprogrammingPrepareRunEnvelopeV0{
			RequestID:              "request-server-claim-001",
			CorrelationID:          "corr-server-claim-001",
			IdempotencyKey:         idempotencyKey,
			OccurredAt:             "2026-07-13T10:00:00Z",
			RequestedBy:            "operator-ref-server-claim-001",
			AutoprogrammingRequest: request,
		},
	)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	claim, issues := orquestaautoprogramming.BuildAutoprogrammingPrepareRunIdempotencyClaimV0(idempotencyKey, command, command)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return claim
}
