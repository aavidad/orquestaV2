package orquestaappcodexstack

import (
	"context"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type CodexStackRunSupervisorExecutorV0 struct {
	Stack *StackV0
}

var _ orquestamcp.MCPTransportRunSupervisorExecutorV0 = CodexStackRunSupervisorExecutorV0{}

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
		MaxBursts:                  codexStackPositiveOrDefaultV0(input.MaxBursts, 1),
		MaxStepsPerBurst:           codexStackPositiveOrDefaultV0(input.MaxStepsPerBurst, 1),
		MaxDispatchesPerWait:       codexStackPositiveOrDefaultV0(input.MaxDispatchesPerWait, 1),
		MaxCommands:                codexStackPositiveOrDefaultV0(input.MaxCommands, 1),
		MaxOutboxPerCycle:          codexStackPositiveOrDefaultV0(input.MaxOutboxPerCycle, 1),
		MaxDecisionCycles:          codexStackPositiveOrDefaultV0(input.MaxDecisionCycles, 1),
		MaxExternalWaits:           input.MaxExternalWaits,
		OperationalDirectorPlanRef: strings.TrimSpace(input.OperationalDirectorPlanRef),
	}
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
			MaxBursts:            codexStackPositiveOrDefaultV0(input.MaxBursts, 1),
			MaxStepsPerBurst:     codexStackPositiveOrDefaultV0(input.MaxStepsPerBurst, 1),
			MaxDispatchesPerWait: codexStackPositiveOrDefaultV0(input.MaxDispatchesPerWait, 1),
			MaxCommands:          codexStackPositiveOrDefaultV0(input.MaxCommands, 1),
			MaxOutboxPerCycle:    codexStackPositiveOrDefaultV0(input.MaxOutboxPerCycle, 1),
			MaxDecisionCycles:    codexStackPositiveOrDefaultV0(input.MaxDecisionCycles, 1),
			MaxExternalWaits:     input.MaxExternalWaits,
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
		ProcessRef:   strings.TrimSpace(snapshot.ProcessRef),
		EvidenceRefs: compactStringsV0(snapshot.EvidenceRefs),
	}
}
