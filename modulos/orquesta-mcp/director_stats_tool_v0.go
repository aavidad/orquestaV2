package orquestamcp

import (
	"context"
	"strings"

	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
	AppRef               string `json:"app_ref,omitempty"`
	ExternalJobRef       string `json:"external_job_ref,omitempty"`
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
	ExternalJob     *MCPDirectorExternalJobStatsV0                   `json:"external_job,omitempty"`
	Stats           *orquestacionnucleoapp.DirectorRunStatsV0        `json:"stats,omitempty"`
	DecisionContext *orquestaobservability.DirectorDecisionContextV0 `json:"decision_context,omitempty"`
	Errores         []MCPValidationIssueV0                           `json:"errores_publicos,omitempty"`
}

type MCPDirectorExternalJobStatsRequestV0 struct {
	RunRef         string `json:"run_ref,omitempty"`
	AppRef         string `json:"app_ref,omitempty"`
	ExternalJobRef string `json:"external_job_ref"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

type MCPDirectorExternalJobStatsV0 struct {
	AppRef       string   `json:"app_ref,omitempty"`
	JobRef       string   `json:"job_ref"`
	WorkKind     string   `json:"work_kind,omitempty"`
	ChangeRef    string   `json:"change_ref,omitempty"`
	RunRef       string   `json:"run_ref,omitempty"`
	TaskRef      string   `json:"task_ref,omitempty"`
	AgentRef     string   `json:"agent_ref,omitempty"`
	Status       string   `json:"status,omitempty"`
	DeliveryRefs []string `json:"delivery_refs,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type MCPDirectorExternalJobStatsSourcePortV0 interface {
	ResolveDirectorExternalJobStatsV0(
		context.Context,
		MCPDirectorExternalJobStatsRequestV0,
	) (MCPDirectorExternalJobStatsV0, bool, error)
}

type MCPDirectorStatsToolExecutorV0 struct {
	RunStore          orquestacionnucleoapp.RunStorePortV0
	ProcessRegistry   orquestacionnucleoapp.AgentProcessRegistryPortV0
	ProgressSource    orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	AgentUsageSource  orquestacionnucleoapp.AgentUsageStatsProviderPortV0
	ExternalJobSource MCPDirectorExternalJobStatsSourcePortV0
}

func MCPDirectorStatsDescriptorV0() MCPDirectorStatsToolDescriptorV0 {
	return MCPDirectorStatsToolDescriptorV0{
		Name:        MCPDirectorStatsToolNameV0,
		Version:     MCPDirectorStatsToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,app_ref?,external_job_ref?,occurred_at?,include_process_refs?,include_agent_progress?,include_agent_usage?}",
		Output:      "ok:{run_ref,external_job?,stats{progress,closure},decision_context}|error:{errores_publicos}",
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
	externalJobRef := strings.TrimSpace(input.ExternalJobRef)
	var externalJob *MCPDirectorExternalJobStatsV0
	if runRef == "" && externalJobRef != "" {
		resolved, ok, err := executor.resolveExternalJobStatsV0(ctx, input, "")
		if err != nil {
			return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
		}
		if !ok || strings.TrimSpace(resolved.RunRef) == "" {
			return newMCPDirectorStatsErrorV0(input, "external_job_no_disponible", "external_job_ref", "external job no disponible"), nil
		}
		externalJob = &resolved
		runRef = resolved.RunRef
	}
	if runRef == "" {
		return newMCPDirectorStatsErrorV0(input, "run_ref_requerido", "run_ref", "run_ref requerido"), nil
	}
	if executor.RunStore == nil {
		return newMCPDirectorStatsErrorV0(input, "run_store_no_disponible", "run_store", "run_store requerido"), nil
	}
	run, err := executor.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		if externalJobRef != "" {
			resolved, ok, resolveErr := executor.resolveExternalJobStatsV0(ctx, input, "")
			if resolveErr != nil {
				return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
			}
			if ok && strings.TrimSpace(resolved.RunRef) != "" {
				externalJob = &resolved
				runRef = strings.TrimSpace(resolved.RunRef)
				run, err = executor.RunStore.LoadRunV0(ctx, runRef)
			}
		}
	}
	if err != nil {
		return newMCPDirectorStatsErrorV0(input, "run_no_disponible", "run_ref", "run no disponible"), nil
	}
	if externalJobRef != "" && externalJob == nil {
		resolved, ok, err := executor.resolveExternalJobStatsV0(ctx, input, runRef)
		if err != nil {
			return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
		}
		if !ok {
			return newMCPDirectorStatsErrorV0(input, "external_job_no_disponible", "external_job_ref", "external job no disponible"), nil
		}
		externalJob = &resolved
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(run)
	if executor.ProcessRegistry != nil || input.IncludeAgentProgress || input.IncludeAgentUsage {
		var progressSource orquestacionnucleoapp.AgentProgressObservationProviderPortV0
		if input.IncludeAgentProgress {
			progressSource = executor.ProgressSource
		}
		var usageSource orquestacionnucleoapp.AgentUsageStatsProviderPortV0
		if input.IncludeAgentUsage {
			usageSource = executor.AgentUsageSource
		}
		stats = orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsV0(
			ctx,
			run,
			executor.ProcessRegistry,
			progressSource,
			usageSource,
			orquestacionnucleoapp.DirectorProgressSourceRequestV0{
				OccurredAt:        input.OccurredAt,
				CorrelationID:     input.CorrelationID,
				IncludeAgentUsage: input.IncludeAgentUsage,
			},
		)
	}
	if !input.IncludeProcessRefs {
		clearMCPDirectorStatsProcessRefsV0(&stats)
	}
	return MCPDirectorStatsToolResultV0{
		Estado:          MCPDirectorStatsEstadoOKV0,
		RequestID:       strings.TrimSpace(input.RequestID),
		CorrelationID:   firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:          stats.RunRef,
		ExternalJob:     externalJob,
		Stats:           &stats,
		DecisionContext: buildMCPDirectorDecisionContextV0(run, stats, input.OccurredAt),
		Errores:         []MCPValidationIssueV0{},
	}, nil
}

func clearMCPDirectorStatsProcessRefsV0(stats *orquestacionnucleoapp.DirectorRunStatsV0) {
	if stats == nil {
		return
	}
	for index := range stats.Agents {
		stats.Agents[index].Process = nil
	}
}

func (executor MCPDirectorStatsToolExecutorV0) resolveExternalJobStatsV0(
	ctx context.Context,
	input MCPDirectorStatsToolInputV0,
	runRef string,
) (MCPDirectorExternalJobStatsV0, bool, error) {
	if executor.ExternalJobSource == nil {
		return MCPDirectorExternalJobStatsV0{}, false, nil
	}
	return executor.ExternalJobSource.ResolveDirectorExternalJobStatsV0(
		ctx,
		MCPDirectorExternalJobStatsRequestV0{
			RunRef:         firstNonEmptyMCPV0(runRef, input.RunRef),
			AppRef:         strings.TrimSpace(input.AppRef),
			ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
			CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
			OccurredAt:     strings.TrimSpace(input.OccurredAt),
		},
	)
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
