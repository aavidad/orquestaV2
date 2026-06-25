package orquestaweb

import (
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	WebAutoprogrammingPrepareRunInboundEndpointV0 = "/api/v0/autoprogramming/prepare-run"
	WebAutoprogrammingStatusInboundEndpointV0     = "/api/v0/autoprogramming/status"

	WebAutoprogrammingPrepareRunEstadoOKV0    = "ok"
	WebAutoprogrammingPrepareRunEstadoErrorV0 = "error"

	WebAutoprogrammingPrepareRunErrTransporteV0        = "autoprogramming_prepare_run_error_transporte"
	WebAutoprogrammingPrepareRunErrRespuestaInvalidaV0 = "autoprogramming_prepare_run_respuesta_invalida"
	WebAutoprogrammingStatusErrTransporteV0            = "autoprogramming_status_error_transporte"
	WebAutoprogrammingStatusErrRespuestaInvalidaV0     = "autoprogramming_status_respuesta_invalida"
)

type WebAutoprogrammingPrepareRunCommandV0 struct {
	RequestID            string                                                        `json:"request_id,omitempty"`
	CorrelationID        string                                                        `json:"correlation_id,omitempty"`
	IdempotencyKey       string                                                        `json:"idempotency_key,omitempty"`
	Locale               string                                                        `json:"locale,omitempty"`
	OccurredAt           string                                                        `json:"occurred_at,omitempty"`
	RequestedBy          string                                                        `json:"requested_by,omitempty"`
	RequestRef           string                                                        `json:"request_ref,omitempty"`
	ProjectRef           string                                                        `json:"project_ref,omitempty"`
	WorktreeRef          string                                                        `json:"worktree_ref,omitempty"`
	BranchRef            string                                                        `json:"branch_ref,omitempty"`
	Tasks                []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 `json:"tasks,omitempty"`
	WriteSet             []string                                                      `json:"write_set,omitempty"`
	RequiredTests        []string                                                      `json:"required_tests,omitempty"`
	MaxBursts            int                                                           `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int                                                           `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int                                                           `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int                                                           `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int                                                           `json:"max_outbox_per_cycle,omitempty"`
	PriorityScore        int                                                           `json:"priority_score,omitempty"`
	MaxTaskRefs          int                                                           `json:"max_task_refs,omitempty"`
	MaxAreas             int                                                           `json:"max_areas,omitempty"`
	MaxWriteSetEntries   int                                                           `json:"max_write_set_entries,omitempty"`
}
type WebAutoprogrammingPrepareRunViewModelV0 struct {
	SchemaVersion    string                                      `json:"schema_version"`
	Locale           string                                      `json:"locale,omitempty"`
	Estado           string                                      `json:"estado"`
	Accepted         bool                                        `json:"accepted"`
	RunRef           string                                      `json:"run_ref,omitempty"`
	ProjectRef       string                                      `json:"project_ref,omitempty"`
	WorktreeRef      string                                      `json:"worktree_ref,omitempty"`
	BranchRef        string                                      `json:"branch_ref,omitempty"`
	PhaseID          string                                      `json:"phase_id,omitempty"`
	WorkflowTaskRefs []string                                    `json:"workflow_task_refs,omitempty"`
	WaitAgentRefs    []string                                    `json:"wait_agent_refs,omitempty"`
	GoalSpecs        []orquestagoal.GoalWorkSpecV0               `json:"goal_specs,omitempty"`
	Continue         *WebAutoprogrammingContinueRequestV0        `json:"continue,omitempty"`
	ErroresPublicos  []WebAutoprogrammingPrepareRunPublicIssueV0 `json:"errores_publicos,omitempty"`
}
type WebAutoprogrammingContinueRequestV0 struct {
	RunRef                     string   `json:"run_ref"`
	OperationalDirectorPlanRef string   `json:"operational_director_plan_ref,omitempty"`
	OccurredAt                 string   `json:"occurred_at,omitempty"`
	CorrelationID              string   `json:"correlation_id,omitempty"`
	RequestedBy                string   `json:"requested_by,omitempty"`
	MaxBursts                  int      `json:"max_bursts,omitempty"`
	MaxStepsPerBurst           int      `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait       int      `json:"max_dispatches_per_wait,omitempty"`
	WaitAgentRefs              []string `json:"wait_agent_refs,omitempty"`
	MaxCommands                int      `json:"max_commands,omitempty"`
	MaxOutboxPerCycle          int      `json:"max_outbox_per_cycle,omitempty"`
}
type WebAutoprogrammingPrepareRunPublicIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewWebAutoprogrammingPrepareRunViewModelV0(locale string, result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0) WebAutoprogrammingPrepareRunViewModelV0 {
	vm := WebAutoprogrammingPrepareRunViewModelV0{
		SchemaVersion:    "web_autoprogramming_prepare_run.v0",
		Locale:           normalizeDirectorStatsLocaleV0(locale),
		Estado:           webAutoprogrammingPrepareRunEstadoV0(result.Estado),
		Accepted:         result.Accepted,
		RunRef:           trimV0(result.RunRef),
		ProjectRef:       trimV0(result.ProjectRef),
		WorktreeRef:      trimV0(result.WorktreeRef),
		BranchRef:        trimV0(result.BranchRef),
		PhaseID:          trimV0(result.PhaseID),
		WorkflowTaskRefs: compactStringsV0(result.WorkflowTaskRefs),
		WaitAgentRefs:    compactStringsV0(result.WaitAgentRefs),
		GoalSpecs:        append([]orquestagoal.GoalWorkSpecV0(nil), result.GoalSpecs...),
		ErroresPublicos:  webAutoprogrammingPrepareRunIssuesV0(result.Errores),
	}
	if result.Continue != nil {
		vm.Continue = webAutoprogrammingContinueRequestV0(*result.Continue)
	}
	return vm
}

func webAutoprogrammingPrepareRunEstadoV0(value string) string {
	if trimV0(value) == WebAutoprogrammingPrepareRunEstadoOKV0 {
		return WebAutoprogrammingPrepareRunEstadoOKV0
	}
	return WebAutoprogrammingPrepareRunEstadoErrorV0
}

func webAutoprogrammingContinueRequestV0(value orquestamcp.MCPAutoprogrammingContinueRequestV0) *WebAutoprogrammingContinueRequestV0 {
	return &WebAutoprogrammingContinueRequestV0{
		RunRef:                     trimV0(value.RunRef),
		OperationalDirectorPlanRef: trimV0(value.OperationalDirectorPlanRef),
		OccurredAt:                 trimV0(value.OccurredAt),
		CorrelationID:              trimV0(value.CorrelationID),
		RequestedBy:                trimV0(value.RequestedBy),
		MaxBursts:                  value.MaxBursts,
		MaxStepsPerBurst:           value.MaxStepsPerBurst,
		MaxDispatchesPerWait:       value.MaxDispatchesPerWait,
		WaitAgentRefs:              compactStringsV0(value.WaitAgentRefs),
		MaxCommands:                value.MaxCommands,
		MaxOutboxPerCycle:          value.MaxOutboxPerCycle,
	}
}

func webAutoprogrammingPrepareRunIssuesV0(values []orquestamcp.MCPValidationIssueV0) []WebAutoprogrammingPrepareRunPublicIssueV0 {
	out := make([]WebAutoprogrammingPrepareRunPublicIssueV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebAutoprogrammingPrepareRunPublicIssueV0{
			Code:    trimV0(value.Code),
			Field:   trimV0(value.Field),
			Message: trimV0(value.Message),
		})
	}
	return out
}

func compactAutoprogrammingTasksV0(values []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0) []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0 {
	out := make([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		taskRef := trimV0(value.TaskRef)
		area := trimV0(value.Area)
		if taskRef == "" || area == "" {
			continue
		}
		key := taskRef + "\x00" + area
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
			TaskRef:            taskRef,
			Area:               area,
			Title:              trimV0(value.Title),
			Objective:          trimV0(value.Objective),
			Context:            compactStringsV0(value.Context),
			ContextRefs:        compactStringsV0(value.ContextRefs),
			AcceptanceCriteria: compactStringsV0(value.AcceptanceCriteria),
			RequiredTests:      compactStringsV0(value.RequiredTests),
			CompactRules:       compactStringsV0(value.CompactRules),
		})
	}
	return out
}
