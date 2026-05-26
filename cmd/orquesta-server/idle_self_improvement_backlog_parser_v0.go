package main

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

type idleSelfImprovementBacklogSectionV0 struct {
	Ref, Heading, Objective        string
	SourcePath                     string
	SourceKind, Owner              string
	LocalAlias, RelatedTXX         string
	LocalState, LocalEntryHash     string
	Scope, Criteria, Tests         []string
	ManualVerifications            []string
	Dependencies, Inputs, Outputs  []string
	StateEvidenceRefs              []string
	Completed, NeedsDocumentReview bool
	SourceLine                     int
	SourceIndexLine                int
}

type idleSelfImprovementBacklogStateV0 struct {
	Completed           bool
	NeedsDocumentReview bool
	EvidenceRefs        []string
}

func parseIdleSelfImprovementBacklogSectionsV0(content string) []idleSelfImprovementBacklogSectionV0 {
	return parseIdleSelfImprovementBacklogSectionsFromDocumentV0(content, idleSelfImprovementBacklogDocRelV0)
}

func parseIdleSelfImprovementBacklogSectionsFromDocumentV0(
	content string,
	sourcePath string,
) []idleSelfImprovementBacklogSectionV0 {
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
		section := idleSelfImprovementParseBacklogSectionV0(heading, lines[index+1:next], index+1)
		section.SourcePath = firstNonEmptyServerStackV0(strings.TrimSpace(sourcePath), idleSelfImprovementBacklogDocRelV0)
		sections = append(sections, section)
	}
	return sections
}

func idleSelfImprovementParseBacklogSectionV0(
	heading string,
	lines []string,
	sourceLine int,
) idleSelfImprovementBacklogSectionV0 {
	state := idleSelfImprovementSectionStateV0(lines)
	tests, manualVerifications := idleSelfImprovementSectionTestsV0(lines)
	return idleSelfImprovementBacklogSectionV0{
		Ref:                 idleSelfImprovementHeadingRefV0(heading),
		Heading:             strings.TrimPrefix(heading, "## "),
		Objective:           idleSelfImprovementSectionValueV0(lines, "Objetivo:"),
		Scope:               idleSelfImprovementSectionListV0(lines, "Alcance:"),
		Criteria:            idleSelfImprovementSectionListV0(lines, "Criterios:"),
		Dependencies:        idleSelfImprovementSectionDependenciesV0(lines),
		Inputs:              idleSelfImprovementSectionIOV0(lines, "Entrada:", "Entradas:"),
		Outputs:             idleSelfImprovementSectionIOV0(lines, "Salida:", "Salidas:"),
		Tests:               tests,
		ManualVerifications: manualVerifications,
		StateEvidenceRefs:   state.EvidenceRefs,
		Completed:           state.Completed,
		NeedsDocumentReview: state.NeedsDocumentReview,
		SourceLine:          sourceLine,
	}
}

func idleSelfImprovementSectionStateV0(lines []string) idleSelfImprovementBacklogStateV0 {
	stateValue := strings.ToLower(idleSelfImprovementSectionValueV0(lines, "Estado:"))
	if idleSelfImprovementBacklogTextContainsAnyV0(stateValue,
		"completada", "completado", "cerrada", "cerrado", "hecha", "hecho", "done",
	) {
		return idleSelfImprovementBacklogStateV0{
			Completed:    true,
			EvidenceRefs: []string{"evidence-ref-autoprogramming-backlog-state-canonical"},
		}
	}
	if strings.TrimSpace(stateValue) != "" {
		return idleSelfImprovementBacklogStateV0{}
	}
	if idleSelfImprovementSectionHasAmbiguousEvidenceV0(lines) {
		return idleSelfImprovementBacklogStateV0{
			NeedsDocumentReview: true,
			EvidenceRefs:        []string{"evidence-ref-autoprogramming-backlog-ambiguous-evidence"},
		}
	}
	return idleSelfImprovementBacklogStateV0{}
}

func idleSelfImprovementSectionHasAmbiguousEvidenceV0(lines []string) bool {
	text := strings.ToLower(strings.Join(lines, "\n"))
	return idleSelfImprovementBacklogTextContainsAnyV0(text,
		"evidencia focal:", "evidencia ejecutada", "evidencia local:",
		"validacion focal:", "validación focal:",
	)
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
		"mejora futura", "futuro", "futura", "experimental",
		"rotacion de sesiones", "rotación de sesiones",
		"session rotation", "handoff estructurado",
	) {
		return 3
	}
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"nucleo", "núcleo", "core", "director", "automejora",
		"autoprogramacion", "autoprogramación", "self-improvement",
		"self improvement", "orchestration-core", "orquesta-core",
		"orquesta-director", "orquesta-autoprogramming",
	) {
		return 0
	}
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"claude", "gemini", "ollama", "vllm", "conector de agente",
		"conectores de agentes", "agent connector", "agent connectors",
	) {
		return 1
	}
	if idleSelfImprovementBacklogTextContainsAnyV0(text,
		"web", "opes", "cli", "mcp", "complemento", "complementos",
		"plugin", "plugins", "modulos/orquesta-web", "modulos/orquesta-opes",
		"modulos/orquesta-cli", "modulos/orquesta-mcp",
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

func idleSelfImprovementSectionDependenciesV0(lines []string) []string {
	values := append(
		idleSelfImprovementSectionListV0(lines, "Depende:"),
		idleSelfImprovementSectionListV0(lines, "Dependencias:")...,
	)
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
			if normalized := idleSelfImprovementNormalizeDependencyRefV0(part); normalized != "" {
				out = append(out, normalized)
			}
		}
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementSectionIOV0(lines []string, labels ...string) []string {
	var out []string
	for _, label := range labels {
		out = append(out, idleSelfImprovementSectionListV0(lines, label)...)
	}
	return compactServerStackStringsV0(out)
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
