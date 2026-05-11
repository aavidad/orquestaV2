package orquestacorereplanner

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	ErrReviewReworkSignalInvalidoV0        = "review_rework_signal_invalido"
	ErrReviewReworkStatusNoSoportadoV0     = "review_rework_status_no_soportado"
	ErrReviewReworkActionNoSoportadaV0     = "review_rework_action_no_soportada"
	ErrReviewReworkSignalPayloadInvalidoV0 = "payload_invalido"
)

type ReviewReworkSignalV0 struct {
	SignalRef        string                                    `json:"signal_ref"`
	RunRef           string                                    `json:"run_ref"`
	TaskRef          string                                    `json:"task_ref"`
	ReviewRequestRef string                                    `json:"review_request_ref"`
	DeliveryRef      string                                    `json:"delivery_ref"`
	ReviewResultRef  string                                    `json:"review_result_ref"`
	ReviewStatus     orquestacoreworkflow.ReviewResultStatusV0 `json:"review_status"`
	RequestedAction  ReplanRecommendedActionV0                 `json:"requested_action"`
	ReasonRef        string                                    `json:"reason_ref,omitempty"`
	Summary          string                                    `json:"summary"`
	EvidenceRefs     []string                                  `json:"evidence_refs,omitempty"`
}

type ReviewResultReplanInputV0 struct {
	ReplanRef       string
	SignalRef       string
	RunRef          string
	TaskRef         string
	RequestedAction ReplanRecommendedActionV0
	ReasonRef       string
	Summary         string
	EvidenceRefs    []string
	ReviewResult    orquestacoreworkflow.ReviewResultV0
}

type ReviewReworkSignalErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err ReviewReworkSignalErrorV0) Error() string {
	return err.Code
}
