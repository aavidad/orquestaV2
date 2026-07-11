package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestCodexGoalWorkspaceAdapterV0ProvisionsAndResolvesAfterRestart(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       repo,
		WorkspaceRoot:       filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback:  "project-ref-fallback",
		WorktreeRefFallback: "worktree-ref-fallback",
	}
	first, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-001", RequestRef: "run-ref-workspace-adapter-001", ProjectRef: "project-ref-packet-001",
	})
	if err != nil {
		t.Fatalf("first prepare: %v", err)
	}
	second, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-002",
	})
	if err != nil {
		t.Fatalf("second prepare: %v", err)
	}
	if first.ProjectWorkDir == second.ProjectWorkDir {
		t.Fatalf("workspaces must differ: first=%q second=%q", first.ProjectWorkDir, second.ProjectWorkDir)
	}
	if err := os.WriteFile(filepath.Join(first.ProjectWorkDir, "only-first.txt"), []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(second.ProjectWorkDir, "only-first.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second workspace observes first mutation: %v", err)
	}

	restarted := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       repo,
		WorkspaceRoot:       adapter.WorkspaceRoot,
		ProjectRefFallback:  adapter.ProjectRefFallback,
		WorktreeRefFallback: adapter.WorktreeRefFallback,
	}
	resolved, err := restarted.ResolveCodexGoalWorkspaceV0(context.Background(), orquestaruntimecodexgoal.CodexGoalObservationRequestV0{
		GoalRef: "goal-ref-workspace-adapter-001",
	})
	if err != nil || resolved.ProjectWorkDir != first.ProjectWorkDir {
		t.Fatalf("resolved=%+v err=%v first=%+v", resolved, err, first)
	}
}

func TestCodexGoalWorkspaceAdapterV0RejectsConflictingRequestIdentity(t *testing.T) {
	repo := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"),
		ProjectRefFallback: "project-ref-fallback", WorktreeRefFallback: "worktree-ref-fallback",
	}
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef: "goal-ref-workspace-adapter-conflict", RequestRef: "run-ref-workspace-adapter-conflict",
	}
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	packet.ProjectRef = "project-ref-conflict"
	if _, err := adapter.PrepareCodexGoalWorkspaceV0(context.Background(), packet); !errors.Is(err, errCodexGoalWorkspaceAdapterConflictV0) {
		t.Fatalf("conflicting identity err=%v", err)
	}
}

func TestCodexGoalWorkspaceAdapterV0UsaWorktreeTipadoDelGoalV0(t *testing.T) {
	adapter := codexGoalWorkspaceAdapterV0{
		SourceWorkDir:       t.TempDir(),
		WorkspaceRoot:       t.TempDir(),
		ProjectRefFallback:  "project-ref-fallback",
		WorktreeRefFallback: "worktree-ref-fallback",
	}
	request := adapter.requestForStartV0(orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		GoalRef:    "goal-ref-workspace-typed",
		RequestRef: "run-ref-workspace-typed",
		ProjectRef: "project-ref-workspace-typed",
		ContextRefs: []orquestagoal.GoalContextRefV0{{
			Kind: "worktree",
			Ref:  "worktree-ref-workspace-typed",
		}},
	})
	if request.WorktreeRef != "worktree-ref-workspace-typed" {
		t.Fatalf("worktree_ref=%q", request.WorktreeRef)
	}
}

func TestCodexGoalWorkspaceRootForSourceV0IsStableAndOutsideRepo(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	root := codexGoalWorkspaceRootForSourceV0(repo)
	if root == "" || filepath.Dir(root) != filepath.Dir(repo) || root == repo {
		t.Fatalf("root=%q repo=%q", root, repo)
	}
	if replay := codexGoalWorkspaceRootForSourceV0(repo); replay != root {
		t.Fatalf("root not stable: %q / %q", root, replay)
	}
}

func newCodexGoalWorkspaceAdapterGitRepoV0(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.invalid"},
		{"config", "user.name", "Orquesta Test"},
	} {
		codexGoalWorkspaceAdapterRunGitV0(t, repo, args...)
	}
	if err := os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	codexGoalWorkspaceAdapterRunGitV0(t, repo, "add", "base.txt")
	codexGoalWorkspaceAdapterRunGitV0(t, repo, "commit", "-q", "-m", "base")
	return repo
}

func codexGoalWorkspaceAdapterRunGitV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
