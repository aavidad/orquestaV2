package orquestaweb

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNuevaAppNoDBNoRuntimeGuardV0(t *testing.T) {
	files := nuevaAppProductionGoFilesV0(t)
	for _, file := range files {
		parsed := parseNuevaAppGoFileV0(t, file)
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if nuevaAppForbiddenImportV0(path) {
				t.Fatalf("%s importa dependencia prohibida para web fina: %s", file, path)
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value := strings.Trim(lit.Value, "`\"")
			for _, forbidden := range []string{
				"fabricar-app",
				"/api/proyectos/",
				"materializar_backlog",
				"materializar backlog",
				"runtime_provider",
				"agent_id",
			} {
				if strings.Contains(strings.ToLower(value), forbidden) {
					t.Fatalf("%s contiene literal heredado/prohibido %q en %q", file, forbidden, value)
				}
			}
			return true
		})
	}
}

func nuevaAppProductionGoFilesV0(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	out := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "nueva_app_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, filepath.Clean(name))
	}
	if len(out) == 0 {
		t.Fatalf("no hay ficheros productivos nueva_app_*.go para validar")
	}
	return out
}

func parseNuevaAppGoFileV0(t *testing.T, path string) *ast.File {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return parsed
}

func nuevaAppForbiddenImportV0(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	for _, forbidden := range []string{
		"orquesta/db",
		"orquesta/db/",
		"orquesta/runtime",
		"orquesta/runtime/",
		"orquesta/runtimeagente",
		"orquesta/runtimeagente/",
		"orquesta/cmd",
		"orquesta/cmd/",
		"orquesta/fabricaapp",
		"orquesta/fabricaapp/",
	} {
		if normalized == strings.TrimSuffix(forbidden, "/") || strings.HasPrefix(normalized, forbidden) {
			return true
		}
	}
	return false
}
