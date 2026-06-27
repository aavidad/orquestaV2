package orquestamcp

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const (
	MCPAutoprogrammingValidateRequestToolNameV0    = "orquesta.autoprogramming.validate_request.v0"
	MCPAutoprogrammingValidateRequestToolVersionV0 = "v0"
	MCPAutoprogrammingValidateRequestResourceURIV0 = "orquesta://contracts/autoprogramming-request/v0"
	MCPAutoprogrammingValidateRequestEstadoOKV0    = "ok"
	MCPAutoprogrammingValidateRequestEstadoErrorV0 = "error"
)

type MCPAutoprogrammingValidateRequestToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingValidateRequestToolInputV0 struct {
	RequestID              string                                           `json:"request_id,omitempty"`
	CorrelationID          string                                           `json:"correlation_id,omitempty"`
	AutoprogrammingRequest orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request"`
}

type MCPAutoprogrammingValidateRequestToolResultV0 struct {
	Estado           string                                               `json:"estado"`
	RequestID        string                                               `json:"request_id,omitempty"`
	CorrelationID    string                                               `json:"correlation_id,omitempty"`
	Accepted         bool                                                 `json:"accepted"`
	Groups           []orquestaautoprogramming.AutoprogrammingTaskGroupV0 `json:"groups,omitempty"`
	WriteSet         []string                                             `json:"write_set,omitempty"`
	RequiredTests    []string                                             `json:"required_tests,omitempty"`
	ProgrammableWork *MCPAutoprogrammingProgrammableWorkV0                `json:"programmable_work,omitempty"`
	Errores          []MCPValidationIssueV0                               `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingProgrammableWorkV0 struct {
	RequestRef       string                                      `json:"request_ref"`
	ProjectRef       string                                      `json:"project_ref"`
	WorktreeRef      string                                      `json:"worktree_ref"`
	BranchRef        string                                      `json:"branch_ref"`
	WorkProfileRefs  []string                                    `json:"work_profile_refs,omitempty"`
	WorkflowTaskRefs []string                                    `json:"workflow_task_refs,omitempty"`
	Groups           []MCPAutoprogrammingProgrammableWorkGroupV0 `json:"groups,omitempty"`
}

type MCPAutoprogrammingProgrammableWorkGroupV0 struct {
	Area            string   `json:"area"`
	SourceTaskRefs  []string `json:"source_task_refs,omitempty"`
	WorkProfileRef  string   `json:"work_profile_ref"`
	WorkflowTaskRef string   `json:"workflow_task_ref"`
	WorkKind        string   `json:"work_kind"`
	PhaseID         string   `json:"phase_id"`
	Title           string   `json:"title,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	WriteSet        []string `json:"write_set,omitempty"`
	RequiredTests   []string `json:"required_tests,omitempty"`
	Criteria        []string `json:"acceptance_criteria,omitempty"`
	ContextRefs     []string `json:"context_refs,omitempty"`
}

type MCPAutoprogrammingValidateRequestToolExecutorV0 struct{}

func MCPAutoprogrammingValidateRequestDescriptorV0() MCPAutoprogrammingValidateRequestToolDescriptorV0 {
	return MCPAutoprogrammingValidateRequestToolDescriptorV0{
		Name:        MCPAutoprogrammingValidateRequestToolNameV0,
		Version:     MCPAutoprogrammingValidateRequestToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,autoprogramming_request:AutoprogrammingRequestV0}",
		Output:      "ok:{accepted,groups,write_set,required_tests,programmable_work? compacto}|error:{accepted:false,errores_publicos}",
		ResourceURI: MCPAutoprogrammingValidateRequestResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delega validacion y trabajo programable en orquesta-autoprogramming",
			"no ejecuta agentes pruebas VCS comandos ni filesystem",
			"no elige DB runtime ni implementacion de cambio",
		},
	}
}

func (executor MCPAutoprogrammingValidateRequestToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingValidateRequestToolInputV0,
) (MCPAutoprogrammingValidateRequestToolResultV0, error) {
	_ = ctx
	request := autoprogrammingRequestFromMCPV0(input)
	validation := orquestaautoprogramming.ValidateAutoprogrammingRequestV0(request)
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request)
	return newMCPAutoprogrammingValidateRequestResultV0(input, request, validation, work), nil
}

func autoprogrammingRequestFromMCPV0(
	input MCPAutoprogrammingValidateRequestToolInputV0,
) orquestaautoprogramming.AutoprogrammingRequestV0 {
	request := input.AutoprogrammingRequest
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	return request
}

