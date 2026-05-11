package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
	result orquestacionnucleoapp.AutoprogrammingReviewGateResultV0,
	issues []orquestaruntime.ExternalAgentConnectorErrorV0,
) orquestacionnucleoapp.AutoprogrammingReviewGateResultV0 {
	if len(issues) == 0 {
		return result
	}
	for _, issue := range issues {
		for _, code := range issue.Evidence {
			code = strings.TrimSpace(code)
			if code != "" && !codexReviewGateResultHasIssueV0(result, code) {
				result.Issues = append(result.Issues, orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0{
					Code:  code,
					Field: strings.TrimSpace(issue.Field),
				})
			}
		}
	}
	result.Accepted = len(result.Issues) == 0
	return result
}

func codexReviewGateMergeGateIssuesV0(
	result orquestacionnucleoapp.AutoprogrammingReviewGateResultV0,
	issues []orquestacionnucleoapp.AutoprogrammingReviewGateIssueV0,
) orquestacionnucleoapp.AutoprogrammingReviewGateResultV0 {
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code != "" && !codexReviewGateResultHasIssueV0(result, code) {
			result.Issues = append(result.Issues, issue)
		}
	}
	result.Accepted = len(result.Issues) == 0
	return result
}

func codexReviewGateResultHasIssueV0(
	result orquestacionnucleoapp.AutoprogrammingReviewGateResultV0,
	code string,
) bool {
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}
