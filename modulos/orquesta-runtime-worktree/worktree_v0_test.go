package orquestaruntimeworktree

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestVerifyWorktreeWriteSetV0RechazaBorradoAunqueEsteEnWriteSet(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "v1")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	if err := os.Remove(filepath.Join(root, "docs", "manual.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs/manual.md"},
	})
	if len(issues) != 1 || issues[0].Code != WorktreeIssueRemovedPathV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.RemovedPaths) != 1 || result.RemovedPaths[0] != "docs/manual.md" {
		t.Fatalf("removed=%v", result.RemovedPaths)
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

func TestPrepareIsolatedWorktreeV0ConservaBranchRefOpaca(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "go.mod", "module app\n")
	writeWorktreeFileForTestV0(t, root, ".control/agent_ack.json", "pending")

	result, issues := PrepareIsolatedWorktreeV0(context.Background(), WorktreeIsolationRequestV0{
		IsolationRef:   "worktree-isolation-ref-001",
		ProjectRef:     "project-ref-autoprogramming-001",
		WorktreeRef:    "worktree-ref-autoprogramming-001",
		BranchRef:      "branch-ref-autoprogramming-001",
		ProjectWorkDir: root,
		Isolated:       true,
		BaselineRef:    "worktree-baseline-ref-001",
		IgnorePrefixes: []string{".control"},
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if result.SchemaVersion != WorktreeIsolationSchemaVersionV0 ||
		result.Mode != WorktreeIsolationModeIsolatedV0 ||
		result.BranchRef != "branch-ref-autoprogramming-001" ||
		result.WorktreeRef != "worktree-ref-autoprogramming-001" ||
		result.BaselineRef != "worktree-baseline-ref-001" {
		t.Fatalf("result inesperado: %+v", result)
	}
	if len(result.Snapshot.Files) != 1 || result.Snapshot.Files[0].Path != "go.mod" {
		t.Fatalf("snapshot no acotado: %+v", result.Snapshot.Files)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), root) {
		t.Fatalf("resultado publico filtra path absoluto: %s", string(raw))
	}
}

func TestPrepareIsolatedWorktreeV0RechazaRamaNoOpacaONoAislada(t *testing.T) {
	root := t.TempDir()
	_, issues := PrepareIsolatedWorktreeV0(context.Background(), WorktreeIsolationRequestV0{
		IsolationRef:   "worktree-isolation-ref-001",
		ProjectRef:     "project-ref-autoprogramming-001",
		WorktreeRef:    "worktree-ref-autoprogramming-001",
		BranchRef:      "feature/app",
		ProjectWorkDir: root,
		Isolated:       false,
	})
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}
	requireWorktreeIssueFieldV0(t, issues, "branch_ref")
	requireWorktreeIssueFieldV0(t, issues, "isolated")
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

func requireWorktreeIssueFieldV0(t *testing.T, issues []WorktreeIssueV0, field string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Field == field {
			return
		}
	}
	t.Fatalf("no issue for field %q in %+v", field, issues)
}
