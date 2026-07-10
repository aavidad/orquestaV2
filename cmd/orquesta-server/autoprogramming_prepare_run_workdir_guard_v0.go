package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	serverPrepareRunWorkdirInvalidV0    = "autoprogramming_prepare_run_workdir_invalid"
	serverPrepareRunWorkdirMissingV0    = "autoprogramming_prepare_run_workdir_missing"
	serverPrepareRunWorkdirStaleV0      = "autoprogramming_prepare_run_workdir_stale"
	serverPrepareRunWorkdirNotAlignedV0 = "autoprogramming_prepare_run_workdir_not_aligned"
)

type serverWorkdirGuardedPrepareRunExecutorV0 struct {
	inner          orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0
	projectWorkDir string
}

func (executor serverWorkdirGuardedPrepareRunExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) (orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0, error) {
	if issue := validatePrepareRunProjectWorkdirV0(ctx, executor.projectWorkDir); issue != nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(input, []orquestamcp.MCPValidationIssueV0{*issue}), nil
	}
	return executor.inner.Execute(ctx, input)
}

func guardPrepareRunExecutorWorkdirV0(
	inner orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0,
	projectWorkDir string,
) orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0 {
	if inner == nil {
		return nil
	}
	return serverWorkdirGuardedPrepareRunExecutorV0{
		inner:          inner,
		projectWorkDir: projectWorkDir,
	}
}

func validatePrepareRunProjectWorkdirV0(
	ctx context.Context,
	projectWorkDir string,
) *orquestamcp.MCPValidationIssueV0 {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirMissingV0)
	}
	abs, err := filepath.Abs(projectWorkDir)
	if err != nil {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirInvalidV0)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirMissingV0)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(abs) {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirInvalidV0)
	}
	if !serverWorktreeGitOKV0(ctx, abs, "rev-parse", "--is-inside-work-tree") ||
		!serverWorktreeGitOKV0(ctx, abs, "rev-parse", "--verify", "HEAD") {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirInvalidV0)
	}
	root := serverWorktreeGitOutputV0(ctx, abs, "rev-parse", "--show-toplevel")
	if root == "" || filepath.Clean(root) != filepath.Clean(abs) {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirInvalidV0)
	}
	upstream := serverWorktreeGitOutputV0(ctx, abs, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if upstream == "" {
		return nil
	}
	head := serverWorktreeGitOutputV0(ctx, abs, "rev-parse", "HEAD")
	upstreamSHA := serverWorktreeGitOutputV0(ctx, abs, "rev-parse", upstream)
	if head == "" || upstreamSHA == "" || head == upstreamSHA {
		return nil
	}
	if serverWorktreeGitOKV0(ctx, abs, "merge-base", "--is-ancestor", "HEAD", upstream) {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirStaleV0)
	}
	if !serverWorktreeGitOKV0(ctx, abs, "merge-base", "--is-ancestor", upstream, "HEAD") {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirNotAlignedV0)
	}
	return nil
}

func prepareRunWorkdirIssueV0(code string) *orquestamcp.MCPValidationIssueV0 {
	return &orquestamcp.MCPValidationIssueV0{
		Code:    code,
		Field:   "project_work_dir",
		Message: code,
	}
}
