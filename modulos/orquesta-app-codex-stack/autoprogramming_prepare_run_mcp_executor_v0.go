package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type CodexStackAutoprogrammingPrepareRunExecutorV0 struct {
	Stack              *StackV0
	DefaultOccurredAt  string
	DefaultRequestedBy string
}

var _ orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0 = CodexStackAutoprogrammingPrepareRunExecutorV0{}

func NewCodexStackAutoprogrammingPrepareRunExecutorV0(
	stack *StackV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
) CodexStackAutoprogrammingPrepareRunExecutorV0 {
	return CodexStackAutoprogrammingPrepareRunExecutorV0{
		Stack:              stack,
		DefaultOccurredAt:  strings.TrimSpace(defaultOccurredAt),
		DefaultRequestedBy: strings.TrimSpace(defaultRequestedBy),
	}
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) (orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0, error) {
	if executor.Stack == nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
			input,
			"autoprogramming_prepare_run_no_stack",
			"stack",
			"stack requerido",
		), nil
	}
	bridgeRequest := codexStackAutoprogrammingBridgeRequestFromMCPV0(
		input,
		executor.DefaultOccurredAt,
		executor.DefaultRequestedBy,
	)
	result, err := PrepareAutoprogrammingRunFromStackV0(ctx, *executor.Stack, bridgeRequest)
	if err != nil {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
			input,
			"autoprogramming_prepare_run_error",
			"executor",
			err.Error(),
		), nil
	}
	return codexStackAutoprogrammingPrepareRunResultMCPV0(input, result), nil
}

func codexStackAutoprogrammingBridgeRequestFromMCPV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
) AutoprogrammingBridgeRequestV0 {
	request := input.AutoprogrammingRequest
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyAutoprogrammingStackV0(input.RequestID, input.CorrelationID)
	}
	return AutoprogrammingBridgeRequestV0{
		Request:              request,
		OccurredAt:           firstNonEmptyAutoprogrammingStackV0(input.OccurredAt, defaultOccurredAt),
		CorrelationID:        firstNonEmptyAutoprogrammingStackV0(input.CorrelationID, input.RequestID, request.RequestRef),
		RequestedBy:          firstNonEmptyAutoprogrammingStackV0(input.RequestedBy, defaultRequestedBy),
		MaxBursts:            input.MaxBursts,
		MaxStepsPerBurst:     input.MaxStepsPerBurst,
		MaxDispatchesPerWait: input.MaxDispatchesPerWait,
		MaxCommands:          input.MaxCommands,
		MaxOutboxPerCycle:    input.MaxOutboxPerCycle,
	}
}

func codexStackAutoprogrammingPrepareRunResultMCPV0(
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	if !result.Accepted {
		return orquestamcp.NewMCPAutoprogrammingPrepareRunIssuesResultV0(
			input,
			codexStackAutoprogrammingIssuesMCPV0(result.Issues),
		)
	}
	out := orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:           orquestamcp.MCPAutoprogrammingPrepareRunEstadoOKV0,
		RequestID:        firstNonEmptyAutoprogrammingStackV0(input.RequestID, result.Work.RequestRef),
		CorrelationID:    firstNonEmptyAutoprogrammingStackV0(input.CorrelationID, input.RequestID, result.Work.RequestRef),
		Accepted:         true,
		RunRef:           strings.TrimSpace(result.Run.RunID),
		ProjectRef:       strings.TrimSpace(result.Work.ProjectRef),
		WorktreeRef:      strings.TrimSpace(result.Work.WorktreeRef),
		BranchRef:        strings.TrimSpace(result.Work.BranchRef),
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		WorkflowTaskRefs: codexStackAutoprogrammingWorkflowTaskRefsMCPV0(result.Tasks),
		WaitAgentRefs:    compactStringsV0(result.WaitAgentRefs),
		Errores:          []orquestamcp.MCPValidationIssueV0{},
	}
	out.Continue = &orquestamcp.MCPAutoprogrammingContinueRequestV0{
		RunRef:               strings.TrimSpace(result.Continue.RunRef),
		OccurredAt:           strings.TrimSpace(result.Continue.OccurredAt),
		CorrelationID:        strings.TrimSpace(result.Continue.CorrelationID),
		RequestedBy:          strings.TrimSpace(result.Continue.RequestedBy),
		MaxBursts:            result.Continue.MaxBursts,
		MaxStepsPerBurst:     result.Continue.MaxStepsPerBurst,
		MaxDispatchesPerWait: result.Continue.MaxDispatchesPerWait,
		WaitAgentRefs:        compactStringsV0(result.Continue.WaitAgentRefs),
		MaxCommands:          result.Continue.MaxCommands,
		MaxOutboxPerCycle:    result.Continue.MaxOutboxPerCycle,
	}
	return out
}

func codexStackAutoprogrammingWorkflowTaskRefsMCPV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.TaskID)
	}
	return compactStringsV0(refs)
}

func codexStackAutoprogrammingIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []orquestamcp.MCPValidationIssueV0 {
	out := make([]orquestamcp.MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestamcp.MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []orquestamcp.MCPValidationIssueV0{}
	}
	return out
}

func firstNonEmptyAutoprogrammingStackV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
