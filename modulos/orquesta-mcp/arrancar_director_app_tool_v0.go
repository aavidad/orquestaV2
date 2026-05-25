package orquestamcp

import (
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	MCPArrancarDirectorAppToolNameV0    = "orquesta.apps.arrancar_director.v0"
	MCPArrancarDirectorAppToolVersionV0 = "v0"
	MCPArrancarDirectorAppResourceURIV0 = "orquesta://contracts/arrancar-director-app/v0"
	MCPArrancarDirectorAppEstadoOKV0    = "ok"
	MCPArrancarDirectorAppEstadoErrorV0 = "error"
)

type MCPArrancarDirectorAppToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPArrancarDirectorAppToolInputV0 struct {
	RequestID            string                           `json:"request_id,omitempty"`
	CorrelationID        string                           `json:"correlation_id,omitempty"`
	Respuesta            string                           `json:"respuesta,omitempty"`
	AppSpecRequest       orquestafactory.AppSpecRequestV0 `json:"app_spec_request"`
	MaxBursts            int                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int                              `json:"max_outbox_per_cycle,omitempty"`
	MaxExternalWaits     int                              `json:"max_external_waits,omitempty"`
}

type MCPArrancarDirectorAppToolResultV0 struct {
	Estado        string                     `json:"estado"`
	RequestID     string                     `json:"request_id,omitempty"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
	RoutePolicy   MCPAppSpecRoutePolicyV0    `json:"route_policy"`
	AppSpec       MCPAppSpecCompactV0        `json:"app_spec,omitempty"`
	RunRef        string                     `json:"run_ref,omitempty"`
	PhaseID       string                     `json:"phase_id,omitempty"`
	DirectorTask  MCPDirectorTaskCompactV0   `json:"director_task,omitempty"`
	DirectorTasks []MCPDirectorTaskCompactV0 `json:"director_tasks,omitempty"`
	LoopStatus    string                     `json:"loop_status,omitempty"`
	StartedAgents []string                   `json:"started_agents,omitempty"`
	Errores       []MCPValidationIssueV0     `json:"errores_publicos,omitempty"`
	EvidenceRefs  []string                   `json:"evidence_refs,omitempty"`
}

type MCPDirectorTaskCompactV0 struct {
	TaskRef        string `json:"task_ref,omitempty"`
	BrainstormRef  string `json:"brainstorm_ref,omitempty"`
	AgentRequestID string `json:"agent_request_id,omitempty"`
	Capacity       string `json:"capacity,omitempty"`
}

func MCPArrancarDirectorAppDescriptorV0() MCPArrancarDirectorAppToolDescriptorV0 {
	return MCPArrancarDirectorAppToolDescriptorV0{
		Name:        MCPArrancarDirectorAppToolNameV0,
		Version:     MCPArrancarDirectorAppToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,app_spec_request:AppSpecRequestV0(request_kind?,execution_mode?),limits?:{max_external_waits?}}",
		Output:      "ok:{route_policy,app_spec,run_ref,phase_id,director_task,director_tasks,loop_status}|error:{route_policy,errores_publicos}",
		ResourceURI: MCPArrancarDirectorAppResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"entrada operativa preferente para apps nuevas con juicio del Director",
			"no elige proveedor modelo credenciales home runtime ni DB",
			"delegacion en orquesta-app-director-service",
		},
	}
}

func ToStartAppDirectorRequestV0(
	input MCPArrancarDirectorAppToolInputV0,
) orquestaappdirectorservice.StartAppDirectorRequestV0 {
	req := input.AppSpecRequest
	externalRequestID := firstNonEmptyMCPV0(req.RequestID, input.RequestID, input.CorrelationID, req.Nombre)
	externalCorrelationID := firstNonEmptyMCPV0(input.CorrelationID, externalRequestID)
	req.Source = MCPNuevaAppSourceV0
	req.RequestID = neutralAppDirectorRequestRefV0("req", req.Nombre, externalRequestID)
	return orquestaappdirectorservice.StartAppDirectorRequestV0{
		CorrelationID:        neutralAppDirectorRequestRefV0("corr", req.Nombre, externalCorrelationID),
		RequestedBy:          "orquesta-mcp",
		AppSpecRequest:       req,
		MaxBursts:            input.MaxBursts,
		MaxStepsPerBurst:     input.MaxStepsPerBurst,
		MaxDispatchesPerWait: input.MaxDispatchesPerWait,
		MaxCommands:          input.MaxCommands,
		MaxOutboxPerCycle:    input.MaxOutboxPerCycle,
		MaxExternalWaits:     input.MaxExternalWaits,
	}
}

func NewMCPArrancarDirectorAppResultV0(
	result orquestaappdirectorservice.StartAppDirectorResultV0,
) MCPArrancarDirectorAppToolResultV0 {
	if result.Status == orquestaappdirectorservice.StartAppDirectorStatusInvalidV0 {
		return MCPArrancarDirectorAppToolResultV0{
			Estado:        MCPArrancarDirectorAppEstadoErrorV0,
			CorrelationID: strings.TrimSpace(result.CorrelationID),
			RoutePolicy:   mcpPreferredDirectorRoutePolicyV0(),
			Errores:       publicIssuesMCPV0(result.ValidationIssues),
			EvidenceRefs:  compactStringsMCPV0(result.EvidenceRefs),
		}
	}
	return MCPArrancarDirectorAppToolResultV0{
		Estado:        MCPArrancarDirectorAppEstadoOKV0,
		RequestID:     strings.TrimSpace(result.AppSpec.RequestID),
		CorrelationID: strings.TrimSpace(result.CorrelationID),
		RoutePolicy:   mcpPreferredDirectorRoutePolicyV0(),
		AppSpec:       compactAppSpecV0(result.AppSpec),
		RunRef:        strings.TrimSpace(result.Run.RunID),
		PhaseID:       strings.TrimSpace(string(result.Run.CurrentPhase)),
		DirectorTask: MCPDirectorTaskCompactV0{
			TaskRef:        strings.TrimSpace(result.DirectorTask.TaskRef),
			BrainstormRef:  strings.TrimSpace(result.DirectorTask.BrainstormRef),
			AgentRequestID: strings.TrimSpace(result.DirectorTask.AgentRequestID),
			Capacity:       strings.TrimSpace(string(result.DirectorTask.Capacity)),
		},
		DirectorTasks: compactDirectorTasksMCPV0(result.DirectorTasks),
		LoopStatus:    strings.TrimSpace(string(result.LoopStatus)),
		StartedAgents: compactStringsMCPV0(result.StartedAgents),
		EvidenceRefs:  compactStringsMCPV0(result.EvidenceRefs),
	}
}

func mcpPreferredDirectorRoutePolicyV0() MCPAppSpecRoutePolicyV0 {
	return MCPAppSpecRoutePolicyV0{
		Mode:                "operativo_preferente",
		PreferredEntrypoint: MCPArrancarDirectorAppToolNameV0,
		PublicReason:        "apps_nuevas_con_juicio_usan_director_v2",
	}
}

func compactDirectorTasksMCPV0(
	tasks []orquestaappdirectorintake.AppDirectorTaskV0,
) []MCPDirectorTaskCompactV0 {
	if len(tasks) == 0 {
		return nil
	}
	out := make([]MCPDirectorTaskCompactV0, 0, len(tasks))
	seen := map[string]bool{}
	for _, task := range tasks {
		agentRef := strings.TrimSpace(task.AgentRequestID)
		if agentRef == "" || seen[agentRef] {
			continue
		}
		seen[agentRef] = true
		out = append(out, MCPDirectorTaskCompactV0{
			TaskRef:        strings.TrimSpace(task.TaskRef),
			BrainstormRef:  strings.TrimSpace(task.BrainstormRef),
			AgentRequestID: agentRef,
			Capacity:       strings.TrimSpace(string(task.Capacity)),
		})
	}
	return out
}
