package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"strings"
	"time"
)

func claimBootstrapRuntimeOrderParaStart(agente string, proyectoID *int64, excludeOrderID int64) (*RuntimeOrder, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	tipos := []string{"handoff", "resume"}
	placeholders, tipoArgs := runtimeOrderPlaceholders(tipos)
	argsBase := make([]any, 0, len(tipoArgs)+4)
	argsBase = append(argsBase, strings.TrimSpace(agente))
	argsBase = append(argsBase, tipoArgs...)
	for {
		args := append([]any{}, argsBase...)
		q := `
			SELECT id
			FROM runtime_orders
			WHERE agente = ?
			  AND estado = 'pendiente'
				  AND (available_at IS NULL OR available_at <= CURRENT_TIMESTAMP)
			  AND tipo IN (` + placeholders + `)`
		if proyectoID != nil {
			q += ` AND (proyecto_id = ? OR proyecto_id IS NULL)`
			args = append(args, *proyectoID)
		}
		if excludeOrderID > 0 {
			q += ` AND id <> ?`
			args = append(args, excludeOrderID)
		}
		q += `
			ORDER BY
			  CASE
			    WHEN tipo = 'handoff' THEN 0
			    WHEN tipo = 'resume' THEN 1
			    ELSE 9
			  END,
			  id
			LIMIT 1`
		var id int64
		err := DB.QueryRow(q, args...).Scan(&id)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		order, err := claimRuntimeOrderByID(id)
		if err != nil {
			return nil, err
		}
		if order == nil {
			continue
		}
		return order, nil
	}
}

func construirResumePayloadBootstrapDB(prev string, order *RuntimeOrder, mailbox []*RuntimeMailboxMessage, checkpoint *RuntimeCheckpoint) string {
	prev = strings.TrimSpace(prev)
	envelope := map[string]any{}
	if order != nil {
		envelope["runtime_order"] = map[string]any{
			"id":      order.ID,
			"tipo":    order.Tipo,
			"payload": rawJSONOrStringDB(order.PayloadJSON),
		}
	}
	if len(mailbox) > 0 {
		items := make([]map[string]any, 0, len(mailbox))
		for _, msg := range mailbox {
			if msg == nil {
				continue
			}
			items = append(items, map[string]any{
				"id":          msg.ID,
				"from_agente": msg.FromAgente,
				"kind":        msg.Kind,
				"payload":     rawJSONOrStringDB(msg.PayloadJSON),
			})
		}
		if len(items) > 0 {
			envelope["mailbox"] = items
		}
	}
	if checkpoint != nil {
		envelope["checkpoint"] = map[string]any{
			"id":              checkpoint.ID,
			"kind":            checkpoint.CheckpointKind,
			"resumen":         resumenCheckpointPayload(checkpoint),
			"branch":          checkpoint.Branch,
			"cwd":             checkpoint.CWD,
			"payload":         rawJSONOrStringDB(checkpoint.PayloadJSON),
			"resume_strategy": checkpoint.ResumeStrategy,
			"source":          checkpoint.Source,
		}
	}
	if len(envelope) == 0 {
		return prev
	}
	return MergeResumePayloadEnvelope(prev, envelope)
}

func construirResumenBootstrapDB(prev string, order *RuntimeOrder, mailbox []*RuntimeMailboxMessage, checkpoint *RuntimeCheckpoint) string {
	partes := make([]string, 0, 4)
	prev = strings.TrimSpace(prev)
	if limpio := limpiarResumenBootstrapPrevio(prev); limpio != "" {
		partes = append(partes, limpio)
	}
	if order != nil {
		switch strings.TrimSpace(order.Tipo) {
		case "handoff":
			var payload HandoffPayload
			if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err == nil {
				resumen := strings.TrimSpace(payload.ResumenContinuidad)
				if resumen != "" {
					partes = append(partes, "Handoff: "+resumen)
				} else if strings.TrimSpace(payload.Motivo) != "" {
					partes = append(partes, "Handoff: "+strings.TrimSpace(payload.Motivo))
				}
			}
		default:
			partes = append(partes, "Orden pendiente aplicada: "+strings.TrimSpace(order.Tipo))
		}
	}
	if resumen := resumenCheckpointBootstrap(checkpoint, prev); resumen != "" {
		partes = append(partes, resumen)
	}
	if len(mailbox) > 0 {
		partes = append(partes, fmt.Sprintf("Mailbox: %d mensaje(s) inyectados", len(mailbox)))
	}
	return unirPartesUnicasResume(partes)
}

func rawJSONOrStringDB(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed
	}
	return raw
}

func checkpointIDOrZeroDB(cp *RuntimeCheckpoint) int64 {
	if cp == nil {
		return 0
	}
	return cp.ID
}

