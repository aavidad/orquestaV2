package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOPESOperationalDocsGuardV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"modulos/orquesta-opes-bridge/README.md",
		"modulos/orquesta-server/README.md",
		"docs/opes_flujo_temario_operativo_2026-06-02.md",
		"docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md",
		"docs/runbooks/smoke_opes_domain_work_real_2026-05-13.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		blocks := shellBlocksWithNeedleV0(text, "ORQUESTA_OPES_BASE_URL=")
		if len(blocks) == 0 {
			t.Fatalf("%s no contiene comandos OPES revisables", rel)
		}
		for i, block := range blocks {
			if !strings.Contains(block, "ORQUESTA_OPES_TEMPORAL_CONFIRM=1") {
				t.Fatalf("%s bloque OPES %d sin ORQUESTA_OPES_TEMPORAL_CONFIRM=1:\n%s", rel, i+1, block)
			}
			if isEffectfulOPESCommandBlockV0(block) && !strings.Contains(block, "ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux") {
				t.Fatalf("%s bloque OPES con efectos %d sin app_server_tmux:\n%s", rel, i+1, block)
			}
			if isEffectfulOPESCommandBlockV0(block) && strings.Contains(block, "go run ./cmd/orquesta-server run") {
				t.Fatalf("%s bloque OPES con efectos %d usa runtime manual no gestionado:\n%s", rel, i+1, block)
			}
			if strings.Contains(block, "go run ./cmd/orquesta-server run") {
				t.Fatalf("%s bloque OPES %d usa runtime manual no gestionado:\n%s", rel, i+1, block)
			}
		}
	}
}

func TestOPESLocalTaskDocsGuardV0NoReabrenT12CerradoV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"modulos/orquesta-opes-bridge/docs/tareas.md",
		"modulos/orquesta-opes-connector/docs/tareas.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		if strings.Contains(text, "bloqueado verificable 2026-05-27") {
			t.Fatalf("%s reabre T12 stale como bloqueado 2026-05-27", rel)
		}
		if !strings.Contains(text, "resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md") ||
			!strings.Contains(text, "completed_syllabus_package") {
			t.Fatalf("%s no apunta al cierre funcional OPES goal-first 2026-06-28", rel)
		}
	}
}

func TestOPESConnectorDocsGuardV0T12CierreFuncionalVigenteV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"modulos/orquesta-opes-connector/docs/decisiones.md",
		"modulos/orquesta-opes-connector/docs/pruebas.md",
		"modulos/orquesta-opes-connector/docs/tareas.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		for _, stale := range []string{
			"T12 se declara bloqueada",
			"bloqueada por falta de entorno temporal",
			"bloqueado verificable 2026-05-27",
			"smoke real OPES sigue bloqueado",
			"sigue bloqueado sin OPES temporal",
		} {
			if strings.Contains(text, stale) {
				t.Fatalf("%s conserva narrativa stale de T12 bloqueada: %q", rel, stale)
			}
		}
		for _, required := range []string{
			"resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md",
			"completed_syllabus_package",
			"app_server_tmux",
			"OPES temporal",
			"scope duro",
		} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s no documenta requisito vigente %q", rel, required)
			}
		}
	}
}

func TestGoalOperationalDocsGuardV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"cmd/orquesta-server/server_env_registry_v0.go",
		"docs/matriz_pruebas_reales_y_smoke_2026-05-17.md",
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
		"docs/runbooks/handoff_opes_derivados_reales_hasta_local_html_site_2026-06-07.md",
		"modulos/orquesta-server/docs/contratos.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		if strings.Contains(text, "app_server_proxy` solo diagnostico opt-in") ||
			strings.Contains(text, "app_server_proxy` queda como diagnostico opt-in") ||
			strings.Contains(text, "app_server_proxy solo se permite como diagnostico breakglass") ||
			strings.Contains(text, "app_server_proxy` queda solo como diagnostico breakglass") ||
			strings.Contains(text, "app_server_proxy` queda solo como diagnostico opt-in") ||
			strings.Contains(text, "solo debe usarse como diagnostico aislado") {
			t.Fatalf("%s presenta app_server_proxy como diagnostico operativo vigente", rel)
		}
	}
	for _, rel := range []string{
		"docs/orquesta_goal_first_codex_2026-06-25.md",
		"docs/runbooks/smoke_goal_first_app_server_real_2026-06-25.md",
		"docs/runbooks/orquesta_self_programming_remoto_aislado_2026-06-29.md",
		"deploy/self-programming/README.md",
		"deploy/self-programming/goal_context.md",
		"deploy/self-programming/manual_codex_context.md",
		"modulos/orquesta-runtime-codex-goal/README.md",
		"modulos/orquesta-runtime-codex-goal/docs/contratos.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		if bad := goalOperationalDocStdioBackendClaimsV0(text); len(bad) > 0 {
			t.Fatalf("%s presenta stdio como backend Goal operativo: %v", rel, bad)
		}
	}
}

func readOperationalDocGuardV0(t *testing.T, root string, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("leer %s: %v", rel, err)
	}
	return string(data)
}

func shellBlocksWithNeedleV0(text string, needle string) []string {
	var blocks []string
	inBlock := false
	var current []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				block := strings.Join(current, "\n")
				if strings.Contains(block, needle) {
					blocks = append(blocks, block)
				}
				current = nil
				inBlock = false
				continue
			}
			inBlock = true
			current = nil
			continue
		}
		if inBlock {
			current = append(current, line)
		}
	}
	return blocks
}

func isEffectfulOPESCommandBlockV0(block string) bool {
	return strings.Contains(block, "ORQUESTA_OPES_BRIDGE_CONFIRM=1") ||
		strings.Contains(block, "ORQUESTA_OPES_BRIDGE_ENABLED=1") ||
		strings.Contains(block, "opes-temario-cycle")
}

func goalOperationalDocStdioBackendClaimsV0(text string) []string {
	var bad []string
	for _, line := range strings.Split(text, "\n") {
		if !goalOperationalDocLineMentionsStdioV0(line) ||
			goalOperationalDocLineRejectsStdioV0(line) {
			continue
		}
		bad = append(bad, strings.TrimSpace(line))
	}
	return bad
}

func goalOperationalDocLineMentionsStdioV0(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "app_server_stdio") ||
		strings.Contains(line, "--stdio") ||
		strings.Contains(line, "mcp_stdio") ||
		strings.Contains(line, "`stdio`") ||
		strings.Contains(line, " stdio")
}

func goalOperationalDocLineRejectsStdioV0(line string) bool {
	line = strings.ToLower(line)
	for _, marker := range []string{
		"no usa",
		"no uses",
		"no debe",
		"no se usa",
		"no se usan",
		"no se permite",
		"no aceptar",
		"sin stdio",
		"queda retirado",
		"retirado",
		"retirada",
		"bloquear",
		"no caer",
		"nunca",
		"historico",
		"historica",
		"no autorizado",
		"no operativo",
		"regresion",
	} {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}
