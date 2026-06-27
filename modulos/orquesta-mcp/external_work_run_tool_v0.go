package orquestamcp

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

const (
	MCPExternalWorkRunToolNameV0       = "orquesta.external_work.run.v0"
	MCPExternalWorkRunToolVersionV0    = "v0"
	MCPExternalWorkRunResourceURIV0    = "orquesta://contracts/external-work-run/v0"
	MCPExternalWorkRunEstadoOKV0       = "ok"
	MCPExternalWorkRunEstadoErrorV0    = "error"
	MCPExternalWorkRunInputAmbiguousV0 = "external_work_run_input_ambiguous"

	MCPExternalWorkRunRoutePolicyLegacyDirectorLoopV0   = "legacy_director_loop"
	MCPExternalWorkRunRoutePolicyGoalFirstV0            = "goal_first"
	MCPExternalWorkRunNextActionSuperviseLegacyRunV0    = "supervise_legacy_run_or_wait_resident"
	MCPExternalWorkRunNextActionMigrateGoalFirstV0      = "migrate_external_work_to_goal_first"
	MCPExternalWorkRunNextActionObserveGoalV0           = "observe_goal"
	MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0 = "legacy_director_loop"
	MCPExternalWorkRunDirectorExecutionModeGoalFirstV0  = "goal_first"
)

type MCPExternalWorkRunToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPExternalWorkRunToolInputV0 struct {
	RequestID              string                                                `json:"request_id,omitempty"`
	CorrelationID          string                                                `json:"correlation_id,omitempty"`
	ExternalWorkRunRequest orquestaexternalworkrun.StartExternalWorkRunRequestV0 `json:"external_work_run_request,omitempty"`
	AppChangeRequest       orquestaappchange.AppChangeRequestV0                  `json:"app_change_request,omitempty"`
}

type MCPExternalWorkRunToolResultV0 struct {
	Estado                string                      `json:"estado"`
	RoutePolicy           string                      `json:"route_policy,omitempty"`
	DirectorExecutionMode string                      `json:"director_execution_mode,omitempty"`
	RequestID             string                      `json:"request_id,omitempty"`
	CorrelationID         string                      `json:"correlation_id,omitempty"`
	RunRef                string                      `json:"run_ref,omitempty"`
	ProjectRef            string                      `json:"project_ref,omitempty"`
	AppRef                string                      `json:"app_ref,omitempty"`
	ChangeRef             string                      `json:"change_ref,omitempty"`
	GoalRef               string                      `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                      `json:"external_goal_ref,omitempty"`
	DirectorQuestionRef   string                      `json:"director_question_ref,omitempty"`
	EvidenceRefs          []string                    `json:"evidence_refs,omitempty"`
	NextActions           []string                    `json:"next_actions,omitempty"`
	Errores               []MCPExternalWorkRunIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPExternalWorkRunIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func MCPExternalWorkRunDescriptorV0() MCPExternalWorkRunToolDescriptorV0 {
	return MCPExternalWorkRunToolDescriptorV0{
		Name:        MCPExternalWorkRunToolNameV0,
		Version:     MCPExternalWorkRunToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,external_work_run_request?:StartExternalWorkRunRequestV0,app_change_request?:AppChangeRequestV0}",
		Output:      "ok:{route_policy,director_execution_mode,run_ref,change_ref,goal_ref?,next_actions?}|error:{errores_publicos}",
		ResourceURI: MCPExternalWorkRunResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"legacy explicito cuando no hay backend Goal completo",
			"con backend Goal completo la composicion puede devolver route_policy=goal_first y goal_ref",
			"la ruta legacy crea run operativo y encola para loop historico por puertos inyectados",
			"sin OPES, DB, runtime, filesystem ni proveedor hardcodeado",
		},
	}
}

func externalWorkRunRequestFromMCPV0(
	input MCPExternalWorkRunToolInputV0,
) orquestaexternalworkrun.StartExternalWorkRunRequestV0 {
	request := input.ExternalWorkRunRequest
	if request.AppChangeRequest.ChangeRef == "" && input.AppChangeRequest.ChangeRef != "" {
		request.AppChangeRequest = input.AppChangeRequest
	}
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = strings.TrimSpace(input.RequestID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return request
}

func validateExternalWorkRunMCPInputV0(
	input MCPExternalWorkRunToolInputV0,
) []MCPExternalWorkRunIssueV0 {
	if externalWorkRunRequestHasAppChangeV0(input.ExternalWorkRunRequest) &&
		appChangeRequestPresentMCPV0(input.AppChangeRequest) {
		return []MCPExternalWorkRunIssueV0{{
			Code:  MCPExternalWorkRunInputAmbiguousV0,
			Field: "app_change_request",
		}}
	}
	return nil
}

func externalWorkRunRequestHasAppChangeV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) bool {
	return appChangeRequestPresentMCPV0(request.AppChangeRequest)
}

func appChangeRequestPresentMCPV0(
	request orquestaappchange.AppChangeRequestV0,
) bool {
	return strings.TrimSpace(request.RunRef) != "" ||
		strings.TrimSpace(request.AppRef) != "" ||
		strings.TrimSpace(request.ChangeRef) != "" ||
		strings.TrimSpace(request.UserIntent) != "" ||
		request.ExternalWork != nil
}

func newMCPExternalWorkRunInputErrorV0(
	input MCPExternalWorkRunToolInputV0,
	issues []MCPExternalWorkRunIssueV0,
) MCPExternalWorkRunToolResultV0 {
	return MCPExternalWorkRunToolResultV0{
		Estado:        MCPExternalWorkRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores:       issues,
	}
}

func newMCPExternalWorkRunResultV0(
	result orquestaexternalworkrun.StartExternalWorkRunResultV0,
) MCPExternalWorkRunToolResultV0 {
	estado := MCPExternalWorkRunEstadoOKV0
	if result.Status != orquestaexternalworkrun.ExternalWorkRunStatusAcceptedV0 {
		estado = MCPExternalWorkRunEstadoErrorV0
	}
	return MCPExternalWorkRunToolResultV0{
		Estado:                estado,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyLegacyDirectorLoopV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeLegacyLoopV0,
		RequestID:             strings.TrimSpace(result.RequestID),
		CorrelationID:         strings.TrimSpace(result.CorrelationID),
		RunRef:                strings.TrimSpace(result.RunRef),
		ProjectRef:            strings.TrimSpace(result.ProjectRef),
		AppRef:                strings.TrimSpace(result.AppRef),
		ChangeRef:             strings.TrimSpace(result.ChangeRef),
		DirectorQuestionRef:   strings.TrimSpace(result.DirectorQuestionRef),
		EvidenceRefs:          compactStringsMCPV0(result.EvidenceRefs),
		NextActions: compactStringsMCPV0([]string{
			MCPExternalWorkRunNextActionSuperviseLegacyRunV0,
			MCPExternalWorkRunNextActionMigrateGoalFirstV0,
		}),
		Errores: externalWorkRunIssuesMCPV0(result.Issues),
	}
}

func externalWorkRunIssuesMCPV0(
	issues []orquestaexternalworkrun.ExternalWorkRunIssueV0,
) []MCPExternalWorkRunIssueV0 {
	out := make([]MCPExternalWorkRunIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPExternalWorkRunIssueV0{
			Code:  strings.TrimSpace(issue.Code),
			Field: strings.TrimSpace(issue.Field),
		})
	}
	if out == nil {
		return nil
	}
	return out
}
