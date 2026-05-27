package main

import "strings"

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
		if state.PendingExplicit {
			section.PendingExplicit = true
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
		Ref:             ref,
		Heading:         alias + " " + objective,
		Objective:       objective,
		SourcePath:      source.SourcePath,
		SourceKind:      source.SourceKind,
		Owner:           source.Owner,
		LocalAlias:      alias,
		RelatedTXX:      source.RelatedTXX,
		LocalState:      source.State,
		LocalEntryHash:  hash,
		Scope:           []string{source.Owner},
		Tests:           append([]string(nil), source.Tests...),
		SourceLine:      line,
		SourceIndexLine: source.SourceLine,
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
