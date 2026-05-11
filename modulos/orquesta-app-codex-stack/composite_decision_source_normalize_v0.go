package orquestaappcodexstack

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func normalizeCompositeDirectorDecisionBatchV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	bootstrapIDs := compositeBootstrapTaskIDListV0(compositeProgrammingMicrotasksV0(out))
	if len(bootstrapIDs) == 0 {
		return out
	}
	bootstrapRefs := compositeBootstrapTaskIDSetV0(bootstrapIDs)
	for i := range out {
		if out[i].CreateMicrotask == nil {
			continue
		}
		task := &out[i].CreateMicrotask.Task
		if compositeTaskCoversGoModV0(*task) || !compositeTaskNeedsGoBootstrapV0(*task) {
			continue
		}
		if compositeTaskDependsOnAnyV0(*task, bootstrapRefs) {
			continue
		}
		task.DependsOn = append(task.DependsOn, bootstrapIDs[0])
	}
	return out
}

func compositeBootstrapTaskIDListV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, task := range tasks {
		taskID := strings.TrimSpace(task.TaskID)
		if taskID == "" || !compositeTaskCoversGoModV0(task) || seen[taskID] {
			continue
		}
		ids = append(ids, taskID)
		seen[taskID] = true
	}
	return ids
}

func compositeBootstrapTaskIDSetV0(ids []string) map[string]bool {
	refs := map[string]bool{}
	for _, id := range ids {
		refs[strings.TrimSpace(id)] = true
	}
	return refs
}
