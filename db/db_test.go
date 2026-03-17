package db

import "testing"

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

	got := resolverRutaDesdeGitRoot("/home/alberto/Trabajo/PlataformaMunicipal/ContaGrx")
	want := "/home/alberto/Trabajo/PlataformaMunicipal/orquesta/orquesta.db"
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}
