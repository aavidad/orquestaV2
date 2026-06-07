package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementLocalDocIntegrityV0DetectaIDsDuplicadosV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteLocalDocIntegrityTestV0(t, projectDir, "modulos/orquesta-app-codex-stack/docs/tareas.md", `# Tareas

## APP-CODEX-STACK-012

Estado: hecho.

## APP-CODEX-STACK-012

Estado: hecho.
`)

	collisions := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).localDocIntegrityCollisionsV0()
	collision := collisionByCodeForDocIntegrityTestV0(collisions, "duplicate_module_task_doc_id")
	if collision.Code == "" ||
		collision.SectionRef != "app-codex-stack-012" ||
		collision.Message != "path=modulos/orquesta-app-codex-stack/docs/tareas.md;id=APP-CODEX-STACK-012;lines=3,7" {
		t.Fatalf("collision=%+v all=%+v", collision, collisions)
	}
}

func TestIdleSelfImprovementLocalDocIntegrityV0DetectaPendienteConEvidenciaGoTestV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteLocalDocIntegrityTestV0(t, projectDir, "modulos/orquesta-cli/docs/pruebas.md", `# Pruebas

Caso: CLI-P001 ayuda sin efectos laterales
Ultima ejecucion: pendiente

Caso: CLI-P002 envelope JSON
Ultima ejecucion: 2026-05-26, go test -count=1 ./modulos/orquesta-cli; TestCliOutputEnvelopeV0JSONEstable.
`)

	collisions := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).localDocIntegrityCollisionsV0()
	collision := collisionByCodeForDocIntegrityTestV0(collisions, "stale_pending_local_test_execution")
	if collision.Code == "" ||
		collision.SectionRef != "cli-p001" ||
		collision.Message != "path=modulos/orquesta-cli/docs/pruebas.md;case=CLI-P001;line=4" {
		t.Fatalf("collision=%+v all=%+v", collision, collisions)
	}
}

func TestIdleSelfImprovementLocalDocIntegrityV0RailsOfflineNoBloqueaRutasLocalesPublicasV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteLocalDocIntegrityTestV0(t, projectDir, "modulos/orquesta-cli/docs/tareas.md", `# Tareas

## CLI-001

Evidencia publica: .orquesta-smoke-work/run.log
`)

	collisions := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).localDocIntegrityCollisionsV0()
	collision := collisionByCodeForDocIntegrityTestV0(collisions, "documentation_local_path_public_evidence")
	if collision.Code != "" {
		t.Fatalf("rail blando no debe bloquear con rails offline: collision=%+v all=%+v", collision, collisions)
	}
}

func TestIdleSelfImprovementLocalDocIntegrityV0PermiteVariablesYRefsOpacasV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteLocalDocIntegrityTestV0(t, projectDir, "modulos/orquesta-cli/docs/tareas.md", `# Tareas

## CLI-001

Usar ${ORQUESTA_SERVER_STATE_DIR} y state-dir-ref-cli como ejemplo opaco.
Diagnostico operador no exportable: /home/alberto/proyecto/local.log -> state-dir-ref-cli.
`)

	collisions := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).localDocIntegrityCollisionsV0()
	if collision := collisionByCodeForDocIntegrityTestV0(collisions, "documentation_local_path_public_evidence"); collision.Code != "" {
		t.Fatalf("collision inesperada=%+v all=%+v", collision, collisions)
	}
}

func collisionByCodeForDocIntegrityTestV0(
	collisions []orquestaserver.BacklogScanCollisionV0,
	code string,
) orquestaserver.BacklogScanCollisionV0 {
	for _, collision := range collisions {
		if collision.Code == code {
			return collision
		}
	}
	return orquestaserver.BacklogScanCollisionV0{}
}

func mustWriteLocalDocIntegrityTestV0(t *testing.T, projectDir string, rel string, content string) {
	t.Helper()
	path := filepath.Join(projectDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
