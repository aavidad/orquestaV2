package orquestamcp

import (
	"strings"

	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func NewMCPEjecutarOrquestacionAppOKResultV0(
	input MCPEjecutarOrquestacionAppToolInputV0,
	prepared orquestaapprunner.AppOrchestrationPreparedV0,
	result orquestaapprunner.AppOrchestrationRunResultV0,
) MCPEjecutarOrquestacionAppToolResultV0 {
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(result.Run)
	return MCPEjecutarOrquestacionAppToolResultV0{
		Estado:            MCPEjecutarOrquestacionAppEstadoOKV0,
		RequestID:         firstNonEmptyMCPV0(input.RequestID, input.AppSpec.RequestID),
		CorrelationID:     firstNonEmptyMCPV0(input.CorrelationID, "corr-"+result.Run.RunID),
		RoutePolicy:       mcpRoutePolicyFromAppRunnerV0(result.RoutePolicy),
		AppSpec:           compactAppSpecV0(input.AppSpec),
		RunRef:            strings.TrimSpace(result.Run.RunID),
		PhaseID:           strings.TrimSpace(string(result.Run.CurrentPhase)),
		Plan:              compactAppPlanMCPV0(prepared.Plan),
		Progress:          compactAppPlanProgressMCPV0(result.Progress),
		LoopStatus:        strings.TrimSpace(string(result.LoopStatus)),
		StartedAgents:     compactStringsMCPV0(result.StartedAgents),
		DirectorStats:     &stats,
		DirectorLoopStats: result.DirectorLoopStats,
		Attempts:          result.Attempts,
		ExternalWaits:     result.ExternalWaits,
		EvidenceRefs:      compactStringsMCPV0(result.EvidenceRefs),
		Errores:           []MCPValidationIssueV0{},
	}
}

func NewMCPEjecutarOrquestacionAppErrorResultV0(
	input MCPEjecutarOrquestacionAppToolInputV0,
	err error,
) MCPEjecutarOrquestacionAppToolResultV0 {
	return MCPEjecutarOrquestacionAppToolResultV0{
		Estado:        MCPEjecutarOrquestacionAppEstadoErrorV0,
		RequestID:     firstNonEmptyMCPV0(input.RequestID, input.AppSpec.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RoutePolicy: mcpRoutePolicyFromAppRunnerV0(
			orquestaapprunner.AppRunnerPreviewRoutePolicyV0(orquestaapprunner.AppRunnerLegacyEntrypointExecuteV0),
		),
		Errores: []MCPValidationIssueV0{publicPrepareOrchestrationIssueMCPV0(err)},
	}
}