func buscarHandoffPendienteEquivalente(origen, destino string, tareaID *int64, proyectoID *int64) (int64, error) {
	var err error
	origen, err = CanonicalizeAgentName(origen)
	if err != nil {
		return 0, err
	}
	destino, err = CanonicalizeAgentName(destino)
	if err != nil {
		return 0, err
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
		Agente:     &destino,
		ProyectoID: proyectoID,
		Estado:     &estado,
		Tipos:      []string{"handoff"},
	})
	if err != nil {
		return 0, err
	}
	for _, order := range orders {
		if order == nil || order.Tipo != "handoff" {
			continue
		}
		var payload HandoffPayload
		if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
			continue
		}
		payloadOrigen := strings.TrimSpace(payload.AgenteOrigen)
		if payloadOrigen != "" {
			if canonical, err := CanonicalizeAgentName(payloadOrigen); err == nil {
				payloadOrigen = canonical
			}
		}
		payloadDestino := strings.TrimSpace(payload.AgenteDestino)
		if payloadDestino != "" {
			if canonical, err := CanonicalizeAgentName(payloadDestino); err == nil {
				payloadDestino = canonical
			}
		}
		if payloadOrigen != strings.TrimSpace(origen) || payloadDestino != strings.TrimSpace(destino) {
			continue
		}
		switch {
		case tareaID == nil && payload.TareaID == nil:
			return order.ID, nil
		case tareaID != nil && payload.TareaID != nil && *tareaID == *payload.TareaID:
			return order.ID, nil
		}
	}
	return 0, nil
}

func registrarEvidenciaHandoffReanudadoPorOrden(order *RuntimeOrder, sesionID int64) error {
	if order == nil || strings.TrimSpace(order.Tipo) != "handoff" {
		return nil
	}
	var payload HandoffPayload
	if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
		return nil
	}
	destino := strings.TrimSpace(order.Agente)
	if payload.TareaID != nil && *payload.TareaID > 0 {
		res, err := DB.Exec(`UPDATE tareas SET estado='en_progreso' WHERE id=? AND agente=? AND estado='asignada'`, *payload.TareaID, strings.TrimSpace(destino))
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			nota := fmt.Sprintf("handoff completado por %s en sesion #%d", strings.TrimSpace(destino), sesionID)
			if _, err := DB.Exec(
				`UPDATE tareas SET notas = COALESCE(notas,'') || ? WHERE id=?`,
				formatearAnotacionTarea("orquesta", nota, time.Now().UTC()), *payload.TareaID,
			); err != nil {
				return err
			}
		}
	}
	detalle := fmt.Sprintf("origen=%s destino=%s sesion_destino=%d order=%d", payload.AgenteOrigen, payload.AgenteDestino, sesionID, order.ID)
	Audit(strings.TrimSpace(destino), "handoff_reanudado", "runtime_order", order.ID, detalle)
	emitirNotificacionHandoffReanudado(order, payload, sesionID)
	return nil
}

func emitirNotificacionHandoffReanudado(order *RuntimeOrder, payload HandoffPayload, sesionID int64) {
	if order == nil {
		return
	}
	proyectoID := int64(0)
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		proyectoID = *order.ProyectoID
	}
	destino := strings.TrimSpace(order.Agente)
	texto := strings.TrimSpace(payload.ResumenContinuidad)
	if texto == "" {
		texto = fmt.Sprintf("Handoff %s -> %s reanudado en sesion #%d", strings.TrimSpace(payload.AgenteOrigen), strings.TrimSpace(payload.AgenteDestino), sesionID)
	} else {
		texto = fmt.Sprintf("Handoff %s -> %s reanudado en sesion #%d | %s", strings.TrimSpace(payload.AgenteOrigen), strings.TrimSpace(payload.AgenteDestino), sesionID, texto)
	}
	payloadNotif := map[string]any{
		"order_id":            order.ID,
		"session_id":          sesionID,
		"source_agent":        strings.TrimSpace(payload.AgenteOrigen),
		"destination_agent":   strings.TrimSpace(payload.AgenteDestino),
		"continuity_summary":  strings.TrimSpace(payload.ResumenContinuidad),
		"external_session_id": strings.TrimSpace(payload.ExternalSessionID),
	}
	if payload.TareaID != nil && *payload.TareaID > 0 {
		payloadNotif["task_id"] = *payload.TareaID
	}
	EmitirNotificacion(EventoNotificacion{
		Tipo:       "runtime_handoff",
		ID:         order.ID,
		Agente:     destino,
		Texto:      texto,
		ProyectoID: proyectoID,
		Payload:    payloadNotif,
	})
}

func bootstrapResumenJSON(bootstrap *bootstrapRuntimeData) map[string]any {
	if bootstrap == nil {
		return map[string]any{}
	}
	return map[string]any{
		"order_id":      runtimeOrderIDOrZero(bootstrap.Order),
		"order_tipo":    runtimeOrderTipoOrEmpty(bootstrap.Order),
		"mailbox_count": len(bootstrap.Mailbox),
		"consumidos":    bootstrap.Consumidos,
		"checkpoint_id": checkpointIDOrZeroDB(bootstrap.Checkpoint),
	}
}

