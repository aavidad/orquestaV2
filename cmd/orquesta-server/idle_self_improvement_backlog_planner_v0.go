package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const idleSelfImprovementBacklogDocRelV0 = "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md"

type idleSelfImprovementBacklogSectionV0 struct {
	Ref, Heading, Objective string
	Scope, Criteria, Tests  []string
	Completed               bool
	SourceLine              int
}

func (supervisor serverStackSupervisorV0) PlanIdleSelfImprovementV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestaserver.IdleSelfImprovementPlanResultV0{}, err
	}
	return (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: supervisor.projectWorkDir}).PlanV0(request)
}

type idleSelfImprovementBacklogPlannerV0 struct{ ProjectWorkDir string }

func (planner idleSelfImprovementBacklogPlannerV0) PlanV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, error) {
	base := request.BaseRequest
	maxRequests := request.MaxRequests
	if maxRequests <= 0 {
		maxRequests = orquestaserver.DefaultIdleSelfImprovementMaxRequestsV0
	}
	sections, err := planner.loadBacklogSectionsV0()
	if err != nil {
		return orquestaserver.IdleSelfImprovementPlanResultV0{
			Requests: []orquestaserver.IdleSelfImprovementRequestV0{
				idleSelfImprovementBacklogFallbackRequestV0(base, err.Error()),
			},
			EvidenceRefs: []string{"evidence-ref-autoprogramming-backlog-planner-fallback"},
			Message:      err.Error(),
		}, nil
	}
	idleSelfImprovementPrioritizeBacklogSectionsV0(sections)
	excluded := idleSelfImprovementExcludedRequestRefsV0(request)
	for ref := range planner.completedBacklogRequestRefsV0() {
		excluded[ref] = true
	}
	requests := make([]orquestaserver.IdleSelfImprovementRequestV0, 0, maxRequests)
	for _, section := range sections {
		if section.Completed {
			continue
		}
		next := idleSelfImprovementRequestForBacklogSectionV0(base, section)
		if excluded[next.RequestRef] {
			continue
		}
		requests = append(requests, next)
		if len(requests) >= maxRequests {
			break
		}
	}
	if len(requests) < maxRequests && idleSelfImprovementShouldAddScannerRequestV0(request, requests) {
		scanner := idleSelfImprovementBacklogScannerRequestV0(base, request)
		if !excluded[scanner.RequestRef] {
			requests = append(requests, scanner)
		}
	}
	if len(requests) == 0 {
		if len(excluded) > 0 {
			return orquestaserver.IdleSelfImprovementPlanResultV0{
				Requests:     []orquestaserver.IdleSelfImprovementRequestV0{},
				EvidenceRefs: []string{"evidence-ref-autoprogramming-backlog-known-work"},
				Message:      "backlog_tareas_ya_visibles_en_cola",
			}, nil
		}
		requests = append(requests, idleSelfImprovementBacklogFallbackRequestV0(base, "backlog_sin_tareas_pendientes_detectadas"))
	}
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests:     requests,
		EvidenceRefs: []string{"evidence-ref-autoprogramming-backlog-doc"},
		Message:      "backlog_autoprogramming_planned",
	}, nil
}

func idleSelfImprovementExcludedRequestRefsV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) map[string]bool {
	excluded := map[string]bool{}
	for _, value := range request.KnownRequestRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	for _, value := range request.KnownRunRefs {
		if ref := idleSelfImprovementNormalizeQueuedRequestRefV0(value); ref != "" {
			excluded[ref] = true
		}
	}
	return excluded
}

func idleSelfImprovementNormalizeQueuedRequestRefV0(value string) string {
	value = strings.TrimSpace(value)
	if retryIndex := strings.Index(value, "-retry-"); retryIndex > 0 {
		value = value[:retryIndex]
	}
	return value
}

func idleSelfImprovementShouldAddScannerRequestV0(
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	requests []orquestaserver.IdleSelfImprovementRequestV0,
) bool {
	if strings.TrimSpace(request.Trigger) == "capacity_free" {
		return true
	}
	return len(requests) == 0
}

func idleSelfImprovementBacklogScannerRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	plan orquestaserver.IdleSelfImprovementPlanRequestV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	refSuffix := "scanner-" + idleSelfImprovementBacklogHashV0(base.ProjectRef+"|"+strings.TrimSpace(plan.Trigger))
	request.RequestRef = "request-ref-autoprogramming-backlog-" + refSuffix
	request.CorrelationID = "corr-" + request.RequestRef
	request.FailureKind = "backlog_scan"
	request.FailureSummary = "revisar Orquesta, detectar huecos concretos pendientes y anadirlos al backlog de automejora"
	request.SuggestedArea = "backlog-scan"
	request.WriteSet = []string{
		idleSelfImprovementBacklogDocRelV0,
		"docs/rail_errors_observados_2026-05-23.md",
		"docs/duplicaciones_railes_pendientes_2026-05-24.md",
	}
	request.RequiredTests = append([]string(nil), base.RequiredTests...)
	request.AcceptanceCriteria = compactServerStackStringsV0(append(append([]string(nil), base.AcceptanceCriteria...),
		"detectar huecos reales de nucleo/director/adaptadores sin duplicar tareas ya en cola",
		"anadir secciones Txx concretas al backlog con objetivo, alcance, criterios y tests",
		"no programar cambios amplios desde el scanner; dejar tareas ejecutables para el siguiente ciclo",
		"si no hay huecos nuevos, documentar evidencia breve y no inventar trabajo",
	))
	request.CompactRules = compactServerStackStringsV0(append(append([]string(nil), base.CompactRules...),
		"scanner de backlog: salida compacta y tareas concretas",
	))
	request.ContextRefs = compactServerStackStringsV0(append(append([]string(nil), base.ContextRefs...),
		"backlog-doc-autoprogramacion-2026-05-23",
		"trigger:"+firstNonEmptyServerStackV0(strings.TrimSpace(plan.Trigger), "scanner"),
		"queue_size:"+strconv.Itoa(plan.QueueSize),
		"free_capacity:"+strconv.Itoa(plan.FreeCapacity),
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(append([]string(nil), base.EvidenceRefs...),
		"evidence-ref-autoprogramming-backlog-scanner",
	))
	return request
}
func (planner idleSelfImprovementBacklogPlannerV0) loadBacklogSectionsV0() ([]idleSelfImprovementBacklogSectionV0, error) {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil, errors.New("project_work_dir_required")
	}
	body, err := os.ReadFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0))
	if err != nil {
		return nil, errors.New("autoprogramming_backlog_doc_unavailable")
	}
	return parseIdleSelfImprovementBacklogSectionsV0(string(body)), nil
}

func (planner idleSelfImprovementBacklogPlannerV0) completedBacklogRequestRefsV0() map[string]bool {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return map[string]bool{}
	}
	runtimeDir := filepath.Join(projectDir, ".orquesta-runtime")
	out := map[string]bool{}
	_ = filepath.WalkDir(runtimeDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || entry.Name() != "agent_ack.json" {
			return nil
		}
		runRef := backlogRunRefFromRuntimeACKPathV0(runtimeDir, path)
		if runRef == "" {
			return nil
		}
		if idleSelfImprovementACKCompletedV0(path) {
			out[idleSelfImprovementNormalizeQueuedRequestRefV0(runRef)] = true
		}
		return nil
	})
	return out
}

func backlogRunRefFromRuntimeACKPathV0(runtimeDir string, ackPath string) string {
	rel, err := filepath.Rel(runtimeDir, ackPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return ""
	}
	runRef := strings.TrimSpace(parts[0])
	if !strings.HasPrefix(runRef, "request-ref-autoprogramming-backlog-") {
		return ""
	}
	return runRef
}

