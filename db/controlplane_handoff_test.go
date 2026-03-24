package db

import (
	"encoding/json"
	"testing"
)

func TestCrearHandoffAgenteVivoReasignaTareaYCreaOrden(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	sesionOrigenID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	handleOrigen, err := GetRuntimeHandleBySesionID(sesionOrigenID)
	if err != nil {
		t.Fatalf("get handle origen: %v", err)
	}
	if handleOrigen == nil || handleOrigen.Estado != "activo" {
		t.Fatalf("handle origen inesperado: %+v", handleOrigen)
	}

	if _, err := IniciarSesion("Codex2"); err != nil {
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
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	orderID, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto en rojo", "Continuar desde checkpoint", "ext-123")
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
	if tarea.Agente == nil || *tarea.Agente != "Codex2" {
		t.Fatalf("agente reasignado inesperado: %+v", tarea.Agente)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("estado inesperado: %s", tarea.Estado)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("ordenes inesperadas: %+v", orders)
	}
	if orders[0].Tipo != "handoff" {
		t.Fatalf("tipo inesperado: %s", orders[0].Tipo)
	}

	var payload HandoffPayload
	if err := json.Unmarshal([]byte(orders[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload.AgenteOrigen != "Codex1" || payload.AgenteDestino != "Codex2" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.TareaID == nil || *payload.TareaID != tareaID {
		t.Fatalf("tarea en payload inesperada: %+v", payload.TareaID)
	}

	handleOrigen, err = GetRuntimeHandleBySesionID(sesionOrigenID)
	if err != nil {
		t.Fatalf("handle origen tras handoff: %v", err)
	}
	if handleOrigen == nil || handleOrigen.Estado != "pausado" {
		t.Fatalf("el origen deberia quedar pausado tras el handoff: %+v", handleOrigen)
	}
}

func TestCrearHandoffAgenteVivoExigeHandleActivoEnOrigen(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", nil, "", "", ""); err == nil {
		t.Fatalf("se esperaba error por falta de handle activo")
	}
}

func TestCrearHandoffAgenteVivoReutilizaPendienteEquivalente(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex1"); err != nil {
		t.Fatalf("IniciarSesion origen: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion destino: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Revisar handoff idempotente",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	orderA, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto", "Continuar", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo A: %v", err)
	}
	orderB, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "presupuesto", "Continuar", "ext-123")
	if err != nil {
		t.Fatalf("CrearHandoffAgenteVivo B: %v", err)
	}
	if orderA != orderB {
		t.Fatalf("se esperaba reutilizar la orden pendiente: %d != %d", orderA, orderB)
	}

	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia existir una sola orden pendiente equivalente: %+v", orders)
	}
}
