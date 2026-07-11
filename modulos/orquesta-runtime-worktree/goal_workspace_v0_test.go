package orquestaruntimeworktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestGitGoalWorkspaceProvisionerV0CreatesPhysicalWorkspacesPerGoal(t *testing.T) {
	repo := newGoalWorkspaceGitRepoForTestV0(t)
	root := filepath.Join(t.TempDir(), "runtime")
	connector := GitGoalWorkspaceProvisionerV0{}
	first, issues := connector.PrepareGoalWorkspaceV0(context.Background(), GoalWorkspaceRequestV0{
		RunRef: "run-ref-batch-001", GoalRef: "goal-ref-001", ProjectRef: "project-ref-001",
		WorktreeRef: "worktree-ref-001", SourceWorkDir: repo, WorkspaceRoot: root,
	})
	if len(issues) > 0 {
		t.Fatalf("first issues=%+v", issues)
	}
	second, issues := connector.PrepareGoalWorkspaceV0(context.Background(), GoalWorkspaceRequestV0{
		RunRef: "run-ref-batch-001", GoalRef: "goal-ref-002", ProjectRef: "project-ref-001",
		WorktreeRef: "worktree-ref-001", SourceWorkDir: repo, WorkspaceRoot: root,
	})
	if len(issues) > 0 {
		t.Fatalf("second issues=%+v", issues)
	}
	if first.ProjectWorkDir == second.ProjectWorkDir || first.BaseRevision != second.BaseRevision {
		t.Fatalf("workspaces not isolated: first=%+v second=%+v", first, second)
	}
	if err := os.WriteFile(filepath.Join(first.ProjectWorkDir, "first.txt"), []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(second.ProjectWorkDir, "first.txt")); !os.IsNotExist(err) {
		t.Fatalf("goal sibling observed foreign mutation: err=%v", err)
	}
}

func TestGitGoalWorkspaceProvisionerV0ReplayAndBaseConflict(t *testing.T) {
	repo := newGoalWorkspaceGitRepoForTestV0(t)
	root := filepath.Join(t.TempDir(), "runtime")
	request := GoalWorkspaceRequestV0{
		RunRef: "run-ref-replay-001", GoalRef: "goal-ref-replay-001", ProjectRef: "project-ref-001",
		WorktreeRef: "worktree-ref-001", SourceWorkDir: repo, WorkspaceRoot: root,
	}
	connector := GitGoalWorkspaceProvisionerV0{}
	first, issues := connector.PrepareGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 {
		t.Fatalf("first issues=%+v", issues)
	}
	replayed, issues := connector.PrepareGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || replayed.ProjectWorkDir != first.ProjectWorkDir {
		t.Fatalf("replay=%+v issues=%+v", replayed, issues)
	}
	request.BaseRevision = goalWorkspaceGitOutputForTestV0(t, repo, "rev-parse", "HEAD^")
	if _, issues = connector.PrepareGoalWorkspaceV0(context.Background(), request); !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("base mismatch must conflict: %+v", issues)
	}
}

func TestGitGoalWorkspaceProvisionerV0ConcurrentSameGoalDoesNotDuplicate(t *testing.T) {
	repo := newGoalWorkspaceGitRepoForTestV0(t)
	root := filepath.Join(t.TempDir(), "runtime")
	request := GoalWorkspaceRequestV0{
		RunRef: "run-ref-concurrent-001", GoalRef: "goal-ref-concurrent-001", ProjectRef: "project-ref-001",
		WorktreeRef: "worktree-ref-001", SourceWorkDir: repo, WorkspaceRoot: root,
	}
	connector := GitGoalWorkspaceProvisionerV0{}
	type result struct {
		workspace GoalWorkspaceV0
		issues    []WorktreeIssueV0
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workspace, issues := connector.PrepareGoalWorkspaceV0(context.Background(), request)
			results <- result{workspace: workspace, issues: issues}
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	locked := 0
	for got := range results {
		if len(got.issues) == 0 {
			successes++
			continue
		}
		if goalWorkspaceHasIssueForTestV0(got.issues, WorktreeIssueWorkspaceLockedV0) {
			locked++
		}
	}
	if successes < 1 || successes+locked != 2 {
		t.Fatalf("successes=%d locked=%d", successes, locked)
	}
}

func TestNormalizeGoalWorkspaceRequestV0RejectsRuntimeInsideSource(t *testing.T) {
	repo := t.TempDir()
	_, issues := normalizeGoalWorkspaceRequestV0(GoalWorkspaceRequestV0{
		RunRef: "run-ref-001", GoalRef: "goal-ref-001", ProjectRef: "project-ref-001", WorktreeRef: "worktree-ref-001",
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(repo, ".runtime"),
	})
	if !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueInvalidRequestV0) {
		t.Fatalf("issues=%+v", issues)
	}
}

func newGoalWorkspaceGitRepoForTestV0(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	goalWorkspaceRunGitForTestV0(t, repo, "init", "-q")
	goalWorkspaceRunGitForTestV0(t, repo, "config", "user.email", "test@example.invalid")
	goalWorkspaceRunGitForTestV0(t, repo, "config", "user.name", "Orquesta Test")
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	goalWorkspaceRunGitForTestV0(t, repo, "add", "base.txt")
	goalWorkspaceRunGitForTestV0(t, repo, "commit", "-q", "-m", "base")
	if err := os.WriteFile(filepath.Join(repo, "second.txt"), []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	goalWorkspaceRunGitForTestV0(t, repo, "add", "second.txt")
	goalWorkspaceRunGitForTestV0(t, repo, "commit", "-q", "-m", "second")
	return repo
}

func goalWorkspaceRunGitForTestV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func goalWorkspaceGitOutputForTestV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out[:len(out)-1])
}

func goalWorkspaceHasIssueForTestV0(issues []WorktreeIssueV0, code WorktreeIssueCodeV0) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
