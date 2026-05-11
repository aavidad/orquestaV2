package orquestacorereplanner

type ReplanRecommendedActionV0 string

const (
	ReplanActionSplitTaskV0        ReplanRecommendedActionV0 = "split_task"
	ReplanActionRetryTaskV0        ReplanRecommendedActionV0 = "retry_task"
	ReplanActionReplaceAgentV0     ReplanRecommendedActionV0 = "replace_agent"
	ReplanActionEscalateCapacityV0 ReplanRecommendedActionV0 = "escalate_capacity"
	ReplanActionAskDirectorV0      ReplanRecommendedActionV0 = "ask_director"
	ReplanActionAbortTaskV0        ReplanRecommendedActionV0 = "abort_task"

	ErrReplanProposalInvalidoV0        = "replan_proposal_invalido"
	ErrReplanActionNoSoportadaV0       = "replan_action_no_soportada"
	ErrReplanProposalPayloadInvalidoV0 = "payload_invalido"
	ErrDetalleProhibidoV0              = "detalle_prohibido"
)

type ReplanProposalV0 struct {
	ReplanRef          string                    `json:"replan_ref"`
	RunRef             string                    `json:"run_ref"`
	TaskRef            string                    `json:"task_ref"`
	SourceRef          string                    `json:"source_ref"`
	ReasonCode         string                    `json:"reason_code"`
	RecommendedAction  ReplanRecommendedActionV0 `json:"recommended_action"`
	CapacityRequestRef string                    `json:"capacity_request_ref,omitempty"`
	ReplacementRole    string                    `json:"replacement_role,omitempty"`
	Summary            string                    `json:"summary"`
	EvidenceRefs       []string                  `json:"evidence_refs,omitempty"`
}

type ReplanProposalErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err ReplanProposalErrorV0) Error() string {
	return err.Code
}
