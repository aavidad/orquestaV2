package acceptance_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	commandcore "orquesta/internal/commands"
)

func v20ProductionStructFields(t *testing.T, directory, typeName string) (map[string]bool, bool) {
	t.Helper()
	result := make(map[string]bool)
	found := false
	v20WalkGo(t, directory, false, func(_ string, file *ast.File) {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				found = true
				for _, field := range structure.Fields.List {
					for _, name := range field.Names {
						result[name.Name] = true
					}
				}
			}
		}
	})
	if !found && typeName == "CommandAuditRecord" {
		underlying := reflect.TypeOf(commandcore.CommandAuditRecord{})
		if underlying.Kind() == reflect.Struct {
			found = true
			for index := 0; index < underlying.NumField(); index++ {
				result[underlying.Field(index).Name] = true
			}
		}
	}
	return result, found
}

func v20ProductionMethods(t *testing.T, directory, receiver string) map[string]bool {
	t.Helper()
	result := make(map[string]bool)
	v20WalkGo(t, directory, false, func(_ string, file *ast.File) {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || len(function.Recv.List) != 1 {
				continue
			}
			typeExpression := function.Recv.List[0].Type
			if pointer, ok := typeExpression.(*ast.StarExpr); ok {
				typeExpression = pointer.X
			}
			if identifier, ok := typeExpression.(*ast.Ident); ok && identifier.Name == receiver {
				result[function.Name.Name] = true
			}
		}
	})
	return result
}

func v20ProductionFunctions(t *testing.T, path string) map[string]bool {
	t.Helper()
	result := make(map[string]bool)
	v20WalkGo(t, path, false, func(_ string, parsed *ast.File) {
		for _, declaration := range parsed.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
				result[function.Name.Name] = true
			}
		}
	})
	return result
}

func v20ReadProductionGo(t *testing.T, directory string) string {
	t.Helper()
	var result strings.Builder
	v20WalkGo(t, directory, false, func(path string, _ *ast.File) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		result.Write(content)
		result.WriteByte('\n')
	})
	return result.String()
}

func v20ReadTestGo(t *testing.T, repositoryRoot string) string {
	t.Helper()
	var result strings.Builder
	for _, relative := range []string{
		"internal/commands", "internal/interfaces/httpapi", "internal/interfaces/mcp",
		"internal/interfaces/cli", "internal/adapters/state/sqlite", "internal/bootstrap", "sdk/commands",
	} {
		v20WalkGo(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)), true, func(path string, _ *ast.File) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result.Write(content)
			result.WriteByte('\n')
		})
	}
	return result.String()
}

func v20TestFunctionNames(t *testing.T, repositoryRoot string) map[string]struct{} {
	t.Helper()
	names := make(map[string]struct{})
	for _, relative := range []string{
		"acceptance", "internal/commands", "internal/interfaces/httpapi", "internal/interfaces/mcp",
		"internal/interfaces/cli", "internal/adapters/state/sqlite", "internal/bootstrap", "internal/i18n", "sdk/commands",
	} {
		v20WalkGo(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)), true, func(_ string, file *ast.File) {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Name == nil {
					continue
				}
				names[function.Name.Name] = struct{}{}
			}
		})
	}
	return names
}

func v20WalkGo(t *testing.T, root string, tests bool, visit func(string, *ast.File)) {
	t.Helper()
	info, err := os.Stat(root)
	if err != nil {
		return
	}
	walk := func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") != tests {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}
		visit(path, parsed)
		return nil
	}
	if !info.IsDir() {
		if filepath.Ext(root) != ".go" || strings.HasSuffix(root, "_test.go") != tests {
			return
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), root, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		visit(root, parsed)
		return
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatal(err)
	}
}
