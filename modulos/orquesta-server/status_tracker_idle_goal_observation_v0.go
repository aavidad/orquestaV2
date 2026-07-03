package orquestaserver

import (
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementGoalObservedV0(
	result orquestagoal.GoalWorkResultV0,
	observeErr error,
	now time.Time,
) StateV0 {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	return tracker.updateV0(func(state *StateV0) {
		tracker.applyIdleSelfImprovementGoalObservedV0(state, result, observeErr, now)
	})
}

func (tracker *StatusTrackerV0) applyIdleSelfImprovementGoalObservedV0(
	state *StateV0,
	result orquestagoal.GoalWorkResultV0,
	observeErr error,
	now time.Time,
) {
	tracker.applyIdleSelfImprovementGoalObservedWithClosureV0(state, result, nil, observeErr, now)
}

func (tracker *StatusTrackerV0) applyIdleSelfImprovementGoalObservedWithClosureV0(
	state *StateV0,
	result orquestagoal.GoalWorkResultV0,
	observedClosure *orquestagoal.GoalClosureValidationV0,
	observeErr error,
	now time.Time,
) {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	reasonCode := idleSelfImprovementGoalObservationReasonCodeV0(result, observeErr)
	message := firstNonEmptyConfigStringV0(result.Summary, reasonCode)
	if observeErr != nil {
		message = observeErr.Error()
	}
	state.LastHeartbeatAt = formatTimeV0(now)
	state.IdleSelfImprovementCheck = formatTimeV0(now)
	resultForReason := result
	state.IdleSelfImprovementGoalResult = nil
	if strings.TrimSpace(result.GoalRef) != "" {
		goalResult := copyGoalWorkResultForServerStateV0(result)
		state.IdleSelfImprovementGoalResult = &goalResult
	}
	state.IdleSelfImprovementGoalClosure = nil
	if observeErr == nil && result.Status == orquestagoal.GoalStatusCompleteV0 && state.IdleSelfImprovementGoalSpec != nil {
		closure := orquestagoal.GoalClosureValidationV0{}
		if observedClosure != nil {
			closure = copyGoalClosureValidationForServerStateV0(*observedClosure)
		} else {
			closure = orquestagoal.ValidateGoalWorkClosureV0(*state.IdleSelfImprovementGoalSpec, result)
		}
		goalClosure := copyGoalClosureValidationForServerStateV0(closure)
		state.IdleSelfImprovementGoalClosure = &goalClosure
		if closure.Accepted {
			reasonCode = idleSelfImprovementGoalClosureAcceptedReasonV0
			resultForReason.Status = orquestagoal.GoalStatusAcceptedV0
			resultForReason.EvidenceRefs = compactConfigStringsV0(append(resultForReason.EvidenceRefs, closure.EvidenceRefs...))
			message = firstNonEmptyConfigStringV0(result.Summary, reasonCode)
		}
	}
	counters := map[string]int{
		"attempts":  tracker.idleSelfImprovementAttempts,
		"accepted":  tracker.idleSelfImprovementPrepared,
		"artifacts": len(result.ArtifactRefs),
		"tests":     len(result.RequiredTestResults),
		"receipts":  len(result.DomainReceiptRefs),
	}
	if state.IdleSelfImprovementGoalClosure != nil {
		counters["closure_issues"] = len(state.IdleSelfImprovementGoalClosure.Issues)
	}
	state.IdleSelfImprovementReason = idleSelfImprovementGoalObservedReasonV0(reasonCode, resultForReason, message)
	state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
		serverOperationalMessageInputV0{
			Scope:        "idle_self_improvement",
			ReasonCode:   reasonCode,
			Status:       resultForReason.Status,
			Message:      message,
			GoalRefs:     []string{result.GoalRef, result.ExternalGoalRef},
			EvidenceRefs: resultForReason.EvidenceRefs,
			Counters:     counters,
		},
	)
	tracker.lastIdleSelfImprovementAt = now.UTC()
	tracker.idleSelfImprovementInFlight = false
	tracker.idleSelfImprovementAccepted = idleSelfImprovementGoalObservationKeepsAttemptV0(reasonCode, result, observeErr)
	state.IdleSelfImprovementFlight = false
	state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
	state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	if observeErr != nil || result.Status == orquestagoal.GoalStatusInvalidV0 {
		state.LastError = projectServerOperationalMessageV0("idle_self_improvement", message)
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.IdleSelfImprovementOperationalMessage)
		appendRecentServerErrorV0(state, now, "idle_self_improvement_goal_observation", "idle_self_improvement", message, result.EvidenceRefs)
		return
	}
	state.LastError = ""
	state.LastErrorOperationalMessage = nil
}

