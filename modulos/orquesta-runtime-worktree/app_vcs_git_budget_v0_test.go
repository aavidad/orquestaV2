package orquestaruntimeworktree

import (
	"context"
	"strings"
	"testing"
)

func TestGitAppVCSConnectorV0BloqueaStatusConDemasiadasRutasV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, "a.txt", "a\n")
	writeAppVCSFileV0(t, repo, "b.txt", "b\n")

	result, issues := (GitAppVCSConnectorV0{MaxChangedPaths: 1}).ExecuteAppVCSV0(
		context.Background(),
		AppVCSRequestV0{
			Action:         AppVCSActionCommitV0,
			AppRef:         "app-ref-budget",
			RepoRef:        "repo-ref-budget",
			ProjectWorkDir: repo,
			CommitMessage:  "test: blocked by status path budget",
		},
	)

	if result.Status != AppVCSStatusFailedV0 || len(issues) != 1 ||
		issues[0].Code != AppVCSIssueGitStatusTooManyPathsV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	status := runAppVCSGitV0(t, repo, "status", "--porcelain", "--untracked-files=all")
	if !strings.Contains(status, "?? a.txt") || !strings.Contains(status, "?? b.txt") {
		t.Fatalf("commit no debe hacer git add ni ocultar cambios: %q", status)
	}
	requireAppVCSIssueEvidenceSaneadaV0(t, issues[0], repo)
}

func TestGitAppVCSConnectorV0BloqueaOutputGitDemasiadoGrandeV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)

	result, issues := (GitAppVCSConnectorV0{MaxOutputBytes: 8}).ExecuteAppVCSV0(
		context.Background(),
		AppVCSRequestV0{
			Action:         AppVCSActionReviewRepoV0,
			AppRef:         "app-ref-output",
			RepoRef:        "repo-ref-output",
			ProjectWorkDir: repo,
		},
	)

	if result.Status != AppVCSStatusFailedV0 || len(issues) != 1 ||
		issues[0].Code != AppVCSIssueGitOutputTooLargeV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	requireAppVCSIssueEvidenceSaneadaV0(t, issues[0], repo)
}

func TestGitStagingPromotionConnectorV0BloqueaStatusConDemasiadasRutasV0(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, "a.txt", "a\n")
	writeAppVCSFileV0(t, repo, "b.txt", "b\n")

	result, issues := (GitStagingPromotionConnectorV0{
		VCS: GitAppVCSConnectorV0{MaxChangedPaths: 1},
	}).PromoteStagingWorktreeV0(
		context.Background(),
		StagingPromotionRequestV0{
			PromotionRef:   "promotion-ref-budget",
			ProjectRef:     "project-ref-budget",
			RepoRef:        "repo-ref-budget",
			WorktreeRef:    "worktree-ref-budget",
			BranchRef:      "branch-ref-budget",
			ProjectWorkDir: repo,
			CommitMessage:  "test: blocked promotion",
			WriteSet:       []string{"."},
		},
	)

	if result.Status != StagingPromotionStatusBlockedV0 || len(issues) != 1 ||
		issues[0].Code != WorktreeIssueGitStatusTooManyPathsV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	status := runAppVCSGitV0(t, repo, "status", "--porcelain", "--untracked-files=all")
	if !strings.Contains(status, "?? a.txt") || !strings.Contains(status, "?? b.txt") {
		t.Fatalf("promocion no debe hacer git add: %q", status)
	}
}

func requireAppVCSIssueEvidenceSaneadaV0(t *testing.T, issue AppVCSIssueV0, repo string) {
	t.Helper()
	joined := strings.Join(issue.Evidence, "\n")
	if strings.Contains(joined, repo) ||
		strings.Contains(joined, "README.md") ||
		strings.Contains(joined, "a.txt") ||
		strings.Contains(joined, "b.txt") {
		t.Fatalf("evidencia Git no saneada: %+v", issue)
	}
	if !strings.Contains(joined, "action=git.") {
		t.Fatalf("evidencia Git sin accion compacta: %+v", issue)
	}
}
