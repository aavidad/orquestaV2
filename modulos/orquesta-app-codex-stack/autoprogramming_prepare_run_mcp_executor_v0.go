package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

type CodexStackAutoprogrammingPrepareRunExecutorV0 struct {
	Stack              *StackV0
	DefaultOccurredAt  string
	DefaultRequestedBy string
	QueueWriter        orquestarunqueue.RunQueuePriorityWriterPortV0
	Queue              RunQueueConfigV0
	Clock              orquestafactoryhttp.AppSpecHTTPClockV0
	RuntimeWorkDir     string
}

var _ orquestamcp.MCPTransportAutoprogrammingPrepareRunExecutorV0 = CodexStackAutoprogrammingPrepareRunExecutorV0{}

func NewCodexStackAutoprogrammingPrepareRunExecutorV0(
	stack *StackV0,
	defaultOccurredAt string,
	defaultRequestedBy string,
	queueWriter orquestarunqueue.RunQueuePriorityWriterPortV0,
	queue RunQueueConfigV0,
	clock orquestafactoryhttp.AppSpecHTTPClockV0,
	runtimeWorkDir string,
) CodexStackAutoprogrammingPrepareRunExecutorV0 {
	return CodexStackAutoprogrammingPrepareRunExecutorV0{
		Stack:              stack,
		DefaultOccurredAt:  strings.TrimSpace(defaultOccurredAt),
		DefaultRequestedBy: strings.TrimSpace(defaultRequestedBy),
		QueueWriter:        queueWriter,
		Queue:              normalizeRunQueueConfigV0(queue),
		Clock:              clock,
		RuntimeWorkDir:     strings.TrimSpace(runtimeWorkDir),
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
	if !autoprogrammingPrepareRunNeedsFreshAttemptWithRuntimeV0(run, executor.RuntimeWorkDir) {
		return input
	}
	_ = executor.markStaleAutoprogrammingQueueCandidateV0(ctx, run)
	nextAttempt := autoprogrammingPrepareRetryAttemptV0(requestRef) + 1
	nextRef := autoprogrammingPrepareRetryRefV0(requestRef, input.OccurredAt, stackNowV0(executor.Clock))
	input.RequestID = nextRef
	input.AutoprogrammingRequest.RequestRef = nextRef
	input.AutoprogrammingRequest = reframeAutoprogrammingRequestAfterRepeatedRetriesV0(
		input.AutoprogrammingRequest,
		requestRef,
		nextAttempt,
	)
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
	return AutoprogrammingRunNeedsFreshAttemptV0(run)
}

func autoprogrammingPrepareRunNeedsFreshAttemptWithRuntimeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	runtimeWorkDir string,
) bool {
	return AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(run, runtimeWorkDir)
}

func AutoprogrammingRunNeedsFreshAttemptV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	if strings.TrimSpace(run.RunID) == "" ||
		run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!autoprogrammingPrepareRunHasFailureSignalV0(run) {
		return false
	}
	return !autoprogrammingPrepareRunHasLivePendingAgentV0(run)
}

func AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	runtimeWorkDir string,
) bool {
	if AutoprogrammingRunNeedsFreshAttemptV0(run) {
		return true
	}
	pendingAgents := AutoprogrammingRunPendingAgentRefsV0(run)
	return AutoprogrammingRunHasFreshAttemptFailureSignalV0(run) &&
		len(pendingAgents) > 0 &&
		!autoprogrammingRunHasRuntimeDirForAnyAgentV0(runtimeWorkDir, run.RunID, pendingAgents)
}

func AutoprogrammingRunHasFreshAttemptFailureSignalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return autoprogrammingPrepareRunHasFailureSignalV0(run)
}

func AutoprogrammingRunPendingAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	delivered := autoprogrammingPrepareStringSetV0(run.DeliveredAgents)
	failed := autoprogrammingPrepareStringSetV0(run.FailedAgents)
	lost := autoprogrammingPrepareStringSetV0(run.LostAgents)
	stopped := autoprogrammingPrepareStringSetV0(run.StoppedAgents)
	confirmedStopped := autoprogrammingPrepareStringSetV0(run.ConfirmedStoppedAgents)
	out := make([]string, 0)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if delivered[agentRef] ||
			failed[agentRef] ||
			lost[agentRef] ||
			stopped[agentRef] ||
			confirmedStopped[agentRef] {
			continue
		}
		out = append(out, agentRef)
	}
	return out
}

func autoprogrammingPrepareRunHasFailureSignalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(compactStringsV0(run.FailedAgents)) > 0 ||
		len(compactStringsV0(run.LostAgents)) > 0 ||
		autoprogrammingPrepareRunHasTerminalAssessmentV0(run)
}

func autoprogrammingPrepareRunHasTerminalAssessmentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	for _, raw := range compactStringsV0(run.AgentAssessments) {
		projection, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(raw)
		if !ok {
			continue
		}
		if projection.Action == orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
			return true
		}
		switch projection.Verdict {
		case orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
			orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0,
			orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0:
			return true
		}
	}
	return false
}

func autoprogrammingPrepareRunHasLivePendingAgentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return len(AutoprogrammingRunPendingAgentRefsV0(run)) > 0
}

