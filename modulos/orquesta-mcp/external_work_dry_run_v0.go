package orquestamcp

import (
	"context"
	"math"
	"strings"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	MCPExternalWorkDryRunToolNameV0    = "orquesta.external_work.dry_run.v0"
	MCPExternalWorkDryRunToolVersionV0 = "v0"
	MCPExternalWorkDryRunResourceURIV0 = "orquesta://contracts/external-work-dry-run/v0"

	MCPExternalWorkDryRunEvidenceRefV0  = "evidence-ref-external-work-dry-run-v0"
	MCPExternalWorkDryRunDefaultModelV0 = "codex-goal-runtime-default"
)

type MCPExternalWorkDryRunToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPExternalWorkDryRunToolInputV0 struct {
	MCPExternalWorkRunToolInputV0
	Model string `json:"model,omitempty"`
}

type MCPExternalWorkDryRunToolResultV0 struct {
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
	Spec                  orquestagoal.GoalWorkSpecV0 `json:"spec,omitempty"`
	WriteSet              []string                    `json:"write_set,omitempty"`
	RequiredTests         []string                    `json:"required_tests,omitempty"`
	EstModel              string                      `json:"est_model,omitempty"`
	EstTokens             int                         `json:"est_tokens,omitempty"`
	EstCostUSD            float64                     `json:"est_cost_usd,omitempty"`
	EstWallClock          string                      `json:"est_wall_clock,omitempty"`
	EvidenceRefs          []string                    `json:"evidence_refs,omitempty"`
	Issues                []MCPExternalWorkRunIssueV0 `json:"issues,omitempty"`
	Errores               []MCPExternalWorkRunIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPExternalWorkDryRunToolExecutorV0 struct {
	Config orquestaexternalworkrun.StartExternalWorkRunConfigV0
}

func MCPExternalWorkDryRunDescriptorV0() MCPExternalWorkDryRunToolDescriptorV0 {
	return MCPExternalWorkDryRunToolDescriptorV0{
		Name:        MCPExternalWorkDryRunToolNameV0,
		Version:     MCPExternalWorkDryRunToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,director_execution_mode?:goal_first,model?,external_work_run_request?:StartExternalWorkRunRequestV0,app_change_request?:AppChangeRequestV0}",
		Output:      "ok:{route_policy=goal_first,director_execution_mode=goal_first,spec,write_set,required_tests,est_model,est_tokens,est_cost_usd,est_wall_clock,evidence_refs}|error:{errores_publicos}",
		ResourceURI: MCPExternalWorkDryRunResourceURIV0,
		Invariantes: []string{
			"compila el mismo GoalWorkSpecV0 que la ruta goal-first de external_work.run",
			"no lanza agentes, no crea run, no encola y no escribe en stores",
			"expone refs, write-set, pruebas y contratos sin copiar payloads de dominio",
			"la estimacion es orientativa y no sustituye evidencias de cierre",
			"sin OPES, DB, runtime, filesystem ni proveedor hardcodeado",
		},
	}
}

func (executor MCPExternalWorkDryRunToolExecutorV0) Execute(
	_ context.Context,
	input MCPExternalWorkDryRunToolInputV0,
) (MCPExternalWorkDryRunToolResultV0, error) {
	return BuildExternalWorkDryRunV0(input, executor.Config), nil
}

func BuildExternalWorkDryRunV0(
	input MCPExternalWorkDryRunToolInputV0,
	config orquestaexternalworkrun.StartExternalWorkRunConfigV0,
) MCPExternalWorkDryRunToolResultV0 {
	runInput := input.MCPExternalWorkRunToolInputV0
	if issues := validateExternalWorkRunMCPInputV0(runInput); len(issues) > 0 {
		return newMCPExternalWorkDryRunInputErrorV0(input, issues)
	}
	request := orquestaexternalworkrun.PrepareStartExternalWorkRunRequestV0(
		externalWorkRunRequestFromMCPV0(runInput),
		config,
	)
	spec, issues := orquestaexternalworkrun.BuildExternalWorkGoalWorkSpecV0(request, config)
	if len(issues) > 0 {
		return newMCPExternalWorkDryRunExternalWorkErrorV0(input, request, issues)
	}
	spec.DirectorKind = orquestagoal.GoalDirectorKindCodexGoalV0
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	if goalIssues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(goalIssues) > 0 {
		return newMCPExternalWorkDryRunGoalSpecErrorV0(input, request, goalIssues)
	}
	estimate := externalWorkDryRunEstimateV0(spec, input.Model)
	return MCPExternalWorkDryRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoOKV0,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(request.RequestID),
		CorrelationID:         strings.TrimSpace(request.CorrelationID),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		GoalRef:               strings.TrimSpace(spec.GoalRef),
		Spec:                  spec,
		WriteSet:              externalWorkDryRunWriteSetV0(spec),
		RequiredTests:         externalWorkDryRunRequiredTestsV0(spec),
		EstModel:              estimate.Model,
		EstTokens:             estimate.Tokens,
		EstCostUSD:            estimate.CostUSD,
		EstWallClock:          estimate.WallClock,
		EvidenceRefs: compactStringsMCPV0(append(
			[]string{MCPExternalWorkDryRunEvidenceRefV0},
			spec.EvidenceRefs...,
		)),
	}
}

