package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

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
	input = executor.freshAttemptForStaleAutoprogrammingRunV0(ctx, input)
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

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) freshAttemptForStaleAutoprogrammingRunV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0 {
	if executor.Stack == nil || executor.Stack.Stores.RunStore == nil {
		return input
	}
	requestRef := firstNonEmptyAutoprogrammingStackV0(
		input.AutoprogrammingRequest.RequestRef,
		input.RequestID,
		input.CorrelationID,
	)
	if strings.TrimSpace(requestRef) == "" {
		return input
	}
	run, err := executor.Stack.Stores.RunStore.LoadRunV0(ctx, requestRef)
	if err != nil {
		return input
	}
	if !autoprogrammingPrepareRunNeedsFreshAttemptV0(run) {
		return input
	}
	_ = executor.markStaleAutoprogrammingQueueCandidateV0(ctx, run)
	nextRef := autoprogrammingPrepareRetryRefV0(requestRef, input.OccurredAt, stackNowV0(executor.Clock))
	input.RequestID = nextRef
	input.AutoprogrammingRequest.RequestRef = nextRef
	if strings.TrimSpace(input.CorrelationID) == "" ||
		strings.TrimSpace(input.CorrelationID) == "corr-"+requestRef ||
		strings.TrimSpace(input.CorrelationID) == requestRef {
		input.CorrelationID = "corr-" + nextRef
	}
	return input
}

func autoprogrammingPrepareRunNeedsFreshAttemptV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if strings.TrimSpace(run.RunID) == "" ||
		run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(compactStringsV0(run.StartedAgents)) == 0 ||
		!autoprogrammingPrepareRunHasFailureSignalV0(run) {
		return false
	}
	return !autoprogrammingPrepareRunHasLivePendingAgentV0(run)
}

func autoprogrammingPrepareRunHasFailureSignalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactStringsV0(run.FailedAgents)) > 0 ||
		len(compactStringsV0(run.LostAgents)) > 0
}

func autoprogrammingPrepareRunHasLivePendingAgentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	delivered := autoprogrammingPrepareStringSetV0(run.DeliveredAgents)
	failed := autoprogrammingPrepareStringSetV0(run.FailedAgents)
	lost := autoprogrammingPrepareStringSetV0(run.LostAgents)
	stopped := autoprogrammingPrepareStringSetV0(run.StoppedAgents)
	confirmedStopped := autoprogrammingPrepareStringSetV0(run.ConfirmedStoppedAgents)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if delivered[agentRef] ||
			failed[agentRef] ||
			lost[agentRef] ||
			stopped[agentRef] ||
			confirmedStopped[agentRef] {
			continue
		}
		return true
	}
	return false
}

func autoprogrammingPrepareStringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range compactStringsV0(values) {
		out[value] = true
	}
	return out
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markStaleAutoprogrammingQueueCandidateV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if executor.QueueWriter == nil {
		return nil
	}
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        strings.TrimSpace(run.RunID),
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        firstNonEmptyAutoprogrammingStackV0(run.ProjectRef, run.AppSpecRef, run.RunID),
		Status:        orquestarunqueue.RunStatusStoppedV0,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_stale_run_retried",
		IdempotencyKey: "idem-run-queue-autoprogramming-stale-" + autoprogrammingPrepareRetrySafeRefV0(run.RunID),
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-stale-run-retried",
			"evidence-ref-autoprogramming-old-run-preserved",
		},
	})
	return err
}

func autoprogrammingPrepareRetryRefV0(requestRef string, occurredAt string, now time.Time) string {
	base := strings.TrimSpace(requestRef)
	if base == "" {
		base = "request-ref-autoprogramming"
	}
	stamp := strings.TrimSpace(occurredAt)
	if stamp == "" && !now.IsZero() {
		stamp = now.UTC().Format("20060102T150405Z")
	}
	hash := autoprogrammingPrepareRetryHashV0(base + "|" + stamp)
	return base + "-retry-" + hash
}

func autoprogrammingPrepareRetryHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:12]
}

func autoprogrammingPrepareRetrySafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 80 {
		return value
	}
	return value[:67] + "-" + autoprogrammingPrepareRetryHashV0(value)
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
