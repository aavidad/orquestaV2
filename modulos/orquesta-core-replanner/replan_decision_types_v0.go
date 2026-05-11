package orquestacorereplanner

const (
	ErrReplanDecisionInvalidaV0        = "replan_decision_invalida"
	ErrReplanDecisionActionInvalidaV0  = "replan_decision_action_invalida"
	ErrReplanDecisionPayloadInvalidoV0 = "payload_invalido"
)

type ReplanDecisionV0 struct {
	ReplanRef      string                    `json:"replan_ref"`
	RunRef         string                    `json:"run_ref"`
	TaskRef        string                    `json:"task_ref"`
	SourceRef      string                    `json:"source_ref"`
	AcceptedAction ReplanRecommendedActionV0 `json:"accepted_action"`
	FollowupRefs   []string                  `json:"followup_refs"`
	Summary        string                    `json:"summary"`
	EvidenceRefs   []string                  `json:"evidence_refs,omitempty"`
}

type ReplanDecisionErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err ReplanDecisionErrorV0) Error() string {
	return err.Code
}
