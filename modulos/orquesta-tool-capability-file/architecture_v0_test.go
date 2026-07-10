package orquestatoolcapabilityfile

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestToolOperationFileStoreDoesNotImportProductRuntimeV0(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "store_v0.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range parsed.Imports {
		path := strings.Trim(spec.Path.Value, "\"")
		for _, forbidden := range []string{
			"orquesta/modulos/orquesta-app-codex-stack",
			"orquesta/modulos/orquesta-mcp",
			"orquesta/modulos/orquesta-orchestration-core",
			"orquesta/modulos/orquesta-runtime",
			"net/http",
			"os/exec",
		} {
			if path == forbidden || strings.HasPrefix(path, forbidden+"/") {
				t.Fatalf("adapter imports forbidden dependency %s", path)
			}
		}
	}
}
