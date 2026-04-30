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

func TestAutonomiaBatchSnapshotBudgetPauseCargaPresupuestoSoloUnaVez(t *testing.T) {
	prev := autonomiaGetAgenteFn
	t.Cleanup(func() { autonomiaGetAgenteFn = prev })

	calls := 0
	autonomiaGetAgenteFn = func(agente string) (*db.Agente, error) {
		calls++
		return &db.Agente{
			Nombre:            agente,
			Habilitado:        true,
			EstadoCuota:       "activo",
			PresupuestoEstado: "handoff_preventivo",
			CuotaRestantePct:  intPtr(12),
		}, nil
	}

	snapshot := &autonomiaBatchSnapshot{
		agentesByName:     map[string]*db.Agente{"codex1": {Nombre: "Codex1", Habilitado: true, EstadoCuota: "activo"}},
		agentBudgetLoaded: map[string]struct{}{},
		pauseByAgent:      map[string]autonomiaBudgetPauseDecision{},
	}

	shouldPause, _, err := snapshot.budgetPause("Codex1")
	if err != nil {
		t.Fatalf("budgetPause first: %v", err)
	}
	if !shouldPause {
		t.Fatal("deberia pausar con presupuesto preventivo")
	}
	shouldPause, _, err = snapshot.budgetPause("Codex1")
	if err != nil {
		t.Fatalf("budgetPause second: %v", err)
	}
	if !shouldPause {
		t.Fatal("deberia reutilizar decision cacheada")
	}
	if calls != 1 {
		t.Fatalf("deberia cargar presupuesto una sola vez, got=%d", calls)
	}
}

func TestNewAutonomiaBatchSnapshotMarcaPresupuestoPrecargado(t *testing.T) {
	prev := autonomiaGetAgenteFn
	t.Cleanup(func() { autonomiaGetAgenteFn = prev })

	calls := 0
	autonomiaGetAgenteFn = func(agente string) (*db.Agente, error) {
		calls++
		return &db.Agente{Nombre: agente, Habilitado: true}, nil
	}

	snapshot := &autonomiaBatchSnapshot{
		agentesByName:     map[string]*db.Agente{"codex1": {Nombre: "Codex1", Habilitado: true, EstadoCuota: "activo", PresupuestoEstado: "handoff_preventivo", CuotaRestantePct: intPtr(9)}},
		agentBudgetLoaded: map[string]struct{}{"codex1": {}},
		pauseByAgent:      map[string]autonomiaBudgetPauseDecision{},
	}

	shouldPause, _, err := snapshot.budgetPause("Codex1")
	if err != nil {
		t.Fatalf("budgetPause: %v", err)
	}
	if !shouldPause {
		t.Fatal("deberia pausar con presupuesto visible precargado")
	}
	if calls != 0 {
		t.Fatalf("no deberia recargar agente si el presupuesto ya venia precargado, got=%d", calls)
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

func TestAutonomiaBatchSnapshotActiveAssignmentsByAgentCacheaPorAgente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}

	snapshot := &autonomiaBatchSnapshot{
		asignacionesByAgent: map[string][]*db.Asignacion{},
	}

	asignaciones, err := snapshot.activeAssignmentsByAgent("Codex1")
	if err != nil {
		t.Fatalf("activeAssignmentsByAgent first: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0] == nil || asignaciones[0].ProyectoID != proyectoID {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}
	if _, err := db.DB.Exec(`DELETE FROM asignaciones WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("delete asignaciones para forzar cache: %v", err)
	}
	asignaciones, err = snapshot.activeAssignmentsByAgent("Codex1")
	if err != nil {
		t.Fatalf("activeAssignmentsByAgent second: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0] == nil || asignaciones[0].ProyectoID != proyectoID {
		t.Fatalf("deberia reutilizar asignaciones cacheadas: %+v", asignaciones)
	}
}
