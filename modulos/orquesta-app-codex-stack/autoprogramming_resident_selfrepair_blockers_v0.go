package orquestaappcodexstack

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func autoprogrammingResidentEvidenceIsBlockingV0(ref string) bool {
	value := strings.ToLower(strings.TrimSpace(ref))
	if value == "" {
		return false
	}
	if autoprogrammingResidentEvidenceHasHardFailureV0(value) {
		return true
	}
	if autoprogrammingResidentEvidenceHasBlockingGateIssueV0(value) {
		return true
	}
	if autoprogrammingResidentEvidenceIsSoftRailV0(value) {
		return false
	}
	for _, token := range strings.FieldsFunc(value, autoprogrammingResidentEvidenceTokenSeparatorV0) {
		if autoprogrammingResidentEvidenceTokenIsBlockingV0(token) {
			return true
		}
	}
	return false
}

func autoprogrammingResidentEvidenceHasHardFailureV0(value string) bool {
	for _, marker := range []string{
		"required-tests-failed",
		"required-tests-evidence-missing",
		"required-test-failed",
		"required-test-missing",
		"required_test_failed",
		"required_test_missing",
		"test_failed",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func autoprogrammingResidentEvidenceIsSoftRailV0(value string) bool {
	if strings.Contains(value, "gate-followup-required") ||
		strings.Contains(value, "request_followup_review") {
		return true
	}
	hasAdvisory := false
	for _, candidate := range autoprogrammingResidentSoftRailCandidatesV0(value) {
		if orquestaautoprogramming.AutoprogrammingReviewGateIssueCodeBlocksClosureV0(candidate) {
			return false
		}
		if orquestaautoprogramming.AutoprogrammingReviewGateIssueCodeIsAdvisoryV0(candidate) {
			hasAdvisory = true
		}
	}
	return hasAdvisory
}

func autoprogrammingResidentEvidenceHasBlockingGateIssueV0(value string) bool {
	for _, candidate := range autoprogrammingResidentSoftRailCandidatesV0(value) {
		if orquestaautoprogramming.AutoprogrammingReviewGateIssueCodeBlocksClosureV0(candidate) {
			return true
		}
	}
	return false
}

func autoprogrammingResidentSoftRailCandidatesV0(value string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	candidates := []string{value}
	candidates = append(candidates, autoprogrammingResidentSoftRailSegmentsV0(value)...)
	for _, marker := range []string{
		"gate-issue:",
		"gate_issue:",
		"review-gate-issue:",
		"review_gate_issue:",
		"quality-gate-issue:",
		"quality_gate_issue:",
		"ack-pending-rail:",
		"ack_pending_rail:",
		"pending_rail:",
		"soft_rail:",
		"rail_blando:",
	} {
		if index := strings.Index(value, marker); index >= 0 {
			candidates = append(candidates, value[index:])
		}
	}
	return compactStringsV0(candidates)
}

func autoprogrammingResidentSoftRailSegmentsV0(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '#' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

func autoprogrammingResidentEvidenceTokenSeparatorV0(r rune) bool {
	return r == '#' || r == ':' || r == '/' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func autoprogrammingResidentEvidenceTokenIsBlockingV0(token string) bool {
	token = strings.Trim(strings.TrimSpace(token), ".,;")
	if token == "" || token == "not-blocked" || token == "non-blocked" ||
		token == "not_blocked" || token == "non_blocked" || token == "unblocked" {
		return false
	}
	return token == "blocked" || token == "policy_blocked" || token == "policy-blocked" ||
		strings.HasSuffix(token, "_blocked") || strings.HasSuffix(token, "-blocked")
}
