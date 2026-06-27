package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type CodexStackExternalWorkDryRunExecutorV0 struct {
	Inner        orquestamcp.MCPExternalWorkDryRunToolExecutorV0
	DefaultModel string
}

var _ orquestamcp.MCPTransportExternalWorkDryRunExecutorV0 = CodexStackExternalWorkDryRunExecutorV0{}

func NewCodexStackExternalWorkDryRunExecutorV0(
	inner orquestamcp.MCPExternalWorkDryRunToolExecutorV0,
	defaultModel string,
) CodexStackExternalWorkDryRunExecutorV0 {
	return CodexStackExternalWorkDryRunExecutorV0{
		Inner:        inner,
		DefaultModel: strings.TrimSpace(defaultModel),
	}
}

func (executor CodexStackExternalWorkDryRunExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkDryRunToolInputV0,
) (orquestamcp.MCPExternalWorkDryRunToolResultV0, error) {
	if strings.TrimSpace(input.Model) == "" {
		input.Model = strings.TrimSpace(executor.DefaultModel)
	}
	return executor.Inner.Execute(ctx, input)
}

type ExternalWorkDryRunProjectWorkDirGuardExecutorV0 struct {
	Next   orquestamcp.MCPTransportExternalWorkDryRunExecutorV0
	Config ExternalWorkRunProjectWorkDirGuardConfigV0
}

var _ orquestamcp.MCPTransportExternalWorkDryRunExecutorV0 = ExternalWorkDryRunProjectWorkDirGuardExecutorV0{}

func NewExternalWorkDryRunProjectWorkDirGuardExecutorV0(
	next orquestamcp.MCPTransportExternalWorkDryRunExecutorV0,
	config ExternalWorkRunProjectWorkDirGuardConfigV0,
) ExternalWorkDryRunProjectWorkDirGuardExecutorV0 {
	return ExternalWorkDryRunProjectWorkDirGuardExecutorV0{Next: next, Config: config}
}

func (executor ExternalWorkDryRunProjectWorkDirGuardExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkDryRunToolInputV0,
) (orquestamcp.MCPExternalWorkDryRunToolResultV0, error) {
	runGuard := ExternalWorkRunProjectWorkDirGuardExecutorV0{Config: executor.Config}
	if issue, blocked := runGuard.blockingIssueV0(input.MCPExternalWorkRunToolInputV0); blocked {
		return orquestamcp.MCPExternalWorkDryRunToolResultV0{
			Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
			DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
			RequestID: strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(
				input.RequestID,
				input.ExternalWorkRunRequest.RequestID,
			)),
			CorrelationID: strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(
				input.CorrelationID,
				input.ExternalWorkRunRequest.CorrelationID,
				input.RequestID,
			)),
			Errores: []orquestamcp.MCPExternalWorkRunIssueV0{issue},
		}, nil
	}
	if executor.Next == nil {
		return orquestamcp.MCPExternalWorkDryRunToolResultV0{
			Estado:                orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			RoutePolicy:           orquestamcp.MCPExternalWorkRunRoutePolicyGoalFirstV0,
			DirectorExecutionMode: orquestamcp.MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
			Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
				Code:  "external_work_dry_run_guard_next_missing",
				Field: "executor",
			}},
		}, nil
	}
	return executor.Next.Execute(ctx, input)
}

func externalWorkDryRunExecutorV0(
	config ConfigV0,
	queueConfig RunQueueConfigV0,
) orquestamcp.MCPTransportExternalWorkDryRunExecutorV0 {
	executor := orquestamcp.MCPTransportExternalWorkDryRunExecutorV0(
		NewCodexStackExternalWorkDryRunExecutorV0(
			orquestamcp.MCPExternalWorkDryRunToolExecutorV0{
				Config: externalWorkRunStartConfigV0(config, queueConfig),
			},
			firstExternalWorkDryRunModelV0(config.Capacity.ModelRef, config.Codex.Model),
		),
	)
	guard := config.ExternalWorkRunGuard
	if len(guard.Rules) == 0 {
		return executor
	}
	if strings.TrimSpace(guard.ProjectWorkDir) == "" {
		guard.ProjectWorkDir = strings.TrimSpace(config.Codex.ProjectWorkDir)
	}
	return NewExternalWorkDryRunProjectWorkDirGuardExecutorV0(executor, guard)
}

func firstExternalWorkDryRunModelV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
