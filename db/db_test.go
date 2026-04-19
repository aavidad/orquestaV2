package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolverRutaDesdeGitRootRepoOrquesta(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	repo := filepath.Join(base, "orquesta")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("MkdirAll repo: %v", err)
	}
	want := filepath.Join(repo, "orquesta.db")
	if err := os.WriteFile(want, []byte("sqlite"), 0o644); err != nil {
		t.Fatalf("WriteFile db: %v", err)
	}

	got := resolverRutaDesdeGitRoot(repo)
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
	want := filepath.Join(orquestador, "orquesta.db")
	if err := os.WriteFile(want, []byte("sqlite"), 0o644); err != nil {
		t.Fatalf("WriteFile db: %v", err)
	}

	got := resolverRutaDesdeGitRoot(contagrx)
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}
