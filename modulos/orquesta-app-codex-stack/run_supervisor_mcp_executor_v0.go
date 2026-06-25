package orquestaappcodexstack

import (
	"context"
	"strconv"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type CodexStackRunSupervisorExecutorV0 struct {
	Stack *StackV0
}

var _ orquestamcp.MCPTransportRunSupervisorExecutorV0 = CodexStackRunSupervisorExecutorV0{}

const (
	defaultCodexStackRunSupervisorMaxBurstsV0            = 1
	defaultCodexStackRunSupervisorMaxStepsPerBurstV0     = 1
	defaultCodexStackRunSupervisorMaxDispatchesPerWaitV0 = 1
	defaultCodexStackRunSupervisorMaxCommandsV0          = 1
	defaultCodexStackRunSupervisorMaxOutboxPerCycleV0    = 1
	defaultCodexStackRunSupervisorMaxDecisionCyclesV0    = 1
	defaultCodexStackRunSupervisorMaxExternalWaitsV0     = 1

	defaultCodexStackDirectOPESMaxBurstsV0            = 16
	defaultCodexStackDirectOPESMaxStepsPerBurstV0     = 12
	defaultCodexStackDirectOPESMaxDispatchesPerWaitV0 = 8
	defaultCodexStackDirectOPESMaxCommandsV0          = 20
	defaultCodexStackDirectOPESMaxOutboxPerCycleV0    = 8
	defaultCodexStackDirectOPESMaxDecisionCyclesV0    = 4
)

func NewCodexStackRunSupervisorExecutorV0(
	stack *StackV0,
) CodexStackRunSupervisorExecutorV0 {
	return CodexStackRunSupervisorExecutorV0{Stack: stack}
}

func (executor CodexStackRunSupervisorExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, error) {
	if executor.Stack == nil {
		return orquestamcp.NewMCPRunSupervisorErrorResultV0(
			input,
			"run_supervisor_stack_no_configurado",
			"stack",
			"stack requerido",
		), nil
	}
	if result, blocked := rejectAutoprogrammingResidentWaitOverrideV0(input); blocked {
		return result, nil
	}
	input = normalizeAutoprogrammingResidentInputV0(input, *executor.Stack)
	input = executor.Stack.normalizeDirectOPESRunSupervisorInputV0(ctx, input)
	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:             *executor.Stack,
		RunRef:            strings.TrimSpace(input.RunRef),
		DrainRequest:      codexStackRunSupervisorDrainRequestV0(input),
		SupervisorCommand: codexStackRunSupervisorCommandV0(input),
	}
	agentLifecycle := CodexSupervisorAgentLifecyclePortV0(lifecycle)
	if input.ResidentMode && strings.TrimSpace(input.RunRef) == "" {
		agentLifecycle = autoprogrammingResidentLifecycleV0{inner: lifecycle}
	}
	result, err := SuperviseCodexV0(
		ctx,
		CodexSupervisorDepsV0{AgentLifecycle: agentLifecycle},
		CodexSupervisorCommandV0{
			MaxTicks:        input.MaxTicks,
			ContinueMessage: input.ContinueMessage,
		},
	)
	if err != nil {
		return codexStackRunSupervisorErrorResultMCPV0(input, result, err), err
	}
	output := orquestamcp.NewMCPRunSupervisorOKResultV0(
		input,
		result.Last.SessionRef,
		string(result.StopReason),
		result.Ticks,
		codexStackRunSupervisorSnapshotMCPV0(result.Last),
		codexStackRunSupervisorHistoryMCPV0(result.History),
	)
	output.NextActions = codexStackRunSupervisorNextActionsMCPV0(result)
	output.Diagnostics = append(
		output.Diagnostics,
		codexStackRunSupervisorDiagnosticsMCPV0(result.Last.Diagnostics)...,
	)
	output.Diagnostics = append(
		output.Diagnostics,
		executor.Stack.codexStackRunSupervisorQueueDiagnosticsMCPV0(ctx, input, result)...,
	)
	output = addAutoprogrammingResidentEvidenceV0(input, output)
	output = maybePrepareAutoprogrammingResidentSelfRepairV0(ctx, input, *executor.Stack, result, output)
	return output, nil
}

