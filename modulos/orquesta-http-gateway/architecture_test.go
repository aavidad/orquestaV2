package orquestahttpgateway

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestProductionImportsDoNotCrossForbiddenBoundaries(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob production files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		checkProductionImports(t, file)
	}
}

func checkProductionImports(t *testing.T, file string) {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}

	for _, importSpec := range parsed.Imports {
		importPath := unquoteImportPath(t, importSpec)
		if isForbiddenImportPath(importPath) {
			t.Fatalf("%s imports forbidden package %q", file, importPath)
		}
	}
}

func unquoteImportPath(t *testing.T, importSpec *ast.ImportSpec) string {
	t.Helper()

	importPath, err := strconv.Unquote(importSpec.Path.Value)
	if err != nil {
		t.Fatalf("unquote import %s: %v", importSpec.Path.Value, err)
	}

	return importPath
}

func isForbiddenImportPath(importPath string) bool {
	for _, prefix := range []string{
		"orquesta/modulos/orquesta-web",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-factory",
		"orquesta/modulos/orquesta-orchestration-core",
		"orquesta/runtimeagente",
	} {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}

	for _, segment := range strings.Split(importPath, "/") {
		switch segment {
		case "cmd", "db", "codex", "runtime":
			return true
		}
	}

	return false
}