func autoprogrammingRunHasRuntimeDirForAnyAgentV0(
	runtimeWorkDir string,
	runRef string,
	agentRefs []string,
) bool {
	runtimeWorkDir = strings.TrimSpace(runtimeWorkDir)
	runRef = strings.TrimSpace(runRef)
	if runtimeWorkDir == "" || runRef == "" {
		return false
	}
	for _, agentRef := range compactStringsV0(agentRefs) {
		info, err := os.Stat(filepath.Join(runtimeWorkDir, runRef, agentRef))
		if err == nil && info.IsDir() {
			return true
		}
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

func autoprogrammingPrepareRetryAttemptV0(requestRef string) int {
	return strings.Count(strings.TrimSpace(requestRef), "-retry-")
}

func reframeAutoprogrammingRequestAfterRepeatedRetriesV0(
	request orquestaautoprogramming.AutoprogrammingRequestV0,
	previousRunRef string,
	nextAttempt int,
) orquestaautoprogramming.AutoprogrammingRequestV0 {
	if nextAttempt < 2 || len(request.Tasks) == 0 {
		return request
	}
	out := request
	out.Tasks = append([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0(nil), request.Tasks...)
	strategyRef := fmt.Sprintf("retry_strategy:reframe-%02d", nextAttempt)
	previousRef := "previous_run_ref:" + autoprogrammingPrepareRetrySafeRefV0(previousRunRef)
	for index := range out.Tasks {
		out.Tasks[index] = reframeAutoprogrammingTaskAfterRepeatedRetriesV0(
			out.Tasks[index],
			nextAttempt,
			strategyRef,
			previousRef,
		)
	}
	return out
}

func reframeAutoprogrammingTaskAfterRepeatedRetriesV0(
	task orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0,
	nextAttempt int,
	strategyRef string,
	previousRef string,
) orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	task.Context = compactStringsV0(append(
		append([]string(nil), task.Context...),
		fmt.Sprintf("Retry %d: el intento anterior quedo atascado; cambia enfoque antes de reintentar.", nextAttempt),
		"Si el contrato es demasiado amplio, entrega un minimo verificable y deja followups causales.",
	))
	task.ContextRefs = compactStringsV0(append(
		append([]string(nil), task.ContextRefs...),
		strategyRef,
		previousRef,
	))
	task.AcceptanceCriteria = compactStringsV0(append(
		append([]string(nil), task.AcceptanceCriteria...),
		"No repetir literalmente el intento fallido; reencuadrar, dividir o normalizar antes de programar.",
		"El ACK explica que cambio de enfoque se aplico y que pruebas verifican el resultado.",
	))
	task.CompactRules = compactStringsV0(append(
		append([]string(nil), task.CompactRules...),
		fmt.Sprintf("retry_reframed_attempt:%02d", nextAttempt),
		"preferir cambios pequenos verificables si el intento anterior fallo por tamano, payload o contrato",
	))
	if strings.TrimSpace(task.Objective) == "" {
		task.Objective = "Reintento reencuadrado: completar la mejora con el menor cambio verificable."
	}
	return task
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
	if terminalStatus := autoprogrammingPreparedRunTerminalQueueStatusV0(result.Run); terminalStatus != "" {
		return executor.markPreparedRunTerminalV0(ctx, input, result, terminalStatus)
	}
	if blockedStatus, blocked, err := executor.queueStatusForPreparedRunControlV0(ctx, runRef); err != nil {
		return err
	} else if blocked {
		return executor.markPreparedRunControlBlockedV0(ctx, input, result, blockedStatus)
	}
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        orquestarunqueue.RunStatusReadyV0,
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

func autoprogrammingPreparedRunTerminalQueueStatusV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		return orquestarunqueue.RunStatusClosedV0
	}
	return ""
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markPreparedRunTerminalV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
	status string,
) error {
	runRef := strings.TrimSpace(result.Run.RunID)
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        status,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run_terminal_reconciled",
		IdempotencyKey: "idem-run-queue-autoprogramming-terminal-" + runRef,
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-prepare-run-terminal-reconciled",
			"evidence-ref-autoprogramming-run-not-requeued",
		},
	})
	return err
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) queueStatusForPreparedRunControlV0(
	ctx context.Context,
	runRef string,
) (string, bool, error) {
	if executor.Stack == nil || executor.Stack.Stores.RunControl == nil {
		return "", false, nil
	}
	state, err := executor.Stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
		if errorAsRunControlNotFoundV0(err, &notFound) {
			return "", false, nil
		}
		return "", false, err
	}
	evaluation := orquestaruncontrol.EvaluateRunControlV0(state)
	if evaluation.DispatchAllowed || evaluation.StopAgentsAllowed {
		return "", false, nil
	}
	switch orquestaruncontrol.NormalizeRunControlStatusV0(state.Status) {
	case orquestaruncontrol.RunControlStatusPausedV0:
		return orquestarunqueue.RunStatusPausedV0, true, nil
	case orquestaruncontrol.RunControlStatusStopRequestedV0, orquestaruncontrol.RunControlStatusStoppedV0:
		return orquestarunqueue.RunStatusStoppedV0, true, nil
	case orquestaruncontrol.RunControlStatusCancelRequestedV0, orquestaruncontrol.RunControlStatusCanceledV0:
		return orquestarunqueue.RunStatusCanceledV0, true, nil
	default:
		return "", false, nil
	}
}

func (executor CodexStackAutoprogrammingPrepareRunExecutorV0) markPreparedRunControlBlockedV0(
	ctx context.Context,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
	result AutoprogrammingBridgeResultV0,
	status string,
) error {
	runRef := strings.TrimSpace(result.Run.RunID)
	_, err := executor.QueueWriter.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      executor.Queue.QueueRef,
		AppRef:        autoprogrammingQueueAppRefV0(result),
		Status:        status,
		PriorityScore: 0,
		UpdatedAt:     stackNowV0(executor.Clock),
		RequestedBy: firstNonEmptyAutoprogrammingStackV0(
			input.RequestedBy,
			result.Continue.RequestedBy,
			executor.DefaultRequestedBy,
			"orquesta-app-codex-stack-autoprogramming",
		),
		Reason:         "autoprogramming_prepare_run_control_blocked",
		IdempotencyKey: "idem-run-queue-autoprogramming-control-blocked-" + runRef,
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-prepare-run-control-blocked",
			"evidence-ref-autoprogramming-run-not-requeued",
		},
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