func idleSelfImprovementGoalObservationReasonCodeV0(
	result orquestagoal.GoalWorkResultV0,
	observeErr error,
) string {
	if observeErr != nil {
		if strings.TrimSpace(observeErr.Error()) == idleSelfImprovementGoalObserverUnavailableReasonV0 {
			return idleSelfImprovementGoalObserverUnavailableReasonV0
		}
		return idleSelfImprovementGoalObservationErrorReasonV0
	}
	switch result.Status {
	case orquestagoal.GoalStatusRunningV0:
		return idleSelfImprovementGoalRunningReasonV0
	case orquestagoal.GoalStatusCompleteV0:
		return idleSelfImprovementGoalCompletePendingClosureV0
	case orquestagoal.GoalStatusBlockedV0:
		if strings.TrimSpace(result.Summary) == idleSelfImprovementGoalBackendGoneWithoutResultV0 {
			return idleSelfImprovementGoalBackendGoneWithoutResultV0
		}
		if strings.TrimSpace(result.Summary) == idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
			goalWorkResultHasIssueCodeV0(result, idleSelfImprovementGoalHighConsumptionNoProgressReasonV0) {
			return idleSelfImprovementGoalHighConsumptionNoProgressReasonV0
		}
		return idleSelfImprovementGoalBlockedReasonV0
	case orquestagoal.GoalStatusInvalidV0:
		return idleSelfImprovementGoalInvalidReasonV0
	default:
		return idleSelfImprovementGoalInvalidReasonV0
	}
}

func goalWorkResultHasIssueCodeV0(result orquestagoal.GoalWorkResultV0, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}

func idleSelfImprovementGoalObservedReasonV0(
	reasonCode string,
	result orquestagoal.GoalWorkResultV0,
	message string,
) string {
	reason := []string{reasonCode}
	if strings.TrimSpace(result.GoalRef) != "" {
		reason = append(reason, "goal_ref="+strings.TrimSpace(result.GoalRef))
	}
	if strings.TrimSpace(result.ExternalGoalRef) != "" {
		reason = append(reason, "external_goal_ref="+strings.TrimSpace(result.ExternalGoalRef))
	}
	if strings.TrimSpace(result.Status) != "" {
		reason = append(reason, projectServerOperationalReasonFieldV0("status", result.Status))
	}
	if strings.TrimSpace(message) != "" {
		reason = append(reason, projectServerOperationalReasonFieldV0("message", message))
	}
	if len(result.EvidenceRefs) > 0 {
		reason = append(reason, "evidence="+strings.Join(compactServerDiagnosticStringsV0(result.EvidenceRefs), ","))
	}
	return strings.Join(compactConfigStringsV0(reason), ";")
}

func idleSelfImprovementGoalObservationKeepsAttemptV0(
	reasonCode string,
	result orquestagoal.GoalWorkResultV0,
	observeErr error,
) bool {
	if observeErr != nil {
		return true
	}
	if reasonCode == idleSelfImprovementGoalClosureAcceptedReasonV0 {
		return false
	}
	if reasonCode == idleSelfImprovementGoalBackendGoneWithoutResultV0 {
		return false
	}
	switch result.Status {
	case orquestagoal.GoalStatusRunningV0,
		orquestagoal.GoalStatusCompleteV0,
		orquestagoal.GoalStatusBlockedV0:
		return true
	default:
		return false
	}
}

func copyGoalWorkSpecForServerStateV0(spec orquestagoal.GoalWorkSpecV0) orquestagoal.GoalWorkSpecV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	spec.ContextRefs = append([]orquestagoal.GoalContextRefV0(nil), spec.ContextRefs...)
	spec.RuleRefs = append([]orquestagoal.GoalRuleRefV0(nil), spec.RuleRefs...)
	spec.SkillRefs = append([]string(nil), spec.SkillRefs...)
	spec.WriteSet = append([]orquestagoal.GoalWriteScopeV0(nil), spec.WriteSet...)
	spec.RequiredTests = copyGoalRequiredTestsForServerStateV0(spec.RequiredTests)
	spec.AcceptanceCriteria = append([]string(nil), spec.AcceptanceCriteria...)
	spec.ArtifactContracts = copyGoalArtifactContractsForServerStateV0(spec.ArtifactContracts)
	spec.EvidenceRefs = append([]string(nil), spec.EvidenceRefs...)
	spec.ClosurePolicy.RequiredEvidenceRefs = append([]string(nil), spec.ClosurePolicy.RequiredEvidenceRefs...)
	return spec
}

