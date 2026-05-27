package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReceiptAckIssuesOnlyReviewableFailedTestEvidenceV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if issue.Code != orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckArtifactV0) {
			return false
		}
		field := strings.TrimSpace(issue.Field)
		if field != "tests" && field != "test_receipts" {
			return false
		}
		if !codexReceiptAckIssueHasEvidenceV0(issue,
			"failed_test_evidence",
			"required_test_receipt_not_passed",
			"required_test_receipt_exit_code_invalid",
		) {
			return false
		}
	}
	return true
}

func codexReceiptAckIssueHasEvidenceV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
	allowed ...string,
) bool {
	for _, evidence := range issue.Evidence {
		for _, item := range allowed {
			if evidence == item {
				return true
			}
		}
	}
	return false
}
