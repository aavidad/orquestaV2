package orquestaautoprogramming

import "strings"

func AutoprogrammingReviewGateIssueCodeBlocksClosureV0(code string) bool {
	for _, stem := range autoprogrammingReviewGateIssueCodeStemsV0(code) {
		switch stem {
		case "ack_missing",
			"ack_not_completed",
			"go_file_line_budget_strict_blocking",
			"removed_path",
			"truncated_path",
			"renamed_or_moved_path",
			"required_test_missing",
			"required_test_not_executed",
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
