package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
)

func (planner idleSelfImprovementBacklogPlannerV0) loadBacklogSectionsV0() (
	[]idleSelfImprovementBacklogSectionV0,
	error,
) {
	if strings.TrimSpace(planner.ProjectWorkDir) == "" {
		return nil, errors.New("project_work_dir_required")
	}
	docs, err := planner.backlogSectionDocumentRefsV0()
	if err != nil {
		return nil, err
	}
	var sections []idleSelfImprovementBacklogSectionV0
	for _, rel := range docs {
		body, err := planner.readBacklogScanDocumentTextV0(rel)
		if err != nil {
			return nil, errors.New("autoprogramming_backlog_doc_unavailable")
		}
		sections = append(sections, parseIdleSelfImprovementBacklogSectionsFromDocumentV0(body, rel)...)
	}
	sections = append(sections, planner.loadFederatedBacklogSectionsV0()...)
	if planner.SelfAuditBacklogEnabled {
		sections = append(sections, selfAuditBacklogSectionsV0(planner.contextV0(), planner.ProjectWorkDir)...)
	}
	idleSelfImprovementAnnotateBacklogTaskIDAliasIndexV0(sections)
	return sections, nil
}

func (planner idleSelfImprovementBacklogPlannerV0) contextV0() context.Context {
	if planner.Context != nil {
		return planner.Context
	}
	return context.Background()
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScannerDocumentRefsV0() []string {
	return compactServerStackStringsV0(append(
		planner.backlogScannerCoreDocumentRefsV0(),
		planner.federatedBacklogSourceRefsV0()...,
	))
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScannerCoreDocumentRefsV0() []string {
	docs, err := planner.backlogSectionDocumentRefsV0()
	if err != nil || len(docs) == 0 {
		docs = []string{idleSelfImprovementBacklogDocRelV0}
	}
	return compactServerStackStringsV0(append(append([]string(nil), docs...),
		"docs/rail_errors_observados_2026-05-23.md",
		"docs/duplicaciones_railes_pendientes_2026-05-24.md",
	))
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogSectionDocumentRefsV0() ([]string, error) {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil, errors.New("project_work_dir_required")
	}
	body, err := planner.readBacklogScanDocumentTextV0(idleSelfImprovementBacklogDocRelV0)
	if err != nil {
		return nil, errors.New("autoprogramming_backlog_doc_unavailable")
	}
	indexed := idleSelfImprovementBacklogIndexDocumentRefsV0(body)
	if len(indexed) == 0 {
		return []string{idleSelfImprovementBacklogDocRelV0}, nil
	}
	return compactServerStackStringsV0(append(
		[]string{idleSelfImprovementBacklogDocRelV0},
		indexed...,
	)), nil
}

func idleSelfImprovementBacklogIndexDocumentRefsV0(content string) []string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var refs []string
	inIndex := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inIndex = idleSelfImprovementBacklogIndexHeadingV0(trimmed)
			continue
		}
		if !inIndex || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		refs = append(refs, idleSelfImprovementBacklogDocRefsFromIndexLineV0(trimmed)...)
	}
	return compactServerStackStringsV0(refs)
}

func idleSelfImprovementBacklogIndexHeadingV0(heading string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(heading, "## ")))
	return strings.Contains(normalized, "indice vivo") ||
		strings.Contains(normalized, "índice vivo") ||
		strings.Contains(normalized, "backlog shards")
}

func idleSelfImprovementBacklogDocRefsFromIndexLineV0(line string) []string {
	var refs []string
	rest := line
	for {
		_, after, ok := strings.Cut(rest, "`")
		if !ok {
			break
		}
		value, next, ok := strings.Cut(after, "`")
		if !ok {
			break
		}
		if rel := idleSelfImprovementCleanBacklogDocRefV0(value); rel != "" {
			refs = append(refs, rel)
		}
		rest = next
	}
	if len(refs) > 0 {
		return refs
	}
	for _, value := range strings.Fields(strings.TrimPrefix(line, "- ")) {
		if rel := idleSelfImprovementCleanBacklogDocRefV0(value); rel != "" {
			refs = append(refs, rel)
		}
	}
	return refs
}

func idleSelfImprovementCleanBacklogDocRefV0(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "`.,;:")
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "..") {
		return ""
	}
	value = filepath.ToSlash(filepath.Clean(value))
	if !strings.HasSuffix(value, ".md") || strings.HasPrefix(value, ".") {
		return ""
	}
	if !strings.HasPrefix(value, "docs/") && !strings.Contains(value, "/docs/") {
		return ""
	}
	return value
}
