package orquestarails

import "testing"

func TestArchitectureImportForbiddenV0BloqueaSoloImportsConcretos(t *testing.T) {
	policy := ArchitectureImportPolicyV0{
		ExactImports:    []string{"database/sql", "net", "runtime"},
		ImportPrefixes:  []string{"orquesta/modulos/orquesta-runtime-"},
		ImportFragments: []string{"modernc.org/sqlite"},
	}
	blocked := []string{
		"database/sql",
		"net/http",
		"runtime/debug",
		"orquesta/modulos/orquesta-runtime-codex",
		"modernc.org/sqlite",
	}
	for _, path := range blocked {
		if !ArchitectureImportForbiddenV0(path, policy) {
			t.Fatalf("import no bloqueado: %q", path)
		}
	}
	allowed := []string{
		"orquesta/modulos/orquesta-runtime",
		"orquesta/modulos/orquesta-rails",
		"strings",
	}
	for _, path := range allowed {
		if ArchitectureImportForbiddenV0(path, policy) {
			t.Fatalf("import valido bloqueado: %q", path)
		}
	}
}

func TestArchitectureSourceLiteralForbiddenV0PermiteRefsOpacas(t *testing.T) {
	allowed := []string{
		"runtime_ref",
		"sql_policy_ref",
		"http_transport_ref",
		"mcp_surface_ref",
		"context-ref-runtime-provider-modelo-opaco",
	}
	for _, value := range allowed {
		if ArchitectureSourceLiteralForbiddenV0("architecture-test", value) {
			t.Fatalf("literal opaco bloqueado: %q", value)
		}
	}
}

func TestArchitectureSourceLiteralForbiddenV0BloqueaDetalleReal(t *testing.T) {
	blocked := []string{
		"api_key=valor-real",
		"authorization: Bearer token-real",
		"postgres://user:pass@localhost/db",
		"sqlite:///tmp/orquesta.db",
		"/home/alberto/.config/orquesta",
	}
	for _, value := range blocked {
		if !ArchitectureSourceLiteralForbiddenV0("architecture-test", value) {
			t.Fatalf("literal sensible no bloqueado: %q", value)
		}
	}
}
