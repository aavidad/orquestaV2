package orquestaoperatormcp

import "strings"

func ValidateOperatorStatusQueryV0(input OperatorStatusQueryV0) []OperatorMCPIssueV0 {
	var issues []OperatorMCPIssueV0
	issues = appendOpaqueRefIssueV0(issues, "request_ref", input.RequestRef)
	issues = appendOpaqueRefIssueV0(issues, "subject_ref", input.SubjectRef)
	issues = appendOpaqueRefIssueV0(issues, "status_connector_ref", input.StatusConnectorRef)
	for i, section := range input.IncludeSections {
		if strings.TrimSpace(section) == "" || hasOpaqueRefLeakV0(section) {
			issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPSectionInvalidV0, Field: "include_sections"})
			break
		}
		if i >= 12 {
			issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPSectionInvalidV0, Field: "include_sections"})
			break
		}
	}
	return issues
}

func ValidateOperatorSupervisedBurstV0(input OperatorSupervisedBurstRequestV0) []OperatorMCPIssueV0 {
	var issues []OperatorMCPIssueV0
	issues = appendOpaqueRefIssueV0(issues, "request_ref", input.RequestRef)
	issues = appendOpaqueRefIssueV0(issues, "run_ref", input.RunRef)
	issues = appendOpaqueRefIssueV0(issues, "burst_connector_ref", input.BurstConnectorRef)
	issues = appendOpaqueRefIssueV0(issues, "supervision_ref", input.SupervisionRef)
	if input.MaxSteps < 1 || input.MaxSteps > OperatorMCPMaxBurstStepsLimitV0 {
		issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPBudgetInvalidV0, Field: "max_steps"})
	}
	for _, ref := range input.EvidenceRefs {
		issues = appendOpaqueRefIssueV0(issues, "evidence_refs", ref)
	}
	return issues
}

func ValidateOperatorPendingOutboxV0(input OperatorPendingOutboxQueryV0) []OperatorMCPIssueV0 {
	var issues []OperatorMCPIssueV0
	issues = appendOpaqueRefIssueV0(issues, "request_ref", input.RequestRef)
	issues = appendOpaqueRefIssueV0(issues, "subject_ref", input.SubjectRef)
	issues = appendOpaqueRefIssueV0(issues, "outbox_connector_ref", input.OutboxConnectorRef)
	if input.Limit < 1 || input.Limit > OperatorMCPMaxOutboxLimitV0 {
		issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPLimitInvalidV0, Field: "limit"})
	}
	for _, kind := range input.IncludeKinds {
		if strings.TrimSpace(kind) == "" || hasOpaqueRefLeakV0(kind) {
			issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPSectionInvalidV0, Field: "include_kinds"})
			break
		}
	}
	return issues
}

func ValidateOperatorDirectedQueryV0(input OperatorDirectedQueryV0) []OperatorMCPIssueV0 {
	var issues []OperatorMCPIssueV0
	issues = appendOpaqueRefIssueV0(issues, "query_ref", input.QueryRef)
	issues = appendOpaqueRefIssueV0(issues, "target_ref", input.TargetRef)
	issues = appendOpaqueRefIssueV0(issues, "query_connector_ref", input.QueryConnectorRef)
	if strings.TrimSpace(input.Question) == "" || len([]rune(input.Question)) > OperatorMCPMaxQuestionRunesV0 {
		issues = append(issues, OperatorMCPIssueV0{Code: ErrOperatorMCPQuestionInvalidV0, Field: "question"})
	}
	for _, ref := range input.EvidenceRefs {
		issues = appendOpaqueRefIssueV0(issues, "evidence_refs", ref)
	}
	return issues
}
