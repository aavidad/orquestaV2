package db

import (
	"os"
	"path/filepath"
<<<<<<< HEAD
=======
	"strings"
>>>>>>> origin/orq-orquestador-codex2
	"testing"
)

func TestResolverRutaDesdeGitRootRepoOrquesta(t *testing.T) {
	t.Parallel()

	got := resolverRutaDesdeGitRoot("/tmp/PlataformaMunicipal/orquestador")
	want := "/tmp/PlataformaMunicipal/orquestador/orquesta.db"
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}

func TestResolverRutaDesdeGitRootRepoContaGrxUsaSiblingOrquestador(t *testing.T) {
	t.Parallel()

<<<<<<< HEAD
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
=======
	got := resolverRutaDesdeGitRoot("/home/alberto/Trabajo/PlataformaMunicipal/ContaGrx")
	want := "/home/alberto/Trabajo/PlataformaMunicipal/orquestador/orquesta.db"
>>>>>>> origin/orq-orquestador-codex2
	if got != want {
		t.Fatalf("ruta inesperada: %s", got)
	}
}

func TestOpenNuevaBDIncluyeEsquemaExtendido(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	for _, table := range []string{"proyectos", "asignaciones", "conectores", "locks", "worktrees", "runtime_handles", "runtime_orders", "pools_capacidad", "pool_modelos", "decisiones_proyecto", "documentos_externos", "git_merges"} {
		exists, err := TableExists(table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !exists {
			t.Fatalf("tabla %s no creada", table)
		}
	}

	for _, column := range []string{"estado", "cwd", "external_session_id", "resumen_continuidad"} {
		found, err := ColumnExists("sesiones", column)
		if err != nil {
			t.Fatalf("ColumnExists(sesiones,%s): %v", column, err)
		}
		if !found {
			t.Fatalf("columna %s no encontrada en sesiones", column)
		}
	}
}

func TestOpenFallaSinDSNParaDriverExterno(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prevPath := os.Getenv("ORQUESTA_DB")
	prevDriver := os.Getenv("ORQUESTA_DB_DRIVER")
	prevDSN := os.Getenv("ORQUESTA_DB_DSN")
	prevBootstrap := os.Getenv("ORQUESTA_DB_BOOTSTRAP")
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prevPath)
		_ = os.Setenv("ORQUESTA_DB_DRIVER", prevDriver)
		_ = os.Setenv("ORQUESTA_DB_DSN", prevDSN)
		_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", prevBootstrap)
		Close()
	}()

	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DRIVER", "postgres"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DRIVER: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DSN", ""); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DSN: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_BOOTSTRAP", "false"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_BOOTSTRAP: %v", err)
	}

	err := Open()
	if err == nil {
		t.Fatalf("esperaba error por driver externo sin DSN")
	}
	if !strings.Contains(err.Error(), "ORQUESTA_DB_DSN") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestOpenFallaSiBootstrapSeFuerzaEnDriverNoSoportado(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prevPath := os.Getenv("ORQUESTA_DB")
	prevDriver := os.Getenv("ORQUESTA_DB_DRIVER")
	prevDSN := os.Getenv("ORQUESTA_DB_DSN")
	prevBootstrap := os.Getenv("ORQUESTA_DB_BOOTSTRAP")
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prevPath)
		_ = os.Setenv("ORQUESTA_DB_DRIVER", prevDriver)
		_ = os.Setenv("ORQUESTA_DB_DSN", prevDSN)
		_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", prevBootstrap)
		Close()
	}()

	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DRIVER", "mysql"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DRIVER: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DSN", "mysql://example"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DSN: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_BOOTSTRAP", "true"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_BOOTSTRAP: %v", err)
	}

	err := Open()
	if err == nil {
		t.Fatalf("esperaba error por bootstrap no soportado")
	}
	if !strings.Contains(err.Error(), "bootstrap de schema no soportado para driver mysql") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestOpenSQLiteSinBootstrapNoCreaEsquema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prevPath := os.Getenv("ORQUESTA_DB")
	prevDriver := os.Getenv("ORQUESTA_DB_DRIVER")
	prevDSN := os.Getenv("ORQUESTA_DB_DSN")
	prevBootstrap := os.Getenv("ORQUESTA_DB_BOOTSTRAP")
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prevPath)
		_ = os.Setenv("ORQUESTA_DB_DRIVER", prevDriver)
		_ = os.Setenv("ORQUESTA_DB_DSN", prevDSN)
		_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", prevBootstrap)
		Close()
	}()

	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DRIVER", "sqlite"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DRIVER: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DSN", ""); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_DSN: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_BOOTSTRAP", "false"); err != nil {
		t.Fatalf("setenv ORQUESTA_DB_BOOTSTRAP: %v", err)
	}

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	exists, err := TableExists("proyectos")
	if err != nil {
		t.Fatalf("TableExists(proyectos): %v", err)
	}
	if exists {
		t.Fatalf("no esperaba esquema bootstrap con ORQUESTA_DB_BOOTSTRAP=false")
	}
}
