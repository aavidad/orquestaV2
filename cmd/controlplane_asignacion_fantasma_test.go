package cmd

import (
	"database/sql"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarAsignacionesPremiumFantasmaBatchPausaAsignacionSinTrabajoNiRuntime(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir(), Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil || asignacion == nil {
		t.Fatalf("get asignacion activa: %+v err=%v", asignacion, err)
	}

	n, err := procesarAsignacionesPremiumFantasmaBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "bloqueado_por_runtime",
	}}, map[string][]*db.Tarea{}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAsignacionesPremiumFantasmaBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("procesadas=%d", n)
	}
	if activa, err := db.GetAsignacionActivaAgente("Gemini1"); err != nil && err != sql.ErrNoRows {
		t.Fatalf("get asignacion activa final: %v", err)
	} else if activa != nil {
		t.Fatalf("deberia haberse pausado la asignacion activa: %+v", activa)
	}
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: strPtr("Gemini1")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].Estado != db.AsignacionPausada || asignaciones[0].Nota != "sin_trabajo_reactivacion_automatica" {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}
}

func TestProcesarAsignacionesPremiumFantasmaBatchRespetaTrabajoActivo(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir(), Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica_trabajo_activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil || asignacion == nil {
		t.Fatalf("get asignacion activa: %+v err=%v", asignacion, err)
	}

	n, err := procesarAsignacionesPremiumFantasmaBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "bloqueado_por_runtime",
		OpenTasks:       1,
	}}, map[string][]*db.Tarea{
		"Gemini1": {{ID: 615}},
	}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAsignacionesPremiumFantasmaBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("procesadas=%d", n)
	}
	if activa, err := db.GetAsignacionActivaAgente("Gemini1"); err != nil {
		t.Fatalf("get asignacion activa final: %v", err)
	} else if activa == nil {
		t.Fatalf("no deberia haberse pausado la asignacion")
	}
}

func TestProcesarAsignacionesPremiumFantasmaBatchPausaReactivacionAutomaticaSinTrabajo(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: t.TempDir(), Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	asignacion, err := db.GetAsignacionActivaAgente("Gemini1")
	if err != nil || asignacion == nil {
		t.Fatalf("get asignacion activa: %+v err=%v", asignacion, err)
	}

	n, err := procesarAsignacionesPremiumFantasmaBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Gemini1"},
		Asignacion:      asignacion,
		EstadoOperativo: "bloqueado_por_runtime",
	}}, map[string][]*db.Tarea{}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAsignacionesPremiumFantasmaBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("procesadas=%d", n)
	}
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: strPtr("Gemini1")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].Estado != db.AsignacionPausada || asignaciones[0].Nota != "sin_trabajo_reactivacion_automatica" {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchCompactaReactivacionAutomaticaFantasma(t *testing.T) {
	prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia compactar la asignacion fantasma, got=%d", n)
	}

	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: strPtr("Gemini1")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].Estado != db.AsignacionPausada || asignaciones[0].Nota != "sin_trabajo_reactivacion_automatica" {
		t.Fatalf("asignaciones inesperadas: %+v", asignaciones)
	}
}
