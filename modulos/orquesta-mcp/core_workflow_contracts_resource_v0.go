package orquestamcp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	MCPCoreWorkflowContractsResourceNameV0    = "orquesta.core_workflow.contracts.v0"
	MCPCoreWorkflowContractsResourceVersionV0 = "v0"
	MCPCoreWorkflowContractsResourceURIV0     = "orquesta://core-workflow/contracts/v0"
	MCPCoreWorkflowContractsContentTypeV0     = "application/vnd.orquesta.core-workflow.contracts.v0+json"
)

type MCPCoreWorkflowContractsResourceDescriptorV0 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	URI         string `json:"uri"`
	ContentType string `json:"content_type"`
	SummaryKey  string `json:"summary_key"`
}

type MCPCoreWorkflowContractsResourceV0 struct {
	URI           string                             `json:"uri"`
	Version       string                             `json:"version"`
	SummaryKey    string                             `json:"summary_key"`
	Owner         string                             `json:"owner"`
	CanonicalRefs []string                           `json:"canonical_refs"`
	StateShape    MCPCoreWorkflowStateShapeV0        `json:"state_shape"`
	Phases        []string                           `json:"phases"`
	Commands      []MCPCoreWorkflowTransitionGuideV0 `json:"commands"`
	Events        []MCPCoreWorkflowTransitionGuideV0 `json:"events"`
	PublicErrors  []string                           `json:"errores_publicos"`
	Guardrails    []string                           `json:"guardrails"`
}

type MCPCoreWorkflowStateShapeV0 struct {
	SchemaVersion string   `json:"schema_version"`
	Status        []string `json:"status"`
	Refs          []string `json:"refs"`
	Counters      []string `json:"counters"`
}

type MCPCoreWorkflowTransitionGuideV0 struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Phase      string   `json:"phase,omitempty"`
	SummaryKey string   `json:"summary_key,omitempty"`
	Requires   []string `json:"requires,omitempty"`
	Projects   []string `json:"projects,omitempty"`
	Guardrails []string `json:"guardrails,omitempty"`
}

func MCPCoreWorkflowContractsDescriptorV0() MCPCoreWorkflowContractsResourceDescriptorV0 {
	return MCPCoreWorkflowContractsResourceDescriptorV0{
		Name:        MCPCoreWorkflowContractsResourceNameV0,
		Version:     MCPCoreWorkflowContractsResourceVersionV0,
		URI:         MCPCoreWorkflowContractsResourceURIV0,
		ContentType: MCPCoreWorkflowContractsContentTypeV0,
		SummaryKey:  "mcp.resources.core_workflow.contracts.summary.v0",
	}
}

func NewMCPCoreWorkflowContractsResourceV0() MCPCoreWorkflowContractsResourceV0 {
	return MCPCoreWorkflowContractsResourceV0{
		URI:        MCPCoreWorkflowContractsResourceURIV0,
		Version:    MCPCoreWorkflowContractsResourceVersionV0,
		SummaryKey: "mcp.resources.core_workflow.contracts.summary.v0",
		Owner:      "orquesta-core-workflow",
		CanonicalRefs: compactStringsMCPV0([]string{
			"orquesta-core-workflow/docs/contratos.md",
			"orquesta-core-workflow/docs/contratos_estado_fases.md",
			"orquesta-core-workflow/docs/contratos_agentes.md",
			"orquesta-core-workflow/docs/contratos_quality_gates.md",
		}),
		StateShape: MCPCoreWorkflowStateShapeV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
			Status: []string{
				string(orquestacoreworkflow.OrchestrationRunStatusPendingV0),
				string(orquestacoreworkflow.OrchestrationRunStatusActiveV0),
				string(orquestacoreworkflow.OrchestrationRunStatusBlockedV0),
				string(orquestacoreworkflow.OrchestrationRunStatusClosedV0),
			},
			Refs: []string{
				"project_ref", "app_spec_ref", "brainstorms", "votes", "tasks", "function_contracts",
				"decisions", "capacity_requests", "capacity_decisions", "agents", "started_agents",
				"failed_agents", "stopped_agents", "confirmed_stopped_agents", "lost_agents",
				"agent_assessments", "agent_lease_expirations", "concurrency_gates", "deliveries", "reviews",
				"quality_gates", "phase_artifacts", "review_results", "rework_requests", "replan_decisions",
				"accepted_reviews", "closed_tasks", "validations", "closures", "director_questions",
				"director_answers", "director_answered_questions", "blockers",
			},
			Counters: []string{"last_event_id", "last_sequence"},
		},
		Phases:       coreWorkflowPhaseIDsMCPV0(),
		Commands:     coreWorkflowCommandGuidesMCPV0(),
		Events:       coreWorkflowEventGuidesMCPV0(),
		PublicErrors: coreWorkflowPublicErrorsMCPV0(),
		Guardrails: []string{
			"resource_puro",
			"sin_db_runtime_transporte",
			"workflow_puro",
			"solo_refs_opacas",
			"detalle_en_core_workflow",
		},
	}
}

