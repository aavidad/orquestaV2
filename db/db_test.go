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
	contagrx := filepath.Join(base, "ContaGrx")
	orquestador := filepath.Join(base, "orquestador")
	if err := os.MkdirAll(contagrx, 0o755); err != nil {
		t.Fatalf("MkdirAll ContaGrx: %v", err)
	}
	if err := os.MkdirAll(orquestador, 0o755); err != nil {
		t.Fatalf("MkdirAll orquestador: %v", err)
	}
	if err := os.WriteFile(filepath.Join(orquestador, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("WriteFile go.mod: %v", err)
	}

	got := resolverRutaDesdeGitRoot(contagrx)
	want := filepath.Join(orquestador, "orquesta.db")
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}
