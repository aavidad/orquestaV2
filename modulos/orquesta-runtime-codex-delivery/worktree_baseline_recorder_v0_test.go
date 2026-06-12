package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexReceiptWorktreeBaselineRecorderV0NoBloqueaIssueDeSnapshot(t *testing.T) {
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

	resolution, err := recorder.CaptureCodexReceiptWorktreeBaselineV0(
		context.Background(),
		CodexReceiptWorktreeBaselineRequestV0{
			DescriptorRef:  "descriptor-ref-baseline-detail",
			ProjectWorkDir: root,
		},
	)
	if err != nil {
		t.Fatalf("CaptureCodexReceiptWorktreeBaselineV0 bloqueo launch: %v", err)
	}
	if resolution.BaselineRef == "" {
		t.Fatalf("baseline parcial no registrada")
	}
	snapshot, err := recorder.SnapshotStore.LoadWorktreeSnapshotV0(context.Background(), resolution.BaselineRef)
	if err != nil {
		t.Fatalf("LoadWorktreeSnapshotV0: %v", err)
	}
	if len(snapshot.Files) != 0 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	requireCodexDeliveryReceiptReasonV0(t, snapshot.ExclusionReceipts, orquestaruntimeworktree.WorktreeIssueSnapshotFileTooLargeV0)
}

func requireCodexDeliveryReceiptReasonV0(
	t *testing.T,
	receipts []orquestaruntimeworktree.WorktreeLocalArtifactExclusionReceiptV0,
	code orquestaruntimeworktree.WorktreeIssueCodeV0,
) {
	t.Helper()
	for _, receipt := range receipts {
		if receipt.ReasonCode == string(code) && receipt.Count > 0 {
			return
		}
	}
	t.Fatalf("receipts=%+v sin reason=%s", receipts, code)
}
