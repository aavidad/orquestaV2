package orquestadirector

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func normalizeAgentProgressSupervisionInputV0(
	input AgentProgressSupervisionInputV0,
) AgentProgressSupervisionInputV0 {
	input.CommandMeta.CommandID = strings.TrimSpace(input.CommandMeta.CommandID)
	input.CommandMeta.RunID = strings.TrimSpace(input.CommandMeta.RunID)
	input.CommandMeta.IdempotencyKey = strings.TrimSpace(input.CommandMeta.IdempotencyKey)
	input.CommandMeta.CorrelationID = strings.TrimSpace(input.CommandMeta.CorrelationID)
	input.CommandMeta.RequestedBy = strings.TrimSpace(input.CommandMeta.RequestedBy)
	input.CommandMeta.OccurredAt = strings.TrimSpace(input.CommandMeta.OccurredAt)
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	input.TaskRef = strings.TrimSpace(input.TaskRef)
	input.DeliveryRef = strings.TrimSpace(input.DeliveryRef)
	input.AssessmentRef = strings.TrimSpace(input.AssessmentRef)
	input.QuestionID = strings.TrimSpace(input.QuestionID)
	input.Report.ReportID = strings.TrimSpace(input.Report.ReportID)
	input.Report.RunID = strings.TrimSpace(input.Report.RunID)
	input.Report.AgentRequestID = strings.TrimSpace(input.Report.AgentRequestID)
	input.Report.Summary = strings.TrimSpace(input.Report.Summary)
	input.Report.EvidenceRefs = compactSupervisorStringsV0(input.Report.EvidenceRefs)
	return input
}

func validateAgentProgressSupervisionInputV0(input AgentProgressSupervisionInputV0) error {
	if issues := orquestaruntime.ValidateAgentProgressReportV0(input.Report); len(issues) != 0 {
		return agentProgressSupervisionReportErrorV0(input, issues)
	}
	required := map[string]string{
		"command_meta.run_id": input.CommandMeta.RunID,
		"phase_id":            input.PhaseID,
		"assessment_ref":      input.AssessmentRef,
	}
	for field, value := range required {
		if value == "" {
			return agentProgressSupervisionErrorV0(input, "campo requerido", field, nil)
		}
	}
	if input.CommandMeta.RunID != input.Report.RunID {
		return agentProgressSupervisionErrorV0(input, "run_id no coincide", "report.run_id", nil)
	}
	if progressSupervisionNeedsQuestionV0(input) && input.QuestionID == "" {
		return agentProgressSupervisionErrorV0(input, "question_id requerido", "question_id", nil)
	}
	return validateSupervisorOutputFieldsV0(input)
}

func progressSupervisionNeedsQuestionV0(input AgentProgressSupervisionInputV0) bool {
	if agentProgressAuthConfigBlockerV0(input.Report) {
		return true
	}
	if agentProgressCapacityLimitedV0(input.Report) {
		return !agentProgressStopAllowedV0(input)
	}
	if agentProgressNoACKV0(input.Report) {
		return true
	}
	if agentProgressOverBudgetNoActivityV0(input.Report) {
		return !agentProgressStopAllowedV0(input)
	}
	if agentProgressOverBudgetButActiveV0(input.Report) {
		return true
	}
	if agentProgressHasArtifactWithoutAckV0(input.Report) {
		return true
	}
	return input.Report.Status == orquestaruntime.AgentStalledV0 ||
		(input.Report.Status == orquestaruntime.AgentLoopDetectedV0 && !agentProgressStopAllowedV0(input))
}

func validateSupervisorOutputFieldsV0(input AgentProgressSupervisionInputV0) error {
	return nil
}

func agentProgressSupervisionReportErrorV0(
	input AgentProgressSupervisionInputV0,
	issues []orquestaruntime.AgentProgressReportErrorV0,
) AgentProgressSupervisionErrorV0 {
	evidence := make([]string, 0, len(issues))
	for _, issue := range issues {
		evidence = append(evidence, string(issue.Code))
	}
	return agentProgressSupervisionErrorV0(input, "agent_progress_report invalido", "report."+issues[0].Field, evidence)
}

func agentProgressSupervisionErrorV0(
	input AgentProgressSupervisionInputV0,
	message string,
	field string,
	evidence []string,
) AgentProgressSupervisionErrorV0 {
	return AgentProgressSupervisionErrorV0{
		Code:          ErrDirectorAgentProgressSupervisionInvalidaV0,
		Message:       message,
		Field:         field,
		Retryable:     false,
		Evidence:      compactSupervisorStringsV0(evidence),
		CorrelationID: strings.TrimSpace(input.CommandMeta.CorrelationID),
	}
}
