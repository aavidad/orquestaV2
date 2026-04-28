package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/runtimeagente"
	"strings"
	"time"
)

func runtimeOrderStartDebeIgnorarBootstrap(payload map[string]any) bool {
	motivo := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "motivo", "")))
	return strings.Contains(motivo, "manual_rehabilitation") ||
		strings.Contains(motivo, "reanimacion_manual") ||
		strings.Contains(motivo, "rehabilitacion_manual")
}

func runtimeOrderControlRetryDelay() time.Duration {
	seconds := configIntOrDefault("runtime_control_retry_seconds", 5)
	if seconds <= 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderStartSesionAbiertaAjena(agente string, proyectoID *int64) (*Sesion, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
	sesion, err := GetSesionAbierta(agente, nil)
	if err == sql.ErrNoRows || sesion == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if proyectoID == nil || *proyectoID <= 0 || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return nil, nil
	}
	if *sesion.ProyectoID == *proyectoID {
		return nil, nil
	}
	return sesion, nil
}

func cerrarSesionFantasmaActiva(sesionID int64) error {
	if sesionID <= 0 {
		return nil
	}
	sesion, err := sesionIfExists(&sesionID)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		UPDATE sesiones
		SET activa = 0,
		    estado = 'cerrada',
		    fin = CURRENT_TIMESTAMP
		WHERE id = ?
		  AND activa = 1`, sesionID)
	if err != nil {
		return err
	}
	if sesion != nil {
		var activas int
		if err := DB.QueryRow(`SELECT COUNT(1) FROM sesiones WHERE agente=? AND activa=1`, sesion.Agente).Scan(&activas); err != nil {
			return err
		}
		if activas == 0 {
			if _, err := DB.Exec(`UPDATE agentes SET activo=0, estado_sesion='' WHERE nombre=?`, sesion.Agente); err != nil {
				return err
			}
		}
	}
	return nil
}

func asegurarStopSesionActivaAjena(startOrder *RuntimeOrder, sesion *Sesion) error {
	if startOrder == nil || sesion == nil || sesion.ProyectoID == nil || *sesion.ProyectoID <= 0 {
		return nil
	}
	if abierta, err := existeRuntimeOrderAbiertaAgenteProyecto(strings.TrimSpace(startOrder.Agente), sesion.ProyectoID, startOrder.ID, "stop"); err != nil {
		return err
	} else if abierta {
		return nil
	}
	proyectoRef := jsonNumber(*sesion.ProyectoID)
	proyecto, err := GetProyecto(proyectoRef)
	if err == nil && proyecto != nil && strings.TrimSpace(proyecto.Slug) != "" {
		proyectoRef = strings.TrimSpace(proyecto.Slug)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"accion":   "stop",
		"motivo":   fmt.Sprintf("sesion_activa_en_otro_proyecto:%s", strings.TrimSpace(proyectoRef)),
		"por":      "orquesta",
		"proyecto": strings.TrimSpace(proyectoRef),
	})
	if err != nil {
		return err
	}
	var handleID *int64
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		return err
	}
	if handle != nil && handle.ID > 0 {
		handleID = &handle.ID
	}
	var runtimeID *int64
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		return err
	}
	if runtime != nil && runtime.ID > 0 {
		runtimeID = &runtime.ID
	} else if handle != nil && handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtimeID = handle.RuntimeID
	}
	_, err = EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      strings.TrimSpace(startOrder.Agente),
		ProyectoID:  sesion.ProyectoID,
		RuntimeID:   runtimeID,
		HandleID:    handleID,
		Tipo:        "stop",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func reencolarRuntimeOrderControlAt(order *RuntimeOrder, reason string, nextAttempt time.Time) error {
	if order == nil || order.ID <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "runtime control reencolado"
	}
	if nextAttempt.IsZero() {
		nextAttempt = time.Now().UTC().Add(runtimeOrderControlRetryDelay())
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":              false,
		"deferred":        true,
		"deferred_reason": reason,
		"retry_after":     nextAttempt.Format(time.RFC3339Nano),
	})
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado = 'pendiente',
		    resultado_json = ?,
		    error_text = CASE
		        WHEN TRIM(COALESCE(error_text, '')) = '' THEN ?
		        ELSE error_text || ?
		    END,
		    started_at = NULL,
		    finished_at = NULL,
		    claimed_by = '',
		    lease_token = '',
		    lease_expires_at = NULL,
		    available_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		  AND estado NOT IN ('completada','fallida','cancelada','expirada')`,
		resultado,
		reason,
		"\n"+reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}

