package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const AppDirectorIntakePreparedSchemaVersionV0 = "app_director_intake_prepared.v0"

type PrepareAppDirectorIntakeRequestV0 struct {
	RunRef        string                    `json:"run_ref,omitempty"`
	ProjectRef    string                    `json:"project_ref,omitempty"`
	OccurredAt    string                    `json:"occurred_at,omitempty"`
	CorrelationID string                    `json:"correlation_id,omitempty"`
	RequestedBy   string                    `json:"requested_by,omitempty"`
	AppSpec       orquestafactory.AppSpecV0 `json:"app_spec"`
}

type AppDirectorIntakePreparedV0 struct {
	SchemaVersion     string                                      `json:"schema_version"`
	Run               orquestacoreworkflow.OrchestrationRunV0     `json:"run"`
	InitialEvents     []orquestacoreworkflow.OrchestrationEventV0 `json:"initial_events,omitempty"`
	DirectorTask      AppDirectorTaskV0                           `json:"director_task"`
	DirectorTasks     []AppDirectorTaskV0                         `json:"director_tasks,omitempty"`
	CandidateProvider AppDirectorCandidateProviderV0              `json:"candidate_provider"`
	EvidenceRefs      []string                                    `json:"evidence_refs,omitempty"`
}

type AppDirectorTaskV0 struct {
	TaskRef        string                                                     `json:"task_ref"`
	TopicRef       string                                                     `json:"topic_ref"`
	BrainstormRef  string                                                     `json:"brainstorm_ref"`
	ClaimRef       string                                                     `json:"claim_ref"`
	CapacityRef    string                                                     `json:"capacity_ref"`
	AgentRequestID string                                                     `json:"agent_request_id"`
	PhaseID        orquestacoreworkflow.OrchestrationPhaseIDV0                `json:"phase_id"`
	Role           string                                                     `json:"role"`
	Capacity       orquestacoreworkflow.OrchestrationCapacityRecommendationV0 `json:"capacity"`
	Summary        string                                                     `json:"summary"`
	WriteSet       []string                                                   `json:"write_set"`
	EvidenceRefs   []string                                                   `json:"evidence_refs,omitempty"`
}

type AppDirectorIntakeIssueV0 struct {
	Field string `json:"field"`
}

func (issue AppDirectorIntakeIssueV0) Error() string {
	return "app_director_intake_invalido: " + issue.Field
}
