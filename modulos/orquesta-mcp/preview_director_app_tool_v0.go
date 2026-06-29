package orquestamcp

import (
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	MCPPreviewDirectorAppToolNameV0    = "orquesta.apps.director.preview.v0"
	MCPPreviewDirectorAppToolVersionV0 = "v0"
	MCPPreviewDirectorAppResourceURIV0 = "orquesta://contracts/director-app-preview/v0"
)

type MCPPreviewDirectorAppToolResultV0 struct {
	Estado                string                                                      `json:"estado"`
	RequestID             string                                                      `json:"request_id,omitempty"`
	CorrelationID         string                                                      `json:"correlation_id,omitempty"`
	RoutePolicy           MCPAppSpecRoutePolicyV0                                     `json:"route_policy"`
	AppSpec               MCPAppSpecCompactV0                                         `json:"app_spec,omitempty"`
	RunRef                string                                                      `json:"run_ref,omitempty"`
	DirectorExecutionMode string                                                      `json:"director_execution_mode,omitempty"`
	GoalSpecSummary       MCPGoalWorkSpecSummaryV0                                    `json:"goal_spec_summary,omitempty"`
	Estimate              orquestaappdirectorservice.AppDirectorGoalPreviewEstimateV0 `json:"estimate,omitempty"`
	GoalSpecIssues        []orquestagoal.GoalWorkIssueV0                              `json:"goal_spec_issues,omitempty"`
	Errores               []MCPValidationIssueV0                                      `json:"errores_publicos,omitempty"`
	EvidenceRefs          []string                                                    `json:"evidence_refs,omitempty"`
}

func NewMCPPreviewDirectorAppResultV0(
	preview orquestaappdirectorservice.StartAppDirectorGoalPreviewV0,
) MCPPreviewDirectorAppToolResultV0 {
	if preview.Status == orquestaappdirectorservice.StartAppDirectorStatusInvalidV0 {
		return MCPPreviewDirectorAppToolResultV0{
			Estado:         MCPArrancarDirectorAppEstadoErrorV0,
			CorrelationID:  strings.TrimSpace(preview.CorrelationID),
			RoutePolicy:    mcpPreferredDirectorPreviewRoutePolicyV0(),
			GoalSpecIssues: append([]orquestagoal.GoalWorkIssueV0(nil), preview.GoalSpecIssues...),
			Errores:        publicIssuesMCPV0(preview.ValidationIssues),
			EvidenceRefs:   compactStringsMCPV0(preview.EvidenceRefs),
		}
	}
	return MCPPreviewDirectorAppToolResultV0{
		Estado:                MCPArrancarDirectorAppEstadoOKV0,
		RequestID:             strings.TrimSpace(preview.AppSpec.RequestID),
		CorrelationID:         strings.TrimSpace(preview.CorrelationID),
		RoutePolicy:           mcpPreferredDirectorPreviewRoutePolicyV0(),
		AppSpec:               compactAppSpecV0(preview.AppSpec),
		RunRef:                strings.TrimSpace(preview.Run.RunID),
		DirectorExecutionMode: strings.TrimSpace(preview.DirectorExecutionMode),
		GoalSpecSummary:       mcpGoalWorkSpecSummaryV0(preview.GoalSpec),
		Estimate:              preview.Estimate,
		GoalSpecIssues:        append([]orquestagoal.GoalWorkIssueV0(nil), preview.GoalSpecIssues...),
		EvidenceRefs:          compactStringsMCPV0(preview.EvidenceRefs),
	}
}

func mcpPreferredDirectorPreviewRoutePolicyV0() MCPAppSpecRoutePolicyV0 {
	return MCPAppSpecRoutePolicyV0{
		Mode:                "preview_goal_first",
		PreferredEntrypoint: MCPPreviewDirectorAppToolNameV0,
		PublicReason:        "preview_compila_goal_work_spec_sin_lanzar_agentes",
	}
}
