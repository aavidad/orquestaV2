package orquestaautoprogramming

import "testing"

func TestAutoprogrammingReviewGateDestructiveIssuesBloqueanCierreV0(t *testing.T) {
	for _, code := range []string{
		"gate-issue:removed_path:docs/manual.md",
		"gate-issue:truncated:docs/manual.md",
		"review-gate-issue:renamed_or_moved:old -> new",
	} {
		t.Run(code, func(t *testing.T) {
			if !AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code) {
				t.Fatalf("code should block closure: %s", code)
			}
			if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
				t.Fatalf("code should not be advisory: %s", code)
			}
			result := AutoprogrammingReviewGateResultFromIssuesV0(
				[]AutoprogrammingReviewGateIssueV0{{Code: code}},
			)
			assertAutoprogrammingReviewGateBlockingV0(t, result)
		})
	}
}
