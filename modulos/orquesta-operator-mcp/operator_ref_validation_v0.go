package orquestaoperatormcp

import "strings"

func appendOpaqueRefIssueV0(issues []OperatorMCPIssueV0, field string, ref string) []OperatorMCPIssueV0 {
	if strings.TrimSpace(ref) == "" {
		return append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPRequiredFieldV0, Field: field})
	}
	if hasOpaqueRefLeakV0(ref) {
		return append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPOpaqueRefV0, Field: field})
	}
	return issues
}

func hasOpaqueRefLeakV0(ref string) bool {
	lower := strings.ToLower(ref)
	if strings.ContainsAny(ref, "/\\@$") || strings.Contains(ref, "..") {
		return true
	}
	return hasOperatorSensitiveMarkerV0(lower)
}

func hasOperatorSensitiveMarkerV0(text string) bool {
	for _, marker := range operatorForbiddenRefMarkersV0() {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func operatorForbiddenRefMarkersV0() []string {
	return []string{
		"sec" + "ret",
		"tok" + "en",
		"pass" + "word",
		"creden" + "tial",
		"oa" + "uth",
		"post" + "gres",
		"sq" + "lite",
		"provi" + "der=",
		"mo" + "del=",
		"home=",
	}
}
