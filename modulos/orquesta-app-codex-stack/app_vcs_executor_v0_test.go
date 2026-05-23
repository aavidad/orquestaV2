package orquestaappcodexstack

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackAppVCSExecutorV0CommitLocalSinFiltrarPath(t *testing.T) {
	repo := initStackAppVCSRepoV0(t)
	writeStackAppVCSFileV0(t, repo, "app.go", "package main\n\nfunc main(){println(\"v2\")}\n")

	result, err := NewCodexStackAppVCSExecutorV0(repo).Execute(context.Background(), orquestamcp.MCPAppVCSToolInputV0{
		RequestID:     "request-ref-stack-vcs-001",
		CorrelationID: "corr-stack-vcs-001",
		Action:        orquestamcp.MCPAppVCSActionCommitV0,
		AppRef:        "app-ref-stack-vcs-001",
		RepoRef:       "repo-ref-stack-vcs-001",
		WorktreeRef:   "worktree-ref-app-vcs-workspaces-20260523-06",
		BranchRef:     "branch-ref-app-vcs-workspaces-20260523-06",
		CommitMessage: "test: commit local app",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAppVCSEstadoOKV0 ||
		result.CommitRef == "" ||
		result.WorktreeRef != "worktree-ref-app-vcs-workspaces-20260523-06" ||
		result.BranchRef != "branch-ref-app-vcs-workspaces-20260523-06" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "app.go" {
		t.Fatalf("changed=%+v", result.ChangedPaths)
	}
}

func initStackAppVCSRepoV0(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runStackAppVCSGitV0(t, repo, "init", "-b", "main")
	runStackAppVCSGitV0(t, repo, "config", "user.name", "Orquesta Test")
	runStackAppVCSGitV0(t, repo, "config", "user.email", "orquesta@example.com")
	writeStackAppVCSFileV0(t, repo, "app.go", "package main\n\nfunc main(){}\n")
	runStackAppVCSGitV0(t, repo, "add", "app.go")
	runStackAppVCSGitV0(t, repo, "commit", "-m", "base")
	return repo
}

func writeStackAppVCSFileV0(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func runStackAppVCSGitV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}
