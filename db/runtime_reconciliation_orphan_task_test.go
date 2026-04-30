package db

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProcesarHigieneRuntimesAutonomosBatchDegradaTareaEnProgresoHuerfanaAAsignada(t *testing.T) {
	prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orphan-task",
		Nombre:  "Orphan Task",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente huerfano",
		Descripcion: "demo",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia reconciliar al menos una tarea huerfana, got=%d", n)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("estado inesperado tras higiene: %s", tarea.Estado)
	}
	if !strings.Contains(tarea.Notas, "degradada automáticamente a asignada por higiene runtime") {
		t.Fatalf("la nota de higiene no quedo registrada: %q", tarea.Notas)
	}
}

func TestProcesarRuntimeOrdersBatchExpiraHandoffPendienteAntigua(t *testing.T) {
	prepararDBTemporal(t)

	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-expiry",
		Nombre:  "Handoff Expiry",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Handoff pendiente",
		Descripcion: "demo",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	payloadJSON, _ := json.Marshal(map[string]any{
		"agente_origen":  "Codex1",
		"agente_destino": "Codex2",
		"tarea_id":       tareaID,
	})
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(payloadJSON),
		Estado:      "pendiente",
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	oldCreated := time.Now().UTC().Add(-25 * time.Hour)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, oldCreated, oldCreated, oldCreated, orderID); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia expirar al menos una handoff pendiente, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if order == nil || order.Estado != "expirada" {
		t.Fatalf("estado inesperado de handoff: %+v", order)
	}
	result := mapFromJSON(order.ResultadoJSON)
	if !boolFromMap(result, "expired") {
		t.Fatalf("resultado de expiracion inesperado: %v", order.ResultadoJSON)
	}
}