func marcarOrigenHandoffPausado(origen string, sesionOrigen *Sesion, handleOrigen *RuntimeHandle) error {
	obj := controlruntime.ObjetivoProceso{}
	if sesionOrigen != nil {
		obj.PID = sesionOrigen.PID
	}
	if handleOrigen != nil {
		obj.HandleKind = handleOrigen.HandleKind
		obj.HandleRef = handleOrigen.HandleRef
		obj.MetadataJSON = handleOrigen.MetadataJSON
	}
	aplicado, pid, err := controlruntime.PausarProceso(obj)
	if err != nil {
		return err
	}
	if sesionOrigen != nil {
		if _, err := DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesionOrigen.ID); err != nil {
			return err
		}
		runtime, err := runtimeHandleRuntime(handleOrigen)
		if err != nil {
			return err
		}
		if runtime == nil {
			runtime, err = GetRuntimeBySesionID(sesionOrigen.ID)
			if err != nil {
				return err
			}
		}
		if runtime != nil {
			if _, err := DB.Exec(`
				UPDATE runtime_instances
				SET logical_state='pausado',
				    process_state=CASE WHEN pid IS NOT NULL THEN 'pausado' ELSE process_state END,
				    last_event_at=CURRENT_TIMESTAMP
				WHERE id=?`, runtime.ID); err != nil {
				return err
			}
		}
	}
	if handleOrigen != nil {
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='pausado',
			    last_seen_at=CURRENT_TIMESTAMP
			WHERE id=?`, handleOrigen.ID); err != nil {
			return err
		}
		runtimeHandleHotReset()
	}
	detalle := fmt.Sprintf("origen=%s control_real=%t pid=%d", strings.TrimSpace(origen), aplicado, pid)
	var handleID int64
	if handleOrigen != nil {
		handleID = handleOrigen.ID
	}
	Audit(strings.TrimSpace(origen), "handoff_origen_pausado", "runtime_handle", handleID, detalle)
	return nil
}

func marcarOrigenHandoffAusente(origen string, sesionOrigen *Sesion, handleOrigen *RuntimeHandle) error {
	var proyectoID *int64
	if sesionOrigen != nil {
		proyectoID = sesionOrigen.ProyectoID
	}
	if err := aparcarSesionActiva(origen, proyectoID); err != nil {
		return err
	}
	if err := MarcarRuntimesCerradosPorAgente(origen); err != nil {
		return err
	}
	if err := MarcarRuntimeHandlesCerradosPorAgente(origen); err != nil {
		return err
	}
	if err := consumirWatchdogMailboxPendiente(origen, proyectoID); err != nil {
		return err
	}
	var handleID int64
	if handleOrigen != nil {
		handleID = handleOrigen.ID
	}
	Audit(strings.TrimSpace(origen), "handoff_origen_aparcado", "runtime_handle", handleID,
		fmt.Sprintf("origen=%s sin_handle_activo=true", strings.TrimSpace(origen)))
	return nil
}

func consumirWatchdogMailboxPendiente(agente string, proyectoID *int64) error {
	var err error
	if strings.TrimSpace(agente) != "" {
		agente, err = CanonicalizeAgentName(agente)
		if err != nil {
			return err
		}
		agente = strings.TrimSpace(agente)
		if agente == "" {
			return nil
		}
	}
	estado := "pendiente"
	mensajes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return err
	}
	consumidos := 0
	for _, msg := range mensajes {
		if msg == nil || strings.TrimSpace(msg.Kind) != "watchdog" {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return err
		}
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return err
		}
		consumidos++
	}
	if consumidos > 0 {
		Audit("orquesta", "handoff_watchdog_mailbox_consumido", "runtime_mailbox", 0,
			fmt.Sprintf("agente=%s consumidos=%d", agente, consumidos))
	}
	return nil
}

func runtimeOrderIDOrZero(order *RuntimeOrder) int64 {
	if order == nil {
		return 0
	}
	return order.ID
}

func runtimeOrderTipoOrEmpty(order *RuntimeOrder) string {
	if order == nil {
		return ""
	}
	return strings.TrimSpace(order.Tipo)
}

func strPtrRuntime(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func ejecutarRuntimeOrderHandoff(order *RuntimeOrder) error {
	var p HandoffPayload
	if err := json.Unmarshal([]byte(order.PayloadJSON), &p); err != nil {
		return fmt.Errorf("payload de handoff inválido: %w", err)
	}
	if strings.TrimSpace(p.AgenteOrigen) == "" {
		p.AgenteOrigen = order.Agente
	}
	if strings.TrimSpace(p.AgenteDestino) == "" {
		return fmt.Errorf("handoff sin agente destino")
	}

	orderID, err := CrearHandoffAgenteVivo(
		p.AgenteOrigen, p.AgenteDestino, p.TareaID,
		p.Motivo, p.ResumenContinuidad, p.ExternalSessionID,
	)
	if err != nil {
		return err
	}
	result := map[string]any{
		"ok":               true,
		"handoff_order_id": orderID,
		"agente_origen":    p.AgenteOrigen,
		"agente_destino":   p.AgenteDestino,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}
