package orquestacoreworkflow

const (
	AgentAssessmentVerdictAcceptableV0      = "acceptable"
	AgentAssessmentVerdictNeedsRevisionV0   = "needs_revision"
	AgentAssessmentVerdictGarbageV0         = "garbage"
	AgentAssessmentVerdictLoopDetectedV0    = "loop_detected"
	AgentAssessmentVerdictCapacityLimitedV0 = "capacity_limited"
	AgentAssessmentVerdictTimeoutV0         = "timeout"

	AgentAssessmentActionContinueV0        = "continue"
	AgentAssessmentActionRequestRevisionV0 = "request_revision"
	AgentAssessmentActionStopAgentV0       = "stop_agent"
	AgentAssessmentActionAskDirectorV0     = "ask_director"

	AgentAssessmentSeverityLowV0      = "low"
	AgentAssessmentSeverityMediumV0   = "medium"
	AgentAssessmentSeverityHighV0     = "high"
	AgentAssessmentSeverityCriticalV0 = "critical"
)

type AssessAgentWorkCommandPayloadV0 struct {
	AssessmentRef  string   `json:"assessment_ref"`
	PhaseID        string   `json:"phase_id"`
	AgentRequestID string   `json:"agent_request_id"`
	TaskRef        string   `json:"task_ref,omitempty"`
	DeliveryRef    string   `json:"delivery_ref,omitempty"`
	Verdict        string   `json:"verdict"`
	Action         string   `json:"action"`
	Severity       string   `json:"severity"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentWorkAssessedPayloadV0 struct {
	AssessmentRef  string   `json:"assessment_ref"`
	PhaseID        string   `json:"phase_id"`
	AgentRequestID string   `json:"agent_request_id"`
	TaskRef        string   `json:"task_ref,omitempty"`
	DeliveryRef    string   `json:"delivery_ref,omitempty"`
	Verdict        string   `json:"verdict"`
	Action         string   `json:"action"`
	Severity       string   `json:"severity"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

func NewAssessAgentWorkCommandV0(meta OrchestrationCommandMetaV0, payload AssessAgentWorkCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandAssessAgentWorkV0, normalizeAssessAgentWorkPayloadV0(payload))
}

func NewAgentWorkAssessedEventV0(meta OrchestrationEventMetaV0, payload AgentWorkAssessedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentWorkAssessedV0, normalizeAgentWorkAssessedPayloadV0(payload))
}
