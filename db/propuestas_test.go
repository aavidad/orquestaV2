package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCrearPropuestaCreaVotosPendientesSinBloquear(t *testing.T) {
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

	if err := RegistrarAgente("CodexX", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	id, err := CrearPropuesta(&Propuesta{
		Titulo:       "Arquitectura MCP",
		Descripcion:  "Servidor MCP como adaptador de entrada",
		Tipo:         "arquitectura",
		PropuestoPor: "CodexX",
		Distribuidor: "CodexX",
	})
	if err != nil {
		t.Fatalf("CrearPropuesta: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de propuesta no valido: %d", id)
	}

	var votos int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ?`, id).Scan(&votos); err != nil {
		t.Fatalf("count votos: %v", err)
	}
	if votos == 0 {
		t.Fatalf("no se crearon votos pendientes")
	}
}

func TestBackfillVotosPendientesRellenaPropuestasAbiertasSinVotos(t *testing.T) {
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

	if _, err := DB.Exec(`DELETE FROM votos`); err != nil {
		t.Fatalf("delete votos: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, estado, propuesto_por, distribuidor)
		VALUES ('OP-777', 'Duplicidad histórica', '', 'arquitectura', 'abierta', 'Codex1', 'Codex1')`); err != nil {
		t.Fatalf("insert propuesta abierta: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, estado, propuesto_por, distribuidor)
		VALUES ('OP-778', 'Ya cerrada', '', 'arquitectura', 'consenso', 'Codex1', 'Codex1')`); err != nil {
		t.Fatalf("insert propuesta cerrada: %v", err)
	}

	if err := BackfillVotosPendientes(); err != nil {
		t.Fatalf("BackfillVotosPendientes: %v", err)
	}

	var abiertos int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = (SELECT id FROM propuestas WHERE codigo='OP-777')`).Scan(&abiertos); err != nil {
		t.Fatalf("count votos abiertos: %v", err)
	}
	if abiertos == 0 {
		t.Fatalf("no se rellenaron votos para propuesta abierta")
	}

	var cerrados int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = (SELECT id FROM propuestas WHERE codigo='OP-778')`).Scan(&cerrados); err != nil {
		t.Fatalf("count votos cerrados: %v", err)
	}
	if cerrados != 0 {
		t.Fatalf("se generaron votos para propuesta cerrada")
	}
}
