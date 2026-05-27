package orquestaappcodexstack

import (
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

const compositeParallelWriteSetCriterionV0 = "write_set_paralelo: usa rutas propias/fragments y deja rutas agregadas para una tarea integradora"

func normalizeCompositeGoAppParallelWriteSetsV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	tasks := compositeInitialProgrammingMicrotasksV0(compositeProgrammingMicrotasksV0(decisions))
	if len(tasks) < 3 || !compositeLooksLikeGoAppPlanV0(tasks) {
		return decisions
	}
	repeated := compositeRepeatedSharedWriteSetsV0(tasks)
	if len(repeated) == 0 {
		return decisions
	}
	for i := range decisions {
		if decisions[i].CreateMicrotask == nil {
			continue
		}
		task := &decisions[i].CreateMicrotask.Task
		if compositeTaskCoversGoModV0(*task) || compositeTaskLooksLikeIntegrationV0(*task) {
			continue
		}
		slug := compositeTaskSlugV0(*task)
		rewritten := compositeRewriteSharedWriteSetForParallelismV0(task.WriteSet, slug, repeated)
		if strings.Join(rewritten, "\n") == strings.Join(compactStringsV0(task.WriteSet), "\n") {
			continue
		}
		task.WriteSet = rewritten
		if !stringInSetV0(task.AcceptanceCriteria, compositeParallelWriteSetCriterionV0) {
			task.AcceptanceCriteria = append(task.AcceptanceCriteria, compositeParallelWriteSetCriterionV0)
		}
	}
	return decisions
}

func compositeRepeatedSharedWriteSetsV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) map[string]bool {
	counts := map[string]int{}
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			normalized := compositeNormalizePathTokenV0(path)
			if compositePathCanSerializeSiblingTasksV0(normalized) {
				counts[normalized]++
			}
		}
	}
	repeated := map[string]bool{}
	for path, count := range counts {
		if count > 1 {
			repeated[path] = true
		}
	}
	return repeated
}

func compositePathCanSerializeSiblingTasksV0(path string) bool {
	if path == "readme.md" ||
		path == ".env.example" ||
		path == "docs" ||
		path == "docs/**" ||
		path == "docs/openapi.yaml" ||
		path == "i18n" ||
		path == "i18n/**" ||
		path == "internal/platform/api" ||
		path == "internal/platform/api/**" ||
		path == "internal/modules" ||
		path == "internal/modules/**" {
		return true
	}
	return strings.HasPrefix(path, "cmd/") ||
		compositeInternalModuleGlobV0(path) != ""
}

func compositeRewriteSharedWriteSetForParallelismV0(
	writeSet []string,
	slug string,
	repeated map[string]bool,
) []string {
	out := make([]string, 0, len(writeSet))
	for _, path := range writeSet {
		normalized := compositeNormalizePathTokenV0(path)
		if !repeated[normalized] {
			out = append(out, path)
			continue
		}
		out = append(out, compositeParallelReplacementPathV0(normalized, slug))
	}
	return compactStringsV0(out)
}

func compositeParallelReplacementPathV0(path string, slug string) string {
	switch path {
	case "internal/platform/api", "internal/platform/api/**":
		return "internal/platform/api/fragments/" + slug
	case "docs", "docs/**":
		return "docs/modules/" + slug + ".md"
	case "docs/openapi.yaml":
		return "docs/openapi/fragments/" + slug + ".yaml"
	case "i18n", "i18n/**":
		return "i18n/modules/" + slug
	case "readme.md":
		return "docs/modules/" + slug + "-readme.md"
	case ".env.example":
		return "docs/config/" + slug + ".env.md"
	case "internal/modules", "internal/modules/**":
		return "internal/modules/" + slug
	}
	if module := compositeInternalModuleGlobV0(path); module != "" && !compositeSlugOwnsModuleV0(slug, module) {
		return "internal/modules/" + slug + "/integrations/" + module
	}
	if strings.HasPrefix(path, "cmd/") {
		return "internal/platform/api/fragments/" + slug
	}
	return path
}

func compositeSlugOwnsModuleV0(slug string, module string) bool {
	return slug == module ||
		strings.HasPrefix(slug, module+"-") ||
		strings.HasPrefix(module, slug+"-")
}

func compositeInternalModuleGlobV0(path string) string {
	const prefix = "internal/modules/"
	const suffix = "/**"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	module := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if module == "" || strings.Contains(module, "/") {
		return ""
	}
	return module
}

func compositeTaskLooksLikeIntegrationV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	text := strings.ToLower(task.Title + " " + task.Summary)
	for _, token := range []string{"final", "smoke", "integracion", "integration", "deploy", "docker"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func compositeTaskSlugV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) string {
	slug := compositeSlugFromTextV0(task.Title)
	if slug == "" {
		slug = compositeSlugFromTextV0(task.TaskID)
	}
	if slug == "" {
		return "area"
	}
	return slug
}

func compositeSlugFromTextV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := make([]string, 0, 8)
	for _, raw := range strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if raw == "" ||
			strings.HasPrefix(raw, "rx") ||
			compositeSlugTokenIsNumericV0(raw) ||
			raw == "task" ||
			raw == "resigrx" {
			continue
		}
		parts = append(parts, raw)
		if len(strings.Join(parts, "-")) >= 48 {
			break
		}
	}
	slug := strings.Join(parts, "-")
	if len(slug) > 48 {
		slug = strings.TrimRight(slug[:48], "-")
	}
	return slug
}

func compositeSlugTokenIsNumericV0(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
