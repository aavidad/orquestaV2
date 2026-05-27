package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectTreeScanHasFileV0PresupuestoYControlFiles(t *testing.T) {
	root := t.TempDir()
	writeFileForProjectTreeScanTestV0(t, root, ".orquesta-runtime/agent_ack.json", "{}")
	writeFileForProjectTreeScanTestV0(t, root, "internal/api/server.go", "package api\n")

	result := ProjectTreeScanHasFileV0(context.Background(), ProjectTreeScanRequestV0{
		ProjectRoot: root,
		Target:      "**",
		Mode:        ProjectTreeScanModeGlobV0,
		MaxEntries:  1,
	})
	if result.Found || !result.BudgetExhausted || result.ReasonCode != ProjectTreeScanReasonBudgetExhaustedV0 {
		t.Fatalf("scan amplio sin presupuesto esperado: %+v", result)
	}

	result = ProjectTreeScanHasFileV0(context.Background(), ProjectTreeScanRequestV0{
		ProjectRoot: root,
		Target:      "**/*.go",
		Mode:        ProjectTreeScanModeGlobV0,
		MaxEntries:  20,
	})
	if !result.Found || result.MatchedPath != "internal/api/server.go" {
		t.Fatalf("glob seguro no encontro fichero producto: %+v", result)
	}

	result = ProjectTreeScanHasFileV0(context.Background(), ProjectTreeScanRequestV0{
		ProjectRoot: root,
		Target:      ".orquesta-runtime",
		Mode:        ProjectTreeScanModeDirV0,
		MaxEntries:  20,
	})
	if result.Found {
		t.Fatalf("control dir no debe contar como evidencia producto: %+v", result)
	}
}

func TestProjectTreeScanFilesV0ListaDirAcotado(t *testing.T) {
	root := t.TempDir()
	writeFileForProjectTreeScanTestV0(t, root, "docs/a.md", "a\n")
	writeFileForProjectTreeScanTestV0(t, root, "docs/b.md", "b\n")
	writeFileForProjectTreeScanTestV0(t, root, "docs/codex_stdout.log", "log\n")

	result := ProjectTreeScanFilesV0(context.Background(), ProjectTreeScanRequestV0{
		ProjectRoot: root,
		Target:      "docs",
		Mode:        ProjectTreeScanModeDirV0,
		MaxEntries:  20,
		MaxResults:  10,
	})
	if !result.Found || len(result.MatchedPaths) != 2 {
		t.Fatalf("recovery dir debe listar solo producto: %+v", result)
	}
}

func TestProjectTreeScanHasFileV0RechazaRutasInseguras(t *testing.T) {
	root := t.TempDir()
	for _, target := range []string{"../secret", "/tmp/secret", "$HOME/.codex", "https://example.test/a"} {
		result := ProjectTreeScanHasFileV0(context.Background(), ProjectTreeScanRequestV0{
			ProjectRoot: root,
			Target:      target,
		})
		if result.ReasonCode != ProjectTreeScanReasonInvalidRequestV0 {
			t.Fatalf("target inseguro %q aceptado: %+v", target, result)
		}
	}
}

func writeFileForProjectTreeScanTestV0(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
