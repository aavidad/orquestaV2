package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarails "orquesta/modulos/orquesta-rails"
)

const appChangeDirectorSourceRailBoundaryV0 = "app_change_director_source"

func appChangeReadyForAutoPlanV0(request orquestaappchange.AppChangeRequestV0) bool {
	criteria := sanitizeAppChangeTaskCriteriaV0(request.AcceptanceCriteria)
	return appChangeHasRunnableScopeV0(request) &&
		len(criteria) > 0 &&
		!containsForbiddenAutoPlanTextV0(criteria...)
}

func appChangeHasRunnableScopeV0(request orquestaappchange.AppChangeRequestV0) bool {
	return len(request.AllowedWriteSet) > 0 || appChangeHasExternalWorkV0(request)
}

func appChangeQuestionReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	questionRef string,
) bool {
	return stringInSetV0(run.DirectorQuestions, questionRef) &&
		!stringInSetV0(run.DirectorAnsweredQuestions, questionRef) &&
		!containsForbiddenAutoPlanTextV0(questionRef)
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

func containsForbiddenAutoPlanTextV0(values ...string) bool {
	for _, value := range values {
		if containsSensitiveAutoPlanTextV0(value) {
			return true
		}
	}
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	for _, value := range values {
		if orquestarails.TextContainsOperationalRawDetailForFieldV0(
			appChangeDirectorSourceRailBoundaryV0,
			"autoplan_readiness",
			value,
		) {
			return true
		}
	}
	return false
}

func containsSensitiveAutoPlanTextV0(value string) bool {
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, fragment := range orquestarails.OperationalSensitiveFragmentsV0 {
		if orquestarails.ContainsFragmentWithBoundaryV0(lower, fragment) {
			return true
		}
	}
	return false
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
