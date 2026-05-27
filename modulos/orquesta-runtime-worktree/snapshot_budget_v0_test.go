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
