package orquestaappcodexstack

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
			Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
			RunRef:         "run-ref-001",
			TaskRef:        "task-ref-001",
			DeliveryRef:    "delivery-ref-001",
			ReceiptRef:     "receipt-ref-001",
			EvidenceRefs:   []string{"evidence-ref-001"},
		},
	); err != nil {
		t.Fatalf("record: %v", err)
	}

	reopened := NewFileDomainWorkArtifactSubmissionLedgerV0(path)
	if ok, err := reopened.HasDomainWorkArtifactSubmissionV0(context.Background(), " idem-001 "); err != nil || !ok {
		t.Fatalf("has reopened ok=%v err=%v", ok, err)
	}
	records, err := reopened.ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef:      "run-ref-001",
			TaskRef:     "task-ref-001",
			DeliveryRef: "delivery-ref-001",
			Status:      DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil || len(records) != 1 || records[0].ReceiptRef != "receipt-ref-001" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	assertAppStackDurableFilePolicyV0(t, filepath.Dir(path), filepath.Base(path))
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

func TestFileDomainWorkArtifactSubmissionLedgerV0LimitaLecturaSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", int(fileDomainWorkArtifactSubmissionLedgerMaxBytesV0)+1)), 0o600); err != nil {
		t.Fatalf("write oversized ledger: %v", err)
	}
	ledger := NewFileDomainWorkArtifactSubmissionLedgerV0(path)
	_, err := ledger.HasDomainWorkArtifactSubmissionV0(context.Background(), "idem-001")
	if err == nil || err.Error() != "domain_work_artifact_file_ledger_size_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
}

func TestFileDomainWorkArtifactSubmissionLedgerV0LimitaRecordsSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	records := make([]string, 0, fileDomainWorkArtifactSubmissionLedgerMaxRecordsV0+1)
	for index := 0; index <= fileDomainWorkArtifactSubmissionLedgerMaxRecordsV0; index++ {
		records = append(records, `{"idempotency_key":"idem-`+string(rune('a'+index%26))+`"}`)
	}
	body := `{"schema_version":"` + fileDomainWorkArtifactSubmissionLedgerSchemaV0 + `","records":[` +
		strings.Join(records, ",") + `]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write ledger: %v", err)
	}
	ledger := NewFileDomainWorkArtifactSubmissionLedgerV0(path)
	_, err := ledger.HasDomainWorkArtifactSubmissionV0(context.Background(), "idem-a")
	if err == nil || err.Error() != "domain_work_artifact_file_ledger_records_limit_exceeded" {
		t.Fatalf("err=%v", err)
	}
}

func assertAppStackDurableFilePolicyV0(t *testing.T, dir string, name string) {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("stat ledger: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("ledger mode=%#o", got)
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "."+name+".*.tmp"))
	if err != nil {
		t.Fatalf("glob temp: %v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temps persistidos=%v", leftovers)
	}
}

func TestInMemoryDomainWorkArtifactSubmissionLedgerV0ClaimYConflicto(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	base := DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-claim-001",
		RunRef:         "run-ref-claim-001",
		TaskRef:        "task-ref-claim-001",
		DeliveryRef:    "delivery-ref-claim-001",
		DomainRef:      "domain-ref-claim",
		JobRef:         "job-ref-claim",
		ArtifactRef:    "artifact-ref-claim",
		ArtifactType:   "content_block",
	}
	for _, status := range []string{
		DomainWorkArtifactSubmissionStatusClaimedV0,
		DomainWorkArtifactSubmissionStatusSubmittingV0,
		DomainWorkArtifactSubmissionStatusAcceptedV0,
	} {
		record := base
		record.Status = status
		if status == DomainWorkArtifactSubmissionStatusAcceptedV0 {
			record.ReceiptRef = "receipt-ref-claim-001"
		}
		if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
			t.Fatalf("record status=%s err=%v", status, err)
		}
	}
	conflict := base
	conflict.Status = DomainWorkArtifactSubmissionStatusAcceptedV0
	conflict.ReceiptRef = "receipt-ref-distinto"
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, conflict); err == nil ||
		!strings.Contains(err.Error(), "domain_work_submit_conflict") {
		t.Fatalf("conflict err=%v", err)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(ctx, DomainWorkArtifactSubmissionRecordFilterV0{
		IdempotencyKey: "idem-claim-001",
	})
	if err != nil || len(records) != 1 || records[0].ReceiptRef != "receipt-ref-claim-001" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestInMemoryDomainWorkArtifactSubmissionLedgerV0PermiteRecuperarRejected(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	base := DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-rejected-retry-001",
		RunRef:         "run-ref-rejected-retry-001",
		TaskRef:        "task-ref-rejected-retry-001",
		DeliveryRef:    "delivery-ref-rejected-retry-001",
		DomainRef:      "opes",
		JobRef:         "job-ref-rejected-retry",
		ArtifactRef:    "artifact-ref-rejected-retry",
		ArtifactType:   "visual_asset",
	}
	rejected := base
	rejected.Status = DomainWorkArtifactSubmissionStatusRejectedV0
	rejected.IssueRefs = []string{"domain-work-submit-artifact-rejected"}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, rejected); err != nil {
		t.Fatalf("record rejected: %v", err)
	}
	submitting := base
	submitting.Status = DomainWorkArtifactSubmissionStatusSubmittingV0
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, submitting); err != nil {
		t.Fatalf("record submitting tras rejected: %v", err)
	}
	accepted := base
	accepted.Status = DomainWorkArtifactSubmissionStatusAcceptedV0
	accepted.ReceiptRef = "receipt-ref-rejected-retry"
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, accepted); err != nil {
		t.Fatalf("record accepted tras retry: %v", err)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(ctx, DomainWorkArtifactSubmissionRecordFilterV0{
		IdempotencyKey: "idem-rejected-retry-001",
	})
	if err != nil || len(records) != 1 ||
		records[0].Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
		records[0].ReceiptRef != "receipt-ref-rejected-retry" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestInMemoryDomainWorkArtifactSubmissionLedgerV0PermiteRecuperarSubmitting(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	base := DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-submitting-retry-001",
		RunRef:         "run-ref-submitting-retry-001",
		TaskRef:        "task-ref-submitting-retry-001",
		DeliveryRef:    "delivery-ref-submitting-retry-001",
		DomainRef:      "opes",
		JobRef:         "job-ref-submitting-retry",
		ArtifactRef:    "artifact-ref-submitting-retry",
		ArtifactType:   "visual_asset",
	}
	for _, status := range []string{
		DomainWorkArtifactSubmissionStatusSubmittingV0,
		DomainWorkArtifactSubmissionStatusRejectedV0,
		DomainWorkArtifactSubmissionStatusClaimedV0,
		DomainWorkArtifactSubmissionStatusSubmittingV0,
		DomainWorkArtifactSubmissionStatusAcceptedV0,
	} {
		record := base
		record.Status = status
		if status == DomainWorkArtifactSubmissionStatusRejectedV0 {
			record.IssueRefs = []string{"domain-work-submit-execute-error"}
		}
		if status == DomainWorkArtifactSubmissionStatusAcceptedV0 {
			record.ReceiptRef = "receipt-ref-submitting-retry"
		}
		if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
			t.Fatalf("record status=%s: %v", status, err)
		}
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(ctx, DomainWorkArtifactSubmissionRecordFilterV0{
		IdempotencyKey: "idem-submitting-retry-001",
	})
	if err != nil || len(records) != 1 ||
		records[0].Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
		records[0].ReceiptRef != "receipt-ref-submitting-retry" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}
