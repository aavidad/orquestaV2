/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"strings"
	"testing"

	"orquesta/db"
)

func TestPrepararBootstrapRuntimeAgenteInyectaHandoffYMailbox(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex0", "programador"); err != nil {
		t.Fatalf("registrar agente origen: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear proyecto: %v", err)
	}
	proyecto := &db.Proyecto{ID: proyectoID, Slug: "orquestador", Nombre: "Orquestador", RutaAbs: tmp, Tipo: db.ProyectoRepo, Activo: true}
	sesionOrigen, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex0",
		ProyectoID:  &proyectoID,
		CWD:         tmp,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	if _, err := db.CrearHandoffAgenteVivo("Codex0", "Codex1", nil, "bootstrap", "continuidad importante", "sess-ext-1"); err != nil {
		t.Fatalf("crear handoff: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"mensaje":"ponte al dia"}`,
	}); err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}
	reglaID, err := db.UpsertRegla(&db.Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "regla-bootstrap-governance",
		Descripcion: "forzar contexto gobernanza en bootstrap",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	if _, err := db.GuardarGovernanceOverride("tester", &db.GovernanceOverride{
		TipoAgente: "programador",
		ScopeTipo:  db.GovernanceScopeProyecto,
		ScopeRef:   "orquestador",
		Entidad:    db.GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     db.GovernanceActionDisable,
	}); err != nil {
		t.Fatalf("guardar override gobernanza: %v", err)
	}

	resume, state, err := prepararBootstrapRuntimeAgente("Codex1", proyecto, nil)
	if err != nil {
		t.Fatalf("preparar bootstrap: %v", err)
	}
	if state == nil || state.Order == nil || state.Order.Tipo != "handoff" {
		t.Fatalf("runtime order bootstrap inesperada: %+v", state)
	}
	if len(state.Mailbox) != 1 {
		t.Fatalf("mailbox inesperada: %+v", state.Mailbox)
	}
	if !strings.Contains(resume.ResumenContinuidad, "continuidad importante") {
		t.Fatalf("resumen continuidad inesperado: %q", resume.ResumenContinuidad)
	}
	if !strings.Contains(resume.ResumePayloadJSON, `"runtime_order"`) {
		t.Fatalf("resume payload sin runtime_order: %s", resume.ResumePayloadJSON)
	}
	if !strings.Contains(resume.ResumePayloadJSON, `"governance_catalog"`) || !strings.Contains(resume.ResumePayloadJSON, `"resolucion_actual":"rol+proyecto"`) {
		t.Fatalf("resume payload sin gobernanza efectiva: %s", resume.ResumePayloadJSON)
	}

	order, err := db.GetRuntimeOrder(state.Order.ID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("estado de orden bootstrap inesperado: %s", order.Estado)
	}
	if sesionOrigen == nil || sesionOrigen.ID == 0 {
		t.Fatalf("sesion origen inesperada: %+v", sesionOrigen)
	}
}
