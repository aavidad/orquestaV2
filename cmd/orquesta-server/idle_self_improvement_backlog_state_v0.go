package main

import (
	"os"
	"path/filepath"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func (planner idleSelfImprovementBacklogPlannerV0) syncBacklogSectionStateV0(
	sections []idleSelfImprovementBacklogSectionV0,
) {
	for index := range sections {
		if sections[index].Completed {
			continue
		}
		state := planner.documentedBacklogSectionStateV0(sections[index])
		if state.Completed && !sections[index].PendingExplicit {
			sections[index].Completed = true
			sections[index].NeedsDocumentReview = false
		}
		if state.PendingExplicit {
			sections[index].PendingExplicit = true
			sections[index].Completed = false
		}
		if state.NeedsDocumentReview {
			sections[index].NeedsDocumentReview = true
		}
		sections[index].StateEvidenceRefs = compactServerStackStringsV0(append(
			sections[index].StateEvidenceRefs,
			state.EvidenceRefs...,
		))
	}
}

func (planner idleSelfImprovementBacklogPlannerV0) documentedBacklogSectionStateV0(
	section idleSelfImprovementBacklogSectionV0,
) idleSelfImprovementBacklogStateV0 {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" || section.Ref == "" {
		return idleSelfImprovementBacklogStateV0{}
	}
	out := idleSelfImprovementBacklogStateV0{}
	for _, rel := range idleSelfImprovementBacklogLocalDocsV0(section) {
		body, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			continue
		}
		state := idleSelfImprovementLocalDocSectionStateV0(string(body), section)
		if state.Completed {
			state.EvidenceRefs = append(state.EvidenceRefs, "evidence-ref-autoprogramming-backlog-local-doc:"+rel)
			return state
		}
		if state.NeedsDocumentReview {
			out.NeedsDocumentReview = true
			out.EvidenceRefs = append(out.EvidenceRefs, state.EvidenceRefs...)
			out.EvidenceRefs = append(out.EvidenceRefs, "evidence-ref-autoprogramming-backlog-local-doc:"+rel)
		}
	}
	out.EvidenceRefs = compactServerStackStringsV0(out.EvidenceRefs)
	return out
}

