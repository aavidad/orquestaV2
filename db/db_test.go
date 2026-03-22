package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolverRutaDesdeGitRootRepoOrquesta(t *testing.T) {
	t.Parallel()

	got := resolverRutaDesdeGitRoot("/tmp/PlataformaMunicipal/orquesta")
	want := "/tmp/PlataformaMunicipal/orquesta/orquesta.db"
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}

func TestResolverRutaDesdeGitRootRepoContaGrxUsaSiblingOrquesta(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	conta := filepath.Join(base, "ContaGrx")
	orquesta := filepath.Join(base, "orquesta")
	if err := os.MkdirAll(conta, 0o755); err != nil {
		t.Fatalf("mkdir conta: %v", err)
	}
	if err := os.MkdirAll(orquesta, 0o755); err != nil {
		t.Fatalf("mkdir orquesta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(orquesta, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	got := resolverRutaDesdeGitRoot(conta)
	want := filepath.Join(orquesta, "orquesta.db")
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}
