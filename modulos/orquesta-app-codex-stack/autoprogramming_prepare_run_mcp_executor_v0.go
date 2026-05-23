package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

type CodexStackAutoprogrammingPrepareRunExecutorV0 struct {
	Stack              *StackV0
	DefaultOccurredAt  string
	DefaultRequestedBy string
	QueueWriter        orquestarunqueue.RunQueuePriorityWriterPortV0
	Queue              RunQueueConfigV0
	Clock              orquestafactoryhttp.AppSpecHTTPClockV0
}

var _ orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0 = CodexStackAutoprogrammingPrepareRunExecutorV0{}

func NewCodexStackAutoprogrammingPrepareRunExecutorV0(
	stack *StackV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
	queueWriter orquestarunqueue.RunQueuePriorityWriterPortV0,
	queue RunQueueConfigV0,
	clock orquestafactoryhttp.AppSpecHTTPClockV0,
) CodexStackAutoprogrammingPrepareRunExecutorV0 {
	return CodexStackAutoprogrammingPrepareRunExecutorV0{
		Stack:              stack,
		DefaultOccurredAt:  strings.TrimSpace(defaultOccurredAt),
		DefaultRequestedBy: strings.TrimSpace(defaultRequestedBy),
		QueueWriter:        queueWriter,
		Queue:              normalizeRunQueueConfigV0(queue),
		Clock:              clock,
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
	if result.Accepted {
		if err := executor.enqueuePreparedRunV0(ctx, input, result); err != nil {
			return orquestamcp.NewMCPAutoprogrammingPrepareRunErrorResultV0(
				input,
				"autoprogramming_prepare_run_queue_error",
				"run_queue",
				err.Error(),
			), nil
		}
	}
	return codexStackAutoprogrammingPrepareRunResultMCPV0(input, result), nil
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) enqueuePreparedRunV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
) error {
	if executor.QueueWriter == nil {
		return fmt.Errorf("run_queue.writer requerido")
	}
	runRef := strings.TrimSpace(result.Run.RunID)
	if runRef == "" {
		return fmt.Errorf("run_ref preparado requerido")
	}
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		PriorityScore: autoprogrammingPrepareRunPriorityScoreV0(input.PriorityScore, executor.Queue.DefaultPriorityScore),
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run",
		IdempotencyKey: "idem-run-queue-autoprogramming-prepare-" + runRef,
		EvidenceRefs:   []string{"evidence-ref-autoprogramming-prepare-run-enqueued"},
	})
	return err
}

func autoprogrammingPrepareRunPriorityScoreV0(requested int, fallback int) int {
	if requested > 0 {
		return requested
	}
	return fallback
}

func autoprogrammingQueueAppRefV0(
	result AutoprogrammingBridgeResultV0,
) string {
	return firstNonEmptyAutoprogrammingStackV0(
		result.Work.ProjectRef,
		result.Run.ProjectRef,
		result.Run.AppSpecRef,
		result.Run.RunID,
	)
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
		RunRef:                     strings.TrimSpace(result.Continue.RunRef),
		OperationalDirectorPlanRef: strings.TrimSpace(result.Continue.OperationalDirectorPlanRef),
		OccurredAt:                 strings.TrimSpace(result.Continue.OccurredAt),
		CorrelationID:              strings.TrimSpace(result.Continue.CorrelationID),
		RequestedBy:                strings.TrimSpace(result.Continue.RequestedBy),
		MaxBursts:                  result.Continue.MaxBursts,
		MaxStepsPerBurst:           result.Continue.MaxStepsPerBurst,
		MaxDispatchesPerWait:       result.Continue.MaxDispatchesPerWait,
		WaitAgentRefs:              compactStringsV0(result.Continue.WaitAgentRefs),
		MaxCommands:                result.Continue.MaxCommands,
		MaxOutboxPerCycle:          result.Continue.MaxOutboxPerCycle,
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
