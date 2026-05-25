package orquestaautoprogramming

import "testing"

func TestAutoprogrammingReviewGateIssueCodeIsAdvisoryV0(t *testing.T) {
	for _, code := range []string{
		"file_too_large",
		"file-too-large",
		"too_large_file",
		"file_too_large:modulos/orquesta-autoprogramming/gate.go",
		"file_outside_write_set",
		"file-outside-write-set",
		"file_outside_write_set:docs/extra.md",
		"artifact_outside_write_set:docs/extra.md",
		"artifact-path-outside-write-set:docs/extra.md",
		"path_outside_write_set:docs/extra.md",
		"outside_write_set",
		"outside-write-set:docs/extra.md",
		"outside_write_set:docs/extra.md",
		"write_set_target_missing",
		"write-set-target-missing:web",
		"missing_write_set_target:web",
		"write_set_destination_missing:web",
		"target_missing_from_write_set:web",
		"write_set_target_missing:web",
		" Write_Set_Target_Missing:WEB ",
		"line_limit_exceeded:README.md",
		"too_many_lines:README.md",
		"gate-issue:file_too_large",
		"gate_issue:file_line_limit_exceeded:README.md",
		"review-gate-issue:outside_of_write_set:docs/extra.md",
		"quality_gate_issue:write_set_path_missing:web",
		"gate-issue:write_set_target_missing:web",
		"file_too_large#policy_blocked",
		"outside-write-set:docs/extra.md#blocked",
		"write-set-target-missing:web#policy-blocked",
		"ack-pending-rail:token#blocked",
		"gate-issue:file_too_large#policy_blocked",
		"ack-pending-rail:token",
		"ack_pending_rail:token",
		"pending_rail:token",
		"rail_pending:token",
		"pending_ack_rail:token",
		"soft_rail:provider",
		"rail_blando:provider",
		"rail-pendiente:provider",
		"ack-pending-rail:operational-detail-marker",
		"ack_files_mismatch",
		"gate-issue:ack-files-mismatch:ack_files_do_not_match_snapshot_changes",
		"go_file_line_budget_exceeded",
		"gate-issue:go-file-line-budget-exceeded:modulos/orquesta-mcp/http.go",
	} {
		if !AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
			t.Fatalf("code=%q advisory=false", code)
		}
	}
	for _, code := range []string{
		"required_test_missing",
		"gate-issue:required_test_missing",
		"ack_missing",
		"delivery_file_missing",
	} {
		if AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code) {
			t.Fatalf("code=%q advisory=true", code)
		}
	}
}

func TestAutoprogrammingReviewGateAdvisoryIssuesPidenFollowupSinBloquearV0(t *testing.T) {
	result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{
		{Code: "file_too_large"},
		{Code: "outside_write_set"},
		{Code: "write_set_target_missing"},
		{Code: "write_set_target_missing:web"},
		{Code: "gate-issue:ack_files_mismatch"},
		{Code: "gate-issue:go_file_line_budget_exceeded"},
	})

	assertAutoprogrammingReviewGateFollowupV0(t, result)
}

func TestAutoprogrammingReviewGatePrefixedHardIssueSigueBloqueandoV0(t *testing.T) {
	for _, code := range []string{
		"gate-issue:required_test_missing",
		"gate-issue:missing_required_test",
		"required-test-missing",
		"gate-issue:required_test_missing:go test ./...",
		"gate-issue:ack_missing",
		"gate-issue:ack-not-completed",
		"failed_required_test:go test ./...",
		"failed-test:go test ./...",
		"required_test_failed:go test ./...",
		"test_failed:go test ./...",
	} {
		if !AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code) {
			t.Fatalf("code=%q blocks_closure=false", code)
		}
		result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{
			{Code: code},
		})

		assertAutoprogrammingReviewGateBlockingV0(t, result)
	}
}
