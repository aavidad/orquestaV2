package orquestaappcodexstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	projectBacklogDecisionSourceBatchLimitV0 = 5
	projectBacklogMaxSubagentsV0             = 6
)

type ProjectBacklogDirectorDecisionSourceV0 struct {
	ProjectWorkDir string
}

type projectBacklogItemV0 struct {
	ID           string
	Title        string
	Objective    string
	Dependencies []string
	Output       []string
	Validation   []string
}

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = ProjectBacklogDirectorDecisionSourceV0{}

func (source ProjectBacklogDirectorDecisionSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if !projectBacklogShouldEmitV0(request) {
		return nil, nil
	}
	path, ok := projectBacklogFilePathV0(source.ProjectWorkDir)
	if !ok {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	items := parseProjectBacklogItemsV0(string(data))
	if len(items) == 0 {
		return nil, nil
	}
	return projectBacklogDecisionsV0(request, items, filepath.Base(strings.TrimSpace(source.ProjectWorkDir))), nil
}

func projectBacklogShouldEmitV0(
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) bool {
	run := request.Run
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		strings.TrimSpace(run.RunID) == "" ||
		len(compactStringsV0(run.Tasks)) == 0 ||
		len(compactStringsV0(run.FunctionContracts)) == 0 ||
		drainRunHasPendingExternalAgentsV0(run, nil) ||
		!stackDrainRunHasAllTasksDeliveredOrClosedV0(run) {
		return false
	}
	kind := strings.TrimSpace(request.RequestKind)
	return kind == "" || kind == orquestafactory.RequestKindCrearAppCompletaV0
}

func projectBacklogFilePathV0(projectWorkDir string) (string, bool) {
	projectWorkDir = strings.TrimSpace(projectWorkDir)
	if projectWorkDir == "" {
		return "", false
	}
	patterns := []string{
		filepath.Join(projectWorkDir, "orquesta_input", "backlog_orquesta_*.md"),
		filepath.Join(projectWorkDir, "orquesta_input", "backlog_*.md"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			continue
		}
		sort.Strings(matches)
		return matches[0], true
	}
	return "", false
}

func parseProjectBacklogItemsV0(markdown string) []projectBacklogItemV0 {
	lines := strings.Split(markdown, "\n")
	items := make([]projectBacklogItemV0, 0)
	var current *projectBacklogItemV0
	section := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if item, ok := projectBacklogHeadingV0(line); ok {
			if current != nil {
				items = append(items, *current)
			}
			current = &item
			section = ""
			continue
		}
		if current == nil || line == "" {
			continue
		}
		lower := strings.ToLower(strings.TrimSuffix(line, ":"))
		switch lower {
		case "entrada", "salida", "salida esperada", "validacion":
			section = lower
			continue
		}
		if strings.HasPrefix(line, "Objetivo:") {
			current.Objective = strings.TrimSpace(strings.TrimPrefix(line, "Objetivo:"))
			continue
		}
		if strings.HasPrefix(line, "Dependencias:") {
			current.Dependencies = projectBacklogDependencyIDsV0(strings.TrimSpace(strings.TrimPrefix(line, "Dependencias:")))
			continue
		}
		if !strings.HasPrefix(line, "-") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "-"))
		switch section {
		case "salida", "salida esperada":
			current.Output = append(current.Output, value)
		case "validacion":
			current.Validation = append(current.Validation, value)
		}
	}
	if current != nil {
		items = append(items, *current)
	}
	return items
}

func projectBacklogHeadingV0(line string) (projectBacklogItemV0, bool) {
	if !strings.HasPrefix(line, "## ") {
		return projectBacklogItemV0{}, false
	}
	title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
	id := projectBacklogFirstIDV0(title)
	if id == "" {
		return projectBacklogItemV0{}, false
	}
	title = strings.TrimSpace(strings.TrimPrefix(title, id))
	return projectBacklogItemV0{ID: projectBacklogNormalizeIDV0(id), Title: title}, true
}

func projectBacklogDependencyIDsV0(value string) []string {
	re := regexp.MustCompile(`(?i)RX-\d{3}`)
	matches := re.FindAllString(value, -1)
	out := make([]string, 0, len(matches))
	if strings.Contains(strings.ToLower(value), " a ") && len(matches) >= 2 {
		out = append(out, projectBacklogExpandRangeV0(matches[0], matches[len(matches)-1])...)
	} else {
		for _, match := range matches {
			out = append(out, projectBacklogNormalizeIDV0(match))
		}
	}
	return compactStringsV0(out)
}

