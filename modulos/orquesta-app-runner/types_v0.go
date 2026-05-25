package orquestaapprunner

import (
	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const AppOrchestrationPreparedSchemaVersionV0 = "app_orchestration_prepared.v0"

type PrepareAppOrchestrationRequestV0 struct {
	RunRef        string                    `json:"run_ref,omitempty"`
	ProjectRef    string                    `json:"project_ref,omitempty"`
	OccurredAt    string                    `json:"occurred_at,omitempty"`
	CorrelationID string                    `json:"correlation_id,omitempty"`
	RequestedBy   string                    `json:"requested_by,omitempty"`
	AppSpec       orquestafactory.AppSpecV0 `json:"app_spec"`
}

type AppOrchestrationPreparedV0 struct {
	SchemaVersion     string                                        `json:"schema_version"`
	Run               orquestacoreworkflow.OrchestrationRunV0       `json:"run"`
	Plan              orquestaappplanner.AppMicrotaskPlanV0         `json:"plan"`
	InitialProgress   orquestaappplanner.AppPlanProgressV0          `json:"initial_progress"`
	CandidateProvider orquestaappplanner.AppPlanCandidateProviderV0 `json:"candidate_provider"`
	RoutePolicy       AppRunnerRoutePolicyV0                        `json:"route_policy"`
	EvidenceRefs      []string                                      `json:"evidence_refs,omitempty"`
}

type AppRunnerIssueV0 struct {
	Field string `json:"field"`
}

func (issue AppRunnerIssueV0) Error() string {
	return "app_runner_invalido: " + issue.Field
}