func reutilizarSesionArranquePoolLocal(agente string, proyectoID *int64, conector *Conector, externalSessionID string, plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext, ultima *Sesion, host string, pid *int64) (*Sesion, bool, error) {
	if conector == nil || !strings.EqualFold(strings.TrimSpace(conector.Slug), "ollama_pool_local") || proyectoID == nil || *proyectoID <= 0 {
		return nil, false, nil
	}
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, false, err
	}
	sesion, err := GetSesionActiva(agente, proyectoID)
	if err == sql.ErrNoRows || sesion == nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	herramienta := strings.TrimSpace(sesion.Herramienta)
	conectorSesion := strings.TrimSpace(sesion.ConectorSlug)
	if !strings.EqualFold(herramienta, "ollama_pool_local") && !strings.EqualFold(conectorSesion, "ollama_pool_local") {
		return nil, false, nil
	}
	if ext := strings.TrimSpace(externalSessionID); ext != "" {
		actual := strings.TrimSpace(sesion.ExternalSessionID)
		if actual != "" && !strings.EqualFold(actual, ext) {
			return nil, false, nil
		}
	}
	cwd := strings.TrimSpace(plan.WorkingDir)
	resumePayload := payloadJSONDesdePlanYResume(plan, resume)
	resumen := resumenContinuidadDesdeResume(ultima, resume)
	branch := branchDesdeResume(ultima, resume)
	estado := "activa"
	if err := GuardarSesionActiva(agente, proyectoID, SesionUpdate{
		CWD:                &cwd,
		Herramienta:        &conector.Slug,
		ExternalSessionID:  &externalSessionID,
		ResumePayloadJSON:  &resumePayload,
		ResumenContinuidad: &resumen,
		Branch:             &branch,
		Host:               &host,
		PID:                pid,
		Estado:             &estado,
		Heartbeat:          true,
	}); err != nil {
		return nil, false, err
	}
	actualizada, err := GetSesionActiva(agente, proyectoID)
	if err != nil {
		return nil, false, err
	}
	return actualizada, true, nil
}

func ackBootstrapRuntimeSinLeaseEnStart(startOrder *RuntimeOrder, bootstrap *bootstrapRuntimeData, sesion *Sesion) error {
	if bootstrap == nil || bootstrap.Order != nil {
		return nil
	}
	if len(bootstrap.Mailbox) == 0 {
		return nil
	}
	var sesionID int64
	if sesion != nil {
		sesionID = sesion.ID
	}
	if startOrder != nil && startOrder.ID > 0 {
		if strings.EqualFold(strings.TrimSpace(startOrder.Estado), "ejecutando") {
			resultado := mergeRuntimeOrderResultJSON(startOrder.ResultadoJSON, map[string]any{
				"ok":            true,
				"lease_state":   "waiting_for_evidence",
				"mailbox_count": len(bootstrap.Mailbox),
				"mailbox_ids":   runtimeMailboxIDs(bootstrap.Mailbox),
				"sesion_id":     sesionID,
				"ack_mode":      "runtime_transcript_or_tick",
				"continuidad":   true,
				"order_tipo":    "start",
			})
			if err := MarcarRuntimeOrderEstado(startOrder.ID, startOrder.Estado, resultado, startOrder.ErrorText); err != nil {
				return err
			}
		}
		Audit("orquesta", "runtime_bootstrap_start_ack", "runtime_order", startOrder.ID,
			fmt.Sprintf("sesion_id=%d mailbox_count=%d ack_mode=start_success_no_mailbox_ack", sesionID, len(bootstrap.Mailbox)))
	}
	return nil
}