func projectBacklogFirstIDV0(value string) string {
	re := regexp.MustCompile(`(?i)RX-\d{3}`)
	return re.FindString(value)
}

func projectBacklogExpandRangeV0(first string, last string) []string {
	start := projectBacklogIDNumberV0(first)
	end := projectBacklogIDNumberV0(last)
	if start < 0 || end < start || end-start > 40 {
		return nil
	}
	out := make([]string, 0, end-start+1)
	for index := start; index <= end; index++ {
		out = append(out, "RX"+leftPadInt3V0(index))
	}
	return out
}

func projectBacklogIDNumberV0(id string) int {
	id = projectBacklogNormalizeIDV0(id)
	if len(id) != 5 || !strings.HasPrefix(id, "RX") {
		return -1
	}
	n := 0
	for _, r := range id[2:] {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func projectBacklogNormalizeIDV0(id string) string {
	id = strings.ToUpper(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, "-", "")
	return id
}

func leftPadInt3V0(value int) string {
	return fmt.Sprintf("%03d", value)
}

func projectBacklogDecisionsV0(
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
	items []projectBacklogItemV0,
	projectSlug string,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	run := request.Run
	existingByID := projectBacklogExistingTaskByIDV0(run)
	satisfied := projectBacklogSatisfiedIDsV0(run)
	contract := projectBacklogFunctionContractRefV0(run)
	if contract.ContractRef == "" && contract.FunctionName == "" {
		return nil
	}
	ready := make([]projectBacklogItemV0, 0, projectBacklogDecisionSourceBatchLimitV0)
	for _, item := range items {
		if _, exists := existingByID[item.ID]; exists {
			continue
		}
		if !projectBacklogDependenciesSatisfiedV0(item.Dependencies, satisfied) {
			continue
		}
		ready = append(ready, item)
		if len(ready) >= projectBacklogDecisionSourceBatchLimitV0 {
			break
		}
	}
	if len(ready) == 0 {
		return nil
	}
	projectSlug = projectBacklogSlugV0(projectSlug)
	if projectSlug == "" {
		projectSlug = projectBacklogSlugV0(run.ProjectRef)
	}
	if projectSlug == "" {
		projectSlug = "app"
	}
	decisions := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(ready)+2)
	if !projectBacklogRunAllowsCreateMicrotaskNowV0(run) {
		decisions = append(decisions, projectBacklogOpenPhaseDecisionV0(
			run,
			projectSlug,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			"Volver a planificacion para materializar backlog pendiente.",
		))
	}
	for _, item := range ready {
		decisions = append(decisions, projectBacklogCreateMicrotaskDecisionV0(
			run,
			projectSlug,
			item,
			contract,
			existingByID,
		))
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		decisions = append(decisions, projectBacklogOpenPhaseDecisionV0(
			run,
			projectSlug,
			orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			"Programacion abierta para ejecutar la siguiente ola del backlog.",
		))
	}
	return decisions
}

func projectBacklogRunAllowsCreateMicrotaskNowV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	return run.CurrentPhase == orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 ||
		run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0
}

func projectBacklogOpenPhaseDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	suffix := codexStackDeterministicDigestV0(run.RunID + ":" + projectSlug + ":open:" + string(phase))
	current := strings.TrimSpace(string(run.CurrentPhase))
	if current == "" {
		current = string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	}
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-backlog-open-" + projectSlug + "-" + string(phase) + "-" + suffix,
		RunID:         run.RunID,
		PhaseID:       current,
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-project-backlog-open-" + projectSlug + "-" + string(phase) + "-" + suffix,
		Summary:       reason,
		EvidenceRefs:  []string{"evidence-ref-project-backlog-" + projectSlug},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(phase),
			Reason:  reason,
		},
	}
}

func projectBacklogCreateMicrotaskDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	projectSlug string,
	item projectBacklogItemV0,
	contract orquestadirectoragent.DirectorAgentFunctionContractRefV0,
	existingByID map[string]string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	taskID := projectBacklogTaskIDV0(projectSlug, item)
	suffix := codexStackDeterministicDigestV0(run.RunID + ":" + taskID)
	deps := make([]string, 0, len(item.Dependencies))
	for _, dep := range item.Dependencies {
		if taskRef := strings.TrimSpace(existingByID[dep]); taskRef != "" {
			deps = append(deps, taskRef)
		}
	}
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-project-backlog-" + projectSlug + "-" + strings.ToLower(item.ID) + "-" + suffix,
		RunID:         run.RunID,
		PhaseID:       orquestadirectoragent.DirectorAgentPlanningPhaseIDV0,
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-project-backlog-" + projectSlug + "-" + strings.ToLower(item.ID) + "-" + suffix,
		Summary:       "Materializar item " + item.ID + " del backlog de app completa.",
		EvidenceRefs:  []string{"evidence-ref-project-backlog-" + projectSlug, "evidence-ref-project-backlog-" + strings.ToLower(item.ID)},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:        orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:               taskID,
				RunID:                run.RunID,
				PhaseID:              string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				WorkProfileKind:      string(orquestacoreworkflow.WorkProfileImplementationV0),
				Title:                projectBacklogLimitTextV0(item.ID + " " + item.Title),
				Summary:              projectBacklogLimitTextV0(item.Objective),
				WriteSet:             projectBacklogWriteSetV0(item),
				AcceptanceCriteria:   projectBacklogAcceptanceCriteriaV0(item, projectSlug),
				RequiredTests:        []string{"go test ./..."},
				DependsOn:            compactStringsV0(deps),
				ContextRefs:          []string{"context-ref-project-backlog-" + projectSlug, "context-ref-project-backlog-" + strings.ToLower(item.ID)},
				CohortRef:            "cohort-ref-project-backlog-" + projectSlug,
				WaveRef:              "wave-ref-project-backlog-" + projectSlug + "-" + strings.ToLower(item.ID),
				MaxChildAgents:       projectBacklogMaxSubagentsV0,
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{contract},
			},
		},
	}
}

func projectBacklogFunctionContractRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestadirectoragent.DirectorAgentFunctionContractRefV0 {
	contracts := compactStringsV0(run.FunctionContracts)
	if len(contracts) == 0 {
		return orquestadirectoragent.DirectorAgentFunctionContractRefV0{}
	}
	return orquestadirectoragent.DirectorAgentFunctionContractRefV0{
		ContractRef:  contracts[0],
		FunctionName: "ApplyProjectBacklogItemV0",
	}
}

func projectBacklogExistingTaskByIDV0(run orquestacoreworkflow.OrchestrationRunV0) map[string]string {
	out := map[string]string{}
	for _, taskRef := range compactStringsV0(run.Tasks) {
		key := projectBacklogIDFromTaskRefV0(taskRef)
		if key != "" {
			out[key] = taskRef
		}
	}
	return out
}

func projectBacklogSatisfiedIDsV0(run orquestacoreworkflow.OrchestrationRunV0) map[string]bool {
	out := map[string]bool{}
	for _, refs := range [][]string{run.DeliveredTasks, run.ClosedTasks} {
		for _, taskRef := range compactStringsV0(refs) {
			if key := projectBacklogIDFromTaskRefV0(taskRef); key != "" {
				out[key] = true
			}
		}
	}
	return out
}

func projectBacklogIDFromTaskRefV0(taskRef string) string {
	normalized := projectBacklogNormalizeTaskRefV0(taskRef)
	re := regexp.MustCompile(`RX\d{3}`)
	return re.FindString(normalized)
}

func projectBacklogNormalizeTaskRefV0(taskRef string) string {
	taskRef = strings.ToUpper(strings.TrimSpace(taskRef))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, taskRef)
}

func projectBacklogDependenciesSatisfiedV0(deps []string, satisfied map[string]bool) bool {
	for _, dep := range compactStringsV0(deps) {
		if !satisfied[dep] {
			return false
		}
	}
	return true
}

func projectBacklogTaskIDV0(projectSlug string, item projectBacklogItemV0) string {
	title := projectBacklogSlugV0(item.Title)
	if title == "" {
		title = "work"
	}
	return "task-" + projectSlug + "-" + strings.ToLower(item.ID) + "-" + title
}

