package orquestaautoprogramming

import "strings"

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
	accepted := len(issues) == 0
	requiresFollowup := autoprogrammingReviewGateRequiresFollowupV0(issues)

	return AutoprogrammingReviewGateResultV0{
		Accepted:          accepted,
		PreserveOutput:    accepted || requiresFollowup,
		RequiresFollowup:  requiresFollowup,
		RecommendedAction: autoprogrammingReviewGateRecommendedActionV0(accepted, requiresFollowup, issues),
		Issues:            issues,
	}
}

func autoprogrammingReviewGateRequiresFollowupV0(
	issues []AutoprogrammingReviewGateIssueV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) != "file_outside_write_set" {
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
	if accepted {
		return AutoprogrammingReviewGateActionAcceptV0
	}
	if requiresFollowup {
		return AutoprogrammingReviewGateActionRequestFollowupReviewV0
	}
	if autoprogrammingReviewGateBlocksClosureV0(issues) {
		return AutoprogrammingReviewGateActionBlockClosureV0
	}
	return AutoprogrammingReviewGateActionRequestChangesV0
}

func autoprogrammingReviewGateBlocksClosureV0(
	issues []AutoprogrammingReviewGateIssueV0,
) bool {
	for _, issue := range issues {
		switch strings.TrimSpace(issue.Code) {
		case "ack_missing",
			"ack_not_completed",
			"required_test_missing",
			"required_test_failed",
			"test_failed":
			return true
		}
	}
	return false
}
