package orquestaappdirectorintake

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const AppDirectorIntakePreparedSchemaVersionV0 = "app_director_intake_prepared.v0"
const AppDirectorInputSpecSchemaVersionV0 = "app_director_input_spec.v0"

const (
	AppDirectorRequestKindCrearAppCompletaV0 = "crear_app_completa"
	AppDirectorRequestKindDocumentarAppV0    = "documentar_app"
	AppDirectorRequestKindPlanificarAppV0    = "planificar_app"
	AppDirectorExecutionModeNormalV0         = "normal"
	AppDirectorExecutionModeDebugV0          = "debug"
)

type PrepareAppDirectorIntakeRequestV0 struct {
	RunRef        string                    `json:"run_ref,omitempty"`
	ProjectRef    string                    `json:"project_ref,omitempty"`
	OccurredAt    string                    `json:"occurred_at,omitempty"`
	CorrelationID string                    `json:"correlation_id,omitempty"`
	RequestedBy   string                    `json:"requested_by,omitempty"`
	AppSpec       orquestafactory.AppSpecV0 `json:"app_spec"`
}

type PrepareAppDirectorInputRequestV0 struct {
	RunRef        string                 `json:"run_ref,omitempty"`
	ProjectRef    string                 `json:"project_ref,omitempty"`
	OccurredAt    string                 `json:"occurred_at,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	RequestedBy   string                 `json:"requested_by,omitempty"`
	AppSpec       AppDirectorInputSpecV0 `json:"app_spec"`
}

type AppDirectorInputSpecV0 struct {
	SchemaVersion    string                             `json:"schema_version"`
	SpecID           string                             `json:"spec_id"`
	CreatedAt        string                             `json:"created_at"`
	App              AppDirectorInputAppV0              `json:"app"`
	Platforms        []string                           `json:"platforms,omitempty"`
	Data             AppDirectorInputDataV0             `json:"data,omitempty"`
	I18N             AppDirectorInputI18NV0             `json:"i18n,omitempty"`
	Quality          AppDirectorInputQualityV0          `json:"quality,omitempty"`
	AgentPreferences AppDirectorInputAgentPreferencesV0 `json:"agent_preferences,omitempty"`
	RequestKind      string                             `json:"request_kind,omitempty"`
	ExecutionMode    string                             `json:"execution_mode,omitempty"`
	Validation       AppDirectorInputValidationV0       `json:"validation"`
}

type AppDirectorInputAppV0 struct {
	Nombre      string `json:"nombre,omitempty"`
	Objetivo    string `json:"objetivo,omitempty"`
	Descripcion string `json:"descripcion,omitempty"`
	TipoApp     string `json:"tipo_app,omitempty"`
	Slug        string `json:"slug"`
}

type AppDirectorInputDataV0 struct {
	Needs               []string `json:"needs,omitempty"`
	PersistenceRequired bool     `json:"persistence_required,omitempty"`
}

type AppDirectorInputI18NV0 struct {
	Enabled bool `json:"enabled,omitempty"`
}

type AppDirectorInputQualityV0 struct {
	Tests string `json:"tests,omitempty"`
}

type AppDirectorInputAgentPreferencesV0 struct {
	Autonomy string `json:"autonomy,omitempty"`
}

type AppDirectorInputValidationV0 struct {
	Estado string `json:"estado"`
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
