package orquestaautoprogramming

type AutoprogrammingReviewGateRecommendedActionV0 string

const (
	AutoprogrammingReviewGateActionAcceptV0                AutoprogrammingReviewGateRecommendedActionV0 = "accept"
	AutoprogrammingReviewGateActionRequestFollowupReviewV0 AutoprogrammingReviewGateRecommendedActionV0 = "request_followup_review"
	AutoprogrammingReviewGateActionBlockClosureV0          AutoprogrammingReviewGateRecommendedActionV0 = "block_closure"
	AutoprogrammingReviewGateActionRequestChangesV0        AutoprogrammingReviewGateRecommendedActionV0 = "request_changes"
)

func autoprogrammingReviewGateResultV0(
	issues []AutoprogrammingReviewGateIssueV0,
) AutoprogrammingReviewGateResultV0 {
	issues = append([]AutoprogrammingReviewGateIssueV0(nil), issues...)
	accepted := !autoprogrammingReviewGateHasBlockingIssueV0(issues)
	requiresFollowup := autoprogrammingReviewGateRequiresFollowupV0(issues)

	return AutoprogrammingReviewGateResultV0{
		Accepted:          accepted,
		PreserveOutput:    accepted || requiresFollowup,
		RequiresFollowup:  requiresFollowup,
		RecommendedAction: autoprogrammingReviewGateRecommendedActionV0(accepted, requiresFollowup, issues),
		Issues:            issues,
	}
}

func AutoprogrammingReviewGateResultFromIssuesV0(
	issues []AutoprogrammingReviewGateIssueV0,
) AutoprogrammingReviewGateResultV0 {
	return autoprogrammingReviewGateResultV0(issues)
}

func autoprogrammingReviewGateRequiresFollowupV0(
	issues []AutoprogrammingReviewGateIssueV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if !autoprogrammingReviewGateIssueIsAdvisoryV0(issue) {
			return false
		}
	}
	return true
}

func autoprogrammingReviewGateRecommendedActionV0(
	accepted bool,
	requiresFollowup bool,
	issues []AutoprogrammingReviewGateIssueV0,
) AutoprogrammingReviewGateRecommendedActionV0 {
	if requiresFollowup {
		return AutoprogrammingReviewGateActionRequestFollowupReviewV0
	}
	if accepted {
		return AutoprogrammingReviewGateActionAcceptV0
	}
	if autoprogrammingReviewGateBlocksClosureV0(issues) {
		return AutoprogrammingReviewGateActionBlockClosureV0
	}
	return AutoprogrammingReviewGateActionRequestChangesV0
}

func autoprogrammingReviewGateHasBlockingIssueV0(
	issues []AutoprogrammingReviewGateIssueV0,
) bool {
	for _, issue := range issues {
		if !autoprogrammingReviewGateIssueIsAdvisoryV0(issue) {
			return true
		}
	}
	return false
}

func autoprogrammingReviewGateIssueIsAdvisoryV0(issue AutoprogrammingReviewGateIssueV0) bool {
	return AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(issue.Code)
}

func AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(code string) bool {
	if AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code) {
		return false
	}
	for _, stem := range autoprogrammingReviewGateIssueCodeStemsV0(code) {
		if stem == "file_too_large" ||
			stem == "file_outside_write_set" ||
			stem == "write_set_target_missing" ||
			stem == "ack_pending_rail" ||
			stem == "ack_files_mismatch" ||
			stem == "go_file_line_budget_exceeded" ||
			stem == "replaced_large_delta" ||
			stem == "worktree_snapshot_file_too_large" ||
			stem == "worktree_snapshot_too_many_files" ||
			stem == "worktree_snapshot_too_large" ||
			stem == "worktree_snapshot_unreadable" {
			return true
		}
	}
	return false
}

func autoprogrammingReviewGateBlocksClosureV0(
	issues []AutoprogrammingReviewGateIssueV0,
) bool {
	for _, issue := range issues {
		if AutoprogrammingReviewGateIssueCodeBlocksClosureV0(issue.Code) {
			return true
		}
	}
	return false
}
