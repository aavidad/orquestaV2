package orquestadirectorscheduler

import (
	"strings"

	orquestaagentprogress "orquesta/modulos/orquesta-agent-progress"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func normalizeSchedulableProgressSupervisionCandidatesV0(
	candidates []SchedulableProgressSupervisionCandidateV0,
) []SchedulableProgressSupervisionCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableProgressSupervisionCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeSchedulableProgressSupervisionCandidateV0(candidate))
	}
	return normalized
}

func normalizeSchedulableProgressSupervisionCandidateV0(
	candidate SchedulableProgressSupervisionCandidateV0,
) SchedulableProgressSupervisionCandidateV0 {
	return SchedulableProgressSupervisionCandidateV0{
		CandidateRef:     strings.TrimSpace(candidate.CandidateRef),
		SupervisionInput: normalizeSchedulerProgressSupervisionInputV0(candidate.SupervisionInput),
		EvidenceRefs:     compactSchedulerStringsV0(candidate.EvidenceRefs),
	}
}

func normalizeSchedulerProgressSupervisionInputV0(
	input orquestadirector.AgentProgressSupervisionInputV0,
) orquestadirector.AgentProgressSupervisionInputV0 {
	input.CommandMeta = normalizeSchedulerCommandMetaV0(input.CommandMeta)
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	input.TaskRef = strings.TrimSpace(input.TaskRef)
	input.DeliveryRef = strings.TrimSpace(input.DeliveryRef)
	input.AssessmentRef = strings.TrimSpace(input.AssessmentRef)
	input.QuestionID = strings.TrimSpace(input.QuestionID)
	input.Report.ReportID = strings.TrimSpace(input.Report.ReportID)
	input.Report.RunID = strings.TrimSpace(input.Report.RunID)
	input.Report.AgentRequestID = strings.TrimSpace(input.Report.AgentRequestID)
	input.Report.Status = orquestaagentprogress.AgentProgressStatusV0(
		strings.TrimSpace(string(input.Report.Status)),
	)
	input.Report.BudgetStatus = orquestaagentprogress.AgentProgressBudgetStatusV0(
		strings.TrimSpace(string(input.Report.BudgetStatus)),
	)
	input.Report.BudgetReason = strings.TrimSpace(input.Report.BudgetReason)
	input.Report.Summary = strings.TrimSpace(input.Report.Summary)
	input.Report.StartedAt = strings.TrimSpace(input.Report.StartedAt)
	input.Report.LastActivityAt = strings.TrimSpace(input.Report.LastActivityAt)
	input.Report.LastAckAt = strings.TrimSpace(input.Report.LastAckAt)
	input.Report.EvidenceRefs = compactSchedulerStringsV0(input.Report.EvidenceRefs)
	return input
}
