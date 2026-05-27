package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanRequiredTestsByTaskV0(
	ctx context.Context,
	runRef string,
	store AppDirectorWorkflowTaskStorePortV0,
	activeStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) (map[string][]string, error) {
	fallback := operationalDirectorPlanRequiredTestsFallbackByTaskV0(requiredTests, matches)
	if store == nil {
		return fallback, nil
	}
	taskRefs := compactServiceRefsV0(activeStep.TaskRefs)
	if len(taskRefs) == 0 {
		for _, match := range matches {
			taskRefs = append(taskRefs, match.TaskRef)
		}
		taskRefs = compactServiceRefsV0(taskRefs)
	}
	if len(taskRefs) == 0 {
		return fallback, nil
	}
	tasks, err := store.LoadWorkflowTasksV0(ctx, runRef, taskRefs)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return fallback, nil
	}
	requiredSet := serviceStringSetV0(requiredTests)
	for _, task := range tasks {
		taskRef := strings.TrimSpace(task.TaskID)
		if taskRef == "" {
			continue
		}
		taskRequired := compactServiceRefsV0(task.RequiredTests)
		if len(taskRequired) == 0 {
			continue
		}
		if len(requiredSet) > 0 {
			scoped := make([]string, 0, len(taskRequired))
			for _, required := range taskRequired {
				if requiredSet[strings.TrimSpace(required)] {
					scoped = append(scoped, required)
				}
			}
			if len(scoped) > 0 {
				fallback[taskRef] = compactServiceRefsV0(scoped)
				continue
			}
		}
		fallback[taskRef] = taskRequired
	}
	return fallback, nil
}

func operationalDirectorPlanRequiredTestsFallbackByTaskV0(
	requiredTests []string,
	matches []operationalDirectorPlanAcceptedReviewMatchV0,
) map[string][]string {
	out := make(map[string][]string, len(matches))
	requiredTests = compactServiceRefsV0(requiredTests)
	for _, match := range matches {
		taskRef := strings.TrimSpace(match.TaskRef)
		if taskRef == "" {
			continue
		}
		out[taskRef] = append([]string(nil), requiredTests...)
	}
	return out
}

func operationalDirectorPlanRequiredTestsForMatchV0(
	requiredTestsByTask map[string][]string,
	fallbackRequiredTests []string,
	match operationalDirectorPlanAcceptedReviewMatchV0,
) []string {
	taskRef := strings.TrimSpace(match.TaskRef)
	if taskRef != "" && requiredTestsByTask != nil {
		if tests, ok := requiredTestsByTask[taskRef]; ok {
			return compactServiceRefsV0(tests)
		}
	}
	return compactServiceRefsV0(fallbackRequiredTests)
}

func serviceStringSetV0(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range compactServiceRefsV0(values) {
		out[strings.TrimSpace(value)] = true
	}
	return out
}
