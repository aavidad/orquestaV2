package orquestacoreworkflow

const OrchestrationRunSchemaVersionV0 = "orchestration_run.v0"

type OrchestrationPhaseIDV0 string
type OrchestrationRunStatusV0 string
type OrchestrationPhaseStatusV0 string
type OrchestrationCapacityRecommendationV0 string
type OrchestrationValidationCodeV0 string

const (
	OrchestrationPhaseDescubrimientoV0            OrchestrationPhaseIDV0 = "descubrimiento"
	OrchestrationPhaseBrainstormingArquitecturaV0 OrchestrationPhaseIDV0 = "brainstorming_arquitectura"
	OrchestrationPhaseVotacionYDecisionV0         OrchestrationPhaseIDV0 = "votacion_y_decision"
	OrchestrationPhasePlanificacionMicrotareasV0  OrchestrationPhaseIDV0 = "planificacion_microtareas"
	OrchestrationPhaseProgramacionV0              OrchestrationPhaseIDV0 = "programacion"
	OrchestrationPhaseDocumentacionV0             OrchestrationPhaseIDV0 = "documentacion"
	OrchestrationPhaseIntegracionV0               OrchestrationPhaseIDV0 = "integracion"
	OrchestrationPhaseRevisionV0                  OrchestrationPhaseIDV0 = "revision"
	OrchestrationPhaseValidacionFinalV0           OrchestrationPhaseIDV0 = "validacion_final"
	OrchestrationPhaseCierreV0                    OrchestrationPhaseIDV0 = "cierre"
)

const (
	OrchestrationRunStatusPendingV0 OrchestrationRunStatusV0 = "pendiente"
	OrchestrationRunStatusActiveV0  OrchestrationRunStatusV0 = "activa"
	OrchestrationRunStatusBlockedV0 OrchestrationRunStatusV0 = "bloqueada"
	OrchestrationRunStatusClosedV0  OrchestrationRunStatusV0 = "cerrada"
)

const (
	OrchestrationPhaseStatusPendingV0 OrchestrationPhaseStatusV0 = "pendiente"
	OrchestrationPhaseStatusActiveV0  OrchestrationPhaseStatusV0 = "activa"
	OrchestrationPhaseStatusClosedV0  OrchestrationPhaseStatusV0 = "cerrada"
	OrchestrationPhaseStatusBlockedV0 OrchestrationPhaseStatusV0 = "bloqueada"
)

const (
	OrchestrationCapacityLowV0    OrchestrationCapacityRecommendationV0 = "low"
	OrchestrationCapacityMediumV0 OrchestrationCapacityRecommendationV0 = "medium"
	OrchestrationCapacityHighV0   OrchestrationCapacityRecommendationV0 = "high"
	OrchestrationCapacityXHighV0  OrchestrationCapacityRecommendationV0 = "xhigh"
)

const (
	OrchestrationRunInvalidoV0         OrchestrationValidationCodeV0 = "run_invalido"
	OrchestrationFaseInvalidaV0        OrchestrationValidationCodeV0 = "fase_invalida"
	OrchestrationEstadoInconsistenteV0 OrchestrationValidationCodeV0 = "estado_inconsistente"
	OrchestrationFaseNoSoportadaV0     OrchestrationValidationCodeV0 = "fase_no_soportada"
	OrchestrationDetalleProhibidoV0    OrchestrationValidationCodeV0 = "detalle_prohibido"
)

type OrchestrationRunV0 struct {
	SchemaVersion             string                   `json:"schema_version"`
	RunID                     string                   `json:"run_id"`
	ProjectRef                string                   `json:"project_ref"`
	AppSpecRef                string                   `json:"app_spec_ref"`
	Status                    OrchestrationRunStatusV0 `json:"status"`
	CurrentPhase              OrchestrationPhaseIDV0   `json:"current_phase"`
	Phases                    []OrchestrationPhaseV0   `json:"phases"`
	Brainstorms               []string                 `json:"brainstorms"`
	Votes                     []string                 `json:"votes"`
	Tasks                     []string                 `json:"tasks"`
	FunctionContracts         []string                 `json:"function_contracts"`
	Decisions                 []string                 `json:"decisions"`
	CapacityRequests          []string                 `json:"capacity_requests"`
	CapacityDecisions         []string                 `json:"capacity_decisions"`
	Agents                    []string                 `json:"agents"`
	StartedAgents             []string                 `json:"started_agents"`
	FailedAgents              []string                 `json:"failed_agents"`
	StoppedAgents             []string                 `json:"stopped_agents"`
	ConfirmedStoppedAgents    []string                 `json:"confirmed_stopped_agents"`
	AgentAssessments          []string                 `json:"agent_assessments"`
	AgentLeaseExpirations     []string                 `json:"agent_lease_expirations"`
	ConcurrencyGates          []string                 `json:"concurrency_gates"`
	QualityGates              []string                 `json:"quality_gates"`
	PhaseArtifacts            []string                 `json:"phase_artifacts"`
	Deliveries                []string                 `json:"deliveries"`
	DeliveredTasks            []string                 `json:"delivered_tasks"`
	DeliveredAgents           []string                 `json:"delivered_agents"`
	Reviews                   []string                 `json:"reviews"`
	ReviewResults             []string                 `json:"review_results"`
	ReworkRequests            []string                 `json:"rework_requests"`
	ReplanDecisions           []string                 `json:"replan_decisions"`
	AcceptedReviews           []string                 `json:"accepted_reviews"`
	ClosedTasks               []string                 `json:"closed_tasks"`
	Validations               []string                 `json:"validations"`
	Closures                  []string                 `json:"closures"`
	DirectorQuestions         []string                 `json:"director_questions"`
	DirectorAnswers           []string                 `json:"director_answers"`
	DirectorAnsweredQuestions []string                 `json:"director_answered_questions"`
	Blockers                  []string                 `json:"blockers"`
	CommandEffects            []string                 `json:"command_effects,omitempty"`
	LastEventID               string                   `json:"last_event_id"`
	LastSequence              int64                    `json:"last_sequence"`
}

type OrchestrationPhaseV0 struct {
	ID                  OrchestrationPhaseIDV0                `json:"id"`
	Status              OrchestrationPhaseStatusV0            `json:"status"`
	OpenedAt            string                                `json:"opened_at,omitempty"`
	ClosedAt            string                                `json:"closed_at,omitempty"`
	EntryCriteria       []string                              `json:"entry_criteria"`
	ExitCriteria        []string                              `json:"exit_criteria"`
	EvidenceRequired    []string                              `json:"evidence_required"`
	RecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"recommended_capacity"`
}

type OrchestrationValidationIssueV0 struct {
	Code  OrchestrationValidationCodeV0 `json:"code"`
	Field string                        `json:"field,omitempty"`
}

func (issue OrchestrationValidationIssueV0) Error() string {
	if issue.Field == "" {
		return string(issue.Code)
	}
	return string(issue.Code) + ": " + issue.Field
}