func codexStackRunSupervisorDrainRequestV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
) DrainRunRequestV0 {
	return DrainRunRequestV0{
		RunRef:                     strings.TrimSpace(input.RunRef),
		OccurredAt:                 strings.TrimSpace(input.OccurredAt),
		CorrelationID:              firstNonEmptyQueuedSourceV0(input.CorrelationID, input.RequestID),
		MaxBursts:                  codexStackPositiveOrDefaultV0(input.MaxBursts, defaultCodexStackRunSupervisorMaxBurstsV0),
		MaxStepsPerBurst:           codexStackPositiveOrDefaultV0(input.MaxStepsPerBurst, defaultCodexStackRunSupervisorMaxStepsPerBurstV0),
		MaxDispatchesPerWait:       codexStackPositiveOrDefaultV0(input.MaxDispatchesPerWait, defaultCodexStackRunSupervisorMaxDispatchesPerWaitV0),
		MaxCommands:                codexStackPositiveOrDefaultV0(input.MaxCommands, defaultCodexStackRunSupervisorMaxCommandsV0),
		MaxOutboxPerCycle:          codexStackPositiveOrDefaultV0(input.MaxOutboxPerCycle, defaultCodexStackRunSupervisorMaxOutboxPerCycleV0),
		MaxDecisionCycles:          codexStackPositiveOrDefaultV0(input.MaxDecisionCycles, defaultCodexStackRunSupervisorMaxDecisionCyclesV0),
		MaxExternalWaits:           codexStackPositiveOrDefaultV0(input.MaxExternalWaits, defaultCodexStackRunSupervisorMaxExternalWaitsV0),
		OperationalDirectorPlanRef: strings.TrimSpace(input.OperationalDirectorPlanRef),
	}
}

func (stack StackV0) normalizeDirectOPESRunSupervisorInputV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) orquestamcp.MCPRunSupervisorToolInputV0 {
	if input.ResidentMode || strings.TrimSpace(input.RunRef) == "" || stack.Stores.RunStore == nil {
		return input
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, strings.TrimSpace(input.RunRef))
	if err != nil || !codexStackRunLooksOPESDirectWorkV0(run.ProjectRef, run.AppSpecRef) {
		return input
	}
	if input.MaxBursts <= 0 {
		input.MaxBursts = defaultCodexStackDirectOPESMaxBurstsV0
	}
	if input.MaxStepsPerBurst <= 0 {
		input.MaxStepsPerBurst = defaultCodexStackDirectOPESMaxStepsPerBurstV0
	}
	if input.MaxDispatchesPerWait <= 0 {
		input.MaxDispatchesPerWait = defaultCodexStackDirectOPESMaxDispatchesPerWaitV0
	}
	if input.MaxCommands <= 0 {
		input.MaxCommands = defaultCodexStackDirectOPESMaxCommandsV0
	}
	if input.MaxOutboxPerCycle <= 0 {
		input.MaxOutboxPerCycle = defaultCodexStackDirectOPESMaxOutboxPerCycleV0
	}
	if input.MaxDecisionCycles <= 0 {
		input.MaxDecisionCycles = defaultCodexStackDirectOPESMaxDecisionCyclesV0
	}
	return input
}

func codexStackRunLooksOPESDirectWorkV0(projectRef string, appSpecRef string) bool {
	projectRef = strings.ToLower(strings.TrimSpace(projectRef))
	appSpecRef = strings.ToLower(strings.TrimSpace(appSpecRef))
	return projectRef == "opes" ||
		strings.HasPrefix(projectRef, "opes-") ||
		strings.HasPrefix(appSpecRef, "app-spec-external-work-opes")
}

func codexStackPositiveOrDefaultV0(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func codexStackRunSupervisorCommandV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
) orquestarunsupervisor.RunSupervisorCommandV0 {
	return orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          strings.TrimSpace(input.QueueRef),
		MaxTicks:          1,
		MaxRunsPerTick:    input.MaxRunsPerTick,
		MaxExecutions:     input.MaxExecutions,
		StopOnNoExecution: true,
		AllowRepeatedRuns: input.AllowRepeatedRuns,
		OccurredAt:        codexStackRunSupervisorTimeV0(input.OccurredAt),
		CorrelationID:     firstNonEmptyQueuedSourceV0(input.CorrelationID, input.RequestID),
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            codexStackPositiveOrDefaultV0(input.MaxBursts, defaultCodexStackRunSupervisorMaxBurstsV0),
			MaxStepsPerBurst:     codexStackPositiveOrDefaultV0(input.MaxStepsPerBurst, defaultCodexStackRunSupervisorMaxStepsPerBurstV0),
			MaxDispatchesPerWait: codexStackPositiveOrDefaultV0(input.MaxDispatchesPerWait, defaultCodexStackRunSupervisorMaxDispatchesPerWaitV0),
			MaxCommands:          codexStackPositiveOrDefaultV0(input.MaxCommands, defaultCodexStackRunSupervisorMaxCommandsV0),
			MaxOutboxPerCycle:    codexStackPositiveOrDefaultV0(input.MaxOutboxPerCycle, defaultCodexStackRunSupervisorMaxOutboxPerCycleV0),
			MaxDecisionCycles:    codexStackPositiveOrDefaultV0(input.MaxDecisionCycles, defaultCodexStackRunSupervisorMaxDecisionCyclesV0),
			MaxExternalWaits:     codexStackPositiveOrDefaultV0(input.MaxExternalWaits, defaultCodexStackRunSupervisorMaxExternalWaitsV0),
		},
	}
}

