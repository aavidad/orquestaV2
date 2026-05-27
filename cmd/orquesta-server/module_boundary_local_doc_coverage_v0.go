package main

import (
	"os"
	"path/filepath"
	"strings"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type moduleBoundaryLocalDocSpecV0 struct {
	ModuleRef string
	Reason    string
}

var moduleBoundaryLocalDocSpecsV0 = []moduleBoundaryLocalDocSpecV0{
	{ModuleRef: "modulos/orquesta-rails", Reason: "rails"},
	{ModuleRef: "modulos/orquesta-domain-work-http", Reason: "http"},
	{ModuleRef: "modulos/orquesta-factory-http", Reason: "factory-http"},
	{ModuleRef: "modulos/orquesta-context", Reason: "context-boundary"},
	{ModuleRef: "cmd/orquesta-server", Reason: "composition-root"},
}

func (planner idleSelfImprovementBacklogPlannerV0) moduleBoundaryLocalAgentDocCollisionsV0(
	projectDir string,
) []orquestaserver.BacklogScanCollisionV0 {
	var collisions []orquestaserver.BacklogScanCollisionV0
	for _, spec := range moduleBoundaryLocalDocSpecsV0 {
		ok, sourceRef := moduleBoundaryLocalDocCoveredV0(projectDir, spec.ModuleRef)
		if ok {
			continue
		}
		message := "module=" + spec.ModuleRef + ";reason=" + spec.Reason + ";missing=AGENTS.md"
		if sourceRef != "" {
			message += ";source=" + sourceRef + ";issue=insufficient_local_agent_doc"
		}
		collisions = append(collisions, orquestaserver.BacklogScanCollisionV0{
			Code:       "module_boundary_local_agent_doc_missing",
			SectionRef: spec.ModuleRef,
			Message:    message,
			EvidenceRefs: []string{
				"evidence-ref-autoprogramming-module-boundary-local-agent-doc-coverage",
				"evidence-ref-autoprogramming-t133",
			},
		})
	}
	return collisions
}

func moduleBoundaryLocalDocCoveredV0(projectDir string, moduleRef string) (bool, string) {
	agentRef := filepath.ToSlash(filepath.Join(moduleRef, "AGENTS.md"))
	if moduleBoundaryLocalDocFileSufficientV0(filepath.Join(projectDir, filepath.FromSlash(agentRef))) {
		return true, agentRef
	}
	readmeRef := filepath.ToSlash(filepath.Join(moduleRef, "README.md"))
	if moduleBoundaryLocalDocFileSufficientV0(filepath.Join(projectDir, filepath.FromSlash(readmeRef))) {
		return true, readmeRef
	}
	return false, readmeRef
}

func moduleBoundaryLocalDocFileSufficientV0(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return moduleBoundaryLocalDocContentSufficientV0(string(body))
}

func moduleBoundaryLocalDocContentSufficientV0(content string) bool {
	normalized := strings.ToLower(content)
	required := []string{
		"responsabilidad",
		"capa",
		"imports prohibidos",
		"pruebas focales",
		"docs vigentes",
	}
	for _, token := range required {
		if !strings.Contains(normalized, token) {
			return false
		}
	}
	return true
}
