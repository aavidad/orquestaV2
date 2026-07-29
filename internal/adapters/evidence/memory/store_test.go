package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
)

func TestStoreIsConcurrentAndIdempotentByManifestDigest(t *testing.T) {
	store := New()
	manifest := application.BehaviorEvidenceManifest{
		ManifestDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	const workers = 64
	receipts := make(chan application.BehaviorEvidenceReceipt, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func(offset int) {
			defer wait.Done()
			receipt, err := store.AcceptBehaviorEvidenceManifest(
				t.Context(),
				manifest,
				time.Date(2026, 7, 29, 17, 0, offset, 0, time.UTC),
			)
			if err != nil {
				t.Errorf("accept: %v", err)
				return
			}
			receipts <- receipt
		}(index)
	}
	wait.Wait()
	close(receipts)

	var first application.BehaviorEvidenceReceipt
	for receipt := range receipts {
		if first.Schema == "" {
			first = receipt
			continue
		}
		if receipt != first {
			t.Fatalf("idempotency drift: first=%+v receipt=%+v", first, receipt)
		}
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.receipts) != 1 {
		t.Fatalf("receipts=%d", len(store.receipts))
	}
}

func TestStoreHonorsCancelledContextWithoutMutation(t *testing.T) {
	store := New()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := store.AcceptBehaviorEvidenceManifest(
		ctx,
		application.BehaviorEvidenceManifest{
			ManifestDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		time.Now(),
	); err == nil {
		t.Fatal("cancelled context accepted")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.receipts) != 0 {
		t.Fatalf("cancelled call mutated store: %d", len(store.receipts))
	}
}