func idleSelfImprovementACKCompletedV0(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var ack struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &ack); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(ack.Status), "completed")
}
func parseIdleSelfImprovementBacklogSectionsV0(content string) []idleSelfImprovementBacklogSectionV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	sections := []idleSelfImprovementBacklogSectionV0{}
	for index := 0; index < len(lines); index++ {
		heading := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(heading, "## T") {
			continue
		}
		next := len(lines)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			if strings.HasPrefix(strings.TrimSpace(lines[cursor]), "## ") {
				next = cursor
				break
			}
		}
		sectionLines := lines[index+1 : next]
		section := idleSelfImprovementBacklogSectionV0{
			Ref:        idleSelfImprovementHeadingRefV0(heading),
			Heading:    strings.TrimPrefix(heading, "## "),
			SourceLine: index + 1,
		}
		section.Objective = idleSelfImprovementSectionValueV0(sectionLines, "Objetivo:")
		section.Scope = idleSelfImprovementSectionListV0(sectionLines, "Alcance:")
		section.Criteria = idleSelfImprovementSectionListV0(sectionLines, "Criterios:")
		section.Tests = idleSelfImprovementSectionTestsV0(sectionLines)
		section.Completed = idleSelfImprovementSectionCompletedV0(sectionLines)
		sections = append(sections, section)
	}
	return sections
}

func idleSelfImprovementPrioritizeBacklogSectionsV0(sections []idleSelfImprovementBacklogSectionV0) {
	sort.SliceStable(sections, func(i, j int) bool {
		return idleSelfImprovementBacklogSectionPriorityV0(sections[i]) <
			idleSelfImprovementBacklogSectionPriorityV0(sections[j])
	})
}

func idleSelfImprovementBacklogSectionPriorityV0(section idleSelfImprovementBacklogSectionV0) int {
	text := strings.ToLower(strings.Join(append([]string{
		section.Ref,
		section.Heading,
		section.Objective,
	}, section.Scope...), " "))
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"nucleo",
		"núcleo",
		"core",
		"director",
		"automejora",
		"autoprogramacion",
		"autoprogramación",
		"self-improvement",
		"self improvement",
		"orchestration-core",
		"orquesta-core",
		"orquesta-director",
		"orquesta-autoprogramming",
	) {
		return 0
	}
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"claude",
		"gemini",
		"ollama",
		"vllm",
		"conector de agente",
		"conectores de agentes",
		"agent connector",
		"agent connectors",
	) {
		return 1
	}
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"web",
		"opes",
		"cli",
		"mcp",
		"complemento",
		"complementos",
		"plugin",
		"plugins",
		"modulos/orquesta-web",
		"modulos/orquesta-opes",
		"modulos/orquesta-cli",
		"modulos/orquesta-mcp",
	) {
		return 2
	}
	return 1
}

