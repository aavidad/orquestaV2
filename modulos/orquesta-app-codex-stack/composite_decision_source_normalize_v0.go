package orquestaappcodexstack

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func normalizeCompositeDirectorDecisionBatchV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	out = normalizeCompositeGoAppWriteSetsV0(out)
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

func normalizeCompositeGoAppWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	decisions = normalizeCompositeGoAppRootWriteSetsV0(decisions)
	decisions = normalizeCompositeGoAppWebWriteSetsV0(decisions)
	if compositeAnyTaskWriteSetHasPathV0(compositeProgrammingMicrotasksV0(decisions), "README.md") {
		return decisions
	}
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if !compositeTaskLooksLikeCompleteGoAppWorkV0(*task) {
			continue
		}
		task.WriteSet = append(task.WriteSet, "README.md")
	}
	return decisions
}

func normalizeCompositeGoAppWebWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	if compositeAnyTaskWriteSetHasPathV0(compositeProgrammingMicrotasksV0(decisions), "web") {
		return decisions
	}
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if !compositeTaskLooksLikeCompleteGoAppWorkV0(*task) ||
			!compositeTaskMentionsAnyV0(*task, "web") {
			continue
		}
		task.WriteSet = append(task.WriteSet, "web")
	}
	return decisions
}

func normalizeCompositeGoAppRootWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if !compositeTaskWriteSetHasPathV0(*task, ".") ||
			!compositeTaskLooksLikeRootGoAppWorkV0(*task) {
			continue
		}
		task.WriteSet = []string{"go.mod", "cmd/server", "internal", "web", "README.md"}
	}
	return decisions
}

func compositeTaskLooksLikeRootGoAppWorkV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	return compositeTaskRequiresGoTestAllV0(task) &&
		compositeTaskMentionsAnyV0(task, "go", "go.mod", "cmd/server", "modulo")
}

func compositeAnyTaskWriteSetHasPathV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	for _, task := range tasks {
		if compositeTaskWriteSetHasPathV0(task, want) {
			return true
		}
	}
	return false
}

func compositeTaskLooksLikeCompleteGoAppWorkV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	return compositeTaskCoversGoModV0(task) &&
		compositeTaskCoversCmdEntrypointV0(task) &&
		compositeTaskCoversInternalScopeV0(task) &&
		compositeTaskRequiresGoTestAllV0(task)
}

func compositeTaskCoversCmdEntrypointV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, path := range task.WriteSet {
		normalized := compositeNormalizePathTokenV0(path)
		if normalized == "cmd" ||
			normalized == "cmd/**" ||
			normalized == "cmd/server" ||
			strings.HasPrefix(normalized, "cmd/") {
			return true
		}
	}
	return false
}

func compositeTaskCoversInternalScopeV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, path := range task.WriteSet {
		normalized := compositeNormalizePathTokenV0(path)
		if normalized == "internal" ||
			normalized == "internal/**" ||
			strings.HasPrefix(normalized, "internal/") {
			return true
		}
	}
	return false
}

func compositeTaskWriteSetHasPathV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	want = compositeNormalizePathTokenV0(want)
	for _, path := range task.WriteSet {
		if compositeNormalizePathTokenV0(path) == want {
			return true
		}
	}
	return false
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
