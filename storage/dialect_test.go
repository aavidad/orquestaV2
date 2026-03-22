package storage

import "testing"

func TestRebindQueryNoCambiaSQLiteNiMySQL(t *testing.T) {
	t.Parallel()

	query := `SELECT * FROM tareas WHERE estado = ? AND agente = ?`
	for _, driver := range []string{"sqlite", "mysql"} {
		if got := RebindQuery(driver, query); got != query {
			t.Fatalf("%s rebinding inesperado: %s", driver, got)
		}
	}
}

func TestRebindQueryConviertePlaceholdersEnPostgres(t *testing.T) {
	t.Parallel()

	got := RebindQuery("postgres", `SELECT * FROM tareas WHERE estado = ? AND agente = ?`)
	want := `SELECT * FROM tareas WHERE estado = $1 AND agente = $2`
	if got != want {
		t.Fatalf("query inesperada:\n%s", got)
	}
}

func TestRebindQueryRespetaInterrogantesEnLiterales(t *testing.T) {
	t.Parallel()

	got := RebindQuery("postgres", `SELECT '?' AS lit, "?" AS ident, estado = ? FROM tareas WHERE nota = 'a?b'`)
	want := `SELECT '?' AS lit, "?" AS ident, estado = $1 FROM tareas WHERE nota = 'a?b'`
	if got != want {
		t.Fatalf("query inesperada:\n%s", got)
	}
}

func TestDialectPlaceholderStyle(t *testing.T) {
	t.Parallel()

	if got := DialectForDriver("sqlite").PlaceholderStyle(); got != "qmark" {
		t.Fatalf("placeholder sqlite inesperado: %s", got)
	}
	if got := DialectForDriver("mysql").PlaceholderStyle(); got != "qmark" {
		t.Fatalf("placeholder mysql inesperado: %s", got)
	}
	if got := DialectForDriver("postgres").PlaceholderStyle(); got != "numbered" {
		t.Fatalf("placeholder postgres inesperado: %s", got)
	}
}

func TestDialectSupportsSchemaBootstrap(t *testing.T) {
	t.Parallel()

	if !DialectForDriver("sqlite").SupportsSchemaBootstrap() {
		t.Fatalf("sqlite deberia soportar bootstrap de schema")
	}
	if DialectForDriver("postgres").SupportsSchemaBootstrap() {
		t.Fatalf("postgres no deberia anunciar bootstrap de schema SQLite-first")
	}
	if DialectForDriver("mysql").SupportsSchemaBootstrap() {
		t.Fatalf("mysql no deberia anunciar bootstrap de schema SQLite-first")
	}
}
