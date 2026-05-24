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
			stem == "ack_pending_rail" {
			return true
		}
	}
	return false
}

func AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code string) bool {
	for _, stem := range autoprogrammingReviewGateIssueCodeStemsV0(code) {
		switch stem {
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

func autoprogrammingReviewGateNormalizeIssueCodeV0(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	for {
		next, ok := autoprogrammingReviewGateTrimIssuePrefixV0(code)
		if ok {
			code = strings.TrimSpace(next)
			continue
		}
		return code
	}
}

func autoprogrammingReviewGateTrimIssuePrefixV0(code string) (string, bool) {
	for _, prefix := range []string{
		"gate-issue:",
		"gate_issue:",
		"review-gate-issue:",
		"review_gate_issue:",
		"quality-gate-issue:",
		"quality_gate_issue:",
	} {
		if next, ok := strings.CutPrefix(code, prefix); ok {
			return next, true
		}
	}
	return "", false
}

func autoprogrammingReviewGateIssueCodeStemV0(code string) string {
	code = autoprogrammingReviewGateNormalizeIssueCodeV0(code)
	stem := autoprogrammingReviewGateIssueCodeHeadV0(code)
	stem = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(stem)
	stem = strings.Join(strings.FieldsFunc(stem, func(r rune) bool { return r == '_' }), "_")
	return autoprogrammingReviewGateCanonicalStemV0(stem)
}

func autoprogrammingReviewGateIssueCodeHeadV0(code string) string {
	head := strings.TrimSpace(code)
	for _, separator := range []string{":", "#", "|", ";"} {
		if value, _, ok := strings.Cut(head, separator); ok {
			head = value
		}
	}
	return strings.TrimSpace(head)
}

func autoprogrammingReviewGateCanonicalStemV0(stem string) string {
	switch stem {
	case "outside_write_set",
		"outside_writeset",
		"outside_of_write_set",
		"out_of_write_set",
		"outside_allowed_write_set",
		"outside_write_scope",
		"outside_scope",
		"file_outside_writeset",
		"files_outside_write_set",
		"file_outside_allowed_write_set",
		"file_outside_scope",
		"artifact_outside_write_set",
		"artifact_outside_writeset",
		"artifact_out_of_write_set",
		"artifact_path_outside_write_set",
		"artifact_path_out_of_write_set",
		"path_outside_scope",
		"path_outside_write_set":
		return "file_outside_write_set"
	case "missing_write_set_target",
		"write_set_missing_target",
		"write_set_target_absent",
		"missing_write_set_destination",
		"write_set_destination_missing",
		"missing_write_set_path",
		"write_set_path_missing",
		"missing_write_set_entry",
		"write_set_entry_missing",
		"missing_target",
		"target_missing",
		"target_missing_from_write_set":
		return "write_set_target_missing"
	case "too_large_file",
		"large_file",
		"file_large",
		"file_too_big",
		"file_too_long",
		"file_exceeds_limit",
		"file_size_over_limit",
		"file_over_line_limit",
		"file_line_limit_exceeded",
		"line_limit_exceeded",
		"max_lines_exceeded",
		"max_line_limit_exceeded",
		"line_count_exceeded",
		"too_many_lines",
		"oversized_file":
		return "file_too_large"
	case "ack_pending_rail",
		"pending_rail",
		"rail_pendiente",
		"pending_rail_ack",
		"rail_pending",
		"rail_pending_ack",
		"ack_rail_pending",
		"pending_ack_rail",
		"pending_sensitive_rail",
		"sensitive_rail_pending",
		"soft_rail",
		"rail_blando",
		"rail_dudoso",
		"doubtful_rail",
		"suspect_rail":
		return "ack_pending_rail"
	case "missing_required_test":
		return "required_test_missing"
	case "failed_required_test":
		return "required_test_failed"
	case "failed_test":
		return "test_failed"
	case "ack_absent":
		return "ack_missing"
	case "ack_not_complete":
		return "ack_not_completed"
	default:
		return stem
	}
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
