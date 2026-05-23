package orquestaappcodexstack

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	AutoprogrammingResidentDefaultMaxTicksV0          = 24
	AutoprogrammingResidentDefaultMaxRunsPerTickV0    = 1
	AutoprogrammingResidentDefaultMaxBurstsV0         = 4
	AutoprogrammingResidentDefaultMaxStepsPerBurstV0  = 6
	AutoprogrammingResidentDefaultMaxDispatchesV0     = 4
	AutoprogrammingResidentDefaultMaxCommandsV0       = 20
	AutoprogrammingResidentDefaultMaxOutboxV0         = 4
	AutoprogrammingResidentDefaultMaxDecisionCyclesV0 = 1
	AutoprogrammingResidentDefaultMaxExternalWaitsV0  = 2
)

func normalizeAutoprogrammingResidentInputV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	stack StackV0,
) orquestamcp.MCPRunSupervisorToolInputV0 {
	if !input.ResidentMode {
		return input
	}
	if strings.TrimSpace(input.QueueRef) == "" {
		input.QueueRef = normalizeRunQueueConfigV0(stack.RunQueue).QueueRef
	}
	input.AllowRepeatedRuns = true
	if input.MaxTicks <= 0 {
		input.MaxTicks = AutoprogrammingResidentDefaultMaxTicksV0
	}
	if input.MaxRunsPerTick <= 0 {
		input.MaxRunsPerTick = AutoprogrammingResidentDefaultMaxRunsPerTickV0
	}
	if input.MaxBursts <= 0 {
		input.MaxBursts = AutoprogrammingResidentDefaultMaxBurstsV0
	}
	if input.MaxStepsPerBurst <= 0 {
		input.MaxStepsPerBurst = AutoprogrammingResidentDefaultMaxStepsPerBurstV0
	}
	if input.MaxDispatchesPerWait <= 0 {
		input.MaxDispatchesPerWait = AutoprogrammingResidentDefaultMaxDispatchesV0
	}
	if input.MaxCommands <= 0 {
		input.MaxCommands = AutoprogrammingResidentDefaultMaxCommandsV0
	}
	if input.MaxOutboxPerCycle <= 0 {
		input.MaxOutboxPerCycle = AutoprogrammingResidentDefaultMaxOutboxV0
	}
	if input.MaxDecisionCycles <= 0 {
		input.MaxDecisionCycles = AutoprogrammingResidentDefaultMaxDecisionCyclesV0
	}
	if input.MaxExternalWaits <= 0 {
		input.MaxExternalWaits = AutoprogrammingResidentDefaultMaxExternalWaitsV0
	}
	if strings.TrimSpace(input.ContinueMessage) == "" {
		input.ContinueMessage = DefaultCodexSupervisorContinueMessageV0
	}
	return input
}

func addAutoprogrammingResidentEvidenceV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	if !input.ResidentMode || result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		return result
	}
	result.EvidenceRefs = compactStringsV0(append(
		result.EvidenceRefs,
		"evidence-ref-autoprogramming-resident-mode",
		"evidence-ref-autoprogramming-resident-backlog-queue",
		"evidence-ref-autoprogramming-resident-review-replan-close",
	))
	if result.Last.Status != "" {
		result.Last.EvidenceRefs = compactStringsV0(append(result.Last.EvidenceRefs, result.EvidenceRefs...))
	}
	return result
}
