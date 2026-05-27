package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func captureCodexDeliveryWorktreeBaselineForTestV0(
	t *testing.T,
	projectDir string,
) orquestaruntimeworktree.WorktreeSnapshotV0 {
	t.Helper()
	snapshot, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(
		context.Background(),
		orquestaruntimeworktree.WorktreeSnapshotRequestV0{
			SnapshotRef:    "worktree-snapshot-ref-test",
			ProjectWorkDir: projectDir,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("baseline issues=%+v", issues)
	}
	return snapshot
}

func writeCodexDeliveryProjectFileForTestV0(
	t *testing.T,
	projectDir string,
	rel string,
	content string,
) {
	t.Helper()
	path := filepath.Join(projectDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
