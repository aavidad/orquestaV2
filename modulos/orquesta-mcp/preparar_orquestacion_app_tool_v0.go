package orquestamcp

import (
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	MCPPrepararOrquestacionAppToolNameV0    = "orquesta.apps.preparar_orquestacion.v0"
	MCPPrepararOrquestacionAppToolVersionV0 = "v0"
	MCPPrepararOrquestacionAppResourceURIV0 = "orquesta://contracts/preparar-orquestacion-app/v0"
	MCPPrepararOrquestacionAppEstadoOKV0    = "ok"
	MCPPrepararOrquestacionAppEstadoErrorV0 = "error"
	MCPPrepararOrquestacionAppErrorCodeV0   = "preparar_orquestacion_app_invalida"
	MCPPrepararOrquestacionAppErrorMsgV0    = "preparacion_orquestacion_invalida"
)

type MCPPrepararOrquestacionAppToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPPrepararOrquestacionAppToolInputV0 struct {
	RequestID     string                    `json:"request_id,omitempty"`
	CorrelationID string                    `json:"correlation_id,omitempty"`
	Respuesta     string                    `json:"respuesta,omitempty"`
	RunRef        string                    `json:"run_ref,omitempty"`
	ProjectRef    string                    `json:"project_ref,omitempty"`
	OccurredAt    string                    `json:"occurred_at,omitempty"`
	RequestedBy   string                    `json:"requested_by,omitempty"`
	AppSpec       orquestafactory.AppSpecV0 `json:"app_spec"`
}

type MCPPrepararOrquestacionAppToolResultV0 struct {
	Estado        string                  `json:"estado"`
	RequestID     string                  `json:"request_id,omitempty"`
	CorrelationID string                  `json:"correlation_id,omitempty"`
	AppSpec       MCPAppSpecCompactV0     `json:"app_spec,omitempty"`
	RunRef        string                  `json:"run_ref,omitempty"`
	PhaseID       string                  `json:"phase_id,omitempty"`
	Plan          MCPAppPlanCompactV0     `json:"plan,omitempty"`
	Progress      MCPAppPlanProgressMCPV0 `json:"progress,omitempty"`
	EvidenceRefs  []string                `json:"evidence_refs,omitempty"`
	Errores       []MCPValidationIssueV0  `json:"errores_publicos,omitempty"`
}

type MCPAppPlanCompactV0 struct {
	SchemaVersion string                `json:"schema_version,omitempty"`
	RunRef        string                `json:"run_ref,omitempty"`
	AppRef        string                `json:"app_ref,omitempty"`
	Units         int                   `json:"units"`
	UnitRefs      []MCPAppPlanUnitMCPV0 `json:"unit_refs,omitempty"`
	EvidenceRefs  []string              `json:"evidence_refs,omitempty"`
}

type MCPAppPlanUnitMCPV0 struct {
	TaskRef             string   `json:"task_ref,omitempty"`
	Title               string   `json:"title,omitempty"`
	PhaseID             string   `json:"phase_id,omitempty"`
	Role                string   `json:"role,omitempty"`
	Capacity            string   `json:"capacity,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	DependsOnDeliveries []string `json:"depends_on_deliveries,omitempty"`
	WriteSet            []string `json:"write_set,omitempty"`
}

type MCPAppPlanProgressMCPV0 struct {
	TotalUnits        int      `json:"total_units"`
	DeliveredUnits    int      `json:"delivered_units"`
	Complete          bool     `json:"complete"`
	ReadyTaskRefs     []string `json:"ready_task_refs,omitempty"`
	BlockedTaskRefs   []string `json:"blocked_task_refs,omitempty"`
	PendingTaskRefs   []string `json:"pending_task_refs,omitempty"`
	DeliveredTaskRefs []string `json:"delivered_task_refs,omitempty"`
}

func MCPPrepararOrquestacionAppDescriptorV0() MCPPrepararOrquestacionAppToolDescriptorV0 {
	return MCPPrepararOrquestacionAppToolDescriptorV0{
		Name:        MCPPrepararOrquestacionAppToolNameV0,
		Version:     MCPPrepararOrquestacionAppToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,project_ref?,app_spec:AppSpecV0}",
		Output:      "ok:{app_spec,run_ref,phase_id,plan,progress}|error:{errores_publicos}",
		ResourceURI: MCPPrepararOrquestacionAppResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delegacion en orquesta-app-runner",
			"no expone CandidateProvider ni elige DB runtime proveedor modelo o HOME",
		},
	}
}

func ToPrepareAppOrchestrationRequestMCPV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) orquestaapprunner.PrepareAppOrchestrationRequestV0 {
	return orquestaapprunner.PrepareAppOrchestrationRequestV0{
		RunRef:        neutralPrepareOrchestrationRunRefMCPV0(input),
		ProjectRef:    neutralPrepareOrchestrationProjectRefMCPV0(input),
		OccurredAt:    input.OccurredAt,
		CorrelationID: neutralPrepareOrchestrationCorrelationMCPV0(input),
		RequestedBy:   "orquesta-app-runner",
		AppSpec:       input.AppSpec,
	}
}
