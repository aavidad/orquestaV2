package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexReceiptWorktreeBaselineRecorderV0ConservaDetalleDeIssue(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "grande.txt"), []byte("contenido"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	recorder := CodexReceiptWorktreeBaselineRecorderV0{
		SnapshotStore: orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0(),
		SnapshotReadBudget: orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0{
			MaxFiles:      10,
			MaxFileBytes:  1,
			MaxTotalBytes: 1024,
		},
	}

	_, err := recorder.CaptureCodexReceiptWorktreeBaselineV0(
		context.Background(),
		CodexReceiptWorktreeBaselineRequestV0{
			DescriptorRef:  "descriptor-ref-baseline-detail",
			ProjectWorkDir: root,
		},
	)
	if err == nil {
		t.Fatalf("CaptureCodexReceiptWorktreeBaselineV0 sin error")
	}
	got := err.Error()
	for _, want := range []string{
		"codex_worktree_baseline: capture_failed",
		"code=worktree_snapshot_file_too_large",
		"field=file",
		"evidence=grande.txt",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error=%q missing=%q", got, want)
		}
	}
	if strings.Contains(got, root) {
		t.Fatalf("error filtra path absoluto: %q", got)
	}
}
