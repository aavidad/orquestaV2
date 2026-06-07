package orquestaoperatormcp

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

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
	return orquestarails.TextContainsOperationalSensitiveDetailV0(text)
}
