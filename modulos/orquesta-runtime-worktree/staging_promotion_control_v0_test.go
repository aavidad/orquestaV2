package orquestaruntimeworktree

import (
	"context"
	"strings"
	"testing"
)

func TestGitStagingPromotionConnectorV0NoPromocionaControlFilesConWriteSetRaizV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, "feature.md", "feature\n")
	writeAppVCSFileV0(t, repo, ".orquesta-runtime/run/agent_ack.json", "{}")
	writeAppVCSFileV0(t, repo, ".orquesta-local-runtime-20260525/run/debug.log", "log")
	writeAppVCSFileV0(t, repo, ".orquesta-smoke-work/run.log", "log")
	writeAppVCSFileV0(t, repo, "orquesta.env", "SECRET=value")
	writeAppVCSFileV0(t, repo, "orquesta.db", "db")

	result, issues := (GitStagingPromotionConnectorV0{}).PromoteStagingWorktreeV0(
		context.Background(),
		StagingPromotionRequestV0{
			PromotionRef:   "promotion-ref-control-001",
			ProjectRef:     "project-ref-control-001",
			RepoRef:        "repo-ref-control-001",
			WorktreeRef:    "worktree-ref-control-001",
			BranchRef:      "branch-ref-control-001",
			ProjectWorkDir: repo,
			CommitMessage:  "test: promote product only",
			WriteSet:       []string{"."},
		},
	)
	if len(issues) > 0 || result.Status != StagingPromotionStatusPromotedV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "feature.md" {
		t.Fatalf("changed=%+v", result.ChangedPaths)
	}
	if len(result.Issues) != 5 || result.Issues[0].Code != WorktreeIssueControlPathV0 {
		t.Fatalf("control issues=%+v", result.Issues)
	}
	if !worktreeReceiptCategoryForTestV0(result.ExclusionReceipts, "smoke_work_dir") {
		t.Fatalf("receipts=%+v", result.ExclusionReceipts)
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "agent_ack.json") {
		t.Fatalf("control file fue promocionado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), ".orquesta-local-runtime") {
		t.Fatalf("control dir local fue promocionado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), ".orquesta-smoke-work") {
		t.Fatalf("smoke dir local fue promocionado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "orquesta.env") {
		t.Fatalf("config local fue promocionada")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "orquesta.db") {
		t.Fatalf("db local fue promocionada")
	}
}
