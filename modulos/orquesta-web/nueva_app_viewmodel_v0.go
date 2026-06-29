package orquestaweb

import (
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type WebNuevaAppEstadoV0 string

const (
	WebNuevaAppEstadoInicial       WebNuevaAppEstadoV0 = "inicial"
	WebNuevaAppEstadoEnviando      WebNuevaAppEstadoV0 = "enviando"
	WebNuevaAppEstadoValida        WebNuevaAppEstadoV0 = "valida"
	WebNuevaAppEstadoDirector      WebNuevaAppEstadoV0 = "director_arrancado"
	WebNuevaAppEstadoGoalPreview   WebNuevaAppEstadoV0 = "goal_preview"
	WebNuevaAppEstadoRequiereDatos WebNuevaAppEstadoV0 = "requiere_datos"
	WebNuevaAppEstadoInvalida      WebNuevaAppEstadoV0 = "invalida"
	WebNuevaAppEstadoError         WebNuevaAppEstadoV0 = "error"
)

type WebNuevaAppViewModelV0 struct {
	RequestID              string                      `json:"request_id"`
	Locale                 string                      `json:"locale"`
	Estado                 WebNuevaAppEstadoV0         `json:"estado"`
	ResumenApp             WebNuevaAppResumenV0        `json:"resumen_app"`
	BacklogPreview         WebNuevaAppBacklogPreviewV0 `json:"backlog_preview"`
	DefaultsAplicados      []WebNuevaAppDefaultV0      `json:"defaults_aplicados"`
	Warnings               []WebNuevaAppIssueV0        `json:"warnings"`
	ErroresPublicos        []WebNuevaAppIssueV0        `json:"errores_publicos"`
	PreguntasAbiertas      []string                    `json:"preguntas_abiertas"`
	Fases                  []WebNuevaAppFaseV0         `json:"fases"`
	Microtareas            []WebNuevaAppMicrotareaV0   `json:"microtareas"`
	ContratosRequeridos    []string                    `json:"contratos_requeridos"`
	Riesgos                []string                    `json:"riesgos"`
	BacklogSchemaVersion   string                      `json:"backlog_schema_version,omitempty"`
	AppSpecSchemaVersion   string                      `json:"app_spec_schema_version,omitempty"`
	SpecID                 string                      `json:"spec_id,omitempty"`
	ValidationEstadoFuente string                      `json:"validation_estado_fuente,omitempty"`
	Director               *WebNuevaAppDirectorV0      `json:"director,omitempty"`
	GoalPreview            *WebNuevaAppGoalPreviewV0   `json:"goal_preview,omitempty"`
}

type WebNuevaAppResumenV0 struct {
	Nombre        string   `json:"nombre,omitempty"`
	Slug          string   `json:"slug,omitempty"`
	Objetivo      string   `json:"objetivo,omitempty"`
	Descripcion   string   `json:"descripcion,omitempty"`
	TipoApp       string   `json:"tipo_app,omitempty"`
	RequestKind   string   `json:"request_kind,omitempty"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
	Locale        string   `json:"locale,omitempty"`
	DefaultLocale string   `json:"default_locale,omitempty"`
	I18NEnabled   bool     `json:"i18n_enabled"`
	Plataformas   []string `json:"plataformas,omitempty"`
	DeployTarget  string   `json:"deploy_target,omitempty"`
}

type WebNuevaAppDefaultV0 struct {
	Campo  string `json:"campo"`
	Valor  string `json:"valor"`
	Motivo string `json:"motivo"`
}

type WebNuevaAppIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type WebNuevaAppFaseV0 struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Objetivo string `json:"objetivo"`
	Orden    int    `json:"orden"`
}

type WebNuevaAppMicrotareaV0 struct {
	ID               string   `json:"id"`
	Fase             string   `json:"fase"`
	ModuloSugerido   string   `json:"modulo_sugerido"`
	Objetivo         string   `json:"objetivo"`
	WriteSetPrevisto []string `json:"write_set_previsto"`
	Contrato         string   `json:"contrato"`
	Validacion       string   `json:"validacion"`
	Bloqueos         []string `json:"bloqueos"`
}

type WebNuevaAppDirectorV0 struct {
	RunRef                string                            `json:"run_ref"`
	PhaseID               string                            `json:"phase_id,omitempty"`
	DirectorExecutionMode string                            `json:"director_execution_mode,omitempty"`
	LoopStatus            string                            `json:"loop_status,omitempty"`
	DirectorTasks         []WebNuevaAppDirectorTaskV0       `json:"director_tasks,omitempty"`
	StartedAgents         []string                          `json:"started_agents,omitempty"`
	GoalRef               string                            `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                            `json:"external_goal_ref,omitempty"`
	GoalStatus            string                            `json:"goal_status,omitempty"`
	GoalLaunchReceipt     *orquestagoal.GoalLaunchReceiptV0 `json:"goal_launch_receipt,omitempty"`
	EvidenceRefs          []string                          `json:"evidence_refs,omitempty"`
}

type WebNuevaAppDirectorTaskV0 struct {
	TaskRef        string `json:"task_ref,omitempty"`
	BrainstormRef  string `json:"brainstorm_ref,omitempty"`
	AgentRequestID string `json:"agent_request_id,omitempty"`
	Capacity       string `json:"capacity,omitempty"`
}

type WebNuevaAppGoalPreviewV0 struct {
	RunRef                string                           `json:"run_ref,omitempty"`
	GoalRef               string                           `json:"goal_ref,omitempty"`
	DirectorExecutionMode string                           `json:"director_execution_mode,omitempty"`
	WorkKind              string                           `json:"work_kind,omitempty"`
	WorkProfileKind       string                           `json:"work_profile_kind,omitempty"`
	DirectorKind          string                           `json:"director_kind,omitempty"`
	SpecHash              string                           `json:"spec_hash,omitempty"`
	ContextRefs           []string                         `json:"context_refs,omitempty"`
	RuleRefs              []string                         `json:"rule_refs,omitempty"`
	RequiredTestRefs      []string                         `json:"required_test_refs,omitempty"`
	ArtifactTypes         []string                         `json:"artifact_types,omitempty"`
	SpecSummary           WebGoalWorkSpecSummaryV0         `json:"goal_spec_summary,omitempty"`
	Estimate              WebNuevaAppGoalPreviewEstimateV0 `json:"estimate,omitempty"`
	EvidenceRefs          []string                         `json:"evidence_refs,omitempty"`
}

type WebGoalWorkSpecSummaryV0 struct {
	SchemaVersion            string   `json:"schema_version"`
	GoalRef                  string   `json:"goal_ref,omitempty"`
	RunRef                   string   `json:"run_ref,omitempty"`
	DirectorKind             string   `json:"director_kind,omitempty"`
	SpecHash                 string   `json:"spec_hash,omitempty"`
	ContextRefs              []string `json:"context_refs,omitempty"`
	RuleRefs                 []string `json:"rule_refs,omitempty"`
	RequiredTestRefs         []string `json:"required_test_refs,omitempty"`
	ArtifactTypes            []string `json:"artifact_types,omitempty"`
	ContextRefCount          int      `json:"context_ref_count,omitempty"`
	RuleRefCount             int      `json:"rule_ref_count,omitempty"`
	WriteSetCount            int      `json:"write_set_count,omitempty"`
	RequiredTestCount        int      `json:"required_test_count,omitempty"`
	AcceptanceCriteriaCount  int      `json:"acceptance_criteria_count,omitempty"`
	ArtifactContractCount    int      `json:"artifact_contract_count,omitempty"`
	ClosureRequiresTests     bool     `json:"closure_requires_tests,omitempty"`
	ClosureRequiresArtifacts bool     `json:"closure_requires_artifacts,omitempty"`
}

type WebNuevaAppGoalPreviewEstimateV0 struct {
	TokenBudget       int    `json:"token_budget,omitempty"`
	MaxRuntimeSeconds int    `json:"max_runtime_seconds,omitempty"`
	MaxSubgoals       int    `json:"max_subgoals,omitempty"`
	MaxReworkGoals    int    `json:"max_rework_goals,omitempty"`
	WriteSetItems     int    `json:"write_set_items,omitempty"`
	RequiredTests     int    `json:"required_tests,omitempty"`
	ArtifactContracts int    `json:"artifact_contracts,omitempty"`
	CostTier          string `json:"cost_tier,omitempty"`
}

func NewWebNuevaAppViewModelV0(spec orquestafactory.AppSpecV0, backlog orquestafactory.BacklogInicialPropuestoV0) WebNuevaAppViewModelV0 {
	return WebNuevaAppViewModelV0{
		RequestID:              spec.RequestID,
		Locale:                 spec.Locale,
		Estado:                 estadoFromSpecV0(spec),
		ResumenApp:             resumenFromSpecV0(spec),
		BacklogPreview:         backlogPreviewFromBacklogV0(backlog),
		DefaultsAplicados:      defaultsFromSpecV0(spec.DefaultsApplied),
		Warnings:               issuesFromFactoryV0(spec.Validation.Warnings),
		ErroresPublicos:        issuesFromFactoryV0(spec.Validation.Errores),
		PreguntasAbiertas:      compactStringsV0(append(spec.Scope.PreguntasAbiertas, backlog.PreguntasAbiertas...)),
		Fases:                  fasesFromBacklogV0(backlog.Fases),
		Microtareas:            microtareasFromBacklogV0(backlog.Microtareas),
		ContratosRequeridos:    compactStringsV0(backlog.ContratosRequeridos),
		Riesgos:                compactStringsV0(backlog.Riesgos),
		BacklogSchemaVersion:   backlog.SchemaVersion,
		AppSpecSchemaVersion:   spec.SchemaVersion,
		SpecID:                 spec.SpecID,
		ValidationEstadoFuente: spec.Validation.Estado,
	}
}

func NewWebNuevaAppErrorViewModelV0(requestID, locale string, errores []orquestafactory.ValidationIssue) WebNuevaAppViewModelV0 {
	return WebNuevaAppViewModelV0{
		RequestID:           trimV0(requestID),
		Locale:              trimV0(locale),
		Estado:              WebNuevaAppEstadoInvalida,
		BacklogPreview:      emptyBacklogPreviewV0(),
		ErroresPublicos:     issuesFromFactoryV0(errores),
		PreguntasAbiertas:   []string{},
		Fases:               []WebNuevaAppFaseV0{},
		Microtareas:         []WebNuevaAppMicrotareaV0{},
		ContratosRequeridos: []string{},
		Riesgos:             []string{},
	}
}
