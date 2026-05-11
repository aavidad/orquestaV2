package orquestacorereplanner

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	ErrAgentReworkSignalInvalidoV0        = "agent_rework_signal_invalido"
	ErrAgentReworkVerdictNoSoportadoV0    = "agent_rework_verdict_no_soportado"
	ErrAgentReworkActionNoSoportadaV0     = "agent_rework_action_no_soportada"
	ErrAgentReworkSignalPayloadInvalidoV0 = "payload_invalido"
)

type AgentReworkSignalV0 struct {
	SignalRef        string                    `json:"signal_ref"`
	RunRef           string                    `json:"run_ref"`
	TaskRef          string                    `json:"task_ref"`
	AgentRequestID   string                    `json:"agent_request_id"`
	SourceRef        string                    `json:"source_ref"`
	AssessmentStatus string                    `json:"assessment_status"`
	AssessmentAction string                    `json:"assessment_action"`
	RequestedAction  ReplanRecommendedActionV0 `json:"requested_action"`
	ReplacementRole  string                    `json:"replacement_role,omitempty"`
	ReasonRef        string                    `json:"reason_ref,omitempty"`
	Summary          string                    `json:"summary"`
	EvidenceRefs     []string                  `json:"evidence_refs,omitempty"`
}

type AgentWorkAssessmentReplanInputV0 struct {
	ReplanRef       string
	SignalRef       string
	RunRef          string
	TaskRef         string
	RequestedAction ReplanRecommendedActionV0
	ReplacementRole string
	ReasonRef       string
	Summary         string
	EvidenceRefs    []string
	Assessment      orquestacoreworkflow.AgentWorkAssessedPayloadV0
}

type AgentReworkSignalErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err AgentReworkSignalErrorV0) Error() string {
	return err.Code
}
