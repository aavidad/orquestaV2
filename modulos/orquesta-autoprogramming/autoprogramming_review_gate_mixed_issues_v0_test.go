package orquestaautoprogramming

import "testing"

func TestAutoprogrammingReviewGateMixedSoftRailNoOcultaBloqueoV0(t *testing.T) {
	for _, code := range []string{
		"gate-issue:file_too_large#gate-issue:ack_missing#blocked",
		"file_outside_write_set:docs/extra.md|required_test_failed:go test ./...",
		"ack-pending-rail:token;gate-issue:required_test_missing",
	} {
		if !AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code) {
			t.Fatalf("code=%q blocks_closure=false", code)
		}
		if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
			t.Fatalf("code=%q advisory=true", code)
		}
		result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{{Code: code}})
		assertAutoprogrammingReviewGateBlockingV0(t, result)
	}
}

func TestAutoprogrammingReviewGateSoftRailSuffixCompatSigueAdvisoryV0(t *testing.T) {
	for _, code := range []string{
		"file_too_large#policy_blocked",
		"outside-write-set:docs/extra.md#blocked",
		"ack-pending-rail:token#blocked",
		"artifact_paths_outside_write_set:docs/extra.md#blocked",
		"destino-write-set-faltante:web#policy_blocked",
		"fichero-demasiado-grande:README.md#blocked",
		"detector-dudoso:provider#blocked",
	} {
		if !AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
			t.Fatalf("code=%q advisory=false", code)
		}
	}
}

func TestAutoprogrammingReviewGateSoftRailAliasNoOcultaBloqueoV0(t *testing.T) {
	code := "fichero-demasiado-grande:README.md#gate-issue:required-test-failed"
	if !AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code) {
		t.Fatalf("code=%q blocks_closure=false", code)
	}
	if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
		t.Fatalf("code=%q advisory=true", code)
	}
}
