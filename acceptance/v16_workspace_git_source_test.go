package acceptance_test

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"orquesta/internal/application"
)

func v16RequireOutboundInterface(t *testing.T, directory string, required v16RequiredPort) {
	t.Helper()
	methods, found := v16InterfaceMethods(t, directory, required.Name)
	if !found {
		t.Errorf("V16_RED application outbound port %s missing", required.Name)
		return
	}
	for _, name := range required.Methods {
		signature, ok := methods[name]
		if !ok {
			t.Errorf("V16_RED %s lacks %s", required.Name, name)
			continue
		}
		if !strings.Contains(signature, "context.Context") || strings.Count(signature, "ports.") < 2 ||
			!strings.Contains(signature, "error") {
			t.Errorf("V16_RED %s.%s must use context plus neutral ports request/result: %s", required.Name, name, signature)
		}
	}
}

func v16InterfaceMethods(t *testing.T, directory, typeName string) (map[string]string, bool) {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typeName {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					return nil, true
				}
				methods := make(map[string]string)
				for _, field := range iface.Methods.List {
					if len(field.Names) != 1 {
						continue
					}
					var rendered bytes.Buffer
					if err := format.Node(&rendered, token.NewFileSet(), field.Type); err != nil {
						t.Fatalf("render %s.%s: %v", typeName, field.Names[0].Name, err)
					}
					methods[field.Names[0].Name] = rendered.String()
				}
				return methods, true
			}
		}
	}
	return nil, false
}

func v16RequireUseCase(t *testing.T, orchestrator reflect.Type, name string) {
	t.Helper()
	method, ok := orchestrator.MethodByName(name)
	if !ok {
		t.Errorf("V16_RED Orchestrator lacks %s", name)
		return
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if method.Type.NumIn() != 4 || method.Type.In(1) != contextType ||
		method.Type.In(2) != reflect.TypeOf(application.Access{}) ||
		method.Type.In(3).Name() != name+"Request" || method.Type.NumOut() != 2 ||
		method.Type.Out(0).Name() != name+"Result" || method.Type.Out(1) != errorType {
		t.Errorf("V16_RED %s signature=%s", name, method.Type)
	}
}

func v16StructFieldTypeCount(t *testing.T, directory, typeName, fieldType string) int {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
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
					return 0
				}
				count := 0
				for _, field := range structure.Fields.List {
					identifier, ok := field.Type.(*ast.Ident)
					if ok && identifier.Name == fieldType {
						count += len(field.Names)
					}
				}
				return count
			}
		}
	}
	return 0
}

func v16ProductionTypeExists(t *testing.T, directory, typeName string) bool {
	t.Helper()
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == typeName {
					return true
				}
			}
		}
	}
	return false
}

func v16ReadProductionGo(t *testing.T, directory string) string {
	t.Helper()
	var source strings.Builder
	for _, file := range v16ProductionGoFiles(t, directory, false) {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source.Write(content)
		source.WriteByte('\n')
	}
	return source.String()
}

func v16ReadProductionGoOptional(t *testing.T, directory string) string {
	t.Helper()
	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("stat %s: %v", directory, err)
	}
	return v16ReadProductionGo(t, directory)
}

func v16ReadGoTests(t *testing.T, directories ...string) string {
	t.Helper()
	var source strings.Builder
	for _, directory := range directories {
		if _, err := os.Stat(directory); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("V16_RED test package missing: %s", directory)
				continue
			}
			t.Fatalf("stat test package %s: %v", directory, err)
		}
		for _, file := range v16ProductionGoFiles(t, directory, true) {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			source.Write(content)
			source.WriteByte('\n')
		}
	}
	return source.String()
}

func v16ProductionGoFiles(t *testing.T, directory string, tests bool) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", directory, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			(strings.HasSuffix(entry.Name(), "_test.go") != tests) {
			continue
		}
		files = append(files, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(files)
	return files
}
