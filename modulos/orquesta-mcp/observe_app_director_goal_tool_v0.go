package orquestamcp

import (
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	MCPObserveAppDirectorGoalToolNameV0    = "orquesta.apps.observe_director_goal.v0"
	MCPObserveAppDirectorGoalToolVersionV0 = "v0"
	MCPObserveAppDirectorGoalResourceURIV0 = "orquesta://contracts/observe-app-director-goal/v0"
	MCPObserveAppDirectorGoalEstadoOKV0    = "ok"
	MCPObserveAppDirectorGoalEstadoErrorV0 = "error"
)

type MCPObserveAppDirectorGoalToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPObserveAppDirectorGoalToolInputV0 struct {
	RequestID     string `json:"request_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	RunRef        string `json:"run_ref"`
	OccurredAt    string `json:"occurred_at,omitempty"`
	RequestedBy   string `json:"requested_by,omitempty"`
}

type MCPObserveAppDirectorGoalToolResultV0 struct {
	Estado             string                 `json:"estado"`
	RequestID          string                 `json:"request_id,omitempty"`
	CorrelationID      string                 `json:"correlation_id,omitempty"`
	RunRef             string                 `json:"run_ref,omitempty"`
	RunStatus          string                 `json:"run_status,omitempty"`
	GoalRef            string                 `json:"goal_ref,omitempty"`
	ExternalGoalRef    string                 `json:"external_goal_ref,omitempty"`
	GoalStatus         string                 `json:"goal_status,omitempty"`
	ClosureStatus      string                 `json:"closure_status,omitempty"`
	ClosureAccepted    bool                   `json:"closure_accepted,omitempty"`
	ClosureNeedsRework bool                   `json:"closure_needs_rework,omitempty"`
	Summary            string                 `json:"summary,omitempty"`
	ArtifactRefs       []string               `json:"artifact_refs,omitempty"`
	DomainReceiptRefs  []string               `json:"domain_receipt_refs,omitempty"`
	EvidenceRefs       []string               `json:"evidence_refs,omitempty"`
	ClosureIssues      []MCPValidationIssueV0 `json:"closure_issues,omitempty"`
	Errores            []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

func MCPObserveAppDirectorGoalDescriptorV0() MCPObserveAppDirectorGoalToolDescriptorV0 {
	return MCPObserveAppDirectorGoalToolDescriptorV0{
		Name:        MCPObserveAppDirectorGoalToolNameV0,
		Version:     MCPObserveAppDirectorGoalToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref,occurred_at?,requested_by?}",
		Output:      "ok:{run_ref,run_status?,goal_ref,goal_status,closure_status?,closure_accepted?,artifact_refs?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPObserveAppDirectorGoalResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"observa un goal ya lanzado por run_ref",
			"no ejecuta loop legacy ni arranca proveedor",
			"la validacion de cierre vive en orquesta-app-director-service",
		},
	}
}

func ToObserveAppDirectorGoalRequestV0(
	input MCPObserveAppDirectorGoalToolInputV0,
) orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0 {
	return orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
		RunRef:        strings.TrimSpace(input.RunRef),
		OccurredAt:    strings.TrimSpace(input.OccurredAt),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RequestedBy:   firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-observe-director-goal"),
	}
}

func NewMCPObserveAppDirectorGoalResultV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	goalResult := result.GoalResult
	return MCPObserveAppDirectorGoalToolResultV0{
		Estado:             MCPObserveAppDirectorGoalEstadoOKV0,
		RequestID:          strings.TrimSpace(input.RequestID),
		CorrelationID:      firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:             firstNonEmptyMCPV0(result.RunRef, input.RunRef),
		RunStatus:          strings.TrimSpace(string(result.Run.Status)),
		GoalRef:            strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:    strings.TrimSpace(result.ExternalGoalRef),
		GoalStatus:         strings.TrimSpace(result.Status),
		ClosureStatus:      strings.TrimSpace(result.Closure.Status),
		ClosureAccepted:    result.Closure.Accepted,
		ClosureNeedsRework: result.Closure.NeedsRework,
		Summary:            strings.TrimSpace(goalResult.Summary),
		ArtifactRefs:       compactStringsMCPV0(goalResult.ArtifactRefs),
		DomainReceiptRefs:  compactStringsMCPV0(goalResult.DomainReceiptRefs),
		EvidenceRefs:       compactStringsMCPV0(result.EvidenceRefs),
		ClosureIssues:      goalWorkIssuesMCPV0(result.Closure.Issues),
		Errores:            []MCPValidationIssueV0{},
	}
}

func NewMCPObserveAppDirectorGoalErrorResultV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	code string,
	field string,
	message string,
) MCPObserveAppDirectorGoalToolResultV0 {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "observe_app_director_goal_error"
	}
	return MCPObserveAppDirectorGoalToolResultV0{
		Estado:        MCPObserveAppDirectorGoalEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		Errores: []MCPValidationIssueV0{{
			Code:    code,
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, code)),
		}},
	}
}

func goalWorkIssuesMCPV0(values []orquestagoal.GoalWorkIssueV0) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if code == "" {
			code = "goal_issue"
		}
		out = append(out, MCPValidationIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(value.Field),
			Message: code,
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
