package orquestacorereplanner

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	ErrAgentFailedReplanSignalInvalidoV0        = "agent_failed_replan_signal_invalido"
	ErrAgentFailedReplanActionNoSoportadaV0     = "agent_failed_replan_action_no_soportada"
	ErrAgentFailedReplanSignalPayloadInvalidoV0 = "payload_invalido"
)

type AgentFailedReplanSignalV0 struct {
	SignalRef         string                    `json:"signal_ref"`
	RunRef            string                    `json:"run_ref"`
	TaskRef           string                    `json:"task_ref"`
	AgentRequestID    string                    `json:"agent_request_id"`
	SourceRef         string                    `json:"source_ref"`
	FailureReasonCode string                    `json:"failure_reason_code"`
	Retryable         bool                      `json:"retryable"`
	RequestedAction   ReplanRecommendedActionV0 `json:"requested_action"`
	ReplacementRole   string                    `json:"replacement_role,omitempty"`
	ReasonRef         string                    `json:"reason_ref,omitempty"`
	Summary           string                    `json:"summary"`
	EvidenceRefs      []string                  `json:"evidence_refs,omitempty"`
}

type AgentFailedReplanInputV0 struct {
	ReplanRef       string
	SignalRef       string
	RunRef          string
	TaskRef         string
	RequestedAction ReplanRecommendedActionV0
	ReplacementRole string
	ReasonRef       string
	Summary         string
	EvidenceRefs    []string
	AgentFailed     orquestacoreworkflow.AgentFailedPayloadV0
}

type AgentFailedReplanSignalErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err AgentFailedReplanSignalErrorV0) Error() string {
	return err.Code
}
