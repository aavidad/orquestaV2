package orquestaautoprogramming

import "strings"

func autoprogrammingReviewGateIssueCodeStemsV0(code string) []string {
	candidates := []string{strings.TrimSpace(code)}
	for _, part := range strings.FieldsFunc(code, autoprogrammingReviewGateIssueCodeFragmentSeparatorV0) {
		candidates = append(candidates, strings.TrimSpace(part))
	}
	stems := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		stem := autoprogrammingReviewGateIssueCodeStemV0(candidate)
		if stem != "" && !autoprogrammingReviewGateStringInSetV0(stems, stem) {
			stems = append(stems, stem)
		}
	}
	return stems
}

func autoprogrammingReviewGateIssueCodeFragmentSeparatorV0(r rune) bool {
	return r == '#' || r == '|' || r == ';'
}

func autoprogrammingReviewGateStringInSetV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
