package orquestacoreworkflow

import "strings"

func orchestrationRunIsEmptyV0(run OrchestrationRunV0) bool {
	return strings.TrimSpace(run.RunID) == "" &&
		strings.TrimSpace(run.ProjectRef) == "" &&
		strings.TrimSpace(run.AppSpecRef) == "" &&
		strings.TrimSpace(string(run.Status)) == "" &&
		strings.TrimSpace(string(run.CurrentPhase)) == "" &&
		len(run.Phases) == 0
}

func isSupportedCapacityRecommendationV0(capacity OrchestrationCapacityRecommendationV0) bool {
	switch strings.TrimSpace(string(capacity)) {
	case string(OrchestrationCapacityLowV0), string(OrchestrationCapacityMediumV0),
		string(OrchestrationCapacityHighV0), string(OrchestrationCapacityXHighV0):
		return true
	default:
		return false
	}
}

func cloneOrchestrationPhaseV0(phase OrchestrationPhaseV0) OrchestrationPhaseV0 {
	phase.EntryCriteria = cloneStringsV0(phase.EntryCriteria)
	phase.ExitCriteria = cloneStringsV0(phase.ExitCriteria)
	phase.EvidenceRequired = cloneStringsV0(phase.EvidenceRequired)
	return phase
}

func cloneStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func normalizePhaseIDV0(phase OrchestrationPhaseIDV0) OrchestrationPhaseIDV0 {
	return OrchestrationPhaseIDV0(strings.TrimSpace(string(phase)))
}

func normalizeRunStatusV0(status OrchestrationRunStatusV0) OrchestrationRunStatusV0 {
	return OrchestrationRunStatusV0(strings.TrimSpace(string(status)))
}

func normalizePhaseStatusV0(status OrchestrationPhaseStatusV0) OrchestrationPhaseStatusV0 {
	return OrchestrationPhaseStatusV0(strings.TrimSpace(string(status)))
}

func issueV0(code OrchestrationValidationCodeV0, field string) OrchestrationValidationIssueV0 {
	return OrchestrationValidationIssueV0{Code: code, Field: field}
}
