package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaserver "orquesta/modulos/orquesta-server"
)

var localDocPathRedactionForbiddenFragmentsV0 = []string{
	"/home/",
	"\\home\\",
	"/users/",
	"\\users\\",
	"c:\\users\\",
	"$home",
	"~/",
	".orquesta-runtime",
	".orquesta-server",
	".orquesta-smoke-work",
	".orquesta-codex-runtime",
	".orquesta-local-runtime",
	".orquesta-control",
	".orquesta-runs",
	".orquesta-worktrees",
	"agent_ack.json",
	"agent_packet.json",
	"agent_prompt.txt",
	"agent_shutdown_checkpoint_ack.json",
	"director_decisions.json",
	"orquesta_shutdown_request.json",
	"codex_stdout.log",
	"codex_stderr.log",
	"codex_last_message.txt",
	"server.log",
	"logs/",
	".ssl-key.log",
	"transcript=",
	"raw_transcript=",
}

func (planner idleSelfImprovementBacklogPlannerV0) localDocPathRedactionCollisionsV0(
	projectDir string,
) []orquestaserver.BacklogScanCollisionV0 {
	if !orquestarails.RailsEnforcedV0() {
		return nil
	}
	var collisions []orquestaserver.BacklogScanCollisionV0
	for _, rel := range localDocPathRedactionRefsV0(projectDir) {
		body, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			continue
		}
		for _, line := range localDocPathRedactionProblemLinesV0(string(body)) {
			collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
				Code:       "documentation_local_path_public_evidence",
				SectionRef: "documentation-local-path-redaction",
				Message:    "path=" + rel + ";line=" + strconv.Itoa(line) + ";kind=local_path_public_evidence",
				EvidenceRefs: []string{
					"evidence-ref-autoprogramming-local-doc-integrity-linter",
					"evidence-ref-autoprogramming-local-doc-path-redaction",
				},
			})
		}
	}
	return collisions
}

func localDocPathRedactionRefsV0(projectDir string) []string {
	refs := []string{
		idleSelfImprovementBacklogDocRelV0,
		"docs/rail_errors_observados_2026-05-23.md",
		"docs/duplicaciones_railes_pendientes_2026-05-24.md",
	}
	for _, pattern := range []string{
		filepath.Join(projectDir, "modulos", "*", "docs", "*.md"),
		filepath.Join(projectDir, "modulos", "*", "README.md"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			rel, err := filepath.Rel(projectDir, match)
			if err == nil {
				refs = append(refs, filepath.ToSlash(rel))
			}
		}
	}
	return compactServerStackStringsV0(refs)
}

func localDocPathRedactionProblemLinesV0(content string) []int {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []int
	for index, line := range lines {
		if localDocPathLineNeedsRedactionV0(line) {
			out = append(out, index+1)
		}
	}
	return out
}

func localDocPathLineNeedsRedactionV0(line string) bool {
	if !orquestarails.RailsEnforcedV0() {
		return false
	}
	normalized := strings.ToLower(strings.ReplaceAll(line, `\/`, "/"))
	if localDocPathLineMarkedNonExportableV0(normalized) {
		return false
	}
	for _, fragment := range localDocPathRedactionForbiddenFragmentsV0 {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func localDocPathLineMarkedNonExportableV0(normalized string) bool {
	for _, marker := range []string{
		"no exportable",
		"no_exportable",
		"no se exporta",
		"diagnostico operador",
		"diagnóstico operador",
		"opt-in local",
		"solo local",
		"redactado",
		"redacted",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
