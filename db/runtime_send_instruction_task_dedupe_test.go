package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEjecutarRuntimeOrderSendInstructionSupersedeOlderTaskScopedOrder(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "send-task-dedupe",
		Nombre:  "Send Task Dedupe",
		RutaAbs: filepath.Join(tmp, "repo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	olderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"slice viejo","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("encolar older: %v", err)
	}
	newerID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"slice nuevo","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("encolar newer: %v", err)
	}
	older, err := GetRuntimeOrder(olderID)
	if err != nil || older == nil {
		t.Fatalf("get older: %+v err=%v", older, err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(older); err != nil {
		t.Fatalf("ejecutar older: %v", err)
	}
	older, err = GetRuntimeOrder(olderID)
	if err != nil {
		t.Fatalf("get older final: %v", err)
	}
	if older == nil || older.Estado != "completada" {
		t.Fatalf("older deberia quedar completada por supersede: %+v", older)
	}
	if !strings.Contains(older.ResultadoJSON, `"superseded_reason":"covered_by_newer_task_order:`+jsonNumber(newerID)+`"`) {
		t.Fatalf("older sin reason de supersede por task order: %s", older.ResultadoJSON)
	}
}
