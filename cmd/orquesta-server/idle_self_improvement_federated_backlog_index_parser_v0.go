package main

import "strings"

func idleSelfImprovementParseFederatedBacklogSourcesV0(content string) []idleSelfImprovementFederatedBacklogSourceV0 {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var sources []idleSelfImprovementFederatedBacklogSourceV0
	inIndex := false
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inIndex = idleSelfImprovementFederatedBacklogIndexHeadingV0(trimmed)
			continue
		}
		if !inIndex || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		sourceLine := index + 1
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		cursor := index + 1
		for ; cursor < len(lines); cursor++ {
			next := strings.TrimSpace(lines[cursor])
			if next == "" || strings.HasPrefix(next, "## ") || strings.HasPrefix(next, "- ") {
				break
			}
			item += " " + next
		}
		index = cursor - 1
		source := idleSelfImprovementParseFederatedBacklogSourceLineV0("- " + item)
		source.SourceLine = sourceLine
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
		value = strings.TrimSpace(value)
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
