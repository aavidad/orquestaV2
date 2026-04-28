package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/runtimeagente"
)

func runtimeOrderSendInstructionEsContinuidadDirectaServidor(payload map[string]any) bool {
	if payload == nil || runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false
	}
	fromAgente := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "from_agente", "")))
	if fromAgente != "server" && fromAgente != "orquesta" {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "kind", "")))
	if kind != "" && kind != "instruction" {
		return false
	}
	classification := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "classification", "")))
	return classification == "blocked" || int64FromAny(payload["transcript_id"]) > 0
}

func runtimeOrderSendInstructionDirectaAbsorbidaPorTrabajo(order *RuntimeOrder, payload map[string]any, now time.Time) (string, error) {
	if order == nil || !runtimeOrderSendInstructionEsContinuidadDirectaServidor(payload) {
		return "", nil
	}
	tareaID, err := GetTareaActivaIDPorAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID)
	if err != nil {
		return "", err
	}
	if tareaID <= 0 {
		return "", nil
	}
	var handle *RuntimeHandle
	if order.HandleID != nil && *order.HandleID > 0 {
		handle, err = GetRuntimeHandle(*order.HandleID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	if handle == nil {
		handle, err = GetRuntimeHandleCanonicoRecienteAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID)
		if err != nil {
			return "", err
		}
	}
	if handle == nil || !RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) {
		return "", nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return "", err
	}
	if snap == nil {
		return "", nil
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return "", nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running", "idle":
		return fmt.Sprintf("direct_instruction_absorbed_by_active_task:%d", tareaID), nil
	default:
		return "", nil
	}
}

func runtimeOrderSendInstructionAutonomiaReasignadaAbsorbidaPorTrabajo(order *RuntimeOrder, payload map[string]any, now time.Time) (string, error) {
	if order == nil || payload == nil {
		return "", nil
	}
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "kind", "")), "autonomia") {
		return "", nil
	}
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "accion", "")), "continuar_trabajo") {
		return "", nil
	}
	if !boolFromMap(payload, "post_remediation") {
		return "", nil
	}
	tareaID := int64FromAny(payload["tarea_id"])
	if tareaID <= 0 {
		return "", nil
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		return "", err
	}
	if tarea == nil || tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), strings.TrimSpace(order.Agente)) {
		return "", nil
	}
	if tarea.Estado != TareaEnProgreso {
		return "", nil
	}
	var handle *RuntimeHandle
	if order.HandleID != nil && *order.HandleID > 0 {
		handle, err = GetRuntimeHandle(*order.HandleID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	if handle == nil {
		handle, err = GetRuntimeHandleCanonicoRecienteAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID)
		if err != nil {
			return "", err
		}
	}
	if handle == nil || !RuntimeHandleSnapshotIsFresh(handle, 2*time.Minute) {
		return "", nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return "", err
	}
	if snap == nil {
		return "", nil
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return "", nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running", "working", "idle":
		return fmt.Sprintf("mailbox_instruction_absorbed_by_active_task:%d", tareaID), nil
	default:
		return "", nil
	}
}

func RuntimeOrderSendInstructionAbsorbidaPorTrabajoVivo(order *RuntimeOrder, now time.Time) (bool, string, error) {
	if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
		return false, "", nil
	}
	payload := mapFromJSON(order.PayloadJSON)
	reason, err := runtimeOrderSendInstructionDirectaAbsorbidaPorTrabajo(order, payload, now.UTC())
	if err != nil {
		return false, "", err
	}
	if strings.TrimSpace(reason) == "" {
		reason, err = runtimeOrderSendInstructionAutonomiaReasignadaAbsorbidaPorTrabajo(order, payload, now.UTC())
	}
	if err != nil {
		return false, "", err
	}
	if strings.TrimSpace(reason) == "" {
		return false, "", nil
	}
	return true, reason, nil
}

func runtimeOrderSendInstructionReceiptBloqueoIgnoraSnapshot(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	if strings.EqualFold(driver, "ollama_pool_local") {
		return true
	}
	return strings.EqualFold(transport, "api") && boolFromMap(meta, "pool_local")
}

func runtimeOrderSendInstructionReceiptTimedOut(order *RuntimeOrder, msg *RuntimeMailboxMessage, now time.Time) bool {
	notifiedAt := runtimeOrderSendInstructionNotifiedAt(order, msg)
	if notifiedAt.IsZero() {
		return false
	}
	return now.UTC().Sub(notifiedAt) > runtimeOrderSendInstructionReceiptTimeout()
}

func runtimeOrderSendInstructionProvieneMailbox(payload map[string]any) bool {
	return runtimeOrderSendInstructionMailboxID(payload) > 0
}

func runtimeOrderSendInstructionTexto(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	texto := strings.TrimSpace(stringFromMap(payload, "texto", ""))
	instruction := strings.TrimSpace(stringFromMap(payload, "instruction", ""))
	if strings.EqualFold(strings.TrimSpace(anyToString(payload["mailbox_kind"])), "pipeline_local") {
		if instruction != "" {
			return instruction
		}
		return texto
	}
	if texto != "" {
		return texto
	}
	return instruction
}

func runtimeOrderSendInstructionTrackingResult(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	out := map[string]any{}
	if mailboxKind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); mailboxKind != "" {
		out["mailbox_kind"] = mailboxKind
	}
	if accion := strings.TrimSpace(stringFromMap(payload, "accion", "")); accion != "" {
		out["accion"] = accion
	}
	if !boolFromMap(payload, "post_remediation") {
		if len(out) == 0 {
			return nil
		}
		return out
	}
	out["post_remediation"] = true
	if verificationKey := strings.TrimSpace(stringFromMap(payload, "verification_key", "")); verificationKey != "" {
		out["verification_key"] = verificationKey
	}
	if remediationKind := strings.TrimSpace(stringFromMap(payload, "remediation_kind", "")); remediationKind != "" {
		out["remediation_kind"] = remediationKind
	}
	if originAgent := strings.TrimSpace(stringFromMap(payload, "origin_agent", "")); originAgent != "" {
		out["origin_agent"] = originAgent
	}
	if taskID := int64FromAny(payload["tarea_id"]); taskID > 0 {
		out["tarea_id"] = taskID
	}
	return out
}

func runtimeOrderSendInstructionEsMicroprogramacion(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "source", "")), "microprogramacion") {
		return true
	}
	_, ok := payload["microprogramacion"]
	return ok
}

func runtimeOrderSendInstructionDebeEsperarReceiptInteractivo(order *RuntimeOrder, handle *RuntimeHandle, payload map[string]any) bool {
	if order == nil || handle == nil {
		return false
	}
	if !runtimeOrderSendInstructionEsMicroprogramacion(payload) && !runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	deliveryMode := RuntimeHandleMailboxDeliveryMode(handle)
	if runtimeOrderSendInstructionEsPipelinePremium(payload) &&
		runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) == runtimeagente.MailboxDeliverySessionResume &&
		runtimeHandleSolicitaSessionResume(handle, deliveryMode) {
		return true
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) {
		return false
	}
	if strings.EqualFold(driver, "tmux_cli_session") || strings.EqualFold(transport, "tmux") {
		return true
	}
	if strings.EqualFold(driver, "ollama_pool_local") {
		return true
	}
	return strings.EqualFold(transport, "api") && boolFromMap(meta, "pool_local")
}

func runtimeOrderSendInstructionMailboxID(payload map[string]any) int64 {
	if payload == nil {
		return 0
	}
	return int64FromAny(payload["mailbox_id"])
}
