package orquestaappdirectorservice

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada(t *testing.T) {
	for _, file := range productionGoFilesForServiceTestV0(t) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("import path %s: %v", spec.Path.Value, err)
			}
			if serviceImportForbiddenV0(path) {
				t.Fatalf("%s importa dependencia prohibida %q", file, path)
			}
		}
		for _, literal := range serviceSourceStringLiteralsV0(parsed) {
			if serviceSourceStringLiteralForbiddenV0(literal) {
				t.Fatalf("%s contiene literal sensible o adaptador concreto %q", file, literal)
			}
		}
	}
}

func productionGoFilesForServiceTestV0(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Clean(name))
	}
	if len(files) == 0 {
		t.Fatalf("no hay ficheros go de produccion")
	}
	return files
}

func TestServiceSourceStringLiteralForbiddenV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	values := []string{
		"context-ref-runtime-provider-modelo-opaco",
		"domain-ref-review_pedagogical-assemble_topic-a1",
		"adapter-ref-codex-fake-temporal",
		"oauth-policy-ref-sin-valor",
		"db-policy-ref-sql-neutral",
	}
	for _, value := range values {
		if serviceSourceStringLiteralForbiddenV0(value) {
			t.Fatalf("literal opaco bloqueado por vocabulario normal: %q", value)
		}
	}
}

func TestServiceSourceStringLiteralForbiddenV0BloqueaValoresSensiblesYConectores(t *testing.T) {
	values := []string{
		"postgres://user:pass@localhost/db",
		"mysql://user:pass@localhost/db",
		"sqlite:///tmp/orquesta.db",
		"dsn=postgres://user:pass@localhost/db",
		"authorization: Bearer token-real",
		"api_key=valor-real",
		"client_secret=valor-real",
		"/home/alberto/.config/orquesta",
	}
	for _, value := range values {
		if !serviceSourceStringLiteralForbiddenV0(value) {
			t.Fatalf("literal sensible no bloqueado: %q", value)
		}
	}
}

func serviceSourceStringLiteralsV0(file *ast.File) []string {
	values := []string{}
	ast.Inspect(file, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		values = append(values, value)
		return true
	})
	return values
}

func serviceSourceStringLiteralForbiddenV0(value string) bool {
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, forbidden := range forbiddenServiceStringLiteralFragmentsV0() {
		if strings.Contains(lower, forbidden) {
			return true
		}
	}
	return false
}

func forbiddenServiceStringLiteralFragmentsV0() []string {
	return []string{
		"postgres://",
		"mysql://",
		"sqlite://",
		"mongodb://",
		"dsn=",
		"database_url=",
		"authorization:",
		"bearer ",
		"api_key=",
		"api-key=",
		"access_token=",
		"access-token=",
		"refresh_token=",
		"refresh-token=",
		"client_secret=",
		"client-secret=",
		"password=",
		"passwd=",
		"pwd=",
		"secret=",
		"credential=",
		"-----begin ",
		"/home/",
		"/users/",
		`c:\users\`,
		"~/",
	}
}

func serviceImportForbiddenV0(path string) bool {
	for _, forbidden := range forbiddenServiceExactImportsV0() {
		if path == forbidden {
			return true
		}
	}
	for _, prefix := range forbiddenServiceImportPrefixesV0() {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, fragment := range forbiddenServiceImportFragmentsV0() {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

func forbiddenServiceExactImportsV0() []string {
	return []string{
		"database/sql",
		"net",
		"net/http",
		"os",
		"os/exec",
	}
}

func forbiddenServiceImportPrefixesV0() []string {
	return []string{
		"net/",
		"orquesta/cmd",
		"orquesta/db",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-domain-work-sql",
		"orquesta/modulos/orquesta-http-gateway",
		"orquesta/modulos/orquesta-mcp",
		"orquesta/modulos/orquesta-operator-mcp",
		"orquesta/modulos/orquesta-opes-",
		"orquesta/modulos/orquesta-run-file",
		"orquesta/modulos/orquesta-runtime-",
		"orquesta/modulos/orquesta-state-file",
		"orquesta/modulos/orquesta-web",
	}
}

func forbiddenServiceImportFragmentsV0() []string {
	return []string{
		"codex",
		"modernc.org/sqlite",
		"github.com/mattn/go-sqlite3",
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"github.com/go-sql-driver/mysql",
	}
}
