package main

import (
	"os"
	"path/filepath"
	"strings"
)

type idleSelfImprovementFederatedBacklogSourceV0 struct {
	SourcePath, SourceKind string
	Owner, State           string
	RelatedTXX             string
	Aliases, Tests         []string
	SourceLine             int
}

func (planner idleSelfImprovementBacklogPlannerV0) loadFederatedBacklogSectionsV0() []idleSelfImprovementBacklogSectionV0 {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil
	}
	sources := planner.federatedBacklogSourcesV0()
	var sections []idleSelfImprovementBacklogSectionV0
	for _, source := range sources {
		if !idleSelfImprovementFederatedBacklogSourceRunnableV0(source) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(projectDir, source.SourcePath))
		if err != nil {
			continue
		}
		sections = append(sections, idleSelfImprovementParseFederatedLocalSectionsV0(string(body), source)...)
	}
	return sections
}

func (planner idleSelfImprovementBacklogPlannerV0) federatedBacklogSourceRefsV0() []string {
	var refs []string
	for _, source := range planner.federatedBacklogSourcesV0() {
		refs = append(refs, source.SourcePath)
	}
	return compactServerStackStringsV0(refs)
}

func (planner idleSelfImprovementBacklogPlannerV0) federatedBacklogSourcesV0() []idleSelfImprovementFederatedBacklogSourceV0 {
	projectDir := strings.TrimSpace(planner.ProjectWorkDir)
	if projectDir == "" {
		return nil
	}
	body, err := os.ReadFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0))
	if err != nil {
		return nil
	}
	return idleSelfImprovementParseFederatedBacklogSourcesV0(string(body))
}

func idleSelfImprovementParseFederatedBacklogSourcesV0(content string) []idleSelfImprovementFederatedBacklogSourceV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var sources []idleSelfImprovementFederatedBacklogSourceV0
	inIndex := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inIndex = idleSelfImprovementFederatedBacklogIndexHeadingV0(trimmed)
			continue
		}
		if !inIndex || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		source := idleSelfImprovementParseFederatedBacklogSourceLineV0(trimmed)
		source.SourceLine = index + 1
		if source.SourcePath != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

func idleSelfImprovementFederatedBacklogIndexHeadingV0(heading string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(heading, "## ")))
	return strings.Contains(normalized, "indice federado") ||
		strings.Contains(normalized, "índice federado") ||
		strings.Contains(normalized, "federated backlog")
}

func idleSelfImprovementParseFederatedBacklogSourceLineV0(line string) idleSelfImprovementFederatedBacklogSourceV0 {
	fields := idleSelfImprovementFederatedBacklogFieldsV0(strings.TrimPrefix(line, "- "))
	return idleSelfImprovementFederatedBacklogSourceV0{
		SourcePath: idleSelfImprovementCleanBacklogDocRefV0(firstNonEmptyServerStackV0(
			fields["source_path"], fields["source"], fields["fuente"],
		)),
		SourceKind: firstNonEmptyServerStackV0(fields["source_kind"], fields["tipo"], "module_tasks"),
		Owner:      idleSelfImprovementCleanWriteSetEntryV0(firstNonEmptyServerStackV0(fields["owner"], fields["dueno"], fields["dueño"])),
		State:      strings.ToLower(firstNonEmptyServerStackV0(fields["state"], fields["estado"])),
		RelatedTXX: strings.ToUpper(firstNonEmptyServerStackV0(fields["related_txx"], fields["txx"], fields["relacionado"])),
		Aliases:    idleSelfImprovementFederatedCSVV0(firstNonEmptyServerStackV0(fields["aliases"], fields["alias"])),
		Tests:      idleSelfImprovementFederatedTestsV0(firstNonEmptyServerStackV0(fields["tests"], fields["pruebas"])),
	}
}

func idleSelfImprovementFederatedBacklogFieldsV0(text string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(text, ";") {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), "` ")
		if key != "" && value != "" {
			out[key] = value
		}
	}
	return out
}

