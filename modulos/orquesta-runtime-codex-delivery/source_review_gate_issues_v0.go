package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReviewGateIssuesEvaluableV0(
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	for _, issue := range issues {
		if string(issue.Code) != string(orquestaruntimecodex.CodexConnectorAckArtifactV0) {
			return false
		}
	}
	return true
}

func codexReviewGateMergeConnectorIssuesV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) orquestaautoprogramming.AutoprogrammingReviewGateResultV0 {
	if len(issues) == 0 {
		return result
	}
	for _, issue := range issues {
		for _, code := range issue.Evidence {
			code = strings.TrimSpace(code)
			if code != "" && !codexReviewGateResultHasIssueV0(result, code) {
				result.Issues = append(result.Issues, orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
					Code:  code,
					Field: strings.TrimSpace(issue.Field),
				})
			}
		}
	}
	return orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(result.Issues)
}

func codexReviewGateMergePendingRailEvidenceV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) orquestaautoprogramming.AutoprogrammingReviewGateResultV0 {
	for _, ref := range orquestaruntimecodex.CodexAgentAckPendingRailEvidenceRefsV0(ack) {
		ref = strings.TrimSpace(ref)
		if ref == "" || codexReviewGateResultHasIssueV0(result, ref) {
			continue
		}
		result.Issues = append(result.Issues, orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
			Code:  ref,
			Field: "agent_ack",
		})
	}
	return orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(result.Issues)
}

func codexReviewGateMergeIncompleteRequiredEvidenceV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) orquestaautoprogramming.AutoprogrammingReviewGateResultV0 {
	if !orquestaruntimecodex.CodexAgentAckDeclaresIncompleteRequiredEvidenceV0(ack) ||
		codexReviewGateResultHasIssueV0(result, "required_test_not_executed") {
		return result
	}
	result.Issues = append(result.Issues, orquestaautoprogramming.AutoprogrammingReviewGateIssueV0{
		Code:    "required_test_not_executed",
		Field:   "agent_ack",
		Message: "el ACK declara evidencia obligatoria no ejecutada o incompleta",
	})
	return orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(result.Issues)
}

func codexReviewGateMergeGateIssuesV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
	issues []orquestaautoprogramming.AutoprogrammingReviewGateIssueV0,
) orquestaautoprogramming.AutoprogrammingReviewGateResultV0 {
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code != "" && !codexReviewGateResultHasIssueV0(result, code) {
			result.Issues = append(result.Issues, issue)
		}
	}
	return orquestaautoprogramming.AutoprogrammingReviewGateResultFromIssuesV0(result.Issues)
}

func codexReviewGateResultHasIssueV0(
	result orquestaautoprogramming.AutoprogrammingReviewGateResultV0,
	code string,
) bool {
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}
