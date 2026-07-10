package main

import (
	"context"
	"os"
	"os/exec"
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
	info, err := os.Stat(projectWorkDir)
	if err != nil || !info.IsDir() {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirMissingV0)
	}
	if !prepareRunGitOKV0(ctx, projectWorkDir, "rev-parse", "--is-inside-work-tree") ||
		!prepareRunGitOKV0(ctx, projectWorkDir, "rev-parse", "--verify", "HEAD") {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirInvalidV0)
	}
	upstream := strings.TrimSpace(prepareRunGitOutputV0(ctx, projectWorkDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
	if upstream == "" {
		return nil
	}
	head := strings.TrimSpace(prepareRunGitOutputV0(ctx, projectWorkDir, "rev-parse", "HEAD"))
	upstreamSHA := strings.TrimSpace(prepareRunGitOutputV0(ctx, projectWorkDir, "rev-parse", upstream))
	if head == "" || upstreamSHA == "" || head == upstreamSHA {
		return nil
	}
	if prepareRunGitOKV0(ctx, projectWorkDir, "merge-base", "--is-ancestor", "HEAD", upstream) {
		return prepareRunWorkdirIssueV0(serverPrepareRunWorkdirStaleV0)
	}
	if !prepareRunGitOKV0(ctx, projectWorkDir, "merge-base", "--is-ancestor", upstream, "HEAD") {
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

func prepareRunGitOKV0(ctx context.Context, repo string, args ...string) bool {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	return cmd.Run() == nil
}

func prepareRunGitOutputV0(ctx context.Context, repo string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
