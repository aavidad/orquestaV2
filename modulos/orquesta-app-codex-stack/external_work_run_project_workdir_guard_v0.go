package orquestaappcodexstack

import (
	"context"
	"path/filepath"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	ExternalWorkRunProjectWorkDirMismatchV0 = "external_work_project_work_dir_mismatch"
)

type ExternalWorkRunProjectWorkDirGuardConfigV0 struct {
	ProjectWorkDir string
	Rules          []ExternalWorkRunProjectWorkDirGuardRuleV0
}

type ExternalWorkRunProjectWorkDirGuardRuleV0 struct {
	ProjectRef             string
	RequiredProjectWorkDir string
	EvidenceRef            string
}

type ExternalWorkRunProjectWorkDirGuardExecutorV0 struct {
	Next   orquestamcp.MCPTransportExternalWorkRunExecutorV0
	Config ExternalWorkRunProjectWorkDirGuardConfigV0
}

func NewExternalWorkRunProjectWorkDirGuardExecutorV0(
	next orquestamcp.MCPTransportExternalWorkRunExecutorV0,
	config ExternalWorkRunProjectWorkDirGuardConfigV0,
) ExternalWorkRunProjectWorkDirGuardExecutorV0 {
	return ExternalWorkRunProjectWorkDirGuardExecutorV0{Next: next, Config: config}
}

func (executor ExternalWorkRunProjectWorkDirGuardExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunToolResultV0, error) {
	if issue, blocked := executor.blockingIssueV0(input); blocked {
		return orquestamcp.MCPExternalWorkRunToolResultV0{
			Estado:        orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			RequestID:     strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(input.RequestID, input.ExternalWorkRunRequest.RequestID)),
			CorrelationID: strings.TrimSpace(firstExternalWorkRunGuardNonEmptyV0(input.CorrelationID, input.ExternalWorkRunRequest.CorrelationID, input.RequestID)),
			Errores:       []orquestamcp.MCPExternalWorkRunIssueV0{issue},
		}, nil
	}
	if executor.Next == nil {
		return orquestamcp.MCPExternalWorkRunToolResultV0{
			Estado: orquestamcp.MCPExternalWorkRunEstadoErrorV0,
			Errores: []orquestamcp.MCPExternalWorkRunIssueV0{{
				Code:  "external_work_run_guard_next_missing",
				Field: "executor",
			}},
		}, nil
	}
	return executor.Next.Execute(ctx, input)
}

func (executor ExternalWorkRunProjectWorkDirGuardExecutorV0) blockingIssueV0(
	input orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunIssueV0, bool) {
	refs := externalWorkRunGuardProjectRefsV0(input)
	for _, rule := range executor.Config.Rules {
		if !externalWorkRunGuardRuleMatchesV0(rule, refs) {
			continue
		}
		if sameExternalWorkRunGuardPathV0(executor.Config.ProjectWorkDir, rule.RequiredProjectWorkDir) {
			return orquestamcp.MCPExternalWorkRunIssueV0{}, false
		}
		field := "project_work_dir"
		if strings.TrimSpace(rule.EvidenceRef) != "" {
			field = field + ":" + strings.TrimSpace(rule.EvidenceRef)
		}
		return orquestamcp.MCPExternalWorkRunIssueV0{
			Code:  ExternalWorkRunProjectWorkDirMismatchV0,
			Field: field,
		}, true
	}
	return orquestamcp.MCPExternalWorkRunIssueV0{}, false
}

func externalWorkRunGuardProjectRefsV0(input orquestamcp.MCPExternalWorkRunToolInputV0) []string {
	request := input.ExternalWorkRunRequest
	appChange := request.AppChangeRequest
	if !externalWorkRunGuardHasAppChangeV0(appChange) && externalWorkRunGuardHasAppChangeV0(input.AppChangeRequest) {
		appChange = input.AppChangeRequest
	}
	refs := []string{
		request.ProjectRef,
		appChange.AppRef,
	}
	if appChange.ExternalWork != nil {
		refs = append(refs, appChange.ExternalWork.ProjectRef)
	}
	return compactExternalWorkRunGuardStringsV0(refs)
}

func externalWorkRunGuardHasAppChangeV0(request orquestaappchange.AppChangeRequestV0) bool {
	return strings.TrimSpace(request.AppRef) != "" ||
		strings.TrimSpace(request.ChangeRef) != "" ||
		strings.TrimSpace(request.UserIntent) != "" ||
		request.ExternalWork != nil
}

func externalWorkRunGuardRuleMatchesV0(
	rule ExternalWorkRunProjectWorkDirGuardRuleV0,
	refs []string,
) bool {
	projectRef := strings.TrimSpace(rule.ProjectRef)
	if projectRef == "" || strings.TrimSpace(rule.RequiredProjectWorkDir) == "" {
		return false
	}
	for _, ref := range refs {
		if strings.EqualFold(strings.TrimSpace(ref), projectRef) {
			return true
		}
	}
	return false
}

func sameExternalWorkRunGuardPathV0(left string, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	return left != "." && right != "." && left == right
}

func compactExternalWorkRunGuardStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[strings.ToLower(trimmed)] {
			continue
		}
		seen[strings.ToLower(trimmed)] = true
		out = append(out, trimmed)
	}
	return out
}

func firstExternalWorkRunGuardNonEmptyV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var _ orquestamcp.MCPTransportExternalWorkRunExecutorV0 = ExternalWorkRunProjectWorkDirGuardExecutorV0{}
