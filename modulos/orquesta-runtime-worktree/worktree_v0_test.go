package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyWorktreeWriteSetV0AceptaCambiosDentroDelWriteSet(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "internal/api/handler.go", "v1")
	writeWorktreeFileForTestV0(t, root, ".control/agent_ack.json", "pending")
	baseline := captureWorktreeSnapshotForTestV0(t, root, []string{".control"})

	writeWorktreeFileForTestV0(t, root, "internal/api/handler.go", "v2")
	writeWorktreeFileForTestV0(t, root, ".control/agent_ack.json", "done")
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"internal/api/handler.go"},
		IgnorePrefixes: []string{".control"},
	})
	if len(issues) > 0 || !result.OK {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "internal/api/handler.go" {
		t.Fatalf("changed=%v", result.ChangedPaths)
	}
}

func TestVerifyWorktreeWriteSetV0RechazaCambioFueraDelWriteSet(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "README.md", "v1")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	writeWorktreeFileForTestV0(t, root, "README.md", "v2")
	writeWorktreeFileForTestV0(t, root, "docs/extra.md", "extra")
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"README.md"},
	})
	if len(issues) != 1 || issues[0].Code != WorktreeIssueOutsideWriteSetV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.OutsideWriteSet) != 1 || result.OutsideWriteSet[0] != "docs/extra.md" {
		t.Fatalf("outside=%v", result.OutsideWriteSet)
	}
}

func TestVerifyWorktreeWriteSetV0PermiteRootParaAppNueva(t *testing.T) {
	root := t.TempDir()
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	writeWorktreeFileForTestV0(t, root, "go.mod", "module app\n")
	writeWorktreeFileForTestV0(t, root, "internal/app/app.go", "package app\n")
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"."},
	})
	if len(issues) > 0 || !result.OK || len(result.ChangedPaths) != 2 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
}

func TestVerifyWorktreeWriteSetV0RechazaRequestInsegura(t *testing.T) {
	root := t.TempDir()
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	_, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"../fuera"},
	})
	if len(issues) == 0 || issues[0].Code != WorktreeIssueInvalidRequestV0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func captureWorktreeSnapshotForTestV0(
	t *testing.T,
	root string,
	ignore []string,
) WorktreeSnapshotV0 {
	t.Helper()
	snapshot, issues := CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-test",
		ProjectWorkDir: root,
		IgnorePrefixes: ignore,
	})
	if len(issues) > 0 {
		t.Fatalf("snapshot issues=%+v", issues)
	}
	return snapshot
}

func writeWorktreeFileForTestV0(
	t *testing.T,
	root string,
	rel string,
	content string,
) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
