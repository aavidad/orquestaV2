package orquestaautoprogramming

import "testing"

func TestAutoprogrammingReviewGateSoftRailsV0NoBloqueanCierreAceptado(t *testing.T) {
	result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{
		{Code: "gate-issue:file_too_large"},
		{Code: "review-gate-issue:file_outside_write_set:docs/extra.md"},
		{Code: "quality_gate_issue:write_set_target_missing:web"},
		{Code: "ack-pending-rail:provider"},
	})

	assertAutoprogrammingReviewGateFollowupV0(t, result)
	if !result.Accepted || !result.PreserveOutput {
		t.Fatalf("soft rails deben conservar entrega aceptada: %+v", result)
	}
}

func TestAutoprogrammingReviewGateSoftRailsV0NoOcultanBloqueoReal(t *testing.T) {
	result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{
		{Code: "gate-issue:file_too_large"},
		{Code: "gate-issue:required_test_failed"},
	})

	assertAutoprogrammingReviewGateBlockingV0(t, result)
}