func MCPCoreWorkflowTransitionByNameV0(value string) (MCPCoreWorkflowTransitionGuideV0, bool) {
	needle := normalizeCoreWorkflowLookupMCPV0(value)
	for _, item := range append(coreWorkflowCommandGuidesMCPV0(), coreWorkflowEventGuidesMCPV0()...) {
		summaryKey := coreWorkflowTransitionSummaryKeyMCPV0(item.Kind, item.Name)
		if needle == normalizeCoreWorkflowLookupMCPV0(item.Name) ||
			needle == normalizeCoreWorkflowLookupMCPV0(item.Kind+"/"+item.Name) ||
			needle == normalizeCoreWorkflowLookupMCPV0(summaryKey) {
			item.SummaryKey = summaryKey
			return item, true
		}
	}
	return MCPCoreWorkflowTransitionGuideV0{}, false
}

func coreWorkflowPhaseIDsMCPV0() []string {
	ids := orquestacoreworkflow.SupportedOrchestrationPhaseIDsV0()
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return compactStringsMCPV0(out)
}

func coreWorkflowCommandGuidesMCPV0() []MCPCoreWorkflowTransitionGuideV0 {
	return []MCPCoreWorkflowTransitionGuideV0{
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandStartRunV0, "command", "", []string{"project_ref", "app_spec_ref"}, []string{"run"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandOpenPhaseV0, "command", "", []string{"phase_id"}, []string{"current_phase"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandClosePhaseV0, "command", "", []string{"phase_id", "closure_ref"}, []string{"phase.status=cerrada"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandBlockRunV0, "command", "", []string{"blocker_id", "reason_code"}, []string{"blockers", "status=bloqueada"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandAskDirectorV0, "command", "", []string{"question_id", "summary"}, []string{"director_questions", "outbox:director"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandAnswerDirectorQuestionV0, "command", "", []string{"answer_id", "question_id", "decision"}, []string{"director_answers", "director_answered_questions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestBrainstormV0, "command", string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0), []string{"scope", "summary"}, []string{"brainstorms"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestVoteV0, "command", string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0), []string{"vote_request_id"}, []string{"votes"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandAcceptDecisionV0, "command", string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0), []string{"vote_ref", "decision_ref"}, []string{"decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandPublishFunctionContractV0, "command", string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0), []string{"decision_ref", "contract_ref"}, []string{"function_contracts"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0, "command", string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0), []string{"task_id", "contract_ref publicado"}, []string{"tasks"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestCapacityV0, "command", "", []string{"capacity_request_id"}, []string{"capacity_requests", "outbox:capacity"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterCapacityDecisionV0, "command", "", []string{"capacity_request_id", "decision_ref"}, []string{"capacity_decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestAgentV0, "command", "", []string{"agent_request_id"}, []string{"agents", "outbox:agent_launcher"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterAgentStartedV0, "command", "", []string{"agent_request_id", "launch_ref"}, []string{"started_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterAgentFailedV0, "command", "", []string{"agent_request_id", "failure_ref"}, []string{"failed_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterAgentLeaseExpiredV0, "command", "", []string{"agent_request_id", "lease_ref"}, []string{"agent_lease_expirations"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandStopAgentV0, "command", "", []string{"agent_request_id", "stop_request_ref"}, []string{"stopped_agents", "outbox:agent_launcher"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterAgentStopConfirmedV0, "command", "", []string{"agent_request_id", "confirmation_ref"}, []string{"confirmed_stopped_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterAgentLostV0, "command", "", []string{"agent_request_id", "loss_ref"}, []string{"lost_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandAssessAgentWorkV0, "command", "", []string{"agent_request_id", "assessment_ref"}, []string{"agent_assessments"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRecordConcurrencyGateV0, "command", "", []string{"gate_ref", "decision"}, []string{"concurrency_gates"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRecordQualityGateV0, "command", "", []string{"run_ref", "gate_ref", "phase_id", "subject_ref", "decision", "summary"}, []string{"quality_gates"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterPhaseArtifactV0, "command", "", []string{"artifact_ref", "phase_id", "agent_ref"}, []string{"phase_artifacts"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterDeliveryV0, "command", string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), []string{"task_id", "agent_ref", "delivery_ref"}, []string{"deliveries"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestReviewV0, "command", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"delivery_ref", "review_request_id"}, []string{"reviews"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandAcceptReviewV0, "command", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"review_request_id", "accepted_review_ref"}, []string{"accepted_reviews"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRecordReviewResultV0, "command", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"review_request_id", "review_result_ref", "status"}, []string{"review_results"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRequestReworkV0, "command", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"rework_request_ref", "review_result_ref"}, []string{"rework_requests"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0, "command", "", []string{"replan_decision_ref", "rework_request_ref"}, []string{"replan_decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandCloseTaskV0, "command", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"task_id", "delivery_ref", "accepted_review_ref"}, []string{"closed_tasks"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandRegisterFinalValidationV0, "command", string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0), []string{"closed_task_ref", "validation_ref"}, []string{"validations"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationCommandCloseRunV0, "command", string(orquestacoreworkflow.OrchestrationPhaseCierreV0), []string{"validation_ref", "closure_ref"}, []string{"closures", "status=cerrada"}),
	}
}

func coreWorkflowEventGuidesMCPV0() []MCPCoreWorkflowTransitionGuideV0 {
	return []MCPCoreWorkflowTransitionGuideV0{
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventRunStartedV0, "event", "", []string{"project_ref", "app_spec_ref"}, []string{"run"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventPhaseOpenedV0, "event", "", []string{"phase_id"}, []string{"current_phase"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventPhaseClosedV0, "event", "", []string{"phase_id", "closure_ref"}, []string{"phase.status=cerrada"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventRunBlockedV0, "event", "", []string{"blocker_id", "reason_code"}, []string{"blockers", "status=bloqueada"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventDirectorQuestionRaisedV0, "event", "", []string{"question_id", "summary"}, []string{"director_questions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventDirectorQuestionAnsweredV0, "event", "", []string{"answer_id", "question_id", "decision"}, []string{"director_answers", "director_answered_questions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0), []string{"brainstorm_request_id"}, []string{"brainstorms"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventVoteRequestedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0), []string{"vote_request_id"}, []string{"votes"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventArchitectureDecisionAcceptedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0), []string{"decision_ref"}, []string{"decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventFunctionContractPublishedV0, "event", string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0), []string{"contract_ref"}, []string{"function_contracts"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0, "event", string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0), []string{"task_id"}, []string{"tasks"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventCapacityRequestedV0, "event", "", []string{"capacity_request_id"}, []string{"capacity_requests"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventCapacityDecidedV0, "event", "", []string{"capacity_request_id", "decision_ref"}, []string{"capacity_decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentRequestedV0, "event", "", []string{"agent_request_id"}, []string{"agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentStartedV0, "event", "", []string{"agent_request_id", "launch_ref"}, []string{"started_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentFailedV0, "event", "", []string{"agent_request_id", "failure_ref"}, []string{"failed_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentLeaseExpiredV0, "event", "", []string{"agent_request_id", "lease_ref"}, []string{"agent_lease_expirations"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentStopRequestedV0, "event", "", []string{"agent_request_id", "stop_request_ref"}, []string{"stopped_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentStopConfirmedV0, "event", "", []string{"agent_request_id", "confirmation_ref"}, []string{"confirmed_stopped_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentLostV0, "event", "", []string{"agent_request_id", "loss_ref"}, []string{"lost_agents"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventAgentWorkAssessedV0, "event", "", []string{"agent_request_id", "assessment_ref"}, []string{"agent_assessments"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventConcurrencyGateRecordedV0, "event", "", []string{"gate_ref", "decision"}, []string{"concurrency_gates"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, "event", "", []string{"run_ref", "gate_ref", "phase_id", "subject_ref", "decision", "summary"}, []string{"quality_gates"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0, "event", "", []string{"artifact_ref", "phase_id", "agent_ref"}, []string{"phase_artifacts"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, "event", string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0), []string{"delivery_ref"}, []string{"deliveries"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventReviewRequestedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"review_request_id"}, []string{"reviews"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"accepted_review_ref"}, []string{"accepted_reviews"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"review_result_ref", "status"}, []string{"review_results"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventReworkRequestedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"rework_request_ref", "review_result_ref"}, []string{"rework_requests"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, "event", "", []string{"replan_decision_ref", "rework_request_ref"}, []string{"replan_decisions"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventTaskClosedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseRevisionV0), []string{"task_id"}, []string{"closed_tasks"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0, "event", string(orquestacoreworkflow.OrchestrationPhaseValidacionFinalV0), []string{"validation_ref"}, []string{"validations"}),
		coreWorkflowTransitionMCPV0(orquestacoreworkflow.OrchestrationEventRunClosedV0, "event", string(orquestacoreworkflow.OrchestrationPhaseCierreV0), []string{"closure_ref"}, []string{"closures", "status=cerrada"}),
	}
}

func coreWorkflowTransitionMCPV0(name string, kind string, phase string, requires []string, projects []string) MCPCoreWorkflowTransitionGuideV0 {
	return MCPCoreWorkflowTransitionGuideV0{
		Name:     strings.TrimSpace(name),
		Kind:     strings.TrimSpace(kind),
		Phase:    strings.TrimSpace(phase),
		Requires: compactStringsMCPV0(requires),
		Projects: compactStringsMCPV0(projects),
	}
}

func coreWorkflowTransitionSummaryKeyMCPV0(kind string, name string) string {
	return "mcp.core_workflow." + normalizeCoreWorkflowLookupMCPV0(kind) + "." + normalizeCoreWorkflowLookupMCPV0(name) + ".summary.v0"
}

func coreWorkflowPublicErrorsMCPV0() []string {
	return compactStringsMCPV0([]string{
		orquestacoreworkflow.ErrComandoNoSoportadoV0,
		orquestacoreworkflow.ErrComandoInvalidoV0,
		orquestacoreworkflow.ErrIdempotencyKeyRequeridaV0,
		orquestacoreworkflow.ErrTransicionInvalidaV0,
		orquestacoreworkflow.ErrEventoNoSoportadoV0,
		orquestacoreworkflow.ErrEventoInvalidoV0,
		orquestacoreworkflow.ErrSecuenciaInvalidaV0,
		orquestacoreworkflow.ErrPayloadInvalidoV0,
		orquestacoreworkflow.ErrDetalleProhibidoV0,
		string(orquestacoreworkflow.OrchestrationRunInvalidoV0),
		string(orquestacoreworkflow.OrchestrationFaseNoSoportadaV0),
		string(orquestacoreworkflow.OrchestrationEstadoInconsistenteV0),
	})
}

func normalizeCoreWorkflowLookupMCPV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "/", "-")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	return normalized
}
