package orquestamcp

import (
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	MCPEjecutarOrquestacionAppToolNameV0    = "orquesta.apps.ejecutar_orquestacion.v0"
	MCPEjecutarOrquestacionAppToolVersionV0 = "v0"
	MCPEjecutarOrquestacionAppResourceURIV0 = "orquesta://contracts/ejecutar-orquestacion-app/v0"
	MCPEjecutarOrquestacionAppEstadoOKV0    = "ok"
	MCPEjecutarOrquestacionAppEstadoErrorV0 = "error"
)

type MCPEjecutarOrquestacionAppToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPEjecutarOrquestacionAppToolInputV0 struct {
	RequestID                 string                    `json:"request_id,omitempty"`
	CorrelationID             string                    `json:"correlation_id,omitempty"`
	Respuesta                 string                    `json:"respuesta,omitempty"`
	DirectorExecutionMode     string                    `json:"director_execution_mode,omitempty"`
	RequireDirectorV2         bool                      `json:"require_director_v2,omitempty"`
	RunRef                    string                    `json:"run_ref,omitempty"`
	ProjectRef                string                    `json:"project_ref,omitempty"`
	OccurredAt                string                    `json:"occurred_at"`
	RequestedBy               string                    `json:"requested_by,omitempty"`
	MaxBursts                 int                       `json:"max_bursts,omitempty"`
	MaxStepsPerBurst          int                       `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait      int                       `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands               int                       `json:"max_commands,omitempty"`
	MaxOutboxPerCycle         int                       `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits          int                       `json:"max_external_waits,omitempty"`
	UseAutonomousDirectorLoop bool                      `json:"use_autonomous_director_loop,omitempty"`
	AppSpec                   orquestafactory.AppSpecV0 `json:"app_spec"`
}

type MCPEjecutarOrquestacionAppToolResultV0 struct {
	Estado            string                                               `json:"estado"`
	RequestID         string                                               `json:"request_id,omitempty"`
	CorrelationID     string                                               `json:"correlation_id,omitempty"`
	RoutePolicy       MCPAppSpecRoutePolicyV0                              `json:"route_policy"`
	AppSpec           MCPAppSpecCompactV0                                  `json:"app_spec,omitempty"`
	RunRef            string                                               `json:"run_ref,omitempty"`
	PhaseID           string                                               `json:"phase_id,omitempty"`
	Plan              MCPAppPlanCompactV0                                  `json:"plan,omitempty"`
	Progress          MCPAppPlanProgressMCPV0                              `json:"progress,omitempty"`
	LoopStatus        string                                               `json:"loop_status,omitempty"`
	StartedAgents     []string                                             `json:"started_agents,omitempty"`
	DirectorStats     *orquestacionnucleoapp.DirectorRunStatsV0            `json:"director_stats,omitempty"`
	DirectorLoopStats *orquestacionnucleoapp.AutonomousDirectorLoopStatsV0 `json:"director_loop_stats,omitempty"`
	Attempts          int                                                  `json:"attempts,omitempty"`
	ExternalWaits     int                                                  `json:"external_waits,omitempty"`
	EvidenceRefs      []string                                             `json:"evidence_refs,omitempty"`
	Errores           []MCPValidationIssueV0                               `json:"errores_publicos,omitempty"`
}

func MCPEjecutarOrquestacionAppDescriptorV0() MCPEjecutarOrquestacionAppToolDescriptorV0 {
	return MCPEjecutarOrquestacionAppToolDescriptorV0{
		Name:        MCPEjecutarOrquestacionAppToolNameV0,
		Version:     MCPEjecutarOrquestacionAppToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,respuesta?,director_execution_mode:legacy_director_loop,require_director_v2?,run_ref?,project_ref?,occurred_at?,requested_by?,max_bursts?,max_steps_per_burst?,max_dispatches_per_wait?,max_commands?,max_outbox_per_cycle?,max_external_waits?,use_autonomous_director_loop?,app_spec:AppSpecV0}",
		Output:      "ok:{route_policy,app_spec,run_ref,phase_id,plan,progress,loop_status,started_agents,director_stats,director_loop_stats,evidence_refs?}|error:{route_policy,errores_publicos}",
		ResourceURI: MCPEjecutarOrquestacionAppResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"compatibilidad legacy explicita mediante orquesta-app-runner",
			"solo ejecuta el loop historico con director_execution_mode=legacy_director_loop",
			"si requiere Director V2 bloquea con razon publica",
			"los efectos salen solo por puertos inyectados",
			"no elige DB runtime proveedor modelo HOME OAuth ni credenciales",
		},
	}
}

func ToRunPreparedAppOrchestrationRequestMCPV0(
	input MCPEjecutarOrquestacionAppToolInputV0,
	prepared orquestaapprunner.AppOrchestrationPreparedV0,
) orquestaapprunner.RunPreparedAppOrchestrationRequestV0 {
	prepareInput := prepareInputFromRunInputMCPV0(input)
	return orquestaapprunner.RunPreparedAppOrchestrationRequestV0{
		Prepared:                  prepared,
		OccurredAt:                input.OccurredAt,
		CorrelationID:             neutralPrepareOrchestrationCorrelationMCPV0(prepareInput),
		RequestedBy:               "orquesta-app-runner",
		RequireDirectorV2:         input.RequireDirectorV2,
		MaxBursts:                 input.MaxBursts,
		MaxStepsPerBurst:          input.MaxStepsPerBurst,
		MaxDispatchesPerWait:      input.MaxDispatchesPerWait,
		MaxCommands:               input.MaxCommands,
		MaxOutboxPerCycle:         input.MaxOutboxPerCycle,
		MaxExternalWaits:          input.MaxExternalWaits,
		UseAutonomousDirectorLoop: input.UseAutonomousDirectorLoop,
	}
}
