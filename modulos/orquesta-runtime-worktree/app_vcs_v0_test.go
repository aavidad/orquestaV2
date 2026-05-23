package orquestaruntimeworktree

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitAppVCSConnectorV0CommitLocalYPushPendienteReintentable(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	writeAppVCSFileV0(t, repo, "README.md", "v2\n")

	result, issues := (GitAppVCSConnectorV0{CommandTimeout: 2 * time.Second}).ExecuteAppVCSV0(
		context.Background(),
		AppVCSRequestV0{
			Action:         AppVCSActionCommitV0,
			AppRef:         "app-ref-vcs-test",
			RepoRef:        "repo-ref-vcs-test",
			WorktreeRef:    "worktree-ref-vcs-test",
			BranchRef:      "branch-ref-vcs-test",
			ProjectWorkDir: repo,
			CommitMessage:  "test: checkpoint local",
			AllowPush:      true,
		},
	)

	if result.Status != AppVCSStatusPendingPushV0 || !result.PushPending || !result.Retryable {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if result.CommitRef == "" || result.CommitShortRef == "" {
		t.Fatalf("commit ref vacio: %+v", result)
	}
	if len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "README.md" {
		t.Fatalf("changed paths=%+v", result.ChangedPaths)
	}
	if len(issues) != 1 || issues[0].Code != AppVCSIssuePushPendingV0 {
		t.Fatalf("issues=%+v", issues)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), repo) {
		t.Fatalf("resultado publico filtra path absoluto: %s", raw)
	}
	if strings.TrimSpace(runAppVCSGitV0(t, repo, "status", "--porcelain")) != "" {
		t.Fatalf("repo debe quedar limpio tras commit local")
	}
}

func TestGitAppVCSConnectorV0PreparaRepoSinPathLocal(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	result, issues := (GitAppVCSConnectorV0{}).ExecuteAppVCSV0(context.Background(), AppVCSRequestV0{
		Action:         AppVCSActionPrepareRepoV0,
		AppRef:         "app-ref-vcs-test",
		RepoRef:        "repo-ref-vcs-test",
		ProjectWorkDir: repo,
	})
	if len(issues) > 0 || result.Status != AppVCSStatusCompletedV0 || result.CommitRef == "" {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), repo) {
		t.Fatalf("resultado publico filtra path absoluto: %s", raw)
	}
}

func TestGitAppVCSConnectorV0ReviewRepoLimpioSinPathLocal(t *testing.T) {
	repo := initAppVCSTestRepoV0(t)
	result, issues := (GitAppVCSConnectorV0{}).ExecuteAppVCSV0(context.Background(), AppVCSRequestV0{
		Action:         AppVCSActionReviewRepoV0,
		AppRef:         "app-ref-vcs-review",
		RepoRef:        "repo-ref-vcs-review",
		WorktreeRef:    "worktree-ref-orquesta-autoprog-git-review-20260523",
		BranchRef:      "branch-ref-orquesta-autoprog-git-review-20260523",
		ProjectWorkDir: repo,
	})
	if len(issues) > 0 || result.Status != AppVCSStatusCleanV0 || result.CommitRef == "" {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if len(result.ChangedPaths) != 0 ||
		result.WorktreeRef != "worktree-ref-orquesta-autoprog-git-review-20260523" ||
		result.BranchRef != "branch-ref-orquesta-autoprog-git-review-20260523" {
		t.Fatalf("refs/changed inesperados: %+v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), repo) {
		t.Fatalf("resultado publico filtra path absoluto: %s", raw)
	}
	if strings.TrimSpace(runAppVCSGitV0(t, repo, "status", "--porcelain")) != "" {
		t.Fatalf("review_repo no debe modificar repo")
	}
}

func initAppVCSTestRepoV0(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runAppVCSGitV0(t, repo, "init", "-b", "main")
	runAppVCSGitV0(t, repo, "config", "user.name", "Orquesta Test")
	runAppVCSGitV0(t, repo, "config", "user.email", "orquesta@example.com")
	writeAppVCSFileV0(t, repo, "README.md", "v1\n")
	runAppVCSGitV0(t, repo, "add", "README.md")
	runAppVCSGitV0(t, repo, "commit", "-m", "base")
	return repo
}

func writeAppVCSFileV0(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func runAppVCSGitV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}
