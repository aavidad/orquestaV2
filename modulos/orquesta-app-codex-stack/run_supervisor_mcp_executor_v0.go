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
	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:             *executor.Stack,
		RunRef:            strings.TrimSpace(input.RunRef),
		DrainRequest:      codexStackRunSupervisorDrainRequestV0(input),
		SupervisorCommand: codexStackRunSupervisorCommandV0(input),
	}
	result, err := SuperviseCodexV0(
		ctx,
		CodexSupervisorDepsV0{AgentLifecycle: lifecycle},
		CodexSupervisorCommandV0{
			MaxTicks:        input.MaxTicks,
			ContinueMessage: input.ContinueMessage,
		},
	)
	if err != nil {
		return orquestamcp.NewMCPRunSupervisorErrorResultV0(
			input,
			"run_supervisor_execute_error",
			"executor",
			"run_supervisor_execute_error",
		), err
	}
	return orquestamcp.NewMCPRunSupervisorOKResultV0(
		input,
		result.Last.SessionRef,
		string(result.StopReason),
		result.Ticks,
		codexStackRunSupervisorSnapshotMCPV0(result.Last),
		codexStackRunSupervisorHistoryMCPV0(result.History),
	), nil
}

func codexStackRunSupervisorDrainRequestV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
) DrainRunRequestV0 {
	return DrainRunRequestV0{
		RunRef:                     strings.TrimSpace(input.RunRef),
		OccurredAt:                 strings.TrimSpace(input.OccurredAt),
		CorrelationID:              firstNonEmptyQueuedSourceV0(input.CorrelationID, input.RequestID),
		MaxBursts:                  input.MaxBursts,
		MaxStepsPerBurst:           input.MaxStepsPerBurst,
		MaxDispatchesPerWait:       input.MaxDispatchesPerWait,
		MaxCommands:                input.MaxCommands,
		MaxOutboxPerCycle:          input.MaxOutboxPerCycle,
		MaxDecisionCycles:          input.MaxDecisionCycles,
		MaxExternalWaits:           input.MaxExternalWaits,
		OperationalDirectorPlanRef: strings.TrimSpace(input.OperationalDirectorPlanRef),
	}
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
			MaxBursts:            input.MaxBursts,
			MaxStepsPerBurst:     input.MaxStepsPerBurst,
			MaxDispatchesPerWait: input.MaxDispatchesPerWait,
			MaxCommands:          input.MaxCommands,
			MaxOutboxPerCycle:    input.MaxOutboxPerCycle,
			MaxDecisionCycles:    input.MaxDecisionCycles,
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
