package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyWorktreeWriteSetV0ClasificaTruncadoFuerteV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", strings.Repeat("linea larga\n", 40))
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "resumen\n")

	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs/manual.md"},
	})

	if len(issues) != 1 || issues[0].Code != WorktreeIssueTruncatedPathV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.TruncatedPaths) != 1 || result.TruncatedPaths[0] != "docs/manual.md" {
		t.Fatalf("truncated=%v", result.TruncatedPaths)
	}
	requireWorktreeDestructiveKindV0(t, result, WorktreeDestructiveTruncatedV0)
}

func TestVerifyWorktreeWriteSetV0ClasificaRenameAmbiguoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "contenido estable\n")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	if err := os.Rename(
		filepath.Join(root, "docs", "manual.md"),
		filepath.Join(root, "docs", "manual-renamed.md"),
	); err != nil {
		t.Fatalf("rename: %v", err)
	}

	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs"},
	})

	if len(issues) != 1 || issues[0].Code != WorktreeIssueRenamedOrMovedV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.RenamedOrMovedPaths) != 1 ||
		result.RenamedOrMovedPaths[0] != "docs/manual.md -> docs/manual-renamed.md" {
		t.Fatalf("renamed=%v", result.RenamedOrMovedPaths)
	}
	requireWorktreeDestructiveKindV0(t, result, WorktreeDestructiveRenamedOrMovedV0)
}

func TestVerifyWorktreeWriteSetV0ClasificaReemplazoConDeltaGrandeV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", strings.Repeat("a", 12000))
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", strings.Repeat("b", 7000))

	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs/manual.md"},
	})

	if len(issues) != 1 || issues[0].Code != WorktreeIssueReplacedLargeV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.ReplacedLargeDelta) != 1 || result.ReplacedLargeDelta[0] != "docs/manual.md" {
		t.Fatalf("replaced=%v", result.ReplacedLargeDelta)
	}
	requireWorktreeDestructiveKindV0(t, result, WorktreeDestructiveReplacedLargeDeltaV0)
}

func TestVerifyWorktreeWriteSetV0PermiteEdicionNormalDentroDelWriteSetV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "linea uno\nlinea dos\n")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "linea uno\nlinea dos corregida\n")

	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs/manual.md"},
	})

	if len(issues) > 0 || !result.OK {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.DestructiveChanges) > 0 {
		t.Fatalf("destructive=%+v", result.DestructiveChanges)
	}
}

func requireWorktreeDestructiveKindV0(
	t *testing.T,
	result WorktreeVerifyResultV0,
	kind string,
) {
	t.Helper()
	for _, change := range result.DestructiveChanges {
		if change.Kind == kind {
			return
		}
	}
	t.Fatalf("missing destructive kind %q in %+v", kind, result.DestructiveChanges)
}
