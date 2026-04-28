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

func TestRuntimeOrderSendInstructionDuplicadaMasRecientePorTareaNoConfundePrefijoNumerico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "send-task-dedupe-prefix",
		Nombre:  "Send Task Dedupe Prefix",
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
		PayloadJSON: `{"to_agente":"Codex1","texto":"slice viejo","accion":"continuar_trabajo","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("encolar older: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"slice otra tarea","accion":"continuar_trabajo","tarea_id":410}`,
	}); err != nil {
		t.Fatalf("encolar newer: %v", err)
	}
	older, err := GetRuntimeOrder(olderID)
	if err != nil || older == nil {
		t.Fatalf("get older: %+v err=%v", older, err)
	}
	payload, err := runtimeOrderPayloadMap(older.PayloadJSON)
	if err != nil {
		t.Fatalf("payload map: %v", err)
	}
	got, err := runtimeOrderSendInstructionDuplicadaMasRecientePorTarea(older, payload)
	if err != nil {
		t.Fatalf("duplicada por tarea: %v", err)
	}
	if got != nil {
		t.Fatalf("no deberia confundir tarea 41 con 410: %+v", got)
	}
}

func TestRuntimeOrderSendInstructionDuplicadaMasRecientePorTareaNoMezclaMailboxYDirecta(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "send-task-dedupe-mode",
		Nombre:  "Send Task Dedupe Mode",
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
		PayloadJSON: `{"to_agente":"Codex1","texto":"desde mailbox","accion":"continuar_trabajo","mailbox_id":77,"mailbox_kind":"autonomia","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("encolar older: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"directa","accion":"continuar_trabajo","tarea_id":41}`,
	}); err != nil {
		t.Fatalf("encolar newer: %v", err)
	}
	older, err := GetRuntimeOrder(olderID)
	if err != nil || older == nil {
		t.Fatalf("get older: %+v err=%v", older, err)
	}
	payload, err := runtimeOrderPayloadMap(older.PayloadJSON)
	if err != nil {
		t.Fatalf("payload map: %v", err)
	}
	got, err := runtimeOrderSendInstructionDuplicadaMasRecientePorTarea(older, payload)
	if err != nil {
		t.Fatalf("duplicada por tarea: %v", err)
	}
	if got != nil {
		t.Fatalf("no deberia mezclar mailbox y directa: %+v", got)
	}
}
