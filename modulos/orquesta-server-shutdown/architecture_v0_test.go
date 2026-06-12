package orquestaservershutdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerShutdownModuleV0NoImportaAdaptadoresConcretos(t *testing.T) {
	forbidden := []string{
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-web",
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-runtime-codex",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/cmd/",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, value := range forbidden {
			if strings.Contains(string(raw), value) {
				t.Fatalf("%s importa adaptador prohibido %s", file, value)
			}
		}
	}
}

func TestServerShutdownModuleV0MantieneFicherosGoBajoPresupuesto(t *testing.T) {
	const maxLines = 299

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		lines := strings.Count(string(raw), "\n")
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			lines++
		}
		if lines > maxLines {
			t.Fatalf("%s tiene %d lineas; presupuesto maximo %d", file, lines, maxLines)
		}
	}
}