func projectBacklogSlugV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsLetter(r), unicode.IsDigit(r):
			continue
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
		if builder.Len() >= 48 {
			break
		}
	}
	return strings.Trim(builder.String(), "-")
}

func projectBacklogWriteSetV0(item projectBacklogItemV0) []string {
	text := strings.ToLower(strings.Join(append([]string{item.ID, item.Title, item.Objective}, item.Output...), "\n"))
	writeSet := []string{"docs"}
	add := func(paths ...string) {
		writeSet = append(writeSet, paths...)
	}
	switch {
	case strings.Contains(text, "bootstrap") || strings.Contains(text, "estructura base"):
		add("go.mod", "cmd", "internal", "README.md", "i18n")
	case strings.Contains(text, "config") || strings.Contains(text, "i18n") || strings.Contains(text, "observabilidad"):
		add(".env.example", "internal/platform/config", "internal/platform/i18n", "internal/platform/observability", "i18n")
	case strings.Contains(text, "openapi"):
		add("docs/openapi.yaml", "internal/platform/http", "cmd")
	case strings.Contains(text, "tenant") || strings.Contains(text, "identity") || strings.Contains(text, "sesion"):
		add("internal/modules/identity", "internal/platform/security")
	case strings.Contains(text, "auth") || strings.Contains(text, "rbac") || strings.Contains(text, "abac"):
		add("internal/modules/identity", "internal/platform/security", "internal/platform/http")
	case strings.Contains(text, "audit") || strings.Contains(text, "auditoria"):
		add("internal/modules/audit", "internal/platform/audit")
	case strings.Contains(text, "resident"):
		add("internal/modules/residents", "internal/platform/http")
	case strings.Contains(text, "care") || strings.Contains(text, "cuidados") || strings.Contains(text, "incidencias"):
		add("internal/modules/care", "internal/platform/http")
	case strings.Contains(text, "medic") || strings.Contains(text, "emar"):
		add("internal/modules/medication", "internal/platform/http")
	case strings.Contains(text, "restric") || strings.Contains(text, "sujec"):
		add("internal/modules/restrictions", "internal/platform/http")
	case strings.Contains(text, "diet") || strings.Contains(text, "appcc"):
		add("internal/modules/diet", "internal/platform/http")
	case strings.Contains(text, "document") || strings.Contains(text, "firma"):
		add("internal/modules/documents", "internal/platform/http")
	case strings.Contains(text, "runbook") || strings.Contains(text, "eipd") || strings.Contains(text, "security"):
		add("docs/runbooks", "internal/platform/diagnostics")
	case strings.Contains(text, "web"):
		add("web", "internal/platform/web")
	case strings.Contains(text, "persist") || strings.Contains(text, "migracion"):
		add("internal/adapters/persistence", "migrations")
	case strings.Contains(text, "docker") || strings.Contains(text, "compose"):
		add("Dockerfile", "docker-compose.yml", "scripts")
	case strings.Contains(text, "smoke"):
		add("tests", "README.md")
	default:
		add("internal")
	}
	return compactStringsV0(writeSet)
}

func projectBacklogAcceptanceCriteriaV0(item projectBacklogItemV0, projectSlug string) []string {
	criteria := []string{
		"objetivo_actual: crear app completa " + projectSlug + " " + strings.ToLower(item.ID),
		"source_ref: project_backlog:" + item.ID,
		"mantener arquitectura hexagonal e i18n en la superficie tocada",
		"un agente padre puede delegar hasta 6 subagentes si ayuda a terminar antes",
		"no borrar ni mover trabajo existente sin evidencia y permiso explicito",
	}
	if len(item.Dependencies) > 0 {
		criteria = append(criteria, "dependencies: "+strings.Join(item.Dependencies, ","))
	}
	if len(item.Output) > 0 {
		criteria = append(criteria, "salida: "+projectBacklogLimitTextV0(strings.Join(item.Output, "; ")))
	}
	if len(item.Validation) > 0 {
		criteria = append(criteria, "validacion: "+projectBacklogLimitTextV0(strings.Join(item.Validation, "; ")))
	}
	return compactStringsV0(criteria)
}

func projectBacklogLimitTextV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 560 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= 560 {
		return value
	}
	return strings.TrimSpace(string(runes[:560]))
}