func idleSelfImprovementBacklogTextContainsAnyV0(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func idleSelfImprovementRequestForBacklogSectionV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	refSuffix := section.Ref + "-" + idleSelfImprovementBacklogHashV0(section.Heading+"|"+section.Objective)
	request.RequestRef = "request-ref-autoprogramming-backlog-" + refSuffix
	request.CorrelationID = "corr-" + request.RequestRef
	request.FailureKind = "backlog_autoprogramming"
	request.FailureSummary = idleSelfImprovementBacklogSummaryV0(section)
	request.SuggestedArea = firstNonEmptyServerStackV0(section.Ref, base.SuggestedArea)
	request.WriteSet = idleSelfImprovementBacklogWriteSetV0(base.WriteSet, section.Scope)
	request.RequiredTests = append([]string(nil), base.RequiredTests...)
	if len(section.Tests) > 0 {
		request.RequiredTests = compactServerStackStringsV0(section.Tests)
	}
	request.AcceptanceCriteria = compactServerStackStringsV0(append(append([]string(nil), base.AcceptanceCriteria...), section.Criteria...))
	request.CompactRules = compactServerStackStringsV0(append(append([]string(nil), base.CompactRules...),
		"el director revisa backlog y genera tareas concretas; no una tarea generica",
		"un agente padre por tarea; subagentes hasta 6 si ayudan",
		"si aparece otro hueco general, registrarlo como nueva automejora y seguir",
	))
	request.ContextRefs = compactServerStackStringsV0(append(append([]string(nil), base.ContextRefs...),
		"backlog-doc-autoprogramacion-2026-05-23",
		"backlog_section:"+section.Ref,
		"backlog_line:"+strconv.Itoa(section.SourceLine),
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(append([]string(nil), base.EvidenceRefs...),
		"evidence-ref-autoprogramming-backlog-doc",
		"evidence-ref-autoprogramming-backlog-section-"+section.Ref,
	))
	return request
}
func idleSelfImprovementBacklogFallbackRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	reason string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request := base
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"backlog_planner_fallback:"+strings.TrimSpace(reason),
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-planner-fallback",
	))
	return request
}
func idleSelfImprovementBacklogSummaryV0(section idleSelfImprovementBacklogSectionV0) string {
	objective := strings.TrimSpace(section.Objective)
	if objective == "" {
		objective = "cerrar seccion pendiente de autoprogramacion"
	}
	return "backlog pendiente " + section.Heading + ": " + objective
}
func idleSelfImprovementBacklogWriteSetV0(base []string, scope []string) []string {
	if len(scope) == 0 {
		return append([]string(nil), base...)
	}
	out := make([]string, 0, len(scope))
	for _, value := range scope {
		out = append(out, idleSelfImprovementBacklogWriteSetEntriesV0(value)...)
	}
	return compactServerStackStringsV0(out)
}
func idleSelfImprovementSectionValueV0(lines []string, label string) string {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, label) {
			return strings.TrimSpace(strings.TrimPrefix(line, label))
		}
	}
	return ""
}
func idleSelfImprovementSectionListV0(lines []string, label string) []string {
	var out []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, label) {
			if value := strings.TrimSpace(strings.TrimPrefix(trimmed, label)); value != "" {
				out = append(out, value)
			}
			inBlock = true
			continue
		}
		if inBlock && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "- ") {
			break
		}
		if !inBlock || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		out = append(out, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}
	return compactServerStackStringsV0(out)
}
func idleSelfImprovementBacklogWriteSetEntriesV0(value string) []string {
	value = strings.TrimSpace(value)
	original := value
	var out []string
	for {
		_, rest, ok := strings.Cut(value, "`")
		if !ok {
			break
		}
		entry, next, ok := strings.Cut(rest, "`")
		if !ok {
			break
		}
		out = append(out, idleSelfImprovementCleanWriteSetEntryV0(entry))
		value = next
	}
	if len(out) > 0 {
		return compactServerStackStringsV0(out)
	}
	if entry := idleSelfImprovementCleanWriteSetEntryV0(original); entry != "" {
		return []string{entry}
	}
	return nil
}
func idleSelfImprovementCleanWriteSetEntryV0(value string) string {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`"))
	for len(value) > 1 && strings.ContainsRune(".,;:", rune(value[len(value)-1])) {
		value = strings.TrimSpace(value[:len(value)-1])
	}
	return value
}
func idleSelfImprovementSectionTestsV0(lines []string) []string {
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "Tests:") {
			continue
		}
		value := strings.TrimSpace(trimmed[strings.Index(trimmed, "Tests:")+len("Tests:"):])
		value = strings.Trim(value, "` ")
		for _, part := range strings.Split(value, " y ") {
			part = strings.Trim(strings.TrimSpace(part), "` .")
			if strings.HasPrefix(part, "go test ") {
				out = append(out, part)
			}
		}
	}
	return compactServerStackStringsV0(out)
}
func idleSelfImprovementSectionCompletedV0(lines []string) bool {
	text := strings.ToLower(strings.Join(lines, "\n"))
	return strings.Contains(text, "estado: completada") ||
		strings.Contains(text, "estado: completado") ||
		strings.Contains(text, "revalidacion final")
}
func idleSelfImprovementHeadingRefV0(heading string) string {
	heading = strings.TrimSpace(strings.TrimPrefix(heading, "## "))
	heading = strings.ToLower(heading)
	var b strings.Builder
	lastDash := false
	for _, r := range heading {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
func idleSelfImprovementBacklogHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:8]
}
func compactServerStackStringsV0(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
