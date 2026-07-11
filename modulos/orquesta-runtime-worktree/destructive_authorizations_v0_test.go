package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyWorktreeWriteSetV0PermiteRenameExactamenteAutorizadoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "contenido estable\n")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	if err := os.Rename(filepath.Join(root, "docs", "manual.md"), filepath.Join(root, "docs", "manual-renamed.md")); err != nil {
		t.Fatalf("rename: %v", err)
	}

	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"docs"},
		DestructiveAuthorizations: []WorktreeDestructiveAuthorizationV0{{
			Kind:         WorktreeDestructiveAuthorizationRenameV0,
			PreviousPath: "docs/manual.md",
			CurrentPath:  "docs/manual-renamed.md",
		}},
	})

	if len(issues) > 0 || !result.OK {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.DestructiveChanges) != 1 ||
		result.DestructiveChanges[0].Kind != WorktreeDestructiveRenamedOrMovedV0 ||
		result.DestructiveChanges[0].PreviousPath != "docs/manual.md" ||
		result.DestructiveChanges[0].CurrentPath != "docs/manual-renamed.md" {
		t.Fatalf("destructive=%+v", result.DestructiveChanges)
	}
}

func TestVerifyWorktreeWriteSetV0BloqueaRenameNoAutorizadoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "docs/manual.md", "contenido estable\n")
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
	if err := os.Rename(filepath.Join(root, "docs", "manual.md"), filepath.Join(root, "docs", "manual-renamed.md")); err != nil {
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
}

func TestVerifyWorktreeWriteSetV0BloqueaAutorizacionDestructivaParcialOEquivocadaV0(t *testing.T) {
	for _, authorization := range []WorktreeDestructiveAuthorizationV0{
		{
			Kind:         WorktreeDestructiveAuthorizationRenameV0,
			PreviousPath: "docs/manual.md",
		},
		{
			Kind:         WorktreeDestructiveAuthorizationRenameV0,
			PreviousPath: "docs/manual.md",
			CurrentPath:  "docs/otro.md",
		},
	} {
		t.Run(string(authorization.Kind)+"-"+authorization.CurrentPath, func(t *testing.T) {
			root := t.TempDir()
			writeWorktreeFileForTestV0(t, root, "docs/manual.md", "contenido estable\n")
			baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
			if err := os.Rename(filepath.Join(root, "docs", "manual.md"), filepath.Join(root, "docs", "manual-renamed.md")); err != nil {
				t.Fatalf("rename: %v", err)
			}

			result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
				Baseline:                  baseline,
				ProjectWorkDir:            root,
				WriteSet:                  []string{"docs"},
				DestructiveAuthorizations: []WorktreeDestructiveAuthorizationV0{authorization},
			})

			if len(issues) != 1 || issues[0].Code != WorktreeIssueRenamedOrMovedV0 {
				t.Fatalf("result=%+v issues=%+v", result, issues)
			}
		})
	}
}

func TestVerifyWorktreeWriteSetV0PermiteRemoveSoloExactoV0(t *testing.T) {
	for _, test := range []struct {
		name          string
		authorization WorktreeDestructiveAuthorizationV0
		wantIssue     bool
	}{
		{
			name: "exacta",
			authorization: WorktreeDestructiveAuthorizationV0{
				Kind: WorktreeDestructiveAuthorizationRemoveV0,
				Path: "docs/manual.md",
			},
		},
		{
			name: "equivocada",
			authorization: WorktreeDestructiveAuthorizationV0{
				Kind: WorktreeDestructiveAuthorizationRemoveV0,
				Path: "docs/otro.md",
			},
			wantIssue: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeWorktreeFileForTestV0(t, root, "docs/manual.md", strings.Repeat("contenido\n", 16))
			baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
			if err := os.Remove(filepath.Join(root, "docs", "manual.md")); err != nil {
				t.Fatalf("remove: %v", err)
			}

			result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
				Baseline:                  baseline,
				ProjectWorkDir:            root,
				WriteSet:                  []string{"docs/manual.md"},
				DestructiveAuthorizations: []WorktreeDestructiveAuthorizationV0{test.authorization},
			})

			if test.wantIssue {
				if len(issues) != 1 || issues[0].Code != WorktreeIssueRemovedPathV0 {
					t.Fatalf("result=%+v issues=%+v", result, issues)
				}
				return
			}
			if len(issues) > 0 || !result.OK {
				t.Fatalf("result=%+v issues=%+v", result, issues)
			}
			requireWorktreeDestructiveKindV0(t, result, WorktreeDestructiveRemovedV0)
		})
	}
}
