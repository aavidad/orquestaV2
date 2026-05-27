package orquestaruntimeworktree

import (
	"context"
	"strings"
	"testing"
)

func TestGitAppVCSConnectorV0ReviewYCommitExcluyenControlFilesV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, ".orquesta-runtime/run/agent_packet.json", "{}")
	writeAppVCSFileV0(t, repo, ".orquesta-local-runtime-20260525/run/debug.log", "log")
	writeAppVCSFileV0(t, repo, ".orquesta-server/state.json", "{}")
	writeAppVCSFileV0(t, repo, ".orquesta-logs/daemon.log", "log")
	writeAppVCSFileV0(t, repo, "logs/server.log", "log")
	writeAppVCSFileV0(t, repo, "orquesta.env", "SECRET=value")
	writeAppVCSFileV0(t, repo, "orquesta.db", "db")
	writeAppVCSFileV0(t, repo, ".orquesta-inbox.md", "local")

	review, reviewIssues := (GitAppVCSConnectorV0{}).ExecuteAppVCSV0(context.Background(), AppVCSRequestV0{
		Action:         AppVCSActionReviewRepoV0,
		AppRef:         "app-ref-control",
		RepoRef:        "repo-ref-control",
		ProjectWorkDir: repo,
	})
	if len(reviewIssues) > 0 || review.Status != AppVCSStatusCleanV0 || len(review.ChangedPaths) != 0 {
		t.Fatalf("review=%+v issues=%+v", review, reviewIssues)
	}
	if len(review.Issues) != 8 || review.Issues[0].Code != AppVCSIssueControlPathV0 {
		t.Fatalf("review control issues=%+v", review.Issues)
	}
	for _, want := range []string{
		"runtime_control_dir",
		"server_state_dir",
		"local_logs",
		"local_operator_config",
		"local_database_state",
		"local_operator_notes",
	} {
		if !worktreeReceiptCategoryForTestV0(review.ExclusionReceipts, want) {
			t.Fatalf("review receipts sin %s: %+v", want, review.ExclusionReceipts)
		}
	}

	writeAppVCSFileV0(t, repo, "feature.md", "feature\n")
	commit, commitIssues := (GitAppVCSConnectorV0{}).ExecuteAppVCSV0(context.Background(), AppVCSRequestV0{
		Action:         AppVCSActionCommitV0,
		AppRef:         "app-ref-control",
		RepoRef:        "repo-ref-control",
		ProjectWorkDir: repo,
		CommitMessage:  "test: product commit",
	})
	if len(commitIssues) > 0 || commit.Status != AppVCSStatusCompletedV0 {
		t.Fatalf("commit=%+v issues=%+v", commit, commitIssues)
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "agent_packet.json") {
		t.Fatalf("control file fue commiteado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), ".orquesta-local-runtime") {
		t.Fatalf("control dir local fue commiteado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), ".orquesta-server") {
		t.Fatalf("state dir local fue commiteado")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "orquesta.env") {
		t.Fatalf("config local fue commiteada")
	}
	if strings.Contains(runAppVCSGitV0(t, repo, "show", "--name-only", "--format=", "HEAD"), "orquesta.db") {
		t.Fatalf("db local fue commiteada")
	}
}
