package orquestacoreworkflow

type ReviewResultStatusV0 string

const (
	ReviewResultStatusAcceptedV0         ReviewResultStatusV0 = "accepted"
	ReviewResultStatusChangesRequestedV0 ReviewResultStatusV0 = "changes_requested"
	ReviewResultStatusRejectedV0         ReviewResultStatusV0 = "rejected"

	ErrReviewResultInvalidoV0          = "review_result_invalido"
	ErrReviewResultStatusNoSoportadoV0 = "status_no_soportado"
	ErrReviewResultPayloadInvalidoV0   = "payload_invalido"
)

type ReviewResultV0 struct {
	ReviewResultRef string               `json:"review_result_ref"`
	ReviewRequestID string               `json:"review_request_id"`
	DeliveryRef     string               `json:"delivery_ref"`
	Status          ReviewResultStatusV0 `json:"status"`
	Summary         string               `json:"summary"`
	EvidenceRefs    []string             `json:"evidence_refs,omitempty"`
	QualityGateRef  string               `json:"quality_gate_ref,omitempty"`
}

type ReviewResultErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err ReviewResultErrorV0) Error() string {
	return err.Code
}
