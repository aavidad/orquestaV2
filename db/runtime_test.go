/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"encoding/json"
	"testing"
)

func TestRegistrarRuntimeHandleReemplazaActivoPrevio(t *testing.T) {
	abrirDBTemporalMemoria(t)

	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	_, err = RegistrarRuntimeHandle(&RuntimeHandle{
		Agente:     "codex1",
		SesionID:   &sesionID,
		Transporte: "pty",
		HandleKind: "process",
		HandleRef:  "proc-1",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeHandle 1: %v", err)
	}

	_, err = RegistrarRuntimeHandle(&RuntimeHandle{
		Agente:     "codex1",
		SesionID:   &sesionID,
		Transporte: "pty",
		HandleKind: "process",
		HandleRef:  "proc-2",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeHandle 2: %v", err)
	}

	h, err := RuntimeHandleActivo("codex1")
	if err != nil {
		t.Fatalf("RuntimeHandleActivo: %v", err)
	}
	if h.HandleRef != "proc-2" {
		t.Fatalf("handle activo inesperado: %+v", h)
	}
}

func TestCrearHandoffAgenteVivoReasignaTareaYCreaOrden(t *testing.T) {
	abrirDBTemporalMemoria(t)

	sesionOrigenID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := RegistrarRuntimeHandle(&RuntimeHandle{
		Agente:     "codex1",
		SesionID:   &sesionOrigenID,
		Transporte: "pty",
		HandleKind: "process",
		HandleRef:  "proc-codex1",
	}); err != nil {
		t.Fatalf("RegistrarRuntimeHandle origen: %v", err)
	}

	sesionDestinoID, err := IniciarSesion("codex2")
	if err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Revisar handoff",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	orderID, err := CrearHandoffAgenteVivo("codex1", "codex2", &tareaID, "presupuesto en rojo", "Continuar desde checkpoint", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo: %v", err)
	}
	if orderID == 0 {
		t.Fatalf("order id inesperado: %d", orderID)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "codex2" {
		t.Fatalf("agente reasignado inesperado: %+v", tarea.Agente)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("estado inesperado: %s", tarea.Estado)
	}

	orders, err := ListarRuntimeOrders("codex2", "pendiente")
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("ordenes inesperadas: %+v", orders)
	}
	if orders[0].Tipo != "handoff" {
		t.Fatalf("tipo inesperado: %s", orders[0].Tipo)
	}
	if orders[0].SesionID == nil || *orders[0].SesionID != sesionDestinoID {
		t.Fatalf("sesion vinculada inesperada: %+v", orders[0].SesionID)
	}

	var payload HandoffPayload
	if err := json.Unmarshal([]byte(orders[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.AgenteOrigen != "codex1" || payload.AgenteDestino != "codex2" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.TareaID == nil || *payload.TareaID != tareaID {
		t.Fatalf("tarea en payload inesperada: %+v", payload.TareaID)
	}
}

func TestCrearHandoffAgenteVivoExigeHandleActivoEnOrigen(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if _, err := IniciarSesion("codex1"); err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := IniciarSesion("codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}

	if _, err := CrearHandoffAgenteVivo("codex1", "codex2", nil, "", "", ""); err == nil {
		t.Fatalf("se esperaba error por falta de handle activo")
	}
}
