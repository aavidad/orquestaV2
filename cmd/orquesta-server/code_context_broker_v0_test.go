package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestServerRGCodeContextProviderV0DevuelveCoincidenciasCompactas(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg no disponible")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "modulos", "demo"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "modulos", "demo", "service.go"),
		[]byte("package demo\n\nfunc BrokerCentral() {}\n"),
		0o644,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	provider := serverRGCodeContextProviderV0{RootDir: root, Command: "rg"}
	result, err := provider.QueryCodeContextV0(context.Background(), orquestacontext.CodeContextQueryV0{
		SchemaVersion: orquestacontext.CodeContextQuerySchemaVersionV0,
		RepositoryRef: "repo-ref-test",
		QueryKind:     orquestacontext.CodeContextQueryKindSearchV0,
		Query:         "BrokerCentral",
		Scope:         []string{"modulos"},
		MaxResults:    5,
		MaxBytes:      3000,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 || len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Results[0].Path != "modulos/demo/service.go" ||
		result.Results[0].Line != 3 {
		t.Fatalf("hit=%+v", result.Results[0])
	}
}

func TestServerRGCodeContextScopesV0RechazaRutasAbsolutasYPadres(t *testing.T) {
	got := serverRGCodeContextScopesV0([]string{
		"/tmp/no",
		"../no",
		"modulos/orquesta-mcp",
		"docs",
	})
	if len(got) != 2 ||
		got[0] != "modulos/orquesta-mcp" ||
		got[1] != "docs" {
		t.Fatalf("scopes=%+v", got)
	}
}

func TestServerLimitedBufferV0DescartaExcesoSinBloquearWriter(t *testing.T) {
	buffer := &serverLimitedBufferV0{maxBytes: 8}
	written, err := buffer.Write([]byte("1234567890"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 10 {
		t.Fatalf("written=%d", written)
	}
	if buffer.Len() != 8 || buffer.String() != "12345678" || !buffer.overflow {
		t.Fatalf("buffer len=%d value=%q overflow=%v", buffer.Len(), buffer.String(), buffer.overflow)
	}
	written, err = buffer.Write([]byte("abcdef"))
	if err != nil {
		t.Fatalf("write2: %v", err)
	}
	if written != 6 || buffer.Len() != 8 {
		t.Fatalf("written=%d len=%d", written, buffer.Len())
	}
}
