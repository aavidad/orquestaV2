package orquestaappcodexstack

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestFileDomainWorkArtifactSubmissionLedgerV0PersisteYRecupera(t *testing.T) {
	path := filepath.Join(t.TempDir(), "domain-work-ledger.json")
	ledger := NewFileDomainWorkArtifactSubmissionLedgerV0(path)

	if ok, err := ledger.HasDomainWorkArtifactSubmissionV0(context.Background(), "idem-001"); err != nil || ok {
		t.Fatalf("has inicial ok=%v err=%v", ok, err)
	}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordV0{
			IdempotencyKey: "idem-001",
			RunRef:         "run-ref-001",
			TaskRef:        "task-ref-001",
			DeliveryRef:    "delivery-ref-001",
			ReceiptRef:     "receipt-ref-001",
		},
	); err != nil {
		t.Fatalf("record: %v", err)
	}

	reopened := NewFileDomainWorkArtifactSubmissionLedgerV0(path)
	if ok, err := reopened.HasDomainWorkArtifactSubmissionV0(context.Background(), " idem-001 "); err != nil || !ok {
		t.Fatalf("has reopened ok=%v err=%v", ok, err)
	}
}

func TestFileDomainWorkArtifactSubmissionLedgerV0RespetaContextoCancelado(t *testing.T) {
	ledger := NewFileDomainWorkArtifactSubmissionLedgerV0(filepath.Join(t.TempDir(), "ledger.json"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := ledger.HasDomainWorkArtifactSubmissionV0(ctx, "idem-001"); !errors.Is(err, context.Canceled) {
		t.Fatalf("has err=%v", err)
	}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(
		ctx,
		DomainWorkArtifactSubmissionRecordV0{IdempotencyKey: "idem-001"},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("record err=%v", err)
	}
}
