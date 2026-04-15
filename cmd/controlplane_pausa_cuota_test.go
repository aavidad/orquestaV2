package cmd

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestPersistirPausaPorCuotaAutonomiaBloqueaTareaActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Claude1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(2 * time.Hour)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":0,"window_minutes":120,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "2h",
		BudgetSource:    "claude_usage_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto visible: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium en ejecucion",
		Descripcion: "slice",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Claude1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	if err := persistirPausaPorCuotaAutonomia("Claude1", "worker bloqueado por cuota"); err != nil {
		t.Fatalf("persistir pausa cuota: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != db.TareaBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada: %+v", tarea)
	}
	notas := tarea.Notas
	if !strings.Contains(strings.ToLower(notas), "cuota sin relevo sano") {
		t.Fatalf("faltaba nota de bloqueo por cuota: %s", notas)
	}
	agente, err := db.GetAgente("Claude1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil || agente.ReanimarAt == nil || !agente.ReanimarAt.After(time.Now().UTC()) {
		t.Fatalf("la pausa por cuota deberia dejar reanimar_at futuro: %+v", agente)
	}
}
