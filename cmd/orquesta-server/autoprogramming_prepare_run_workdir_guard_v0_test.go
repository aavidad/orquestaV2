package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestAutoprogrammingPrepareRunRuntimeWorkdirGuardV0RechazaWorktreeInvalido(t *testing.T) {
	inner := &recordingPrepareRunExecutorForWorkdirGuardV0{}
	executor := serverWorkdirGuardedPrepareRunExecutorV0{
		inner:          inner,
		projectWorkDir: t.TempDir(),
	}

	result, err := executor.Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-ref-prepare-run-workdir-invalid",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != serverPrepareRunWorkdirInvalidV0 {
		t.Fatalf("result=%+v", result)
	}
	if inner.called {
		t.Fatalf("prepare-run no debe llegar al executor interno con worktree invalido")
	}
}

func TestAutoprogrammingPrepareRunRuntimeWorkdirGuardV0RechazaWorktreeStale(t *testing.T) {
	repo := initPrepareRunGuardRepoV0(t)
	runPrepareRunGuardGitV0(t, repo, "branch", "upstream")
	runPrepareRunGuardGitV0(t, repo, "checkout", "upstream")
	writePrepareRunGuardFileV0(t, repo, "next.txt", "next\n")
	runPrepareRunGuardGitV0(t, repo, "add", "next.txt")
	runPrepareRunGuardGitV0(t, repo, "commit", "-m", "next")
	runPrepareRunGuardGitV0(t, repo, "checkout", "main")
	runPrepareRunGuardGitV0(t, repo, "branch", "--set-upstream-to", "upstream", "main")
	inner := &recordingPrepareRunExecutorForWorkdirGuardV0{}
	executor := serverWorkdirGuardedPrepareRunExecutorV0{
		inner:          inner,
		projectWorkDir: repo,
	}

	result, err := executor.Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-ref-prepare-run-workdir-stale",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != serverPrepareRunWorkdirStaleV0 {
		t.Fatalf("result=%+v", result)
	}
	if inner.called {
		t.Fatalf("prepare-run no debe llegar al executor interno con worktree stale")
	}
}

func TestAutoprogrammingPrepareRunRuntimeWorkdirGuardV0AceptaWorktreeAlineado(t *testing.T) {
	repo := initPrepareRunGuardRepoV0(t)
	inner := &recordingPrepareRunExecutorForWorkdirGuardV0{}
	executor := serverWorkdirGuardedPrepareRunExecutorV0{
		inner:          inner,
		projectWorkDir: repo,
	}

	result, err := executor.Execute(context.Background(), orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-ref-prepare-run-workdir-ok",
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0 || !inner.called {
		t.Fatalf("result=%+v called=%v", result, inner.called)
	}
}

type recordingPrepareRunExecutorForWorkdirGuardV0 struct {
	called bool
}

func (executor *recordingPrepareRunExecutorForWorkdirGuardV0) Execute(
	context.Context,
	orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) (orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0, error) {
	executor.called = true
	return orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:   orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
		Accepted: true,
		RunRef:   "run-ref-prepare-run-workdir-ok",
	}, nil
}

func initPrepareRunGuardRepoV0(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runPrepareRunGuardGitV0(t, repo, "init", "-b", "main")
	runPrepareRunGuardGitV0(t, repo, "config", "user.name", "Orquesta Test")
	runPrepareRunGuardGitV0(t, repo, "config", "user.email", "orquesta-test@example.invalid")
	writePrepareRunGuardFileV0(t, repo, "README.md", "ok\n")
	runPrepareRunGuardGitV0(t, repo, "add", "README.md")
	runPrepareRunGuardGitV0(t, repo, "commit", "-m", "initial")
	return repo
}

func writePrepareRunGuardFileV0(t *testing.T, repo string, rel string, content string) {
	t.Helper()
	path := filepath.Join(repo, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func runPrepareRunGuardGitV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}