func idleSelfImprovementBacklogLocalDocsV0(section idleSelfImprovementBacklogSectionV0) []string {
	out := []string{}
	for _, scope := range idleSelfImprovementBacklogWriteSetV0(nil, section.Scope) {
		if strings.HasPrefix(scope, "modulos/") {
			out = append(out, scope+"/docs/tareas.md", scope+"/docs/decisiones.md", scope+"/docs/pruebas.md", scope+"/README.md")
		}
		if strings.HasPrefix(scope, "docs/") {
			out = append(out, scope)
		}
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementLocalDocSectionStateV0(
	content string,
	section idleSelfImprovementBacklogSectionV0,
) idleSelfImprovementBacklogStateV0 {
	block := idleSelfImprovementLocalDocSectionBlockV0(content, section)
	if block == "" {
		return idleSelfImprovementBacklogStateV0{}
	}
	state := idleSelfImprovementSectionStateV0(strings.Split(block, "\n"))
	if state.Completed {
		state.EvidenceRefs = append(state.EvidenceRefs, "evidence-ref-autoprogramming-backlog-local-doc-state")
		return state
	}
	if state.PendingExplicit {
		state.EvidenceRefs = append(state.EvidenceRefs, "evidence-ref-autoprogramming-backlog-local-doc-state")
		return state
	}
	if idleSelfImprovementSectionHasAmbiguousEvidenceV0(strings.Split(block, "\n")) {
		return idleSelfImprovementBacklogStateV0{
			NeedsDocumentReview: true,
			EvidenceRefs:        []string{"evidence-ref-autoprogramming-backlog-local-doc-ambiguous-evidence"},
		}
	}
	return idleSelfImprovementBacklogStateV0{}
}

func idleSelfImprovementLocalDocSectionBlockV0(
	content string,
	section idleSelfImprovementBacklogSectionV0,
) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	needle := strings.ToLower(section.Ref)
	start := -1
	for index, line := range lines {
		if strings.Contains(strings.ToLower(line), needle) {
			start = index
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for cursor := start + 1; cursor < len(lines); cursor++ {
		trimmed := strings.TrimSpace(lines[cursor])
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			end = cursor
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func (planner idleSelfImprovementBacklogPlannerV0) completedBacklogRequestRefsV0() (
	map[string]bool,
	[]string,
	[]orquestaserver.BacklogScanCollisionV0,
) {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return map[string]bool{}, nil, nil
	}
	return planner.completedBacklogRequestRefsFromRuntimeV0(projectDir)
}

func idleSelfImprovementCompletedBacklogSectionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	completedRequestRefs map[string]bool,
) map[string]bool {
	out := map[string]bool{}
	for _, section := range sections {
		requestRef := idleSelfImprovementNormalizeDependencyRefV0(
			idleSelfImprovementRequestRefForBacklogSectionV0(section),
		)
		if !section.Completed && !completedRequestRefs[requestRef] {
			continue
		}
		out[idleSelfImprovementNormalizeDependencyRefV0(section.Ref)] = true
		out[idleSelfImprovementNormalizeDependencyRefV0(section.Heading)] = true
		out[requestRef] = true
	}
	return out
}

func idleSelfImprovementMarkReconciledPendingSectionsClosedV0(
	sections []idleSelfImprovementBacklogSectionV0,
	completedRequestRefs map[string]bool,
	knownAttempts map[string]int,
) {
	for index := range sections {
		if !sections[index].PendingExplicit {
			continue
		}
		requestRef := idleSelfImprovementNormalizeDependencyRefV0(
			idleSelfImprovementRequestRefForBacklogSectionV0(sections[index]),
		)
		if !completedRequestRefs[requestRef] || knownAttempts[requestRef] < 3 {
			continue
		}
		sections[index].Completed = true
		sections[index].PendingExplicit = false
		sections[index].NeedsDocumentReview = false
		sections[index].StateEvidenceRefs = compactServerStackStringsV0(append(
			sections[index].StateEvidenceRefs,
			"evidence-ref-autoprogramming-backlog-reconciled-pending-ack",
		))
	}
}

func idleSelfImprovementDropExplicitPendingRuntimeCompletionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	completedRequestRefs map[string]bool,
) {
	for _, section := range sections {
		if !section.PendingExplicit {
			continue
		}
		requestRef := idleSelfImprovementNormalizeDependencyRefV0(
			idleSelfImprovementRequestRefForBacklogSectionV0(section),
		)
		delete(completedRequestRefs, requestRef)
	}
}

func idleSelfImprovementDropExplicitPendingKnownExclusionsV0(
	sections []idleSelfImprovementBacklogSectionV0,
	excluded map[string]bool,
) {
	for _, section := range sections {
		if !section.PendingExplicit {
			continue
		}
		requestRef := idleSelfImprovementNormalizeDependencyRefV0(
			idleSelfImprovementRequestRefForBacklogSectionV0(section),
		)
		delete(excluded, requestRef)
	}
}

func idleSelfImprovementBacklogDependenciesSatisfiedV0(
	section idleSelfImprovementBacklogSectionV0,
	completed map[string]bool,
) bool {
	for _, dependency := range section.Dependencies {
		if !completed[idleSelfImprovementNormalizeDependencyRefV0(dependency)] {
			return false
		}
	}
	return true
}

func idleSelfImprovementBacklogCanRunWithDependenciesV0(
	section idleSelfImprovementBacklogSectionV0,
	completed map[string]bool,
) bool {
	if idleSelfImprovementBacklogDependenciesSatisfiedV0(section, completed) {
		return true
	}
	return idleSelfImprovementBacklogHasRunnableDependencyContractV0(section)
}

func idleSelfImprovementBacklogHasRunnableDependencyContractV0(
	section idleSelfImprovementBacklogSectionV0,
) bool {
	return len(section.Dependencies) > 0 && len(section.Inputs) > 0 && len(section.Outputs) > 0
}

func idleSelfImprovementNormalizeDependencyRefV0(value string) string {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), "`.,; ")
	return idleSelfImprovementNormalizeQueuedRequestRefV0(value)
}
