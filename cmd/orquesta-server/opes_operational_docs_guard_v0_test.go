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
		}
	}
}

func TestGoalOperationalDocsGuardV0(t *testing.T) {
	root := findRepoRootForResidualGoFileBudgetTestV0(t)
	for _, rel := range []string{
		"docs/matriz_pruebas_reales_y_smoke_2026-05-17.md",
		"docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
	} {
		text := readOperationalDocGuardV0(t, root, rel)
		if strings.Contains(text, "app_server_proxy` solo diagnostico opt-in") ||
			strings.Contains(text, "app_server_proxy` queda como diagnostico opt-in") {
			t.Fatalf("%s presenta app_server_proxy como diagnostico operativo vigente", rel)
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
