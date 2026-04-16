package cmd

import (
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestAutonomiaBatchSnapshotOperationalStateCacheaPorAgente(t *testing.T) {
	prev := autonomiaOperationalStateResolver
	defer func() { autonomiaOperationalStateResolver = prev }()

	llamadas := 0
	autonomiaOperationalStateResolver = func(agente string) (string, string, error) {
		llamadas++
		if agente != "Codex1" {
			t.Fatalf("agente inesperado: %s", agente)
		}
		return "trabajando", "worker fresco", nil
	}

	snapshot := &autonomiaBatchSnapshot{
		operationalStateByAgent:  map[string]string{},
		operationalDetailByAgent: map[string]string{},
		operationalStateResolved: map[string]struct{}{},
	}

	estado, detalle, err := snapshot.operationalState("Codex1")
	if err != nil {
		t.Fatalf("operationalState first: %v", err)
	}
	if estado != "trabajando" || detalle != "worker fresco" {
		t.Fatalf("estado/detalle inesperados: %q / %q", estado, detalle)
	}
	estado, detalle, err = snapshot.operationalState("Codex1")
	if err != nil {
		t.Fatalf("operationalState second: %v", err)
	}
	if estado != "trabajando" || detalle != "worker fresco" {
		t.Fatalf("estado/detalle inesperados en cache: %q / %q", estado, detalle)
	}
	if llamadas != 1 {
		t.Fatalf("resolver deberia llamarse una sola vez, got=%d", llamadas)
	}
}

func TestAutonomiaBatchSnapshotProjectCacheaPorProyecto(t *testing.T) {
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

	snapshot := &autonomiaBatchSnapshot{
		proyectosByID:   map[int64]*db.Proyecto{},
		proyectosLoaded: map[int64]struct{}{},
	}

	proyecto, err := snapshot.project(proyectoID)
	if err != nil {
		t.Fatalf("project first: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoID {
		t.Fatalf("proyecto inesperado: %+v", proyecto)
	}
	if _, err := db.DB.Exec(`DELETE FROM proyectos WHERE id=?`, proyectoID); err != nil {
		t.Fatalf("delete proyecto para forzar cache: %v", err)
	}
	proyecto, err = snapshot.project(proyectoID)
	if err != nil {
		t.Fatalf("project second: %v", err)
	}
	if proyecto == nil || proyecto.ID != proyectoID {
		t.Fatalf("deberia reutilizar proyecto cacheado: %+v", proyecto)
	}
}
