package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

const compositeDirectorAgentTextListMaxV0 = 24

func normalizeCompositeDirectorDecisionBatchV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	out = normalizeCompositeOpenPhaseDecisionPhasesV0(run, out)
	out = normalizeCompositePublishContractDecisionRefsV0(run, out)
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

func normalizeCompositeOpenPhaseDecisionPhasesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	current := strings.TrimSpace(string(run.CurrentPhase))
	if current == "" {
		current = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	}
	for i := range decisions {
		decisionPhase := strings.TrimSpace(decisions[i].PhaseID)
		if decisions[i].OpenPhase != nil {
			target := strings.TrimSpace(decisions[i].OpenPhase.PhaseID)
			if current != "" && target != "" && (decisionPhase == "" || decisionPhase == target) {
				decisions[i].PhaseID = current
			}
			if target != "" {
				current = target
			}
			continue
		}
		if decisionPhase != "" {
			current = decisionPhase
		}
	}
	return decisions
}

func normalizeCompositePublishContractDecisionRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	accepted := compositeStringSetV0(run.Decisions)
	latestAccepted := ""
	for _, decisionRef := range run.Decisions {
		decisionRef = strings.TrimSpace(decisionRef)
		if decisionRef == "" {
			continue
		}
		latestAccepted = decisionRef
	}
	for i := range decisions {
		if decisions[i].AcceptDecision != nil {
			ref := strings.TrimSpace(decisions[i].AcceptDecision.DecisionRef)
			if ref != "" {
				compositeAddRefV0(accepted, ref)
				latestAccepted = ref
			}
			continue
		}
		if decisions[i].PublishContract == nil {
			continue
		}
		ref := strings.TrimSpace(decisions[i].PublishContract.DecisionRef)
		if ref == "" || accepted[ref] || latestAccepted == "" {
			continue
		}
		decisions[i].PublishContract.DecisionRef = latestAccepted
	}
	return decisions
}

func normalizeCompositeGoAppWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	decisions = normalizeCompositeGoAppRootWriteSetsV0(decisions)
	decisions = normalizeCompositeGoAppWebWriteSetsV0(decisions)
	decisions = normalizeCompositeGoAppParallelWriteSetsV0(decisions)
	if compositeAnyTaskWriteSetHasPathV0(compositeProgrammingMicrotasksV0(decisions), "README.md") {
		return decisions
	}
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if !compositeTaskLooksLikeCompleteGoAppWorkV0(*task) ||
			len(task.WriteSet) >= compositeDirectorAgentTextListMaxV0 {
			continue
		}
		task.WriteSet = append(task.WriteSet, "README.md")
	}
	return decisions
}

func normalizeCompositeGoAppWebWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	if compositeAnyTaskWriteSetHasPathOrChildV0(compositeProgrammingMicrotasksV0(decisions), "web") {
		return decisions
	}
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if !compositeTaskLooksLikeCompleteGoAppWorkV0(*task) ||
			!compositeTaskMentionsAnyV0(*task, "web") ||
			len(task.WriteSet) >= compositeDirectorAgentTextListMaxV0 {
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

func compositeAnyTaskWriteSetHasPathOrChildV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	for _, task := range tasks {
		if compositeTaskWriteSetHasPathOrChildV0(task, want) {
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

func compositeTaskWriteSetHasPathOrChildV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	want = compositeNormalizePathTokenV0(want)
	for _, path := range task.WriteSet {
		normalized := compositeNormalizePathTokenV0(path)
		if normalized == want || strings.HasPrefix(normalized, want+"/") {
			return true
		}
		if want == "web" && compositePathCoversWebSurfaceV0(normalized) {
			return true
		}
	}
	return false
}

func compositePathCoversWebSurfaceV0(path string) bool {
	path = compositeNormalizePathTokenV0(path)
	return path == "internal/webadmin" ||
		path == "internal/webadmin/**" ||
		strings.HasPrefix(path, "internal/webadmin/")
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
