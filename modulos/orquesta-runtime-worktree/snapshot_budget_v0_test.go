package orquestaruntimeworktree

import (
	"context"
	"strings"
	"testing"
)

func TestCaptureWorktreeSnapshotV0AplicaPresupuestoDeLecturaV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "a.txt", "1234")
	writeWorktreeFileForTestV0(t, root, "b.txt", "1234")

	_, issues := CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-files",
		ProjectWorkDir: root,
		MaxFiles:       1,
	})
	requireWorktreeSnapshotIssueV0(t, issues, WorktreeIssueSnapshotTooManyFilesV0, root)

	_, issues = CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-file-bytes",
		ProjectWorkDir: root,
		MaxFileBytes:   3,
	})
	requireWorktreeSnapshotIssueV0(t, issues, WorktreeIssueSnapshotFileTooLargeV0, root)

	_, issues = CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-total",
		ProjectWorkDir: root,
		MaxTotalBytes:  6,
	})
	requireWorktreeSnapshotIssueV0(t, issues, WorktreeIssueSnapshotTooLargeV0, root)
}

func TestCaptureWorktreeSnapshotV0PermiteParcialPorPresupuestoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "a.txt", "12")
	writeWorktreeFileForTestV0(t, root, "b.txt", "1234")
	writeWorktreeFileForTestV0(t, root, "c.txt", "12")

	snapshot, issues := CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-file-bytes-partial",
		ProjectWorkDir: root,
		MaxFileBytes:   3,
		MaxTotalBytes:  32,
		AllowPartial:   true,
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(snapshot.Files) != 2 || snapshot.Files[0].Path != "a.txt" || snapshot.Files[1].Path != "c.txt" {
		t.Fatalf("files=%+v", snapshot.Files)
	}
	if len(snapshot.OmittedPaths) != 1 || snapshot.OmittedPaths[0] != "b.txt" {
		t.Fatalf("omitted=%+v", snapshot.OmittedPaths)
	}
	requireWorktreeReceiptReasonV0(t, snapshot.ExclusionReceipts, WorktreeIssueSnapshotFileTooLargeV0)

	snapshot, issues = CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-files-partial",
		ProjectWorkDir: root,
		MaxFiles:       1,
		MaxFileBytes:   32,
		MaxTotalBytes:  32,
		AllowPartial:   true,
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Path != "a.txt" {
		t.Fatalf("files=%+v", snapshot.Files)
	}
	if len(snapshot.OmittedPaths) != 0 {
		t.Fatalf("omitted=%+v", snapshot.OmittedPaths)
	}
	requireWorktreeReceiptReasonV0(t, snapshot.ExclusionReceipts, WorktreeIssueSnapshotTooManyFilesV0)

	snapshot, issues = CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-total-partial",
		ProjectWorkDir: root,
		MaxFileBytes:   32,
		MaxTotalBytes:  3,
		AllowPartial:   true,
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Path != "a.txt" {
		t.Fatalf("files=%+v", snapshot.Files)
	}
	if len(snapshot.OmittedPaths) != 2 || snapshot.OmittedPaths[0] != "b.txt" || snapshot.OmittedPaths[1] != "c.txt" {
		t.Fatalf("omitted=%+v", snapshot.OmittedPaths)
	}
	requireWorktreeReceiptReasonV0(t, snapshot.ExclusionReceipts, WorktreeIssueSnapshotTooLargeV0)
}

func TestVerifyWorktreeWriteSetV0SnapshotParcialNoMarcaOmitidoComoBorradoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "README.md", "v1")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	writeWorktreeFileForTestV0(t, root, "README.md", "contenido demasiado grande")
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:              baseline,
		ProjectWorkDir:        root,
		WriteSet:              []string{"README.md"},
		MaxSnapshotFiles:      10,
		MaxSnapshotFileBytes:  4,
		MaxSnapshotTotalBytes: 1024,
		AllowPartialSnapshot:  true,
	})
	if len(issues) > 0 || !result.OK {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.RemovedPaths) > 0 || len(result.ChangedPaths) > 0 {
		t.Fatalf("result=%+v", result)
	}
	requireWorktreeReceiptReasonV0(t, result.ExclusionReceipts, WorktreeIssueSnapshotFileTooLargeV0)
	if !worktreeStringInSetV0(
		result.EvidenceRefs,
		"evidence-ref-worktree-snapshot-file-too-large-excluded-v0",
	) {
		t.Fatalf("evidence_refs=%v", result.EvidenceRefs)
	}
}

func TestCaptureWorktreeSnapshotV0HashStreamingCuentaLineasGoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "main.go", "package main\nfunc main() {}")

	snapshot, issues := CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-budget-streaming",
		ProjectWorkDir: root,
		MaxFileBytes:   128,
		MaxTotalBytes:  128,
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if snapshot.ReadBudget.MaxFiles == 0 ||
		snapshot.ReadBudget.MaxFileBytes != 128 ||
		snapshot.ReadBudget.MaxTotalBytes != 128 {
		t.Fatalf("budget no declarado: %+v", snapshot.ReadBudget)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].LineCount != 2 ||
		snapshot.Files[0].Digest == "" {
		t.Fatalf("snapshot inesperado: %+v", snapshot.Files)
	}
}

func requireWorktreeSnapshotIssueV0(
	t *testing.T,
	issues []WorktreeIssueV0,
	code WorktreeIssueCodeV0,
	forbidden string,
) {
	t.Helper()
	if len(issues) != 1 || issues[0].Code != code {
		t.Fatalf("issues=%+v want %s", issues, code)
	}
	payload := issues[0].Field + strings.Join(issues[0].Evidence, ",")
	if strings.Contains(payload, forbidden) {
		t.Fatalf("issue filtra path absoluto: %+v", issues[0])
	}
}

func requireWorktreeReceiptReasonV0(
	t *testing.T,
	receipts []WorktreeLocalArtifactExclusionReceiptV0,
	code WorktreeIssueCodeV0,
) {
	t.Helper()
	for _, receipt := range receipts {
		if receipt.ReasonCode == string(code) && receipt.Count > 0 {
			return
		}
	}
	t.Fatalf("receipts=%+v sin reason=%s", receipts, code)
}
