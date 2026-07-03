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
	Estado                     string                      `json:"estado"`
	RequestID                  string                      `json:"request_id,omitempty"`
	CorrelationID              string                      `json:"correlation_id,omitempty"`
	Action                     string                      `json:"action,omitempty"`
	RunRef                     string                      `json:"run_ref,omitempty"`
	Status                     string                      `json:"status,omitempty"`
	PreviousStatus             string                      `json:"previous_status,omitempty"`
	FinalStatus                string                      `json:"final_status,omitempty"`
	GoalRef                    string                      `json:"goal_ref,omitempty"`
	ExternalGoalRef            string                      `json:"external_goal_ref,omitempty"`
	GoalStatusBefore           string                      `json:"goal_status_before,omitempty"`
	GoalStatusAfter            string                      `json:"goal_status_after,omitempty"`
	GoalControlSignalSent      bool                        `json:"goal_control_signal_sent,omitempty"`
	GoalControlSignalConfirmed bool                        `json:"goal_control_signal_confirmed,omitempty"`
	RecommendedAction          string                      `json:"recommended_action,omitempty"`
	CheckpointRecorded         bool                        `json:"checkpoint_recorded,omitempty"`
	Forced                     bool                        `json:"forced,omitempty"`
	EvidenceRefs               []string                    `json:"evidence_refs,omitempty"`
	Diagnostics                []MCPRunControlDiagnosticV0 `json:"diagnostics,omitempty"`
	Errores                    []MCPValidationIssueV0      `json:"errores_publicos,omitempty"`
}

type MCPRunControlDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func MCPRunControlDescriptorV0() MCPRunControlToolDescriptorV0 {
	return MCPRunControlToolDescriptorV0{
		Name:        MCPRunControlToolNameV0,
		Version:     MCPRunControlToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:pause|resume|stop|cancel,run_ref?|external_job_ref?,app_ref?,requested_by?,reason?,forced?,idempotency_key?,evidence_refs?}",
		Output:      "ok:{run_ref,action,status,previous_status?,final_status?,goal_ref?,external_goal_ref?,goal_status_before?,goal_status_after?,goal_control_signal_sent?,goal_control_signal_confirmed?,recommended_action?,checkpoint_recorded,forced?,evidence_refs?,diagnostics?}|error:{errores_publicos,forced?,evidence_refs?,diagnostics?}",
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

func normalizeMCPRunControlIdentityV0(
	input MCPRunControlToolInputV0,
	headerCorrelationID string,
	headerIdempotencyKey string,
) (MCPRunControlToolInputV0, []MCPValidationIssueV0) {
	identity := NormalizeMCPPublicMutationIdentityV0(MCPPublicMutationIdentityInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		IdempotencyKey:       input.IdempotencyKey,
		HeaderCorrelationID:  headerCorrelationID,
		HeaderIdempotencyKey: headerIdempotencyKey,
		Mutating:             true,
	})
	input.RequestID = identity.RequestID
	input.CorrelationID = identity.CorrelationID
	input.IdempotencyKey = identity.IdempotencyKey
	return input, identity.Issues
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
		FinalStatus:        string(orquestaruncontrol.NormalizeRunControlStatusV0(state.Status)),
		CheckpointRecorded: state.CheckpointRecorded,
		Forced:             state.Forced,
		EvidenceRefs:       compactStringsMCPV0(state.EvidenceRefs),
		Errores:            []MCPValidationIssueV0{},
	}
}

func newMCPRunControlErrorV0(input MCPRunControlToolInputV0, code string, field string) MCPRunControlToolResultV0 {
	normalizedCode := strings.TrimSpace(code)
	evidenceRefs := compactStringsMCPV0(input.EvidenceRefs)
	scope := ""
	if runRef := strings.TrimSpace(input.RunRef); runRef != "" {
		scope = "run:" + runRef
	}
	return MCPRunControlToolResultV0{
		Estado:        MCPRunControlEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        normalizeMCPRunControlActionV0(input.Action),
		RunRef:        strings.TrimSpace(input.RunRef),
		Forced:        input.Forced,
		EvidenceRefs:  evidenceRefs,
		Diagnostics: []MCPRunControlDiagnosticV0{{
			Code:         normalizedCode,
			Scope:        scope,
			EvidenceRefs: evidenceRefs,
		}},
		Errores: []MCPValidationIssueV0{{
			Code:    normalizedCode,
			Field:   strings.TrimSpace(field),
			Message: normalizedCode,
		}},
	}
}

func normalizeMCPRunControlActionV0(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