type externalWorkDryRunEstimateSummaryV0 struct {
	Model     string
	Tokens    int
	CostUSD   float64
	WallClock string
}

func externalWorkDryRunEstimateV0(
	spec orquestagoal.GoalWorkSpecV0,
	model string,
) externalWorkDryRunEstimateSummaryV0 {
	model = strings.TrimSpace(model)
	if model == "" {
		model = MCPExternalWorkDryRunDefaultModelV0
	}
	tokens := spec.Budget.TokenBudget
	if tokens <= 0 {
		tokens = 8000 +
			len(spec.Objective)/4 +
			len(spec.ContextRefs)*220 +
			len(spec.RuleRefs)*250 +
			len(spec.WriteSet)*900 +
			len(spec.RequiredTests)*1800 +
			len(spec.AcceptanceCriteria)*450 +
			len(spec.ArtifactContracts)*800
	}
	if tokens < 4000 {
		tokens = 4000
	}
	return externalWorkDryRunEstimateSummaryV0{
		Model:     model,
		Tokens:    tokens,
		CostUSD:   math.Round((float64(tokens)/1000.0)*0.01*10000) / 10000,
		WallClock: externalWorkDryRunWallClockV0(spec, tokens),
	}
}

func externalWorkDryRunWallClockV0(
	spec orquestagoal.GoalWorkSpecV0,
	tokens int,
) string {
	points := 1 + len(spec.RequiredTests) + len(spec.ArtifactContracts) + len(spec.AcceptanceCriteria)/3
	if tokens > 32000 {
		points += 2
	}
	switch {
	case points <= 3:
		return "5-15m"
	case points <= 7:
		return "15-45m"
	default:
		return "45-120m"
	}
}

func externalWorkDryRunWriteSetV0(spec orquestagoal.GoalWorkSpecV0) []string {
	out := make([]string, 0, len(spec.WriteSet))
	for _, scope := range spec.WriteSet {
		if value := strings.TrimSpace(scope.Path); value != "" {
			out = append(out, value)
		}
	}
	return compactStringsMCPV0(out)
}

func externalWorkDryRunRequiredTestsV0(spec orquestagoal.GoalWorkSpecV0) []string {
	out := make([]string, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		value := strings.TrimSpace(test.Command)
		if value == "" {
			value = strings.TrimSpace(test.TestRef)
		}
		if value != "" {
			out = append(out, value)
		}
	}
	return compactStringsMCPV0(out)
}

func newMCPExternalWorkDryRunInputErrorV0(
	input MCPExternalWorkDryRunToolInputV0,
	issues []MCPExternalWorkRunIssueV0,
) MCPExternalWorkDryRunToolResultV0 {
	return MCPExternalWorkDryRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             strings.TrimSpace(input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores:               issues,
	}
}

func newMCPExternalWorkDryRunExternalWorkErrorV0(
	input MCPExternalWorkDryRunToolInputV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	issues []orquestaexternalworkrun.ExternalWorkRunIssueV0,
) MCPExternalWorkDryRunToolResultV0 {
	return MCPExternalWorkDryRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             firstNonEmptyMCPV0(request.RequestID, input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(request.CorrelationID, input.CorrelationID, input.RequestID),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		Errores:               externalWorkRunIssuesMCPV0(issues),
	}
}

func newMCPExternalWorkDryRunGoalSpecErrorV0(
	input MCPExternalWorkDryRunToolInputV0,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	issues []orquestagoal.GoalWorkIssueV0,
) MCPExternalWorkDryRunToolResultV0 {
	out := make([]MCPExternalWorkRunIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			code = orquestaexternalworkrun.ErrExternalWorkRunGoalSpecInvalidV0
		} else {
			code = orquestaexternalworkrun.ErrExternalWorkRunGoalSpecInvalidV0 + ":" + code
		}
		field := "goal_spec"
		if strings.TrimSpace(issue.Field) != "" {
			field += "." + strings.TrimSpace(issue.Field)
		}
		out = append(out, MCPExternalWorkRunIssueV0{Code: code, Field: field})
	}
	return MCPExternalWorkDryRunToolResultV0{
		Estado:                MCPExternalWorkRunEstadoErrorV0,
		RoutePolicy:           MCPExternalWorkRunRoutePolicyGoalFirstV0,
		DirectorExecutionMode: MCPExternalWorkRunDirectorExecutionModeGoalFirstV0,
		RequestID:             firstNonEmptyMCPV0(request.RequestID, input.RequestID),
		CorrelationID:         firstNonEmptyMCPV0(request.CorrelationID, input.CorrelationID, input.RequestID),
		RunRef:                strings.TrimSpace(request.RunRef),
		ProjectRef:            strings.TrimSpace(request.ProjectRef),
		AppRef:                strings.TrimSpace(request.AppChangeRequest.AppRef),
		ChangeRef:             strings.TrimSpace(request.AppChangeRequest.ChangeRef),
		Errores:               out,
	}
}
