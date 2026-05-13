package orquestamcp

import (
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	MCPRunControlToolNameV0    = "orquesta.runs.control.v0"
	MCPRunControlToolVersionV0 = "v0"
	MCPRunControlResourceURIV0 = "orquesta://contracts/run-control/v0"
	MCPRunControlEstadoOKV0    = "ok"
	MCPRunControlEstadoErrorV0 = "error"
)

type MCPRunControlToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPRunControlToolInputV0 struct {
	RequestID      string   `json:"request_id,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	Action         string   `json:"action"`
	RunRef         string   `json:"run_ref,omitempty"`
	AppRef         string   `json:"app_ref,omitempty"`
	ExternalJobRef string   `json:"external_job_ref,omitempty"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Forced         bool     `json:"forced,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type MCPRunControlToolResultV0 struct {
	Estado             string                 `json:"estado"`
	RequestID          string                 `json:"request_id,omitempty"`
	CorrelationID      string                 `json:"correlation_id,omitempty"`
	Action             string                 `json:"action,omitempty"`
	RunRef             string                 `json:"run_ref,omitempty"`
	Status             string                 `json:"status,omitempty"`
	CheckpointRecorded bool                   `json:"checkpoint_recorded,omitempty"`
	Forced             bool                   `json:"forced,omitempty"`
	EvidenceRefs       []string               `json:"evidence_refs,omitempty"`
	Errores            []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

func MCPRunControlDescriptorV0() MCPRunControlToolDescriptorV0 {
	return MCPRunControlToolDescriptorV0{
		Name:        MCPRunControlToolNameV0,
		Version:     MCPRunControlToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:pause|resume|stop|cancel,run_ref?|external_job_ref?,app_ref?,requested_by?,reason?,forced?,idempotency_key?,evidence_refs?}",
		Output:      "ok:{run_ref,action,status,checkpoint_recorded,forced?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPRunControlResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"una sola tool con action pause resume stop cancel",
			"delega en RunControlWriterPortV0 inyectado",
			"external_job_ref solo resuelve la run asociada por puerto inyectado",
			"no usa DB runtime filesystem ni scheduler interno",
		},
	}
}

func newMCPRunControlResultV0(
	input MCPRunControlToolInputV0,
	state orquestaruncontrol.RunControlStateV0,
) MCPRunControlToolResultV0 {
	return MCPRunControlToolResultV0{
		Estado:             MCPRunControlEstadoOKV0,
		RequestID:          strings.TrimSpace(input.RequestID),
		CorrelationID:      firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:             normalizeMCPRunControlActionV0(input.Action),
		RunRef:             strings.TrimSpace(state.RunRef),
		Status:             string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status)),
		CheckpointRecorded: state.CheckpointRecorded,
		Forced:             state.Forced,
		EvidenceRefs:       compactStringsMCPV0(state.EvidenceRefs),
		Errores:            []MCPValidationIssueV0{},
	}
}

func newMCPRunControlErrorV0(input MCPRunControlToolInputV0, code string, field string) MCPRunControlToolResultV0 {
	return MCPRunControlToolResultV0{
		Estado:        MCPRunControlEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        normalizeMCPRunControlActionV0(input.Action),
		RunRef:        strings.TrimSpace(input.RunRef),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(code),
		}},
	}
}

func normalizeMCPRunControlActionV0(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
