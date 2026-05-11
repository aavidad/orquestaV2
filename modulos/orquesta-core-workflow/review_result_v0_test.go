package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNewReviewResultV0AcceptsValidAccepted(t *testing.T) {
	result := validReviewResultV0()
	result.ReviewResultRef = " review-result-001 "
	result.EvidenceRefs = []string{" docs/contratos_revisiones.md#ReviewResultV0 "}

	got, err := NewReviewResultV0(result)
	if err != nil {
		t.Fatalf("NewReviewResultV0: %v", err)
	}
	if got.ReviewResultRef != "review-result-001" {
		t.Fatalf("review_result_ref=%q", got.ReviewResultRef)
	}
	if got.Status != ReviewResultStatusAcceptedV0 {
		t.Fatalf("status=%q, want %q", got.Status, ReviewResultStatusAcceptedV0)
	}
	if !ReviewResultIsAcceptedV0(got) {
		t.Fatalf("accepted review result should be recognized")
	}
}

func TestReviewResultV0ChangesRequestedAndRejectedDoNotCloseTask(t *testing.T) {
	for _, status := range []ReviewResultStatusV0{
		ReviewResultStatusChangesRequestedV0,
		ReviewResultStatusRejectedV0,
	} {
		t.Run(string(status), func(t *testing.T) {
			result := validReviewResultV0()
			result.Status = status

			got, err := NewReviewResultV0(result)
			if err != nil {
				t.Fatalf("NewReviewResultV0: %v", err)
			}
			if ReviewResultIsAcceptedV0(got) {
				t.Fatalf("status %q must not be treated as accepted", status)
			}
		})
	}
}

func TestValidateReviewResultV0RejectsForbiddenDetails(t *testing.T) {
	cases := map[string]func(*ReviewResultV0){
		"provider":   func(result *ReviewResultV0) { result.Summary = "depende de provider externo" },
		"HOME":       func(result *ReviewResultV0) { result.EvidenceRefs = []string{"$HOME/revision.txt"} },
		"OAuth":      func(result *ReviewResultV0) { result.QualityGateRef = "oauth:gate" },
		"DB":         func(result *ReviewResultV0) { result.ReviewResultRef = "db-result" },
		"prompts":    func(result *ReviewResultV0) { result.Summary = "ver prompts completos" },
		"transcript": func(result *ReviewResultV0) { result.DeliveryRef = "transcript-entrega" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			result := validReviewResultV0()
			mutate(&result)

			err := ValidateReviewResultV0(NormalizeReviewResultV0(result))
			assertReviewResultErrorCodeV0(t, err, ErrDetalleProhibidoV0)
		})
	}
}

func TestValidateReviewResultV0RejectsMassivePayload(t *testing.T) {
	result := validReviewResultV0()
	result.EvidenceRefs = nil
	for index := 0; index < maxReviewResultEvidenceRefsV0; index++ {
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-"+strings.Repeat("x", 160))
	}

	err := ValidateReviewResultV0(NormalizeReviewResultV0(result))
	assertReviewResultErrorV0(t, err, ErrReviewResultPayloadInvalidoV0, "payload")
}

func TestValidateReviewResultV0RejectsEmptyRefs(t *testing.T) {
	cases := map[string]func(*ReviewResultV0){
		"review_result_ref": func(result *ReviewResultV0) { result.ReviewResultRef = " " },
		"review_request_id": func(result *ReviewResultV0) { result.ReviewRequestID = " " },
		"delivery_ref":      func(result *ReviewResultV0) { result.DeliveryRef = " " },
		"evidence_refs":     func(result *ReviewResultV0) { result.EvidenceRefs = []string{"evidence:ok", " "} },
	}

	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			result := validReviewResultV0()
			mutate(&result)

			err := ValidateReviewResultV0(NormalizeReviewResultV0(result))
			assertReviewResultErrorV0(t, err, ErrReviewResultInvalidoV0, field)
		})
	}
}

func TestValidateReviewResultV0RejectsUnknownStatus(t *testing.T) {
	result := validReviewResultV0()
	result.Status = ReviewResultStatusV0("approved")

	err := ValidateReviewResultV0(NormalizeReviewResultV0(result))
	assertReviewResultErrorV0(t, err, ErrReviewResultStatusNoSoportadoV0, "status")
}

func TestReviewResultV0JSONHasNoProviderHomeOAuthDBPromptsOrTranscripts(t *testing.T) {
	result := mustReviewResultV0(t, validReviewResultV0())

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal review result: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("review result JSON is invalid: %s", data)
	}
	serialized := strings.ToLower(string(data))
	for _, forbidden := range []string{
		"provider", "home", "oauth", "db", "database", "prompt", "prompts", "transcript", "transcripts",
	} {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("review result JSON contains forbidden detail %q: %s", forbidden, serialized)
		}
	}
}

func validReviewResultV0() ReviewResultV0 {
	return ReviewResultV0{
		ReviewResultRef: "review-result-001",
		ReviewRequestID: "review-request-001",
		DeliveryRef:     "delivery-001",
		Status:          ReviewResultStatusAcceptedV0,
		Summary:         "Revision compacta de la entrega registrada.",
		EvidenceRefs:    []string{"docs/contratos_revisiones.md#ReviewResultV0"},
		QualityGateRef:  "quality-gate-review-001",
	}
}

func mustReviewResultV0(t *testing.T, result ReviewResultV0) ReviewResultV0 {
	t.Helper()
	normalized, err := NewReviewResultV0(result)
	if err != nil {
		t.Fatalf("NewReviewResultV0: %v", err)
	}
	return normalized
}

func assertReviewResultErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr ReviewResultErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReviewResultErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func assertReviewResultErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr ReviewResultErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected ReviewResultErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
