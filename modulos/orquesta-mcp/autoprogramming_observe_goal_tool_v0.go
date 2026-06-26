package orquestamcp

import (
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
)

const (
	MCPAutoprogrammingObserveGoalToolNameV0    = "orquesta.autoprogramming.observe_goal.v0"
	MCPAutoprogrammingObserveGoalToolVersionV0 = "v0"
	MCPAutoprogrammingObserveGoalResourceURIV0 = "orquesta://contracts/autoprogramming-observe-goal/v0"
	MCPAutoprogrammingObserveGoalEstadoOKV0    = MCPObserveAppDirectorGoalEstadoOKV0
	MCPAutoprogrammingObserveGoalEstadoErrorV0 = MCPObserveAppDirectorGoalEstadoErrorV0
)

type MCPAutoprogrammingObserveGoalToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingObserveGoalToolInputV0 struct {
	RequestID     string `json:"request_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	RunRef        string `json:"run_ref"`
	OccurredAt    string `json:"occurred_at,omitempty"`
	RequestedBy   string `json:"requested_by,omitempty"`
}

type MCPAutoprogrammingObserveGoalToolResultV0 = MCPObserveAppDirectorGoalToolResultV0

func MCPAutoprogrammingObserveGoalDescriptorV0() MCPAutoprogrammingObserveGoalToolDescriptorV0 {
	return MCPAutoprogrammingObserveGoalToolDescriptorV0{
		Name:        MCPAutoprogrammingObserveGoalToolNameV0,
		Version:     MCPAutoprogrammingObserveGoalToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref,occurred_at?,requested_by?}",
		Output:      "ok:{run_ref,run_status?,director_execution_mode?,goal_ref,goal_status,closure_status?,closure_accepted?,artifact_refs?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPAutoprogrammingObserveGoalResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"observa un goal de autoprogramacion ya lanzado por run_ref",
			"no ejecuta loop legacy ni arranca proveedor",
			"la validacion de cierre vive en orquesta-app-director-service sobre GoalWorkStateV0",
		},
	}
}

func ToAutoprogrammingObserveGoalRequestV0(
	input MCPAutoprogrammingObserveGoalToolInputV0,
) orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0 {
	return orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
		RunRef:        strings.TrimSpace(input.RunRef),
		OccurredAt:    strings.TrimSpace(input.OccurredAt),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RequestedBy:   firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-autoprogramming-observe-goal"),
	}
}

func NewMCPAutoprogrammingObserveGoalResultV0(
	input MCPAutoprogrammingObserveGoalToolInputV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) MCPAutoprogrammingObserveGoalToolResultV0 {
	return NewMCPObserveAppDirectorGoalResultV0(
		MCPObserveAppDirectorGoalToolInputV0{
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: strings.TrimSpace(input.CorrelationID),
			RunRef:        strings.TrimSpace(input.RunRef),
			OccurredAt:    strings.TrimSpace(input.OccurredAt),
			RequestedBy:   strings.TrimSpace(input.RequestedBy),
		},
		result,
	)
}

func NewMCPAutoprogrammingObserveGoalErrorResultV0(
	input MCPAutoprogrammingObserveGoalToolInputV0,
	code string,
	field string,
	message string,
) MCPAutoprogrammingObserveGoalToolResultV0 {
	return NewMCPObserveAppDirectorGoalErrorResultV0(
		MCPObserveAppDirectorGoalToolInputV0{
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: strings.TrimSpace(input.CorrelationID),
			RunRef:        strings.TrimSpace(input.RunRef),
			OccurredAt:    strings.TrimSpace(input.OccurredAt),
			RequestedBy:   strings.TrimSpace(input.RequestedBy),
		},
		code,
		field,
		message,
	)
}
