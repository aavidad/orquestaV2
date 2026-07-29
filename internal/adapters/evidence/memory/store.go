package memory

import (
	"context"
	"sync"
	"time"

	"orquesta/internal/application"
)

// Store is an isolated adapter for tests and explicit ephemeral compositions.
// Productive wiring must substitute a durable adapter without changing the
// application port.
type Store struct {
	mu       sync.Mutex
	receipts map[string]application.BehaviorEvidenceReceipt
}

func New() *Store {
	return &Store{receipts: make(map[string]application.BehaviorEvidenceReceipt)}
}

func (store *Store) AcceptBehaviorEvidenceManifest(
	ctx context.Context,
	manifest application.BehaviorEvidenceManifest,
	acceptedAt time.Time,
) (application.BehaviorEvidenceReceipt, error) {
	if err := ctx.Err(); err != nil {
		return application.BehaviorEvidenceReceipt{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if receipt, found := store.receipts[manifest.ManifestDigest]; found {
		return receipt, nil
	}
	receipt := application.BehaviorEvidenceReceipt{
		Schema:         application.BehaviorEvidenceReceiptSchema,
		ReceiptRef:     "behavior-evidence-receipt-" + manifest.ManifestDigest,
		ManifestDigest: manifest.ManifestDigest,
		AcceptedAt:     acceptedAt.UTC().Format(time.RFC3339Nano),
	}
	store.receipts[manifest.ManifestDigest] = receipt
	return receipt, nil
}
