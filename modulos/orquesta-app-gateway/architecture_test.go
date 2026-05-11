package orquestaappgateway

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestProductionImportsNoLegacyOrConcreteRuntimeV0(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("unquote %s: %v", imported.Path.Value, err)
			}
			if forbiddenProductionImportV0(path) {
				t.Fatalf("%s importa dependencia prohibida: %s", file, path)
			}
		}
	}
}

func forbiddenProductionImportV0(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	for _, forbidden := range []string{
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/runtimeagente",
		"orquesta/modulos/orquesta-orchestration-core",
		"github.com/spf13/cobra",
		"github.com/go-sql-driver/mysql",
		"github.com/jackc/pgx",
		"modernc.org/sqlite",
	} {
		if normalized == forbidden || strings.HasPrefix(normalized, forbidden+"/") {
			return true
		}
	}
	for _, marker := range []string{"runtime-codex", "runtime-agentcli"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
