package orquestacionnucleoapp

import (
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeReviewReworkReplanPlanV0(
	plan ReviewReworkReplanPlanV0,
) ReviewReworkReplanPlanV0 {
	plan.CandidateRef = strings.TrimSpace(plan.CandidateRef)
	plan.ReplanRef = strings.TrimSpace(plan.ReplanRef)
	plan.SignalRef = strings.TrimSpace(plan.SignalRef)
	plan.ReworkRequestRef = strings.TrimSpace(plan.ReworkRequestRef)
	plan.TaskRef = strings.TrimSpace(plan.TaskRef)
	plan.ReasonRef = strings.TrimSpace(plan.ReasonRef)
	plan.RequestedAction = orquestacorereplanner.ReplanRecommendedActionV0(
		strings.TrimSpace(string(plan.RequestedAction)),
	)
	plan.CapacityRequestRef = strings.TrimSpace(plan.CapacityRequestRef)
	plan.AgentRequestID = strings.TrimSpace(plan.AgentRequestID)
	plan.AskDirectorQuestionID = strings.TrimSpace(plan.AskDirectorQuestionID)
	plan.AgentRole = strings.TrimSpace(plan.AgentRole)
	plan.MinimumRecommendedCapacity = orquestacoreworkflow.OrchestrationCapacityRecommendationV0(
		strings.TrimSpace(string(plan.MinimumRecommendedCapacity)),
	)
	plan.Summary = strings.TrimSpace(plan.Summary)
	plan.EvidenceRefs = compactStringsV0(plan.EvidenceRefs)
	plan.ReviewResult = orquestacoreworkflow.NormalizeReviewResultV0(plan.ReviewResult)
	plan.SplitTasks = normalizeReviewReworkSplitTasksV0(plan.SplitTasks)
	return plan
}

func normalizeReviewReworkSplitTasksV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []orquestacoreworkflow.WorkflowTaskV0 {
	if len(tasks) == 0 {
		return nil
	}
	normalized := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(tasks))
	for _, task := range tasks {
		normalized = append(normalized, orquestacoreworkflow.NormalizeWorkflowTaskV0(task))
	}
	return normalized
}

func reviewReworkReplanRequestV0(
	request SchedulerCandidateRequestV0,
) ReviewReworkReplanPlanRequestV0 {
	return ReviewReworkReplanPlanRequestV0{
		Run:              request.Run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       request.OccurredAt,
		PreviousStep:     request.PreviousStep,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: request.PreviousDecision,
	}
}

func reviewReworkReplanInputV0(
	request SchedulerCandidateRequestV0,
	plan ReviewReworkReplanPlanV0,
) orquestacorereplanner.ReviewResultReplanInputV0 {
	taskRef := plan.TaskRef
	if taskRef == "" {
		taskRef = reviewReworkTaskFromRunV0(request.Run)
	}
	return orquestacorereplanner.ReviewResultReplanInputV0{
		ReplanRef:       plan.ReplanRef,
		SignalRef:       plan.SignalRef,
		RunRef:          request.Run.RunID,
		TaskRef:         taskRef,
		RequestedAction: plan.RequestedAction,
		ReasonRef:       plan.ReasonRef,
		Summary:         plan.Summary,
		EvidenceRefs:    compactStringsV0(plan.EvidenceRefs),
		ReviewResult:    plan.ReviewResult,
	}
}

func reviewReworkTaskFromRunV0(run orquestacoreworkflow.OrchestrationRunV0) string {
	tasks := compactStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return ""
	}
	return tasks[0]
}

func reviewReworkRunHasRefV0(refs []string, ref string, separator string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	for _, raw := range refs {
		value := strings.TrimSpace(raw)
		if value == ref || strings.HasPrefix(value, ref+separator) {
			return true
		}
	}
	return false
}

func reviewReworkRunContainsRefV0(refs []string, ref string) bool {
	ref = strings.TrimSpace(ref)
	for _, value := range compactStringsV0(refs) {
		if value == ref {
			return true
		}
	}
	return false
}

func reviewReworkSplitPlanSettledV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	plan ReviewReworkReplanPlanV0,
) bool {
	if plan.RequestedAction != orquestacorereplanner.ReplanActionSplitTaskV0 ||
		len(plan.SplitTasks) == 0 ||
		!reviewReworkRunHasRefV0(run.ReplanDecisions, plan.ReplanRef, "#source:") {
		return false
	}
	for _, task := range plan.SplitTasks {
		if !reviewReworkRunContainsRefV0(run.Tasks, task.TaskID) {
			return false
		}
	}
	return true
}

func reviewReworkReplanRequestedByV0(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "orquesta-nucleo-review-rework"
}

func reviewReworkReplanSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
