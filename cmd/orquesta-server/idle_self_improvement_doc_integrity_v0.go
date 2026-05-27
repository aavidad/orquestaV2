package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func (planner idleSelfImprovementBacklogPlannerV0) localDocIntegrityCollisionsV0() []orquestaserver.BacklogScanCollisionV0 {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil
	}
	var collisions []orquestaserver.BacklogScanCollisionV0
	collisions = append(collisions, planner.localTaskDocDuplicateIDCollisionsV0(projectDir)...)
	collisions = append(collisions, planner.localProofDocPendingExecutionCollisionsV0(projectDir)...)
	collisions = append(collisions, planner.localDocPathRedactionCollisionsV0(projectDir)...)
	collisions = append(collisions, planner.moduleBoundaryLocalAgentDocCollisionsV0(projectDir)...)
	return compactBacklogScanCollisionsV0(collisions)
}

func (planner idleSelfImprovementBacklogPlannerV0) localTaskDocDuplicateIDCollisionsV0(
	projectDir string,
) []orquestaserver.BacklogScanCollisionV0 {
	var collisions []orquestaserver.BacklogScanCollisionV0
	for _, rel := range localModuleDocRefsV0(projectDir, "tareas.md") {
		body, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			continue
		}
		for id, lines := range localTaskDocIDsByLineV0(string(body)) {
			if len(lines) < 2 {
				continue
			}
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "duplicate_module_task_doc_id",
				SectionRef: strings.ToLower(id),
				Message:    "path=" + rel + ";id=" + id + ";lines=" + joinIntsV0(lines),
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-local-doc-integrity-linter",
					"evidence-ref-autoprogramming-local-doc-duplicate-id",
				},
			})
		}
	}
	return collisions
}

func (planner idleSelfImprovementBacklogPlannerV0) localProofDocPendingExecutionCollisionsV0(
	projectDir string,
) []orquestaserver.BacklogScanCollisionV0 {
	var collisions []orquestaserver.BacklogScanCollisionV0
	for _, rel := range localModuleDocRefsV0(projectDir, "pruebas.md") {
		body, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			continue
		}
		content := string(body)
		if !strings.Contains(content, "go test -count=1") {
			continue
		}
		for _, stale := range localProofDocPendingExecutionsV0(content) {
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "stale_pending_local_test_execution",
				SectionRef: strings.ToLower(stale.CaseRef),
				Message:    "path=" + rel + ";case=" + stale.CaseRef + ";line=" + strconv.Itoa(stale.Line),
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-local-doc-integrity-linter",
					"evidence-ref-autoprogramming-local-doc-pending-with-go-test-evidence",
				},
			})
		}
	}
	return collisions
}

type localProofDocPendingExecutionV0 struct {
	CaseRef string
	Line    int
}

func localModuleDocRefsV0(projectDir string, filename string) []string {
	pattern := filepath.Join(projectDir, "modulos", "*", "docs", filename)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var refs []string
	for _, match := range matches {
		rel, err := filepath.Rel(projectDir, match)
		if err != nil {
			continue
		}
		refs = append(refs, filepath.ToSlash(rel))
	}
	return compactServerStackStringsV0(refs)
}

func localTaskDocIDsByLineV0(content string) map[string][]int {
	out := map[string][]int{}
	for index, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		id := localTaskDocIDFromLineV0(line)
		if id == "" {
			continue
		}
		out[id] = append(out[id], index+1)
	}
	return out
}

func localTaskDocIDFromLineV0(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
	}
	if strings.HasPrefix(trimmed, "ID:") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "ID:"))
	}
	token := firstLocalDocTokenV0(trimmed)
	if isLocalModuleTaskIDV0(token) {
		return token
	}
	return ""
}

func firstLocalDocTokenV0(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "- "))
	if value == "" {
		return ""
	}
	token := value
	if before, _, ok := strings.Cut(token, " "); ok {
		token = before
	}
	token = strings.Trim(token, "`:.,;")
	return token
}

func isLocalModuleTaskIDV0(value string) bool {
	if value == "" || !strings.Contains(value, "-") {
		return false
	}
	hasDigit := false
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return hasDigit
}

func localProofDocPendingExecutionsV0(content string) []localProofDocPendingExecutionV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []localProofDocPendingExecutionV0
	currentCase := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Caso:") {
			currentCase = firstLocalDocTokenV0(strings.TrimSpace(strings.TrimPrefix(trimmed, "Caso:")))
			continue
		}
		if strings.EqualFold(trimmed, "Ultima ejecucion: pendiente") && currentCase != "" {
			out = append(out, localProofDocPendingExecutionV0{CaseRef: currentCase, Line: index + 1})
		}
	}
	return out
}

func joinIntsV0(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
}
