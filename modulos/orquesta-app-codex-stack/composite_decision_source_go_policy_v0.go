package orquestaappcodexstack

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func compositeInitialProgrammingMicrotasksV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) []orquestadirectoragent.DirectorAgentMicrotaskV0 {
	out := make([]orquestadirectoragent.DirectorAgentMicrotaskV0, 0, len(tasks))
	for _, task := range tasks {
		if compositeTaskComesFromAppChangeV0(task) {
			continue
		}
		out = append(out, task)
	}
	return out
}

func compositeTaskComesFromAppChangeV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	if strings.HasPrefix(strings.TrimSpace(task.TaskID), "task-ref-app-change-") {
		return true
	}
	for _, ref := range task.FunctionContractRefs {
		if strings.TrimSpace(ref.FunctionName) == "ApplyAppChangeV0" ||
			strings.TrimSpace(ref.FunctionName) == "ApplyExternalDomainWorkV0" ||
			strings.Contains(strings.TrimSpace(ref.ContractRef), ":app-change:") {
			return true
		}
	}
	return false
}

func compositeLooksLikeGoAppPlanV0(tasks []orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, task := range tasks {
		if compositeTaskMentionsAnyV0(task, "go test", "go.mod", "cmd/server") {
			return true
		}
		for _, path := range task.WriteSet {
			normalized := compositeNormalizePathTokenV0(path)
			if normalized == "go.mod" ||
				strings.HasPrefix(normalized, "cmd/") ||
				strings.HasPrefix(normalized, "internal/") {
				return true
			}
		}
	}
	return false
}

func compositeTasksRequireGoTestAllV0(tasks []orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, task := range tasks {
		if compositeTaskRequiresGoTestAllV0(task) {
			return true
		}
	}
	return false
}

func compositeTasksCoverWriteSetV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	want = compositeNormalizePathTokenV0(want)
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			if compositeNormalizePathTokenV0(path) == want {
				return true
			}
		}
	}
	return false
}

func compositeTasksCoverCmdEntrypointV0(tasks []orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			normalized := compositeNormalizePathTokenV0(path)
			if normalized == "cmd" ||
				normalized == "cmd/**" ||
				normalized == "cmd/server" ||
				strings.HasPrefix(normalized, "cmd/") {
				return true
			}
		}
	}
	return false
}

func compositeTasksDependOnBootstrapV0(tasks []orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	bootstrapIDs := compositeBootstrapTaskIDsV0(tasks)
	if len(bootstrapIDs) == 0 {
		return false
	}
	for _, task := range tasks {
		if compositeTaskCoversGoModV0(task) || !compositeTaskNeedsGoBootstrapV0(task) {
			continue
		}
		if !compositeTaskDependsOnAnyV0(task, bootstrapIDs) {
			return false
		}
	}
	return true
}

func compositeBootstrapTaskIDsV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) map[string]bool {
	ids := map[string]bool{}
	for _, task := range tasks {
		if compositeTaskCoversGoModV0(task) {
			ids[strings.TrimSpace(task.TaskID)] = true
		}
	}
	return ids
}

func compositeTaskCoversGoModV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, path := range task.WriteSet {
		if compositeNormalizePathTokenV0(path) == "go.mod" {
			return true
		}
	}
	return false
}

func compositeTaskNeedsGoBootstrapV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	if compositeTaskRequiresGoTestAllV0(task) {
		return true
	}
	for _, path := range task.WriteSet {
		normalized := compositeNormalizePathTokenV0(path)
		if strings.HasPrefix(normalized, "cmd/") ||
			strings.HasPrefix(normalized, "internal/") {
			return true
		}
	}
	return false
}

func compositeTaskRequiresGoTestAllV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, test := range task.RequiredTests {
		if strings.Contains(strings.ToLower(strings.TrimSpace(test)), "go test ./...") {
			return true
		}
	}
	return false
}

func compositeTaskDependsOnAnyV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	refs map[string]bool,
) bool {
	for _, dep := range task.DependsOn {
		if refs[strings.TrimSpace(dep)] {
			return true
		}
	}
	return false
}

func compositeTaskMentionsAnyV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	needles ...string,
) bool {
	text := strings.ToLower(strings.Join(append(
		append(
			append([]string{task.Title, task.Summary}, task.AcceptanceCriteria...),
			task.RequiredTests...,
		),
		task.WriteSet...,
	), "\n"))
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func compositeNormalizePathTokenV0(path string) string {
	path = strings.TrimSpace(strings.ToLower(path))
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimSuffix(path, "/")
	return path
}