func idleSelfImprovementFederatedCSVV0(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(strings.Trim(part, "` ")); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementFederatedTestsV0(value string) []string {
	var out []string
	rest := value
	for {
		_, after, ok := strings.Cut(rest, "`")
		if !ok {
			break
		}
		test, next, ok := strings.Cut(after, "`")
		if !ok {
			break
		}
		out = append(out, strings.TrimSpace(test))
		rest = next
	}
	if len(out) > 0 {
		return compactServerStackStringsV0(out)
	}
	return idleSelfImprovementFederatedCSVV0(value)
}

func idleSelfImprovementFederatedBacklogSourceRunnableV0(source idleSelfImprovementFederatedBacklogSourceV0) bool {
	state := strings.ToLower(strings.TrimSpace(source.State))
	return state == "vigente" || state == "promocionada" || state == "promoted" || state == "active"
}

func idleSelfImprovementParseFederatedLocalSectionsV0(
	content string,
	source idleSelfImprovementFederatedBacklogSourceV0,
) []idleSelfImprovementBacklogSectionV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var sections []idleSelfImprovementBacklogSectionV0
	seen := map[string]bool{}
	for index, line := range lines {
		alias, objective, ok := idleSelfImprovementFederatedLocalEntryV0(line, source.Aliases)
		if !ok || seen[alias] {
			continue
		}
		seen[alias] = true
		block := idleSelfImprovementFederatedLocalBlockV0(lines, index)
		if objective == "" {
			objective = idleSelfImprovementSectionValueV0(block, "Objetivo:")
		}
		section := idleSelfImprovementFederatedSectionV0(source, alias, objective, index+1)
		state := idleSelfImprovementSectionStateV0(block)
		if state.Completed {
			section.Completed = true
			section.StateEvidenceRefs = append(section.StateEvidenceRefs, state.EvidenceRefs...)
		}
		if idleSelfImprovementFederatedEntryCoveredV0(line, source) {
			section.Completed = true
			section.StateEvidenceRefs = append(section.StateEvidenceRefs, "evidence-ref-autoprogramming-backlog-federated-related-txx")
		}
		sections = append(sections, section)
	}
	return sections
}

func idleSelfImprovementFederatedLocalEntryV0(line string, aliases []string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "ID:") {
		token := strings.TrimSpace(strings.TrimPrefix(trimmed, "ID:"))
		return token, "", idleSelfImprovementFederatedAliasAllowedV0(token, aliases)
	}
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
	if trimmed == "" {
		return "", "", false
	}
	token, rest, _ := strings.Cut(trimmed, ":")
	token = strings.TrimSpace(token)
	if strings.Contains(token, " ") {
		token, rest, _ = strings.Cut(trimmed, " ")
	}
	if !idleSelfImprovementFederatedAliasAllowedV0(token, aliases) {
		return "", "", false
	}
	return token, strings.Trim(strings.TrimSpace(rest), "-: "), true
}

func idleSelfImprovementFederatedAliasAllowedV0(token string, aliases []string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	for _, alias := range aliases {
		prefix := strings.TrimSuffix(strings.TrimSpace(alias), "*")
		if prefix != "" && strings.HasPrefix(token, prefix) {
			return true
		}
	}
	return len(aliases) == 0 && strings.Contains(token, "-")
}

func idleSelfImprovementFederatedLocalBlockV0(lines []string, start int) []string {
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "ID:") {
			end = index
			break
		}
		if index > start+1 && strings.HasPrefix(trimmed, "```") {
			end = index
			break
		}
	}
	return lines[start:end]
}

func idleSelfImprovementFederatedSectionV0(
	source idleSelfImprovementFederatedBacklogSourceV0,
	alias string,
	objective string,
	line int,
) idleSelfImprovementBacklogSectionV0 {
	ref := idleSelfImprovementHeadingRefV0(alias)
	hash := idleSelfImprovementBacklogHashV0(source.SourcePath + "|" + alias + "|" + objective)
	section := idleSelfImprovementBacklogSectionV0{
		Ref:            ref,
		Heading:        alias + " " + objective,
		Objective:      objective,
		SourcePath:     source.SourcePath,
		SourceKind:     source.SourceKind,
		Owner:          source.Owner,
		LocalAlias:     alias,
		RelatedTXX:     source.RelatedTXX,
		LocalState:     source.State,
		LocalEntryHash: hash,
		Scope:          []string{source.Owner},
		Tests:          append([]string(nil), source.Tests...),
		SourceLine:     line,
	}
	if source.Owner == "" || len(source.Tests) == 0 {
		section.NeedsDocumentReview = true
		section.StateEvidenceRefs = append(section.StateEvidenceRefs, "evidence-ref-autoprogramming-backlog-federated-classification-required")
	}
	return section
}

func idleSelfImprovementFederatedEntryCoveredV0(
	line string,
	source idleSelfImprovementFederatedBacklogSourceV0,
) bool {
	upper := strings.ToUpper(line + " " + source.RelatedTXX)
	return strings.Contains(upper, "CUBIERT") && strings.Contains(upper, "T")
}

func idleSelfImprovementFederatedBacklogContextRefsV0(section idleSelfImprovementBacklogSectionV0) []string {
	if section.LocalAlias == "" {
		return nil
	}
	var refs []string
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_source_kind:", section.SourceKind)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_owner:", section.Owner)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_state:", section.LocalState)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_local_alias:", section.LocalAlias)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_related_txx:", section.RelatedTXX)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_local_entry_hash:", section.LocalEntryHash)
	return compactServerStackStringsV0(refs)
}

func appendFederatedBacklogContextRefV0(refs []string, prefix string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return refs
	}
	return append(refs, prefix+strings.TrimSpace(value))
}
