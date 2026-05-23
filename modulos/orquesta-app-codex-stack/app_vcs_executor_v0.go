package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type CodexStackAppVCSExecutorV0 struct {
	ProjectWorkDir string
	Connector      orquestaruntimeworktree.GitAppVCSConnectorV0
}

var _ orquestamcp.MCPAppVCSExecutorPortV0 = CodexStackAppVCSExecutorV0{}

func NewCodexStackAppVCSExecutorV0(projectWorkDir string) CodexStackAppVCSExecutorV0 {
	return CodexStackAppVCSExecutorV0{ProjectWorkDir: strings.TrimSpace(projectWorkDir)}
}

func (executor CodexStackAppVCSExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAppVCSToolInputV0,
) (orquestamcp.MCPAppVCSToolResultV0, error) {
	input = orquestamcp.NormalizeMCPAppVCSInputV0(input)
	request := orquestaruntimeworktree.AppVCSRequestV0{
		RequestID:      input.RequestID,
		CorrelationID:  input.CorrelationID,
		Action:         appVCSActionFromMCPV0(input.Action),
		AppRef:         input.AppRef,
		RepoRef:        input.RepoRef,
		WorktreeRef:    input.WorktreeRef,
		BranchRef:      input.BranchRef,
		ProjectWorkDir: executor.ProjectWorkDir,
		CommitMessage:  input.CommitMessage,
		CommitPaths:    append([]string(nil), input.CommitPaths...),
		AllowPush:      input.AllowPush,
	}
	result, issues := executor.Connector.ExecuteAppVCSV0(ctx, request)
	if len(issues) > 0 && result.Status != orquestaruntimeworktree.AppVCSStatusPendingPushV0 {
		return appVCSErrorToMCPV0(input, issues), nil
	}
	return appVCSResultToMCPV0(input, result, issues), nil
}

func appVCSActionFromMCPV0(action string) orquestaruntimeworktree.AppVCSActionV0 {
	return orquestaruntimeworktree.AppVCSActionV0(strings.TrimSpace(action))
}

func appVCSResultToMCPV0(
	input orquestamcp.MCPAppVCSToolInputV0,
	result orquestaruntimeworktree.AppVCSResultV0,
	issues []orquestaruntimeworktree.AppVCSIssueV0,
) orquestamcp.MCPAppVCSToolResultV0 {
	return orquestamcp.MCPAppVCSToolResultV0{
		Estado:         orquestamcp.MCPAppVCSEstadoOKV0,
		RequestID:      input.RequestID,
		CorrelationID:  firstNonEmptyAppVCSV0(input.CorrelationID, input.RequestID),
		Action:         string(result.Action),
		Status:         string(result.Status),
		AppRef:         result.AppRef,
		RepoRef:        result.RepoRef,
		WorktreeRef:    result.WorktreeRef,
		BranchRef:      result.BranchRef,
		CommitRef:      result.CommitRef,
		CommitShortRef: result.CommitShortRef,
		ChangedPaths:   compactStringsV0(result.ChangedPaths),
		PushPending:    result.PushPending,
		Retryable:      result.Retryable,
		EvidenceRefs:   compactStringsV0(result.EvidenceRefs),
		Errores:        appVCSIssuesToMCPV0(issues),
	}
}

func appVCSErrorToMCPV0(
	input orquestamcp.MCPAppVCSToolInputV0,
	issues []orquestaruntimeworktree.AppVCSIssueV0,
) orquestamcp.MCPAppVCSToolResultV0 {
	return orquestamcp.MCPAppVCSToolResultV0{
		Estado:        orquestamcp.MCPAppVCSEstadoErrorV0,
		RequestID:     input.RequestID,
		CorrelationID: firstNonEmptyAppVCSV0(input.CorrelationID, input.RequestID),
		Action:        input.Action,
		AppRef:        input.AppRef,
		RepoRef:       input.RepoRef,
		Errores:       appVCSIssuesToMCPV0(issues),
	}
}

func appVCSIssuesToMCPV0(
	issues []orquestaruntimeworktree.AppVCSIssueV0,
) []orquestamcp.MCPValidationIssueV0 {
	out := make([]orquestamcp.MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestamcp.MCPValidationIssueV0{
			Code:    string(issue.Code),
			Field:   issue.Field,
			Message: issue.MessageKey,
		})
	}
	return out
}

func firstNonEmptyAppVCSV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
