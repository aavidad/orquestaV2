package orquestamcp

import (
	"errors"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
	Estado                string                 `json:"estado"`
	RequestID             string                 `json:"request_id,omitempty"`
	CorrelationID         string                 `json:"correlation_id,omitempty"`
	RunRef                string                 `json:"run_ref,omitempty"`
	RunStatus             string                 `json:"run_status,omitempty"`
	DirectorExecutionMode string                 `json:"director_execution_mode,omitempty"`
	GoalRef               string                 `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                 `json:"external_goal_ref,omitempty"`
	GoalStatus            string                 `json:"goal_status,omitempty"`
	ClosureStatus         string                 `json:"closure_status,omitempty"`
	ClosureAccepted       bool                   `json:"closure_accepted,omitempty"`
	ClosureNeedsRework    bool                   `json:"closure_needs_rework,omitempty"`
	Summary               string                 `json:"summary,omitempty"`
	ArtifactRefs          []string               `json:"artifact_refs,omitempty"`
	DomainReceiptRefs     []string               `json:"domain_receipt_refs,omitempty"`
	EvidenceRefs          []string               `json:"evidence_refs,omitempty"`
	ClosureIssues         []MCPValidationIssueV0 `json:"closure_issues,omitempty"`
	Errores               []MCPValidationIssueV0 `json:"errores_publicos,omitempty"`
}

type mcpObserveAppDirectorGoalPublicIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func MCPObserveAppDirectorGoalDescriptorV0() MCPObserveAppDirectorGoalToolDescriptorV0 {
	return MCPObserveAppDirectorGoalToolDescriptorV0{
		Name:        MCPObserveAppDirectorGoalToolNameV0,
		Version:     MCPObserveAppDirectorGoalToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref,occurred_at?,requested_by?}",
		Output:      "ok:{run_ref,run_status?,director_execution_mode?,goal_ref,goal_status,closure_status?,closure_accepted?,artifact_refs?,evidence_refs?}|error:{errores_publicos}",
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
		Estado:                MCPObserveAppDirectorGoalEstadoOKV0,
		RequestID:             strings.TrimSpace(input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:                firstNonEmptyMCPV0(result.RunRef, input.RunRef),
		RunStatus:             strings.TrimSpace(string(result.Run.Status)),
		DirectorExecutionMode: strings.TrimSpace(result.DirectorExecutionMode),
		GoalRef:               strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(result.ExternalGoalRef),
		GoalStatus:            strings.TrimSpace(result.Status),
		ClosureStatus:         strings.TrimSpace(result.Closure.Status),
		ClosureAccepted:       result.Closure.Accepted,
		ClosureNeedsRework:    result.Closure.NeedsRework,
		Summary:               strings.TrimSpace(goalResult.Summary),
		ArtifactRefs:          compactStringsMCPV0(goalResult.ArtifactRefs),
		DomainReceiptRefs:     compactStringsMCPV0(goalResult.DomainReceiptRefs),
		EvidenceRefs:          compactStringsMCPV0(result.EvidenceRefs),
		ClosureIssues:         goalWorkIssuesMCPV0(result.Closure.Issues),
		Errores:               []MCPValidationIssueV0{},
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

func NewMCPObserveAppDirectorGoalErrorResultFromErrorV0(
	input MCPObserveAppDirectorGoalToolInputV0,
	err error,
) (MCPObserveAppDirectorGoalToolResultV0, bool) {
	issue, ok := mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err)
	if !ok {
		return MCPObserveAppDirectorGoalToolResultV0{}, false
	}
	return NewMCPObserveAppDirectorGoalErrorResultV0(input, issue.Code, issue.Field, issue.Message), true
}

func NewMCPAutoprogrammingObserveGoalErrorResultFromErrorV0(
	input MCPAutoprogrammingObserveGoalToolInputV0,
	err error,
) (MCPAutoprogrammingObserveGoalToolResultV0, bool) {
	issue, ok := mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err)
	if !ok {
		return MCPAutoprogrammingObserveGoalToolResultV0{}, false
	}
	issue.Code = strings.Replace(issue.Code, "observe_app_director_goal", "autoprogramming_observe_goal", 1)
	return NewMCPAutoprogrammingObserveGoalErrorResultV0(input, issue.Code, issue.Field, issue.Message), true
}

func mcpObserveAppDirectorGoalPublicIssueFromErrorV0(err error) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	if err == nil {
		return mcpObserveAppDirectorGoalPublicIssueV0{}, false
	}
	var serviceIssue orquestaappdirectorservice.AppDirectorServiceIssueV0
	if errors.As(err, &serviceIssue) {
		return mcpObserveAppDirectorGoalPublicIssueForFieldV0(serviceIssue.Field), true
	}
	var lifecycleIssue orquestagoal.GoalWorkLifecycleIssueErrorV0
	if errors.As(err, &lifecycleIssue) {
		if issue, ok := mcpObserveAppDirectorGoalPublicIssueFromGoalIssuesV0(lifecycleIssue.Issues); ok {
			return issue, true
		}
		return mcpObserveAppDirectorGoalPublicIssueForFieldV0(lifecycleIssue.Field), true
	}
	var coreIssue orquestacionnucleoapp.ErrorV0
	if errors.As(err, &coreIssue) {
		return mcpObserveAppDirectorGoalPublicIssueForCoreErrorV0(coreIssue)
	}
	if mcpObserveAppDirectorGoalLooksLikeMissingStateV0(err.Error()) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalPublicIssueFromGoalIssuesV0(
	issues []orquestagoal.GoalWorkIssueV0,
) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(issue.Field),
			Message: code,
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalPublicIssueForFieldV0(field string) mcpObserveAppDirectorGoalPublicIssueV0 {
	field = strings.TrimSpace(field)
	switch field {
	case "run_ref":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_input_invalid",
			Field:   "run_ref",
			Message: "run_ref_requerido",
		}
	case "ports.goal_state_store":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_store_unbound",
			Field:   "goal_state_store",
			Message: "goal_state_store_not_configured",
		}
	case "ports.goal_observer":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_observer_unbound",
			Field:   "goal_observer",
			Message: "goal_observer_not_configured",
		}
	case "ports.goal_closure_validator":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_closure_validator_unbound",
			Field:   "goal_closure_validator",
			Message: "goal_closure_validator_not_configured",
		}
	case "ports.run_store", "ports.event_sink":
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_closure_reflection_unbound",
			Field:   strings.TrimPrefix(field, "ports."),
			Message: "goal_closure_reflection_not_configured",
		}
	default:
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_dependency_invalid",
			Field:   field,
			Message: "observe_goal_dependency_invalid",
		}
	}
}

func mcpObserveAppDirectorGoalPublicIssueForCoreErrorV0(
	coreIssue orquestacionnucleoapp.ErrorV0,
) (mcpObserveAppDirectorGoalPublicIssueV0, bool) {
	field := strings.TrimSpace(coreIssue.Field)
	message := strings.TrimSpace(coreIssue.Message)
	if field == "app_director_goal_state" && mcpObserveAppDirectorGoalLooksLikeMissingStateV0(message) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	if strings.HasPrefix(field, "app_director_goal_state") {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_invalid",
			Field:   "goal_state",
			Message: "goal_state_invalid",
		}, true
	}
	if coreIssue.Code == orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0 &&
		mcpObserveAppDirectorGoalLooksLikeMissingStateV0(coreIssue.Error()) {
		return mcpObserveAppDirectorGoalPublicIssueV0{
			Code:    "observe_app_director_goal_state_not_found",
			Field:   "goal_state",
			Message: "goal_state_not_found",
		}, true
	}
	return mcpObserveAppDirectorGoalPublicIssueV0{}, false
}

func mcpObserveAppDirectorGoalLooksLikeMissingStateV0(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	return (strings.Contains(normalized, "goal_state") ||
		strings.Contains(normalized, "goal state") ||
		strings.Contains(normalized, "estado de goal")) &&
		(strings.Contains(normalized, "not found") ||
			strings.Contains(normalized, "no encontrado") ||
			strings.Contains(normalized, "no encontrada"))
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
