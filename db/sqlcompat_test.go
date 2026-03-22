package db

import "testing"

func TestBuildInsertIgnoreValuesSQL(t *testing.T) {
	t.Parallel()

	got := buildInsertIgnoreValuesSQL("sqlite", "votos", []string{"propuesta_id", "agente", "posicion"}, []string{"propuesta_id", "agente"})
	want := "INSERT INTO votos (propuesta_id,agente,posicion) VALUES (?,?,?) ON CONFLICT(propuesta_id,agente) DO NOTHING"
	if got != want {
		t.Fatalf("sql sqlite inesperado:\n%s", got)
	}

	got = buildInsertIgnoreValuesSQL("mysql", "votos", []string{"propuesta_id", "agente", "posicion"}, []string{"propuesta_id", "agente"})
	want = "INSERT IGNORE INTO votos (propuesta_id,agente,posicion) VALUES (?,?,?)"
	if got != want {
		t.Fatalf("sql mysql inesperado:\n%s", got)
	}
}

func TestBuildInsertIgnoreSelectSQL(t *testing.T) {
	t.Parallel()

	got := buildInsertIgnoreSelectSQL("postgres", "votos", []string{"propuesta_id", "agente", "posicion", "comentario"}, []string{"propuesta_id", "agente"},
		"SELECT p.id, a.nombre, 'pendiente', '' FROM propuestas p")
	want := "INSERT INTO votos (propuesta_id,agente,posicion,comentario) SELECT p.id, a.nombre, 'pendiente', '' FROM propuestas p ON CONFLICT(propuesta_id,agente) DO NOTHING"
	if got != want {
		t.Fatalf("sql postgres inesperado:\n%s", got)
	}
}

func TestBuildUpsertValuesSQL(t *testing.T) {
	t.Parallel()

	assignments := []upsertAssignment{
		{Column: "posicion"},
		{Column: "comentario"},
		{Column: "updated_at", Expr: "CURRENT_TIMESTAMP"},
	}

	got := buildUpsertValuesSQL("sqlite", "votos", []string{"propuesta_id", "agente", "posicion", "comentario"}, []string{"propuesta_id", "agente"}, assignments)
	want := "INSERT INTO votos (propuesta_id,agente,posicion,comentario) VALUES (?,?,?,?) ON CONFLICT(propuesta_id,agente) DO UPDATE SET posicion=excluded.posicion,comentario=excluded.comentario,updated_at=CURRENT_TIMESTAMP"
	if got != want {
		t.Fatalf("sql sqlite upsert inesperado:\n%s", got)
	}

	got = buildUpsertValuesSQL("mysql", "votos", []string{"propuesta_id", "agente", "posicion", "comentario"}, []string{"propuesta_id", "agente"}, assignments)
	want = "INSERT INTO votos (propuesta_id,agente,posicion,comentario) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE posicion=VALUES(posicion),comentario=VALUES(comentario),updated_at=CURRENT_TIMESTAMP"
	if got != want {
		t.Fatalf("sql mysql upsert inesperado:\n%s", got)
	}
}
