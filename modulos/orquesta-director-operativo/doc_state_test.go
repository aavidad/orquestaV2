package orquestadirectoroperativo

import (
	"os"
	"strings"
	"testing"
)

func TestDirectorOperativoLocalDocsAlineadosConFotoVigenteV0(t *testing.T) {
	docs := []string{
		"README.md",
		"docs/contratos.md",
		"docs/tareas.md",
		"docs/pruebas.md",
	}
	for _, path := range docs {
		body := readLocalDocV0(t, path)
		for _, stale := range []string{
			"quedan pendientes wait por cohorte/ola",
			"recursion Codex real con parent/child refs",
			"smoke con Codex real amplio/recursivo",
		} {
			if strings.Contains(body, stale) {
				t.Fatalf("%s conserva contradiccion stale %q", path, stale)
			}
		}
	}

	readme := readLocalDocV0(t, "README.md")
	for _, required := range []string{
		"CODEX-WAVE-REAL",
		"CODEX-RECURSION-REAL",
		"OPES temporal real de derivados/cierre",
		"contrato puro",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README.md no declara estado vigente %q", required)
		}
	}
}

func readLocalDocV0(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
