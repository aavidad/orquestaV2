package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
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

func codexStackRunSupervisorErrorResultMCPV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	partial CodexSupervisorResultV0,
	err error,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	var drainErr DrainObservationApplyErrorV0
	if errors.As(err, &drainErr) {
		result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
			input,
			"delivery_ack_ingestion_failed",
			firstNonEmptyQueuedSourceV0(drainErr.Field, "drain_observation"),
			codexStackDrainObservationPublicMessageV0(drainErr),
		)
		result.RunRef = firstNonEmptyQueuedSourceV0(drainErr.RunRef, input.RunRef)
		result.Diagnostics = codexStackDrainObservationDiagnosticsMCPV0(drainErr)
		result.EvidenceRefs = compactStringsV0(append(
			codexStackDrainObservationEvidenceRefsV0(drainErr),
			partial.Last.EvidenceRefs...,
		))
		result.NextActions = compactStringsV0([]string{drainErr.NextActionV0()})
		return result
	}
	result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
		input,
		"run_supervisor_execute_error",
		"executor",
		"run_supervisor_execute_error",
	)
	result.RunRef = firstNonEmptyQueuedSourceV0(partial.Last.SessionRef, input.RunRef)
	result.StopReason = string(partial.StopReason)
	result.Ticks = partial.Ticks
	result.Last = codexStackRunSupervisorSnapshotMCPV0(partial.Last)
	result.History = codexStackRunSupervisorHistoryMCPV0(partial.History)
	result.EvidenceRefs = compactStringsV0(partial.Last.EvidenceRefs)
	if len(result.EvidenceRefs) > 0 {
		result.Diagnostics = []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:         "run_supervisor_partial_snapshot",
			Scope:        "run:" + result.RunRef,
			Message:      "executor fallo con snapshot parcial disponible",
			EvidenceRefs: result.EvidenceRefs,
		}}
	}
	return result
}

func codexStackDrainObservationPublicMessageV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"delivery_ack_ingestion_failed",
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
	})
	return strings.Join(parts, " ")
}

func codexStackDrainObservationDiagnosticsMCPV0(
	err DrainObservationApplyErrorV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	message := strings.Join(compactStringsV0([]string{
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
		fmt.Sprintf("agents=%d started=%d failed=%d lost=%d stopped=%d confirmed_stopped=%d deliveries=%d",
			len(compactStringsV0(err.Agents)),
			len(compactStringsV0(err.StartedAgents)),
			len(compactStringsV0(err.FailedAgents)),
			len(compactStringsV0(err.LostAgents)),
			len(compactStringsV0(err.StoppedAgents)),
			len(compactStringsV0(err.ConfirmedStoppedAgents)),
			len(compactStringsV0(err.Deliveries)),
		),
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "drain_observation_apply_failed",
		Scope:        codexStackDrainObservationScopeMCPV0(err),
		Message:      message,
		EvidenceRefs: codexStackDrainObservationEvidenceRefsV0(err),
	}}
}

func codexStackDrainObservationScopeMCPV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"run:" + err.RunRef,
		"task:" + err.TaskRef,
		"agent:" + err.AgentRef,
	})
	return strings.Join(parts, "/")
}

func codexStackDrainObservationEvidenceRefsV0(err DrainObservationApplyErrorV0) []string {
	return compactStringsV0([]string{
		err.ArtifactRef,
		err.DeliveryRef,
		err.TaskRef,
		err.AgentRef,
		err.PhaseID,
	})
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
