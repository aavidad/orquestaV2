package orquestadirector

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	ErrDirectorPostLeaseActionInvalidaV0 = "director_post_lease_action_invalida"

	postLeaseActionSourceGroupV0 = "orquesta-director"

	PostLeaseFollowupStatusStopAgentV0                = "stop_agent"
	PostLeaseFollowupStatusAskDirectorV0              = "ask_director"
	PostLeaseFollowupStatusUnsupportedNeedsDirectorV0 = "unsupported/needs_director"
)

type PostLeaseActionInputV0 struct {
	CommandMeta       orquestacoreworkflow.OrchestrationCommandMetaV0    `json:"command_meta"`
	RunRef            string                                             `json:"run_ref"`
	AgentRequestID    string                                             `json:"agent_request_id"`
	LeaseRef          string                                             `json:"lease_ref"`
	ReasonCode        string                                             `json:"reason_code"`
	ObservedAt        string                                             `json:"observed_at"`
	RecommendedAction orquestacoreworkflow.AgentLeaseRecommendedActionV0 `json:"recommended_action"`
	EvidenceRefs      []string                                           `json:"evidence_refs,omitempty"`
	QuestionID        string                                             `json:"question_id,omitempty"`
}

type PostLeaseActionResultV0 struct {
	RegisterLeaseExpiredCommand orquestacoreworkflow.OrchestrationCommandV0  `json:"register_lease_expired_command"`
	StopAgentCommand            *orquestacoreworkflow.OrchestrationCommandV0 `json:"stop_agent_command,omitempty"`
	AskDirectorCommand          *orquestacoreworkflow.OrchestrationCommandV0 `json:"ask_director_command,omitempty"`
	FollowupStatus              string                                       `json:"followup_status"`
}

type PostLeaseActionErrorV0 struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Field         string   `json:"field,omitempty"`
	Retryable     bool     `json:"retryable"`
	Evidence      []string `json:"evidence,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
}

func (err PostLeaseActionErrorV0) Error() string {
	return err.Code
}
