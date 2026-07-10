package orquestadocumentextraction_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDocumentExtractionArchitectureV0DoesNotImportAdapters(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", "orquesta/modulos/orquesta-document-extraction")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, output)
	}
	for _, imported := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		for _, forbidden := range []string{"database/sql", "net/http", "os", "orquesta/cmd", "orquesta/modulos/orquesta-document-extraction-csv", "orquesta/modulos/orquesta-document-extraction-fake", "orquesta/modulos/orquesta-document-extraction-json", "orquesta/modulos/orquesta-mcp", "orquesta/modulos/orquesta-runtime"} {
			if imported == forbidden || strings.Contains(imported, forbidden) {
				t.Fatalf("forbidden import %q", imported)
			}
		}
	}
}