func codexStackRunSupervisorTimeV0(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func codexStackRunSupervisorHistoryMCPV0(
	history []CodexSupervisorTickV0,
) []orquestamcp.MCPRunSupervisorTickV0 {
	out := make([]orquestamcp.MCPRunSupervisorTickV0, 0, len(history))
	for _, tick := range history {
		out = append(out, orquestamcp.MCPRunSupervisorTickV0{
			TickNumber: tick.TickNumber,
			Action:     strings.TrimSpace(tick.Action),
			Snapshot:   codexStackRunSupervisorSnapshotMCPV0(tick.Snapshot),
		})
	}
	return out
}

func codexStackRunSupervisorSnapshotMCPV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) orquestamcp.MCPRunSupervisorSnapshotV0 {
	return orquestamcp.MCPRunSupervisorSnapshotV0{
		Status:       strings.TrimSpace(string(snapshot.Status)),
		SessionRef:   strings.TrimSpace(snapshot.SessionRef),
		AgentRef:     strings.TrimSpace(snapshot.AgentRef),
		ProcessRef:   strings.TrimSpace(snapshot.ProcessRef),
		EvidenceRefs: compactStringsV0(snapshot.EvidenceRefs),
	}
}

func codexStackRunSupervisorNextActionsMCPV0(
	result CodexSupervisorResultV0,
) []string {
	switch result.Last.Status {
	case CodexSupervisorRuntimeRunningLiveV0:
		return []string{
			"dispatch_started_poll_run_ref_for_ack_or_completion",
			"do_not_relaunch_same_run_ref_while_process_live",
		}
	case CodexSupervisorRuntimeWaitingOutboxV0:
		return []string{
			"waiting_outbox_supervise_again_or_wait_for_capacity",
			"inspect_queue_pressure_if_repeated",
		}
	case CodexSupervisorRuntimeNeedsReplanV0:
		return []string{
			"replan_operational_director_active_step",
			"do_not_mark_run_failed_without_replan",
		}
	case CodexSupervisorRuntimeStopPendingV0:
		return []string{
			"stop_pending_supervise_again_until_runtime_stopped",
			"do_not_mark_run_stopped_without_stop_confirmation",
		}
	}
	if result.StopReason == CodexSupervisorStopDispatchV0 {
		return []string{"dispatch_started_poll_run_ref_for_ack_or_completion"}
	}
	return nil
}

func (stack *StackV0) codexStackRunSupervisorQueueDiagnosticsMCPV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result CodexSupervisorResultV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if stack == nil ||
		stack.Stores.RunQueue == nil ||
		result.Last.Status != CodexSupervisorRuntimeWaitingOutboxV0 {
		return nil
	}
	queue := normalizeRunQueueConfigV0(stack.RunQueue)
	queueRef := firstNonEmptyQueuedSourceV0(input.QueueRef, queue.QueueRef)
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             queueRef,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:    "run_supervisor_queue_pressure_unavailable",
			Scope:   "queue:" + strings.TrimSpace(queueRef),
			Message: "queue_pressure_unavailable",
		}}
	}
	counts := map[string]int{}
	executable := 0
	for _, candidate := range candidates {
		status := strings.TrimSpace(candidate.Status)
		if status == "" {
			status = "unknown"
		}
		counts[status]++
		if orquestarunqueue.IsExecutableRunStatusV0(status) {
			executable++
		}
	}
	message := strings.Join(compactStringsV0([]string{
		"queue_ref=" + strings.TrimSpace(queueRef),
		"total=" + strconv.Itoa(len(candidates)),
		"executable=" + strconv.Itoa(executable),
		"ready=" + strconv.Itoa(counts[orquestarunqueue.RunStatusReadyV0]),
		"running=" + strconv.Itoa(counts[orquestarunqueue.RunStatusRunningV0]),
		"delivered=" + strconv.Itoa(counts[orquestarunqueue.RunStatusDeliveredV0]),
		"stopped=" + strconv.Itoa(counts[orquestarunqueue.RunStatusStoppedV0]),
		"closed=" + strconv.Itoa(counts[orquestarunqueue.RunStatusClosedV0]),
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:    "run_supervisor_queue_pressure",
		Scope:   "queue:" + strings.TrimSpace(queueRef),
		Message: message,
	}}
}
