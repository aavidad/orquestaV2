package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func appChangeReadyForAutoPlanV0(request orquestaappchange.AppChangeRequestV0) bool {
	if !appChangeHasRunnableScopeV0(request) {
		return false
	}
	if len(sanitizeAppChangeTaskCriteriaV0(request.AcceptanceCriteria)) > 0 {
		return true
	}
	if !appChangeHasActionableExternalWorkV0(request) {
		return false
	}
	return len(appChangeTaskCriteriaV0(request)) > 0
}

func appChangeHasRunnableScopeV0(request orquestaappchange.AppChangeRequestV0) bool {
	return len(request.AllowedWriteSet) > 0 || appChangeHasExternalWorkV0(request)
}

func appChangeHasActionableExternalWorkV0(request orquestaappchange.AppChangeRequestV0) bool {
	if !appChangeHasExternalWorkV0(request) || strings.TrimSpace(request.UserIntent) == "" {
		return false
	}
	work := request.ExternalWork
	return strings.TrimSpace(work.WorkKind) != "" ||
		strings.TrimSpace(work.JobRef) != "" ||
		len(work.WorkRefs) > 0
}

func appChangeQuestionReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	refs appChangeRefSetV0,
) bool {
	if !stringInSetV0(run.DirectorQuestions, refs.QuestionRef) {
		return false
	}
	if !stringInSetV0(run.DirectorAnsweredQuestions, refs.QuestionRef) {
		return true
	}
	return !appChangeAutoPlanChainCompleteV0(run, refs)
}

func appChangeAutoPlanChainCompleteV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	refs appChangeRefSetV0,
) bool {
	return run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 &&
		stringInSetV0(run.Tasks, refs.TaskRef)
}

func appChangeReadyForReviewPhaseV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	refs appChangeRefSetV0,
) bool {
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!stringInSetV0(run.Tasks, refs.TaskRef) {
		return false
	}
	tasks := compactAppChangeSourceRefsV0(run.Tasks)
	deliveries := compactAppChangeSourceRefsV0(run.Deliveries)
	return len(tasks) > 0 && len(deliveries) >= len(tasks)
}

func compactAppChangeSourceRefsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func stringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
