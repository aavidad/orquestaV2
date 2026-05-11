package orquestamcp

import (
	"context"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

const (
	MCPDirectorStatsToolNameV0    = "orquesta.director.stats.v0"
	MCPDirectorStatsToolVersionV0 = "v0"
	MCPDirectorStatsResourceURIV0 = "orquesta://contracts/director-stats/v0"
	MCPDirectorStatsEstadoOKV0    = "ok"
	MCPDirectorStatsEstadoErrorV0 = "error"
)

type MCPDirectorStatsToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPDirectorStatsToolInputV0 struct {
	RequestID            string `json:"request_id,omitempty"`
	CorrelationID        string `json:"correlation_id,omitempty"`
	RunRef               string `json:"run_ref"`
	OccurredAt           string `json:"occurred_at,omitempty"`
	IncludeProcessRefs   bool   `json:"include_process_refs,omitempty"`
	IncludeAgentProgress bool   `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    bool   `json:"include_agent_usage,omitempty"`
}

type MCPDirectorStatsToolResultV0 struct {
	Estado          string                                           `json:"estado"`
	RequestID       string                                           `json:"request_id,omitempty"`
	CorrelationID   string                                           `json:"correlation_id,omitempty"`
	RunRef          string                                           `json:"run_ref,omitempty"`
	Stats           *orquestacionnucleoapp.DirectorRunStatsV0        `json:"stats,omitempty"`
	DecisionContext *orquestaobservability.DirectorDecisionContextV0 `json:"decision_context,omitempty"`
	Errores         []MCPValidationIssueV0                           `json:"errores_publicos,omitempty"`
}

type MCPDirectorStatsToolExecutorV0 struct {
	RunStore         orquestacionnucleoapp.RunStorePortV0
	ProcessRegistry  orquestacionnucleoapp.AgentProcessRegistryPortV0
	ProgressSource   orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	AgentUsageSource orquestacionnucleoapp.AgentUsageStatsProviderPortV0
}

func MCPDirectorStatsDescriptorV0() MCPDirectorStatsToolDescriptorV0 {
	return MCPDirectorStatsToolDescriptorV0{
		Name:        MCPDirectorStatsToolNameV0,
		Version:     MCPDirectorStatsToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref,occurred_at?,include_process_refs?,include_agent_progress?,include_agent_usage?}",
		Output:      "ok:{run_ref,stats{progress,closure},decision_context}|error:{errores_publicos}",
		ResourceURI: MCPDirectorStatsResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"consulta el estado por RunStorePortV0 inyectado",
			"enriquece control de procesos solo por AgentProcessRegistryPortV0 opcional",
			"expone cierre bloqueado/ready/cerrado derivado solo del run",
			"devuelve refs opacas sin rutas locales credenciales trazas sensibles ni detalles de runtime",
		},
	}
}

func (executor MCPDirectorStatsToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runRef := strings.TrimSpace(input.RunRef)
	if runRef == "" {
		return newMCPDirectorStatsErrorV0(input, "run_ref_requerido", "run_ref", "run_ref requerido"), nil
	}
	if executor.RunStore == nil {
		return newMCPDirectorStatsErrorV0(input, "run_store_no_disponible", "run_store", "run_store requerido"), nil
	}
	run, err := executor.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return newMCPDirectorStatsErrorV0(input, "run_no_disponible", "run_ref", "run no disponible"), nil
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(run)
	if input.IncludeProcessRefs || input.IncludeAgentProgress || input.IncludeAgentUsage {
		var progressSource orquestacionnucleoapp.AgentProgressObservationProviderPortV0
		if input.IncludeAgentProgress {
			progressSource = executor.ProgressSource
		}
		var registry orquestacionnucleoapp.AgentProcessRegistryPortV0
		if input.IncludeProcessRefs {
			registry = executor.ProcessRegistry
		}
		var usageSource orquestacionnucleoapp.AgentUsageStatsProviderPortV0
		if input.IncludeAgentUsage {
			usageSource = executor.AgentUsageSource
		}
		stats = orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
			ctx,
			run,
			registry,
			progressSource,
			usageSource,
			orquestacionnucleoapp.DirectorProgressSourceRequestV0{
				OccurredAt:    input.OccurredAt,
				CorrelationID: input.CorrelationID,
			},
		)
	}
	return MCPDirectorStatsToolResultV0{
		Estado:          MCPDirectorStatsEstadoOKV0,
		RequestID:       strings.TrimSpace(input.RequestID),
		CorrelationID:   firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:          stats.RunRef,
		Stats:           &stats,
		DecisionContext: buildMCPDirectorDecisionContextV0(run, stats, input.OccurredAt),
		Errores:         []MCPValidationIssueV0{},
	}, nil
}

func newMCPDirectorStatsErrorV0(
	input MCPDirectorStatsToolInputV0,
	code string,
	field string,
	message string,
) MCPDirectorStatsToolResultV0 {
	return MCPDirectorStatsToolResultV0{
		Estado:        MCPDirectorStatsEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}
