package orquestadirectoroperativo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorOperativoDocsV0NoReabrenSmokesCerrados(t *testing.T) {
	root := repoRootFromDirectorOperativoTestV0(t)
	localDocs := map[string]string{
		"README":    "modulos/orquesta-director-operativo/README.md",
		"contratos": "modulos/orquesta-director-operativo/docs/contratos.md",
		"tareas":    "modulos/orquesta-director-operativo/docs/tareas.md",
		"pruebas":   "modulos/orquesta-director-operativo/docs/pruebas.md",
	}
	for name, rel := range localDocs {
		body := readRepoTextV0(t, root, rel)
		for _, stale := range []string{
			"quedan pendientes wait por cohorte/ola",
			"Lo pendiente de este frente no es mas contrato puro, sino smoke con\nCodex real amplio/recursivo",
		} {
			if strings.Contains(body, stale) {
				t.Fatalf("%s reabre estado stale %q", name, stale)
			}
		}
	}

	requiredEvidence := map[string][]string{
		"modulos/orquesta-director-operativo/README.md": {
			"WaitAgentRefs",
			"OPES temporal real de derivados/cierre",
		},
		"modulos/orquesta-director-operativo/docs/tareas.md": {
			"CODEX-WAVE-REAL",
			"CODEX-RECURSION-REAL",
			"OPES temporal real de derivados/cierre",
		},
		"modulos/orquesta-director-operativo/docs/pruebas.md": {
			"CODEX-WAVE-REAL",
			"CODEX-RECURSION-REAL",
			"OPES temporal de derivados/cierre",
		},
		"docs/director_operativo_v1_2026-05-17.md": {
			"CODEX-WAVE-REAL",
			"CODEX-RECURSION-REAL",
			"OPES temporal real de\nderivados/cierre quedo cerrado funcionalmente por goal-first",
		},
		"docs/corte_cierre_generico_director_operativo_2026-05-17.md": {
			"CODEX-WAVE-REAL",
			"CODEX-RECURSION-REAL",
			"OPES temporal real de derivados/cierre quedo cerrado\nfuncionalmente por goal-first",
		},
	}
	for rel, snippets := range requiredEvidence {
		body := readRepoTextV0(t, root, rel)
		for _, snippet := range snippets {
			if !strings.Contains(body, snippet) {
				t.Fatalf("%s no conserva evidencia vigente %q", rel, snippet)
			}
		}
	}
}

func repoRootFromDirectorOperativoTestV0(t *testing.T) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("repo root not found from test cwd: %v", err)
	}
	return root
}

func readRepoTextV0(t *testing.T, root string, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(body)
}