func newMCPAutoprogrammingValidateRequestResultV0(
	input MCPAutoprogrammingValidateRequestToolInputV0,
	request orquestaautoprogramming.AutoprogrammingRequestV0,
	validation orquestaautoprogramming.AutoprogrammingRequestValidationResultV0,
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkResultV0,
) MCPAutoprogrammingValidateRequestToolResultV0 {
	estado := MCPAutoprogrammingValidateRequestEstadoOKV0
	accepted := validation.Accepted && work.Accepted
	issues := validation.Issues
	if validation.Accepted && !work.Accepted {
		issues = work.Issues
	}
	if !accepted {
		estado = MCPAutoprogrammingValidateRequestEstadoErrorV0
	}
	result := MCPAutoprogrammingValidateRequestToolResultV0{
		Estado:        estado,
		RequestID:     strings.TrimSpace(request.RequestRef),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, request.RequestRef),
		Accepted:      accepted,
		Groups:        append([]orquestaautoprogramming.AutoprogrammingTaskGroupV0(nil), validation.Groups...),
		WriteSet:      compactStringsMCPV0(validation.WriteSet),
		RequiredTests: compactStringsMCPV0(validation.RequiredTests),
		Errores:       autoprogrammingRequestIssuesMCPV0(issues),
	}
	if work.Accepted {
		result.ProgrammableWork = newMCPAutoprogrammingProgrammableWorkV0(work.Work)
	}
	return result
}

func newMCPAutoprogrammingProgrammableWorkV0(
	work orquestaautoprogramming.AutoprogrammingProgrammableWorkV0,
) *MCPAutoprogrammingProgrammableWorkV0 {
	out := MCPAutoprogrammingProgrammableWorkV0{
		RequestRef:  strings.TrimSpace(work.RequestRef),
		ProjectRef:  strings.TrimSpace(work.ProjectRef),
		WorktreeRef: strings.TrimSpace(work.WorktreeRef),
		BranchRef:   strings.TrimSpace(work.BranchRef),
	}
	goalReady := strings.TrimSpace(work.GoalMigration.Status) == orquestaautoprogramming.AutoprogrammingGoalMigrationGoalReadyV0
	for _, profile := range work.Profiles {
		if goalReady {
			break
		}
		out.WorkProfileRefs = append(out.WorkProfileRefs, strings.TrimSpace(profile.ProfileRef))
	}
	for _, task := range work.Tasks {
		if goalReady {
			break
		}
		out.WorkflowTaskRefs = append(out.WorkflowTaskRefs, strings.TrimSpace(task.TaskID))
	}
	out.WorkProfileRefs = compactStringsMCPV0(out.WorkProfileRefs)
	out.WorkflowTaskRefs = compactStringsMCPV0(out.WorkflowTaskRefs)
	out.Groups = make([]MCPAutoprogrammingProgrammableWorkGroupV0, 0, len(work.Groups))
	for _, group := range work.Groups {
		task := group.Task
		profile := group.Profile
		workflowTaskRef := strings.TrimSpace(task.TaskID)
		workProfileRef := strings.TrimSpace(profile.ProfileRef)
		workKind := strings.TrimSpace(string(task.WorkProfileKind))
		phaseID := strings.TrimSpace(string(task.PhaseID))
		title := strings.TrimSpace(task.Title)
		summary := strings.TrimSpace(task.Summary)
		writeSet := compactStringsMCPV0(task.WriteSet)
		requiredTests := compactStringsMCPV0(task.RequiredTests)
		criteria := compactStringsMCPV0(task.AcceptanceCriteria)
		contextRefs := compactStringsMCPV0(task.ContextRefs)
		if goalReady {
			workflowTaskRef = ""
			workProfileRef = ""
			workKind = "goal_work_spec"
			phaseID = "goal_first"
			writeSet = compactStringsMCPV0(group.WriteSet)
			requiredTests = compactStringsMCPV0(group.RequiredTests)
			criteria = nil
			contextRefs = nil
		}
		out.Groups = append(out.Groups, MCPAutoprogrammingProgrammableWorkGroupV0{
			Area:            strings.TrimSpace(group.Area),
			SourceTaskRefs:  compactStringsMCPV0(group.TaskRefs),
			WorkProfileRef:  workProfileRef,
			WorkflowTaskRef: workflowTaskRef,
			WorkKind:        workKind,
			PhaseID:         phaseID,
			Title:           title,
			Summary:         summary,
			WriteSet:        writeSet,
			RequiredTests:   requiredTests,
			Criteria:        criteria,
			ContextRefs:     contextRefs,
		})
	}
	return &out
}

func autoprogrammingRequestIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