func copyGoalLaunchReceiptForServerStateV0(receipt orquestagoal.GoalLaunchReceiptV0) orquestagoal.GoalLaunchReceiptV0 {
	receipt.SchemaVersion = orquestagoal.GoalWorkLaunchReceiptSchemaV0
	receipt.Status = strings.TrimSpace(receipt.Status)
	receipt.GoalRef = strings.TrimSpace(receipt.GoalRef)
	receipt.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
	receipt.EvidenceRefs = append([]string(nil), compactConfigStringsV0(receipt.EvidenceRefs)...)
	receipt.Issues = append([]orquestagoal.GoalWorkIssueV0(nil), receipt.Issues...)
	return receipt
}

func copyGoalWorkResultForServerStateV0(result orquestagoal.GoalWorkResultV0) orquestagoal.GoalWorkResultV0 {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	result.ArtifactRefs = append([]string(nil), result.ArtifactRefs...)
	result.ArtifactPaths = append([]string(nil), result.ArtifactPaths...)
	result.MaterializedArtifacts = copyGoalMaterializedArtifactsForServerStateV0(result.MaterializedArtifacts)
	result.Checklist = copyGoalChecklistForServerStateV0(result.Checklist)
	result.RequiredTestResults = copyGoalRequiredTestResultsForServerStateV0(result.RequiredTestResults)
	result.DomainReceiptRefs = append([]string(nil), result.DomainReceiptRefs...)
	result.ReworkPlanRefs = append([]string(nil), result.ReworkPlanRefs...)
	result.EvidenceRefs = append([]string(nil), result.EvidenceRefs...)
	result.Issues = append([]orquestagoal.GoalWorkIssueV0(nil), result.Issues...)
	return result
}

func copyGoalMaterializedArtifactsForServerStateV0(
	artifacts []orquestagoal.GoalMaterializedArtifactV0,
) []orquestagoal.GoalMaterializedArtifactV0 {
	out := append([]orquestagoal.GoalMaterializedArtifactV0(nil), artifacts...)
	for i := range out {
		out[i].EvidenceRefs = append([]string(nil), out[i].EvidenceRefs...)
		out[i].Issues = append([]orquestagoal.GoalWorkIssueV0(nil), out[i].Issues...)
	}
	return out
}

func copyGoalChecklistForServerStateV0(
	checklist orquestagoal.GoalWorkChecklistV0,
) orquestagoal.GoalWorkChecklistV0 {
	checklist.ExpectedRefs = append([]string(nil), checklist.ExpectedRefs...)
	checklist.CompletedRefs = append([]string(nil), checklist.CompletedRefs...)
	checklist.MissingRefs = append([]string(nil), checklist.MissingRefs...)
	checklist.EvidenceRefs = append([]string(nil), checklist.EvidenceRefs...)
	return checklist
}

func copyGoalClosureValidationForServerStateV0(
	validation orquestagoal.GoalClosureValidationV0,
) orquestagoal.GoalClosureValidationV0 {
	validation.EvidenceRefs = append([]string(nil), validation.EvidenceRefs...)
	validation.Issues = append([]orquestagoal.GoalWorkIssueV0(nil), validation.Issues...)
	return validation
}

func copyGoalRequiredTestsForServerStateV0(
	tests []orquestagoal.GoalRequiredTestV0,
) []orquestagoal.GoalRequiredTestV0 {
	out := append([]orquestagoal.GoalRequiredTestV0(nil), tests...)
	for i := range out {
		out[i].AcceptanceCriteria = append([]string(nil), out[i].AcceptanceCriteria...)
		out[i].AcceptanceCriteriaRefs = append([]string(nil), out[i].AcceptanceCriteriaRefs...)
		out[i].EvidenceRefs = append([]string(nil), out[i].EvidenceRefs...)
	}
	return out
}

func copyGoalRequiredTestResultsForServerStateV0(
	results []orquestagoal.GoalRequiredTestResultV0,
) []orquestagoal.GoalRequiredTestResultV0 {
	out := append([]orquestagoal.GoalRequiredTestResultV0(nil), results...)
	for i := range out {
		out[i].EvidenceRefs = append([]string(nil), out[i].EvidenceRefs...)
	}
	return out
}

func copyGoalArtifactContractsForServerStateV0(
	contracts []orquestagoal.GoalArtifactContractV0,
) []orquestagoal.GoalArtifactContractV0 {
	out := append([]orquestagoal.GoalArtifactContractV0(nil), contracts...)
	for i := range out {
		out[i].EvidenceRefs = append([]string(nil), out[i].EvidenceRefs...)
	}
	return out
}
