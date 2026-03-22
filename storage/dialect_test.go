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
