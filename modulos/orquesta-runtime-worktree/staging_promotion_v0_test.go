package orquestaruntimeworktree

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGitStagingPromotionConnectorV0PromocionaYArchivaSinBorrarV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, "README.md", "v2\n")
	archiveDir := filepath.Join(t.TempDir(), "archive")
	connector := GitStagingPromotionConnectorV0{}

	result, issues := connector.PromoteStagingWorktreeV0(context.Background(), StagingPromotionRequestV0{
		PromotionRef:   "promotion-ref-autoprogramming-001",
		RunRef:         "run-ref-autoprogramming-001",
		ProjectRef:     "project-ref-autoprogramming-001",
		RepoRef:        "repo-ref-autoprogramming-001",
		WorktreeRef:    "worktree-ref-autoprogramming-001",
		BranchRef:      "branch-ref-autoprogramming-001",
		ProjectWorkDir: repo,
		CommitMessage:  "test: promote staged autoprogramming",
		WriteSet:       []string{"README.md"},
		EvidenceRefs:   []string{"test-evidence-ref-autoprogramming-001"},
	})
	if len(issues) > 0 ||
		result.Status != StagingPromotionStatusPromotedV0 ||
		result.CommitRef == "" ||
		len(result.ChangedPaths) != 1 ||
		result.ChangedPaths[0] != "README.md" {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if _, err := os.Stat(filepath.Join(repo, "README.md")); err != nil {
		t.Fatalf("README debe seguir vivo: %v", err)
	}

	archive, archiveIssues := connector.ArchiveStagingWorktreeV0(context.Background(), StagingPromotionRequestV0{
		ArchiveRef:   "archive-ref-autoprogramming-001",
		PromotionRef: "promotion-ref-autoprogramming-001",
		RunRef:       "run-ref-autoprogramming-001",
		ProjectRef:   "project-ref-autoprogramming-001",
		RepoRef:      "repo-ref-autoprogramming-001",
		WorktreeRef:  "worktree-ref-autoprogramming-001",
		BranchRef:    "branch-ref-autoprogramming-001",
		ArchiveDir:   archiveDir,
		WriteSet:     []string{"README.md"},
		EvidenceRefs: []string{"test-evidence-ref-autoprogramming-001"},
	})
	if len(archiveIssues) > 0 || archive.Status != StagingPromotionStatusArchivedV0 {
		t.Fatalf("archive=%+v issues=%+v", archive, archiveIssues)
	}
	again, againIssues := connector.ArchiveStagingWorktreeV0(context.Background(), StagingPromotionRequestV0{
		ArchiveRef:   "archive-ref-autoprogramming-001",
		PromotionRef: "promotion-ref-autoprogramming-001",
		RunRef:       "run-ref-autoprogramming-001",
		ProjectRef:   "project-ref-autoprogramming-001",
		RepoRef:      "repo-ref-autoprogramming-001",
		WorktreeRef:  "worktree-ref-autoprogramming-001",
		BranchRef:    "branch-ref-autoprogramming-001",
		ArchiveDir:   archiveDir,
		WriteSet:     []string{"README.md"},
		EvidenceRefs: []string{"test-evidence-ref-autoprogramming-001"},
	})
	if len(againIssues) > 0 || again.Status != StagingPromotionStatusArchivedV0 {
		t.Fatalf("archive replay=%+v issues=%+v", again, againIssues)
	}
	raw, err := os.ReadFile(filepath.Join(archiveDir, "archive-ref-autoprogramming-001.json"))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil || manifest["worktree_ref"] != "worktree-ref-autoprogramming-001" {
		t.Fatalf("manifest err=%v body=%s", err, raw)
	}
}

func TestGitStagingPromotionConnectorV0BloqueaBorradoOCambioFueraDeWriteSetV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	if err := os.Remove(filepath.Join(repo, "README.md")); err != nil {
		t.Fatalf("remove test fixture: %v", err)
	}
	writeAppVCSFileV0(t, repo, "docs/extra.md", "extra\n")

	result, issues := (GitStagingPromotionConnectorV0{}).PromoteStagingWorktreeV0(
		context.Background(),
		StagingPromotionRequestV0{
			PromotionRef:   "promotion-ref-autoprogramming-002",
			RunRef:         "run-ref-autoprogramming-002",
			ProjectRef:     "project-ref-autoprogramming-002",
			RepoRef:        "repo-ref-autoprogramming-002",
			WorktreeRef:    "worktree-ref-autoprogramming-002",
			BranchRef:      "branch-ref-autoprogramming-002",
			ProjectWorkDir: repo,
			CommitMessage:  "test: blocked promotion",
			WriteSet:       []string{"README.md"},
		},
	)
	if result.Status != StagingPromotionStatusBlockedV0 || len(issues) != 2 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	requireWorktreeIssueFieldV0(t, issues, "README.md")
	requireWorktreeIssueFieldV0(t, issues, "docs/extra.md")
}
