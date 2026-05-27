package orquestamcp

import "context"

type MCPAppVCSToolExecutorV0 struct {
	Executor MCPAppVCSExecutorPortV0
}

func NewMCPAppVCSToolExecutorV0(executor MCPAppVCSExecutorPortV0) MCPAppVCSToolExecutorV0 {
	return MCPAppVCSToolExecutorV0{Executor: executor}
}

func (executor MCPAppVCSToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAppVCSToolInputV0,
) (MCPAppVCSToolResultV0, error) {
	var identityIssues []MCPValidationIssueV0
	input, identityIssues = normalizeMCPAppVCSIdentityV0(input, "", "")
	if len(identityIssues) > 0 {
		return NewMCPAppVCSErrorResultV0(
			input,
			identityIssues[0].Code,
			identityIssues[0].Field,
			identityIssues[0].Message,
		), nil
	}
	input = NormalizeMCPAppVCSInputV0(input)
	if issues := ValidateMCPAppVCSInputV0(input); len(issues) > 0 {
		return MCPAppVCSToolResultV0{
			Estado:        MCPAppVCSEstadoErrorV0,
			RequestID:     input.RequestID,
			CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
			Action:        input.Action,
			AppRef:        input.AppRef,
			RepoRef:       input.RepoRef,
			Errores:       issues,
		}, nil
	}
	if executor.Executor == nil {
		return NewMCPAppVCSErrorResultV0(
			input,
			MCPAppVCSExecutorUnavailableV0,
			"executor",
			MCPAppVCSExecutorUnavailableV0,
		), nil
	}
	return executor.Executor.Execute(ctx, input)
}
