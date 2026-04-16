package cmd

import (
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestCargarTareasYBloqueosAutonomiaPorAgenteIgnoraEstadosTerminales(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	agente := "Codex1"
	for _, tc := range []struct {
		titulo  string
		estado  db.EstadoTarea
		visible bool
		blocked bool
	}{
		{titulo: "Asignada", estado: db.EstadoAsignada, visible: true},
		{titulo: "En progreso", estado: db.EstadoEnProgreso, visible: true},
		{titulo: "Bloqueada", estado: db.EstadoBloqueada, blocked: true},
		{titulo: "Completada", estado: db.EstadoCompletada},
		{titulo: "Cancelada", estado: db.EstadoCancelada},
		{titulo: "Backlog", estado: db.EstadoBacklog},
	} {
		id, err := db.CrearTarea(&db.Tarea{
			Titulo:     tc.titulo,
			ProyectoID: &proyectoID,
			CreadoPor:  "test",
			Prioridad:  db.PrioridadMedia,
		})
		if err != nil {
			t.Fatalf("crear tarea %s: %v", tc.titulo, err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET agente=?, estado=? WHERE id=?`, agente, tc.estado, id); err != nil {
			t.Fatalf("update tarea %s: %v", tc.titulo, err)
		}
	}

	activos, bloqueadas, _, err := cargarTareasYBloqueosAutonomiaPorAgente()
	if err != nil {
		t.Fatalf("cargar tareas y bloqueos: %v", err)
	}
	if got := len(activos[agente]); got != 2 {
		t.Fatalf("activas visibles=%d, want 2", got)
	}
	if got := len(bloqueadas[agente]); got != 1 {
		t.Fatalf("bloqueadas visibles=%d, want 1", got)
	}
}

func TestCargarTareasYBloqueosAutonomiaPorAgenteNoCargaBloqueosSiNoHayTareasBloqueadas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	agente := "Codex1"
	for _, estado := range []db.EstadoTarea{db.EstadoAsignada, db.EstadoEnProgreso} {
		id, err := db.CrearTarea(&db.Tarea{
			Titulo:     string(estado),
			ProyectoID: &proyectoID,
			CreadoPor:  "test",
			Prioridad:  db.PrioridadMedia,
		})
		if err != nil {
			t.Fatalf("crear tarea %s: %v", estado, err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET agente=?, estado=? WHERE id=?`, agente, estado, id); err != nil {
			t.Fatalf("update tarea %s: %v", estado, err)
		}
	}

	activos, bloqueadas, bloqueos, err := cargarTareasYBloqueosAutonomiaPorAgente()
	if err != nil {
		t.Fatalf("cargar tareas y bloqueos: %v", err)
	}
	if got := len(activos[agente]); got != 2 {
		t.Fatalf("activas visibles=%d, want 2", got)
	}
	if got := len(bloqueadas[agente]); got != 0 {
		t.Fatalf("bloqueadas visibles=%d, want 0", got)
	}
	if len(bloqueos) != 0 {
		t.Fatalf("no deberia cargar resumen de bloqueos si no hay tareas bloqueadas: %+v", bloqueos)
	}
}
