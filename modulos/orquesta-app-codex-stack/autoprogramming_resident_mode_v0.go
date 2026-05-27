package orquestaappcodexstack

import (
	"strconv"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	AutoprogrammingResidentDefaultMaxTicksV0          = 24
	AutoprogrammingResidentDefaultMaxRunsPerTickV0    = 10
	AutoprogrammingResidentDefaultMaxBurstsV0         = 4
	AutoprogrammingResidentDefaultMaxStepsPerBurstV0  = 6
	AutoprogrammingResidentDefaultMaxDispatchesV0     = 10
	AutoprogrammingResidentDefaultMaxCommandsV0       = 20
	AutoprogrammingResidentDefaultMaxOutboxV0         = 10
	AutoprogrammingResidentDefaultMaxDecisionCyclesV0 = 1
	AutoprogrammingResidentDefaultMaxExternalWaitsV0  = 1
	AutoprogrammingResidentMaxExternalWaitsV0         = 70
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
	} else if input.MaxExternalWaits > AutoprogrammingResidentMaxExternalWaitsV0 {
		input.MaxExternalWaits = AutoprogrammingResidentMaxExternalWaitsV0
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
		"evidence-ref-autoprogramming-resident-external-wait-policy",
	))
	if result.Last.Status != "" {
		result.Last.EvidenceRefs = compactStringsV0(append(result.Last.EvidenceRefs, result.EvidenceRefs...))
	}
	return result
}

func rejectAutoprogrammingResidentWaitOverrideV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, bool) {
	if !input.ResidentMode || input.MaxExternalWaits <= AutoprogrammingResidentMaxExternalWaitsV0 {
		return orquestamcp.MCPRunSupervisorToolResultV0{}, false
	}
	max := strconv.Itoa(AutoprogrammingResidentMaxExternalWaitsV0)
	result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
		input,
		"resident_external_wait_budget_incompatible",
		"max_external_waits",
		"max_external_waits incompatible con modo residente: max="+max,
	)
	result.EvidenceRefs = []string{"evidence-ref-autoprogramming-resident-external-wait-policy"}
	result.Diagnostics = []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "resident_external_wait_budget_incompatible",
		Scope:        "resident_mode",
		Message:      "usar drain real/operador para esperas largas o bajar max_external_waits a " + max,
		EvidenceRefs: result.EvidenceRefs,
	}}
	result.NextActions = []string{"retry_with_resident_max_external_waits_" + max + "_or_use_real_drain_mode"}
	return result, true
}
