package main

import (
	"strings"
	"testing"
)

func TestModuleBoundaryLocalDocCoverageT133V0RepoActualCubiertoV0(t *testing.T) {
	repoRoot := findRepoRootForResidualGoFileBudgetTestV0(t)
	collisions := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir: repoRoot,
	}).moduleBoundaryLocalAgentDocCollisionsV0(repoRoot)
	if len(collisions) > 0 {
		var messages []string
		for _, collision := range collisions {
			messages = append(messages, collision.Message)
		}
		t.Fatalf("modulos sensibles sin guia local publica: %s", strings.Join(messages, "; "))
	}
}

func TestModuleBoundaryLocalDocCoverageT133V0DetectaModuloSinGuiaV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteLocalDocIntegrityTestV0(
		t,
		projectDir,
		"modulos/orquesta-rails/README.md",
		"# orquesta-rails\n\nResumen sin contrato local para agentes.\n",
	)

	collisions := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir: projectDir,
	}).moduleBoundaryLocalAgentDocCollisionsV0(projectDir)
	collision := collisionByCodeForDocIntegrityTestV0(
		collisions,
		"module_boundary_local_agent_doc_missing",
	)
	if collision.Code == "" ||
		collision.SectionRef != "modulos/orquesta-rails" ||
		!strings.Contains(collision.Message, "missing=AGENTS.md") {
		t.Fatalf("collision=%+v all=%+v", collision, collisions)
	}
}
