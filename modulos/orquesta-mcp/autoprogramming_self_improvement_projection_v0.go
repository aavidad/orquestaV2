package orquestamcp

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func selfImprovementIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}

func withMCPAutoprogrammingOperatorAdviceV0(
	proposal orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0,
	advice []MCPAutoprogrammingOperatorAdviceV0,
) orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 {
	for _, item := range advice {
		for _, ref := range compactStringsMCPV0([]string{item.AdviceRef, item.OperatorRef, item.TargetRef, item.RunRef, item.TaskRef}) {
			proposal.ContextRefs = append(proposal.ContextRefs, "operator_advice_ref:"+ref)
		}
		for _, ref := range item.EvidenceRefs {
			proposal.ContextRefs = append(proposal.ContextRefs, "operator_advice_evidence_ref:"+ref)
		}
		if item.Message != "" || item.Action != "" {
			proposal.CompactRules = append(proposal.CompactRules,
				"operator_advice_non_blocking:"+firstNonEmptyMCPV0(item.Action, "advise")+":"+item.Message,
			)
		}
	}
	return proposal
}
