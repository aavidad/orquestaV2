package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func EncolarRuntimeOrder(order *RuntimeOrder) (int64, error) {
	if order == nil || strings.TrimSpace(order.Agente) == "" || strings.TrimSpace(order.Tipo) == "" {
		return 0, sql.ErrNoRows
	}
	var err error
	order.Agente, err = CanonicalizeAgentName(order.Agente)
	if err != nil {
		return 0, err
	}
	payloadJSON, err := normalizarRuntimeOrderPayloadJSON(order.PayloadJSON)
	if err != nil {
		return 0, err
	}
	order.PayloadJSON = payloadJSON
	if strings.TrimSpace(order.ResultadoJSON) == "" {
		order.ResultadoJSON = "{}"
	}
	id, err := insertReturningID(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, runtime_id, handle_id, tipo, payload_json, resultado_json,
			error_text, estado, available_at
		) VALUES (?,?,?,?,?,?,?,?,?,COALESCE(?, CURRENT_TIMESTAMP))`,
		order.Agente, order.ProyectoID, order.RuntimeID, order.HandleID, order.Tipo,
		order.PayloadJSON, order.ResultadoJSON, order.ErrorText, defaultRuntimeOrderEstado(order.Estado), nullableTime(order.AvailableAt),
	)
	if err != nil {
		return 0, err
	}
	if err := runtimeOrdersHotIndexSyncByID(id); err != nil {
		return 0, err
	}
	return id, nil
}

func normalizarRuntimeOrderPayloadJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("payload_json inválido: %w", err)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("payload_json inválido: %w", err)
	}
	return string(data), nil
}

func runtimeOrderPayloadMap(raw string) (map[string]any, error) {
	normalized, err := normalizarRuntimeOrderPayloadJSON(raw)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		return nil, fmt.Errorf("payload_json inválido: %w", err)
	}
	return payload, nil
}

func runtimeOrderSendInstructionReceiptAtOrAfter(candidate, baseline time.Time) bool {
	if candidate.IsZero() {
		return false
	}
	if baseline.IsZero() {
		return true
	}
	return !candidate.UTC().Before(baseline.UTC())
}

func runtimeTranscriptLooksLikeMicroprogramacionPatch(entry *RuntimeTranscriptEntry) bool {
	if entry == nil {
		return false
	}
	return runtimeMicroprogramacionPatchDetected(entry.Text, entry.NormalizedText)
}

func runtimeMicroprogramacionPatchDetected(text, normalized string) bool {
	normalized = strings.TrimSpace(strings.ToLower(normalized))
	if normalized == "" {
		normalized = strings.TrimSpace(strings.ToLower(normalizarTextoTranscript(text)))
	}
	if normalized == "" {
		return false
	}
	switch {
	case strings.Contains(normalized, "patch_unificado"):
		return true
	case strings.Contains(normalized, "// file:"):
		return true
	case strings.Contains(normalized, "diff --git"):
		return true
	case strings.HasPrefix(strings.TrimSpace(text), "--- "):
		return true
	case strings.HasPrefix(strings.TrimSpace(text), "+++ "):
		return true
	case strings.Contains(strings.TrimSpace(text), "\n--- "):
		return true
	case strings.Contains(strings.TrimSpace(text), "\n+++ "):
		return true
	default:
		return false
	}
}

func runtimeTranscriptLooksLikeMicroprogramacionBlockedResponse(text, normalized string) bool {
	normalized = strings.TrimSpace(strings.ToLower(normalized))
	if normalized == "" {
		normalized = strings.TrimSpace(strings.ToLower(normalizarTextoTranscript(text)))
	}
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "si_bloqueo") || strings.Contains(normalized, "si-bloqueo") {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		lineNormalized := strings.TrimSpace(strings.ToLower(normalizarTextoTranscript(line)))
		if strings.HasPrefix(lineNormalized, "bloqueo:") {
			return true
		}
	}
	return strings.HasPrefix(normalized, "bloqueo:")
}

func runtimeOrderSendInstructionEstaNotificada(order *RuntimeOrder) bool {
	if order == nil {
		return false
	}
	if !runtimeOrderSendInstructionExplicitNotifiedAt(order).IsZero() {
		return true
	}
	result := mapFromJSON(order.ResultadoJSON)
	if len(result) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(stringFromMap(result, "dispatch_state", ""))) {
	case "notified", "delivered":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(stringFromMap(result, "delivery_state", ""))) {
	case "notified", "delivered":
		return true
	}
	return false
}

func runtimeOrderSendInstructionDebeCerrarMailboxFaltante(order *RuntimeOrder, payload map[string]any) bool {
	if order == nil || !runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return false
	}
	return runtimeOrderSendInstructionEstaNotificada(order)
}

func runtimeOrderSendInstructionDuplicadaMasReciente(order *RuntimeOrder, payload map[string]any) (*RuntimeOrder, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return nil, nil
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		return nil, nil
	}
	args := []any{
		strings.TrimSpace(order.Agente),
		order.ID,
	}
	whereTask, taskArgs := runtimeJSONIntFieldLikeWhere("payload_json", mailboxID, "mailbox_id")
	args = append(args, taskArgs...)
	query := `
		SELECT id, agente, proyecto_id, runtime_id, handle_id, tipo, payload_json, resultado_json,
		       error_text, estado, claimed_by, lease_token, attempt_count, lease_expires_at,
		       available_at, created_at, started_at, finished_at, updated_at
		FROM runtime_orders
		WHERE tipo = 'send_instruction'
		  AND TRIM(agente) = ?
		  AND id <> ?
		  AND estado IN ('pendiente','tomada','ejecutando')
		  AND ` + whereTask
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		query += ` AND proyecto_id = ?`
		args = append(args, *order.ProyectoID)
	}
	query += `
		ORDER BY created_at DESC, id DESC
		LIMIT 1`
	row := DB.QueryRow(query, args...)
	other, err := scanRuntimeOrder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if other == nil || other.ID <= order.ID {
		return nil, nil
	}
	return other, nil
}

func runtimeOrderSendInstructionDuplicadaMasRecientePorTarea(order *RuntimeOrder, payload map[string]any) (*RuntimeOrder, error) {
	if order == nil {
		return nil, nil
	}
	taskID := runtimeOrderSendInstructionTaskID(payload)
	if taskID <= 0 {
		return nil, nil
	}
	args := []any{
		strings.TrimSpace(order.Agente),
		order.ID,
	}
	whereTask, taskArgs := runtimeJSONIntFieldLikeWhere("payload_json", taskID, "tarea_id", "tarea_objetivo_id")
	args = append(args, taskArgs...)
	query := `
		SELECT id, agente, proyecto_id, runtime_id, handle_id, tipo, payload_json, resultado_json,
		       error_text, estado, claimed_by, lease_token, attempt_count, lease_expires_at,
		       available_at, created_at, started_at, finished_at, updated_at
		FROM runtime_orders
		WHERE tipo = 'send_instruction'
		  AND TRIM(agente) = ?
		  AND id <> ?
		  AND estado IN ('pendiente','tomada','ejecutando')
		  AND ` + whereTask
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		query += ` AND proyecto_id = ?`
		args = append(args, *order.ProyectoID)
	}
	query += `
		ORDER BY created_at DESC, id DESC
		LIMIT 1`
	row := DB.QueryRow(query, args...)
	other, err := scanRuntimeOrder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if other == nil || other.ID <= order.ID {
		return nil, nil
	}
	otherPayload, err := runtimeOrderPayloadMap(other.PayloadJSON)
	if err != nil {
		return nil, err
	}
	if !runtimeOrderSendInstructionTaskScopeEquivalent(payload, otherPayload) {
		return nil, nil
	}
	return other, nil
}

func runtimeJSONIntFieldLikeWhere(column string, value int64, fields ...string) (string, []any) {
	clauses := make([]string, 0, len(fields)*2)
	args := make([]any, 0, len(fields)*2)
	token := strconv.FormatInt(value, 10)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		for _, suffix := range []string{",", "}"} {
			clauses = append(clauses, column+` LIKE ?`)
			args = append(args, `%"`+field+`":`+token+suffix+`%`)
		}
	}
	if len(clauses) == 0 {
		return "1=0", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func runtimeOrderSendInstructionTaskScopeEquivalent(current, other map[string]any) bool {
	if runtimeOrderSendInstructionProvieneMailbox(current) != runtimeOrderSendInstructionProvieneMailbox(other) {
		return false
	}
	if runtimeOrderSendInstructionSemanticKind(current) != runtimeOrderSendInstructionSemanticKind(other) {
		return false
	}
	if strings.TrimSpace(strings.ToLower(anyToString(current["accion"]))) != strings.TrimSpace(strings.ToLower(anyToString(other["accion"]))) {
		return false
	}
	currentVerification := strings.TrimSpace(anyToString(current["verification_key"]))
	otherVerification := strings.TrimSpace(anyToString(other["verification_key"]))
	if currentVerification != "" || otherVerification != "" {
		return currentVerification == otherVerification
	}
	return true
}

func runtimeOrderSendInstructionSemanticKind(payload map[string]any) string {
	kind := strings.TrimSpace(strings.ToLower(anyToString(payload["mailbox_kind"])))
	if kind == "" {
		return "instruction"
	}
	return kind
}

func runtimeOrderSendInstructionTaskID(payload map[string]any) int64 {
	if payload == nil {
		return 0
	}
	if id := int64FromAny(payload["tarea_id"]); id > 0 {
		return id
	}
	return int64FromAny(payload["tarea_objetivo_id"])
}

func runtimeOrderSendInstructionObsoletaPorTarea(order *RuntimeOrder, payload map[string]any) (string, error) {
	if order == nil || !runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return "", nil
	}
	taskID := runtimeOrderSendInstructionTaskID(payload)
	if taskID <= 0 {
		return "", nil
	}
	tarea, err := GetTarea(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Sprintf("task_missing:%d", taskID), nil
		}
		return "", err
	}
	if tarea == nil {
		return fmt.Sprintf("task_missing:%d", taskID), nil
	}
	if tarea.Agente != nil {
		agente := strings.TrimSpace(*tarea.Agente)
		if agente != "" && !strings.EqualFold(agente, strings.TrimSpace(order.Agente)) {
			return fmt.Sprintf("task_reassigned_to:%s", agente), nil
		}
	}
	switch tarea.Estado {
	case TareaCompletada, TareaCancelada, TareaLibre, TareaBacklog:
		return fmt.Sprintf("task_not_actionable:%s", strings.TrimSpace(string(tarea.Estado))), nil
	}
	return "", nil
}

type handoffCreateOptions struct {
	RequireOriginHandle bool
	AuditAction         string
}

func CrearHandoffAgenteVivo(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return crearHandoffAgente(origen, destino, tareaID, motivo, resumenContinuidad, externalSessionID, handoffCreateOptions{
		RequireOriginHandle: true,
		AuditAction:         "handoff_agente_vivo",
	})
}

func CrearHandoffAgenteStale(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	return crearHandoffAgente(origen, destino, tareaID, motivo, resumenContinuidad, externalSessionID, handoffCreateOptions{
		RequireOriginHandle: false,
		AuditAction:         "handoff_agente_stale",
	})
}

func crearHandoffAgente(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string, opts handoffCreateOptions) (int64, error) {
	origen = strings.TrimSpace(origen)
	destino = strings.TrimSpace(destino)
	var err error
	origen, err = CanonicalizeAgentName(origen)
	if err != nil {
		return 0, err
	}
	destino, err = CanonicalizeAgentName(destino)
	if err != nil {
		return 0, err
	}
	if origen == "" || destino == "" {
		return 0, fmt.Errorf("agente origen y destino son obligatorios")
	}
	if origen == destino {
		return 0, fmt.Errorf("origen y destino deben ser distintos")
	}

	sesionOrigen, err := GetSesionAbierta(origen, nil)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("el agente origen %s no tiene sesión abierta", origen)
	}
	if err != nil {
		return 0, err
	}
	handleOrigen, err := runtimeHandleCanonicoRecienteConFallback(origen, nil)
	if err != nil {
		return 0, err
	}
	if opts.RequireOriginHandle && handleOrigen == nil {
		return 0, fmt.Errorf("el agente origen %s no tiene runtime handle activo", origen)
	}

	var (
		proyectoID     *int64
		destinoHandle  *RuntimeHandle
		destinoRuntime *RuntimeInstance
	)
	if sesionOrigen != nil && sesionOrigen.ProyectoID != nil {
		proyectoID = sesionOrigen.ProyectoID
	}

	sesionDestino, err := GetSesionActiva(destino, nil)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if sesionDestino != nil && sesionDestino.ProyectoID != nil {
		proyectoID = sesionDestino.ProyectoID
	}
	destinoHandle, err = runtimeHandleOperativoRecienteConFallback(destino, nil)
	if err != nil {
		return 0, err
	}
	if destinoHandle != nil {
		if destinoHandle.ProyectoID != nil {
			proyectoID = destinoHandle.ProyectoID
		}
		if destinoHandle.RuntimeID != nil {
			destinoRuntime, err = runtimeInstanceIfExists(destinoHandle.RuntimeID)
			if err != nil {
				return 0, err
			}
		} else if destinoHandle.SesionID != nil {
			destinoRuntime, err = GetRuntimeBySesionID(*destinoHandle.SesionID)
			if err != nil {
				return 0, err
			}
		}
	}

	if existenteID, err := buscarHandoffPendienteEquivalente(origen, destino, tareaID, proyectoID); err != nil {
		return 0, err
	} else if existenteID > 0 {
		if err := asegurarEntregaBootstrapHandoffDestino(existenteID, proyectoID, destino, destinoHandle); err != nil {
			return existenteID, err
		}
		return existenteID, nil
	}
	if tareaID != nil {
		t, err := GetTarea(*tareaID)
		if err != nil {
			return 0, fmt.Errorf("tarea #%d no encontrada", *tareaID)
		}
		if t.Agente != nil && *t.Agente != origen {
			return 0, fmt.Errorf("la tarea #%d no pertenece a %s", *tareaID, origen)
		}
		if t.ProyectoID != nil {
			proyectoID = t.ProyectoID
		}
	}
	if err := ValidarCompatibilidadGobernanzaHandoff(origen, destino, proyectoID); err != nil {
		return 0, err
	}

	payload := HandoffPayload{
		AgenteOrigen:       origen,
		AgenteDestino:      destino,
		TareaID:            tareaID,
		Motivo:             strings.TrimSpace(motivo),
		ResumenContinuidad: strings.TrimSpace(resumenContinuidad),
		ExternalSessionID:  strings.TrimSpace(externalSessionID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if tareaID != nil {
		if _, err := tx.Exec(`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, destino, *tareaID); err != nil {
			return 0, err
		}
		nota := fmt.Sprintf("handoff %s→%s", origen, destino)
		if payload.Motivo != "" {
			nota += ": " + payload.Motivo
		}
		if _, err := tx.Exec(
			`UPDATE tareas SET notas = COALESCE(notas,'') || ? WHERE id=?`,
			formatearAnotacionTarea("orquesta", nota, time.Now().UTC()), *tareaID,
		); err != nil {
			return 0, err
		}
	}
	if proyectoID != nil && *proyectoID > 0 {
		notaOrigen := fmt.Sprintf("handoff_cedido_a_%s", destino)
		if payload.Motivo != "" {
			notaOrigen += ": " + payload.Motivo
		}
		if _, err := pausarAsignacionTx(tx, origen, *proyectoID, notaOrigen); err != nil {
			return 0, err
		}

		notaDestino := fmt.Sprintf("handoff_recibido_desde_%s", origen)
		if payload.Motivo != "" {
			notaDestino += ": " + payload.Motivo
		}
		if err := activarAsignacionTx(tx, destino, *proyectoID, notaDestino); err != nil {
			return 0, err
		}
	}

	var runtimeID *int64
	if destinoRuntime != nil {
		runtimeID = &destinoRuntime.ID
	}
	var handleID *int64
	if destinoHandle != nil {
		handleID = &destinoHandle.ID
	}

	orderID, err := insertReturningIDWith(tx, `
		INSERT INTO runtime_orders (
			agente, proyecto_id, runtime_id, handle_id, tipo, payload_json,
			resultado_json, error_text, estado, available_at
		) VALUES (?,?,?,?,?,?,'{}','', 'pendiente', CURRENT_TIMESTAMP)`,
		destino, proyectoID, runtimeID, handleID, "handoff", string(payloadJSON),
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if err := runtimeOrdersHotIndexSyncByID(orderID); err != nil {
		return orderID, err
	}
	if opts.RequireOriginHandle {
		if err := marcarOrigenHandoffPausado(origen, sesionOrigen, handleOrigen); err != nil {
			return 0, err
		}
	} else {
		if err := marcarOrigenHandoffAusente(origen, sesionOrigen, handleOrigen); err != nil {
			return 0, err
		}
	}
	detalle := fmt.Sprintf("%s→%s sesion_origen=%d handle_activo=%t", origen, destino, sesionOrigen.ID, handleOrigen != nil)
	Audit(origen, strings.TrimSpace(opts.AuditAction), "runtime_order", orderID, detalle)
	if err := asegurarEntregaBootstrapHandoffDestino(orderID, proyectoID, destino, destinoHandle); err != nil {
		return orderID, err
	}
	return orderID, nil
}

func asegurarEntregaBootstrapHandoffDestino(orderID int64, proyectoID *int64, destino string, destinoHandle *RuntimeHandle) error {
	if orderID <= 0 {
		return nil
	}
	destino = strings.TrimSpace(destino)
	if destino == "" && destinoHandle != nil {
		destino = strings.TrimSpace(destinoHandle.Agente)
	}
	if destino == "" {
		return nil
	}
	targetProjectID := proyectoID
	if targetProjectID == nil && destinoHandle != nil && destinoHandle.ProyectoID != nil {
		targetProjectID = destinoHandle.ProyectoID
	}
	if targetProjectID == nil || *targetProjectID <= 0 {
		return nil
	}
	if abierta, err := existeRuntimeOrderAbiertaDestino(*targetProjectID, destino, "stop", "start", "restart", "resume"); err != nil {
		return err
	} else if abierta {
		return nil
	}
	proyecto, err := GetProyecto(strconv.FormatInt(*targetProjectID, 10))
	if err != nil {
		return err
	}
	motivo := fmt.Sprintf("handoff_bootstrap:%d", orderID)
	if destinoHandle == nil {
		startOrderID, err := encolarStartBootstrapAgenteProyecto(destino, targetProjectID, strings.TrimSpace(proyecto.Slug), motivo, "orquesta")
		if err != nil {
			return err
		}
		Audit("orquesta", "handoff_destino_reinicio_bootstrap", "runtime_order", orderID,
			fmt.Sprintf("agente=%s proyecto=%s stop_order_id=%d start_order_id=%d", destino, strings.TrimSpace(proyecto.Slug), 0, startOrderID))
		return nil
	}
	var stopOrderID int64
	if runtimeHandlePermiteReinicioBootstrapHandoff(destinoHandle) {
		stopPayload, err := json.Marshal(map[string]any{
			"accion":   "stop",
			"proyecto": strings.TrimSpace(proyecto.Slug),
			"motivo":   motivo,
			"por":      "orquesta",
		})
		if err != nil {
			return err
		}
		stopOrder := &RuntimeOrder{
			Agente:      destino,
			ProyectoID:  targetProjectID,
			Tipo:        "stop",
			PayloadJSON: string(stopPayload),
			HandleID:    &destinoHandle.ID,
		}
		if destinoHandle.RuntimeID != nil && *destinoHandle.RuntimeID > 0 {
			stopOrder.RuntimeID = destinoHandle.RuntimeID
		}
		stopOrderID, err = EncolarRuntimeOrder(stopOrder)
		if err != nil {
			return err
		}
	} else if destinoHandle != nil {
		return nil
	}
	startPayload, err := json.Marshal(map[string]any{
		"accion":   "start",
		"proyecto": strings.TrimSpace(proyecto.Slug),
		"motivo":   motivo,
		"por":      "orquesta",
	})
	if err != nil {
		return err
	}
	startOrderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      destino,
		ProyectoID:  targetProjectID,
		Tipo:        "start",
		PayloadJSON: string(startPayload),
	})
	if err != nil {
		return err
	}
	Audit("orquesta", "handoff_destino_reinicio_bootstrap", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s stop_order_id=%d start_order_id=%d", destino, strings.TrimSpace(proyecto.Slug), stopOrderID, startOrderID))
	return nil
}

func encolarStartBootstrapAgenteProyecto(agente string, proyectoID *int64, proyectoSlug, motivo, actor string) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID == nil || *proyectoID <= 0 {
		return 0, nil
	}
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	if proyectoSlug == "" {
		proyecto, err := GetProyecto(strconv.FormatInt(*proyectoID, 10))
		if err != nil {
			return 0, err
		}
		proyectoSlug = strings.TrimSpace(proyecto.Slug)
	}
	startPayload, err := json.Marshal(map[string]any{
		"accion":   "start",
		"proyecto": proyectoSlug,
		"motivo":   strings.TrimSpace(motivo),
		"por":      strings.TrimSpace(actor),
	})
	if err != nil {
		return 0, err
	}
	return EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      agente,
		ProyectoID:  proyectoID,
		Tipo:        "start",
		PayloadJSON: string(startPayload),
	})
}

func runtimeHandleContextoEntregaScore(handle *RuntimeHandle) int {
	if handle == nil {
		return 0
	}
	return runtimepolicy.RuntimeHandleDeliveryContextScore(
		strings.TrimSpace(handle.MetadataJSON),
		strings.TrimSpace(handle.CapabilitiesJSON),
	)
}

func actualizarRuntimeOrderDestino(orderID int64, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if orderID <= 0 || handle == nil || handle.ID <= 0 {
		return nil
	}
	var runtimeID any
	if runtime != nil && runtime.ID > 0 {
		runtimeID = runtime.ID
	} else if resolvedRuntime, err := runtimeHandleRuntime(handle); err != nil {
		return err
	} else if resolvedRuntime != nil && resolvedRuntime.ID > 0 {
		runtimeID = resolvedRuntime.ID
	} else if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtimeID = *handle.RuntimeID
	}
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET handle_id = ?,
		    runtime_id = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		handle.ID,
		runtimeID,
		orderID,
	)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(orderID)
}

func runtimeHandlePermiteReinicioBootstrapHandoff(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if runtimeHandlePreservesExternalSession(handle) {
		return false
	}
	if RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryBootstrapOnly {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(driver, "tmux_cli_session") {
		return true
	}
	if strings.TrimSpace(handle.Transporte) == "cli" &&
		strings.TrimSpace(handle.HandleKind) == "process" &&
		!runtimepolicy.RuntimeHandleUsaLegacyProcessPTY(meta) {
		return true
	}
	return false
}

func resolverDestinoRuntimeOrderSendInstruction(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (*RuntimeInstance, *RuntimeHandle, error) {
	explicitRuntime := runtime
	explicitHandle := handle
	resolvedRuntime, resolvedHandle, err := resolverDestinoRuntimeOrderCanonico(order, runtime, handle)
	if err != nil {
		return nil, nil, err
	}
	if resolvedHandle == nil && order != nil {
		operationalHandle, operationalErr := runtimeHandleOperativoRecienteConFallback(order.Agente, order.ProyectoID)
		if operationalErr != nil {
			return nil, nil, operationalErr
		}
		if operationalHandle != nil {
			operationalRuntime, runtimeErr := runtimeHandleRuntime(operationalHandle)
			if runtimeErr != nil {
				return nil, nil, runtimeErr
			}
			if !runtimeOrderSendInstructionDebePreferirHandleExplicito(explicitHandle, operationalHandle) {
				if err := refrescarRuntimeOrderDestinoCanonico(order, operationalRuntime, operationalHandle); err != nil {
					return nil, nil, err
				}
				return operationalRuntime, operationalHandle, nil
			}
		}
	}
	if resolvedHandle == nil && order != nil {
		sesionRuntime, sesionHandle, sesionErr := resolverDestinoRuntimeOrderSendInstructionSesionActiva(order)
		if sesionErr != nil {
			return nil, nil, sesionErr
		}
		if sesionHandle != nil {
			if err := refrescarRuntimeOrderDestinoCanonico(order, sesionRuntime, sesionHandle); err != nil {
				return nil, nil, err
			}
			return sesionRuntime, sesionHandle, nil
		}
	}
	if resolvedHandle != nil || resolvedRuntime != nil {
		if runtimeOrderSendInstructionDebePreferirHandleExplicito(explicitHandle, resolvedHandle) {
			if explicitRuntime == nil && explicitHandle != nil {
				explicitRuntime, err = runtimeHandleRuntime(explicitHandle)
				if err != nil {
					return nil, nil, err
				}
			}
			return explicitRuntime, explicitHandle, nil
		}
		return resolvedRuntime, resolvedHandle, nil
	}
	if explicitHandle != nil && runtimeHandleTieneContextoEntrega(explicitHandle) {
		if explicitRuntime == nil {
			explicitRuntime, err = runtimeHandleRuntime(explicitHandle)
			if err != nil {
				return nil, nil, err
			}
		}
		return explicitRuntime, explicitHandle, nil
	}
	if !runtimeOrderSendInstructionConservaHandleExplicito(order, explicitHandle) {
		return resolvedRuntime, resolvedHandle, nil
	}
	if explicitRuntime == nil && explicitHandle != nil {
		explicitRuntime, err = runtimeHandleRuntime(explicitHandle)
		if err != nil {
			return nil, nil, err
		}
	}
	return explicitRuntime, explicitHandle, nil
}

func resolverDestinoRuntimeOrderSendInstructionSesionActiva(order *RuntimeOrder) (*RuntimeInstance, *RuntimeHandle, error) {
	if order == nil {
		return nil, nil, nil
	}
	sesion, err := GetSesionActiva(strings.TrimSpace(order.Agente), order.ProyectoID)
	if err == sql.ErrNoRows || sesion == nil {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(sesion.Herramienta), "ollama_pool_local") &&
		!strings.EqualFold(strings.TrimSpace(sesion.ConectorSlug), "ollama_pool_local") {
		return nil, nil, nil
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	if handle == nil {
		return runtime, nil, nil
	}
	return runtime, handle, nil
}

func runtimeOrderSendInstructionDebePreferirHandleExplicito(explicitHandle, resolvedHandle *RuntimeHandle) bool {
	if explicitHandle == nil || resolvedHandle == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(explicitHandle.Estado)) {
	case "cerrado", "fallido", "fantasma":
		return false
	}
	explicitScore := runtimeHandleContextoEntregaScore(explicitHandle)
	resolvedScore := runtimeHandleContextoEntregaScore(resolvedHandle)
	if explicitHandle.ID == resolvedHandle.ID {
		return explicitScore > resolvedScore
	}
	return explicitScore > resolvedScore &&
		runtimeHandleTieneContextoEntrega(explicitHandle) &&
		!runtimeHandleTieneContextoEntrega(resolvedHandle)
}

func runtimeOrderSendInstructionConservaHandleExplicito(order *RuntimeOrder, handle *RuntimeHandle) bool {
	if order == nil || handle == nil || order.HandleID == nil || *order.HandleID != handle.ID {
		return false
	}
	if strings.TrimSpace(handle.MetadataJSON) != "" {
		return true
	}
	if strings.TrimSpace(handle.HandleKind) != "" || strings.TrimSpace(handle.HandleRef) != "" {
		return true
	}
	return false
}

func runtimeHandleTieneContextoEntrega(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	return runtimepolicy.RuntimeHandleHasDeliveryContext(
		strings.TrimSpace(handle.MetadataJSON),
		strings.TrimSpace(handle.CapabilitiesJSON),
	)
}

func runtimeOrderSendInstructionReceiptTimeout() time.Duration {
	seconds := configIntOrDefault("runtime_send_instruction_receipt_timeout_seconds", 120)
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderSendInstructionExplicitNotifiedAt(order *RuntimeOrder) time.Time {
	if order == nil {
		return time.Time{}
	}
	result := mapFromJSON(order.ResultadoJSON)
	if result == nil {
		return time.Time{}
	}
	if raw := strings.TrimSpace(stringFromMap(result, "delivery_notified_at", "")); raw != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, raw); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}

func runtimeOrderSendInstructionNotifiedAt(order *RuntimeOrder, msg *RuntimeMailboxMessage) time.Time {
	if explicit := runtimeOrderSendInstructionExplicitNotifiedAt(order); !explicit.IsZero() {
		return explicit
	}
	if msg != nil && msg.DeliveredAt != nil && !msg.DeliveredAt.IsZero() {
		return msg.DeliveredAt.UTC()
	}
	return time.Time{}
}

func runtimeOrderSendInstructionReceiptBaseline(order *RuntimeOrder, msg *RuntimeMailboxMessage) time.Time {
	explicitNotifiedAt := runtimeOrderSendInstructionExplicitNotifiedAt(order)
	baseline := explicitNotifiedAt
	if baseline.IsZero() && msg != nil && msg.DeliveredAt != nil && !msg.DeliveredAt.IsZero() {
		baseline = msg.DeliveredAt.UTC()
	}
	if baseline.IsZero() && msg != nil && !msg.CreatedAt.IsZero() {
		baseline = msg.CreatedAt.UTC()
	}
	if order != nil {
		if !order.CreatedAt.IsZero() && order.CreatedAt.UTC().After(baseline) {
			baseline = order.CreatedAt.UTC()
		}
		if order.StartedAt != nil && !order.StartedAt.IsZero() && order.StartedAt.UTC().After(baseline) {
			baseline = order.StartedAt.UTC()
		}
	}
	return baseline
}

func runtimeOrderSendInstructionEsPipelinePremium(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	kind := strings.TrimSpace(strings.ToLower(anyToString(payload["mailbox_kind"])))
	return kind == "pipeline_local"
}

func runtimeOrderSendInstructionPermiteReceiptLastProgress(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	kind := strings.TrimSpace(strings.ToLower(anyToString(payload["mailbox_kind"])))
	switch kind {
	case MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func anyToString(v any) string {
	switch vv := v.(type) {
	case string:
		return vv
	default:
		return ""
	}
}

func runtimeOrderSendInstructionPermiteReceiptTranscriptPatch(payload map[string]any) bool {
	if payload == nil || !runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		return true
	}
	micro, _ := payload["microprogramacion"].(map[string]any)
	formato := strings.TrimSpace(stringFromMap(micro, "formato_salida", ""))
	if formato == "" {
		return true
	}
	formato = strings.ToLower(formato)
	return strings.Contains(formato, "git_worktree")
}

func existeRuntimeOrderAbiertaDestino(proyectoID int64, agente string, tipos ...string) (bool, error) {
	if proyectoID <= 0 || strings.TrimSpace(agente) == "" || len(tipos) == 0 {
		return false, nil
	}
	for _, estado := range []string{"pendiente", "tomada", "ejecutando"} {
		estado := estado
		orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: &proyectoID,
			Estado:     &estado,
			Tipos:      append([]string(nil), tipos...),
			Limit:      1,
		})
		if err != nil {
			return false, err
		}
		if len(orders) > 0 && orders[0] != nil {
			return true, nil
		}
	}
	return false, nil
}

func ClaimNextRuntimeOrder(agente string) (*RuntimeOrder, error) {
	for {
		estado := "pendiente"
		orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
			Agente: stringsPtrTrimmed(agente),
			Estado: &estado,
		})
		if err != nil {
			return nil, err
		}
		if len(orders) == 0 {
			return nil, nil
		}
		sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
		now := time.Now().UTC()
		for _, order := range orders {
			if order == nil || !runtimeOrderPendingReady(order, now) {
				continue
			}
			claimed, err := claimRuntimeOrderByID(order.ID)
			if err != nil {
				return nil, err
			}
			if claimed == nil {
				continue
			}
			return claimed, nil
		}
		return nil, nil
	}
}

func runtimeOrderLeaseDuration() time.Duration {
	seconds := configIntOrDefault("runtime_order_lease_seconds", 120)
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderLeaseDurationForType(tipo string) time.Duration {
	switch strings.TrimSpace(tipo) {
	case "send_instruction":
		seconds := configIntOrDefault("runtime_send_instruction_lease_seconds", 35)
		if seconds <= 0 {
			seconds = 35
		}
		return time.Duration(seconds) * time.Second
	default:
		return runtimeOrderLeaseDuration()
	}
}

func runtimeOrderLeaseDeadline(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Add(runtimeOrderLeaseDuration())
}

func runtimeOrderLeaseDeadlineForType(tipo string, now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Add(runtimeOrderLeaseDurationForType(tipo))
}

func runtimeOrderClaimedBy() string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("control_plane:%s", host)
}

func runtimeOrderLeaseToken(orderID int64, now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return fmt.Sprintf("order:%d:%d", orderID, now.UTC().UnixNano())
}

func ClaimNextBootstrapRuntimeOrder(agente string, proyectoID *int64) (*RuntimeOrder, error) {
	id, err := nextBootstrapRuntimeOrderID(strings.TrimSpace(agente), proyectoID)
	if err != nil || id == 0 {
		return nil, err
	}
	for {
		order, err := claimRuntimeOrderByID(id)
		if err != nil {
			return nil, err
		}
		if order == nil {
			id, err = nextBootstrapRuntimeOrderID(strings.TrimSpace(agente), proyectoID)
			if err != nil || id == 0 {
				return nil, err
			}
			continue
		}
		return order, nil
	}
}

func procesarBootstrapRuntimeOrdersActivosBatch() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	orders, err := seleccionarRuntimeOrdersBatch(SeleccionRuntimeOrdersBatch{
		Estado:       "pendiente",
		Tipos:        []string{"handoff", "resume"},
		Limit:        limit,
		Now:          time.Now().UTC(),
		SortLess:     runtimeOrderByIDLess,
		RequireReady: true,
	})
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, order := range orders {
		handle, err := runtimeHandleActivoParaBootstrapOrder(order)
		if err != nil {
			return processed, err
		}
		if abierta, err := existeRuntimeOrderAbiertaAgenteProyecto(order.Agente, order.ProyectoID, order.ID, "stop", "start", "restart"); err != nil {
			return processed, err
		} else if abierta {
			continue
		}
		motivo := fmt.Sprintf("runtime_bootstrap_coordinated_restart:%s:%d", strings.TrimSpace(order.Tipo), order.ID)
		if handle == nil {
			startOrderID, err := encolarStartBootstrapAgenteProyecto(order.Agente, order.ProyectoID, "", motivo, "orquesta")
			if err != nil {
				return processed, err
			}
			Audit("orquesta", "runtime_bootstrap_coordinated_restart", "runtime_order", order.ID,
				fmt.Sprintf("agente=%s tipo=%s stop_order_id=%d start_order_id=%d", strings.TrimSpace(order.Agente), strings.TrimSpace(order.Tipo), 0, startOrderID))
			processed++
			continue
		}
		stopOrderID, startOrderID, err := EncolarReinicioCoordinadoRuntimeHandle(handle, order.ProyectoID, motivo, "orquesta")
		if err != nil {
			return processed, err
		}
		Audit("orquesta", "runtime_bootstrap_coordinated_restart", "runtime_order", order.ID,
			fmt.Sprintf("agente=%s tipo=%s stop_order_id=%d start_order_id=%d", strings.TrimSpace(order.Agente), strings.TrimSpace(order.Tipo), stopOrderID, startOrderID))
		processed++
	}
	return processed, nil
}

type SeleccionRuntimeOrdersBatch struct {
	Estado       string
	Tipos        []string
	Limit        int
	Now          time.Time
	SortLess     func(left, right *RuntimeOrder) bool
	RequireReady bool
}

func seleccionarRuntimeOrdersBatch(input SeleccionRuntimeOrdersBatch) ([]*RuntimeOrder, error) {
	estado := strings.TrimSpace(input.Estado)
	if estado == "" {
		return nil, nil
	}
	filter := FiltroRuntimeOrders{Estado: &estado}
	if len(input.Tipos) > 0 {
		filter.Tipos = append([]string(nil), input.Tipos...)
	}
	if !input.RequireReady && input.Limit > 0 {
		filter.Limit = input.Limit
	}
	orders, err := ListarRuntimeOrdersVivas(filter)
	if err != nil {
		return nil, err
	}
	tiposWanted := make(map[string]struct{}, len(input.Tipos))
	for _, tipo := range input.Tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo != "" {
			tiposWanted[tipo] = struct{}{}
		}
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	selected := make([]*RuntimeOrder, 0, len(orders))
	for _, order := range orders {
		if order == nil {
			continue
		}
		if input.RequireReady && !runtimeOrderPendingReady(order, now) {
			continue
		}
		if len(tiposWanted) > 0 {
			if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; !ok {
				continue
			}
		}
		selected = append(selected, order)
	}
	sort.Slice(selected, func(i, j int) bool {
		if input.SortLess != nil {
			return input.SortLess(selected[i], selected[j])
		}
		return runtimeOrderByIDLess(selected[i], selected[j])
	})
	if input.Limit > 0 && len(selected) > input.Limit {
		selected = selected[:input.Limit]
	}
	return selected, nil
}

func runtimeOrderByIDLess(left, right *RuntimeOrder) bool {
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	return left.ID < right.ID
}

func runtimeHandleActivoParaBootstrapOrder(order *RuntimeOrder) (*RuntimeHandle, error) {
	if order == nil {
		return nil, nil
	}
	return runtimeHandleCanonicoRecienteConFallback(strings.TrimSpace(order.Agente), order.ProyectoID)
}

func existeRuntimeOrderAbiertaAgenteProyecto(agente string, proyectoID *int64, excludeID int64, tipos ...string) (bool, error) {
	if strings.TrimSpace(agente) == "" || len(tipos) == 0 {
		return false, nil
	}
	return runtimeOrdersHotIndexExists(FiltroRuntimeOrders{
		Agente:     stringsPtrTrimmed(agente),
		ProyectoID: proyectoID,
		Tipos:      append([]string(nil), tipos...),
	}, excludeID)
}

func EncolarReinicioCoordinadoRuntimeHandle(handle *RuntimeHandle, proyectoID *int64, motivo, actor string) (int64, int64, error) {
	if handle == nil || strings.TrimSpace(handle.Agente) == "" {
		return 0, 0, nil
	}
	if strings.TrimSpace(actor) == "" {
		actor = "orquesta"
	}

	projectRef := proyectoID
	if projectRef == nil && handle.ProyectoID != nil && *handle.ProyectoID > 0 {
		projectRef = handle.ProyectoID
	}
	if projectRef == nil && handle.SesionID != nil {
		sesion, err := sesionIfExists(handle.SesionID)
		if err != nil {
			return 0, 0, err
		}
		if sesion != nil && sesion.ProyectoID != nil && *sesion.ProyectoID > 0 {
			projectRef = sesion.ProyectoID
		}
	}

	stopPayload, err := json.Marshal(map[string]any{
		"accion": "stop",
		"motivo": strings.TrimSpace(motivo),
		"por":    strings.TrimSpace(actor),
	})
	if err != nil {
		return 0, 0, err
	}
	stopOrder := &RuntimeOrder{
		Agente:      strings.TrimSpace(handle.Agente),
		ProyectoID:  projectRef,
		Tipo:        "stop",
		PayloadJSON: string(stopPayload),
		HandleID:    &handle.ID,
	}
	if runtime, err := runtimeHandleRuntime(handle); err != nil {
		return 0, 0, err
	} else if runtime != nil && runtime.ID > 0 {
		stopOrder.RuntimeID = &runtime.ID
	} else if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		stopOrder.RuntimeID = handle.RuntimeID
	}
	stopOrderID, err := EncolarRuntimeOrder(stopOrder)
	if err != nil {
		return 0, 0, err
	}

	startPayload, err := json.Marshal(map[string]any{
		"accion": "start",
		"motivo": strings.TrimSpace(motivo),
		"por":    strings.TrimSpace(actor),
	})
	if err != nil {
		return stopOrderID, 0, err
	}
	startOrderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      strings.TrimSpace(handle.Agente),
		ProyectoID:  projectRef,
		Tipo:        "start",
		PayloadJSON: string(startPayload),
	})
	if err != nil {
		return stopOrderID, 0, err
	}
	return stopOrderID, startOrderID, nil
}

func PeekNextBootstrapRuntimeOrder(agente string, proyectoID *int64) (*RuntimeOrder, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	id, err := nextBootstrapRuntimeOrderID(agente, proyectoID)
	if err != nil || id == 0 {
		return nil, err
	}
	return getRuntimeOrderPreferHotIndex(id)
}

func nextBootstrapRuntimeOrderID(agente string, proyectoID *int64) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente: &agente,
		Estado: &estado,
	})
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	tipos := map[string]struct{}{}
	for _, tipo := range runtimeOrderTiposBootstrap() {
		tipos[strings.TrimSpace(tipo)] = struct{}{}
	}
	candidates := make([]*RuntimeOrder, 0, len(orders))
	for _, order := range orders {
		if order == nil || !runtimeOrderPendingReady(order, now) {
			continue
		}
		if _, ok := tipos[strings.TrimSpace(order.Tipo)]; !ok {
			continue
		}
		if proyectoID != nil && order.ProyectoID != nil && *order.ProyectoID != *proyectoID {
			continue
		}
		candidates = append(candidates, order)
	}
	sort.Slice(candidates, func(i, j int) bool {
		leftProject, leftType := runtimeOrderBootstrapPriority(candidates[i], proyectoID)
		rightProject, rightType := runtimeOrderBootstrapPriority(candidates[j], proyectoID)
		if leftProject != rightProject {
			return leftProject < rightProject
		}
		if leftType != rightType {
			return leftType < rightType
		}
		return candidates[i].ID < candidates[j].ID
	})
	if len(candidates) == 0 {
		return 0, nil
	}
	return candidates[0].ID, nil
}

func GetRuntimeOrder(id int64) (*RuntimeOrder, error) {
	return consultarConReintentos(func() (*RuntimeOrder, error) {
		row := DB.QueryRow(runtimeOrderSelectBase()+` WHERE id = ?`, id)
		order, err := scanRuntimeOrder(row)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return order, err
	})
}

func getRuntimeOrderPreferHotIndex(id int64) (*RuntimeOrder, error) {
	if current, ok, err := runtimeOrdersHotIndexGetByID(id); err != nil {
		return nil, err
	} else if ok {
		return current, nil
	}
	return GetRuntimeOrder(id)
}

func ListarRuntimeOrders(filter FiltroRuntimeOrders) ([]*RuntimeOrder, error) {
	if filter.Agente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agente)
		if err != nil {
			return nil, err
		}
		filter.Agente = &agenteCanonico
	}
	if filter.Estado != nil && runtimeOrderEstadoVivo(*filter.Estado) {
		return ListarRuntimeOrdersVivas(filter)
	}
	return consultarConReintentos(func() ([]*RuntimeOrder, error) {
		q := runtimeOrderSelectBase() + ` WHERE 1=1`
		var args []any
		if filter.Agente != nil {
			q += ` AND agente = ?`
			args = append(args, strings.TrimSpace(*filter.Agente))
		}
		if filter.ProyectoID != nil {
			q += ` AND proyecto_id = ?`
			args = append(args, *filter.ProyectoID)
		}
		if filter.Estado != nil {
			q += ` AND estado = ?`
			args = append(args, strings.TrimSpace(*filter.Estado))
		}
		if len(filter.Tipos) > 0 {
			tipos := make([]string, 0, len(filter.Tipos))
			for _, tipo := range filter.Tipos {
				tipo = strings.TrimSpace(tipo)
				if tipo != "" {
					tipos = append(tipos, tipo)
				}
			}
			if len(tipos) > 0 {
				q += ` AND tipo IN (` + strings.TrimRight(strings.Repeat("?,", len(tipos)), ",") + `)`
				for _, tipo := range tipos {
					args = append(args, tipo)
				}
			}
		}
		q += ` ORDER BY id DESC`
		if filter.Limit > 0 {
			q += ` LIMIT ?`
			args = append(args, filter.Limit)
		}
		rows, err := DB.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []*RuntimeOrder
		for rows.Next() {
			order, err := scanRuntimeOrder(rows)
			if err != nil {
				return nil, err
			}
			out = append(out, order)
		}
		return out, rows.Err()
	})
}

func MarcarRuntimeOrderEstado(id int64, estado, resultadoJSON, errorText string) error {
	if strings.TrimSpace(resultadoJSON) == "" {
		resultadoJSON = "{}"
	}
	sync := func(err error) error {
		if err != nil {
			return err
		}
		return runtimeOrdersHotIndexSyncByID(id)
	}
	switch strings.TrimSpace(estado) {
	case "ejecutando":
		leaseUntil := runtimeOrderLeaseDeadline(time.Now().UTC())
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado='ejecutando',
			    resultado_json = ?,
			    error_text = ?,
			    lease_expires_at = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, resultadoJSON, errorText, leaseUntil, id)
		return sync(err)
	case "completada", "fallida", "expirada", "cancelada":
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = ?,
			    resultado_json = ?,
			    error_text = ?,
			    finished_at = CURRENT_TIMESTAMP,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL
			WHERE id = ?`, estado, resultadoJSON, errorText, id)
		return sync(err)
	case "pendiente":
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'pendiente',
			    resultado_json = ?,
			    error_text = ?,
			    started_at = NULL,
			    finished_at = NULL,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, resultadoJSON, errorText, id)
		return sync(err)
	default:
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, estado, id)
		return sync(err)
	}
}

func MarcarRuntimeOrderEjecutando(order *RuntimeOrder) error {
	if order == nil {
		return sql.ErrNoRows
	}
	resultadoJSON := runtimeOrderResultJSONForExecuting(order)
	leaseUntil := runtimeOrderLeaseDeadlineForType(order.Tipo, time.Now().UTC())
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='ejecutando',
		    resultado_json = ?,
		    error_text = '',
		    lease_expires_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, resultadoJSON, leaseUntil, order.ID)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}

func runtimeOrderResultJSONForExecuting(order *RuntimeOrder) string {
	if order == nil {
		return "{}"
	}
	resultado := mapFromJSON(order.ResultadoJSON)
	if len(resultado) == 0 {
		resultado = map[string]any{}
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "start", "resume", "restart":
		delete(resultado, "deferred")
		delete(resultado, "deferred_reason")
		delete(resultado, "retry_after")
		resultado["in_progress"] = true
		resultado["phase"] = "launching"
	}
	data, err := json.Marshal(resultado)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func claimRuntimeOrderByID(id int64) (*RuntimeOrder, error) {
	now := time.Now().UTC()
	current, err := getRuntimeOrderPreferHotIndex(id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}
	leaseUntil := runtimeOrderLeaseDeadlineForType(current.Tipo, now)
	claimedBy := runtimeOrderClaimedBy()
	leaseToken := runtimeOrderLeaseToken(id, now)
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(`
		UPDATE runtime_orders
		SET estado = 'tomada',
		    started_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP,
		    claimed_by = ?,
		    lease_token = ?,
		    lease_expires_at = ?,
		    attempt_count = attempt_count + 1
		WHERE id = ?
		  AND estado = 'pendiente'
		  AND (available_at IS NULL OR available_at <= CURRENT_TIMESTAMP)`,
		claimedBy, leaseToken, leaseUntil, id)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		_ = tx.Rollback()
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := runtimeOrdersHotIndexSyncByID(id); err != nil {
		return nil, err
	}
	if claimed, ok, err := runtimeOrdersHotIndexGetByID(id); err == nil && ok {
		return claimed, nil
	} else if err != nil {
		return nil, err
	}
	return GetRuntimeOrder(id)
}

func resolverHandleParaOrden(order *RuntimeOrder) (*RuntimeHandle, error) {
	if order == nil {
		return nil, nil
	}
	var (
		handle *RuntimeHandle
		err    error
	)
	if order.HandleID != nil {
		handle, err = GetRuntimeHandle(*order.HandleID)
		if err != nil {
			return nil, err
		}
	}
	_, handle, err = resolverDestinoRuntimeOrderCanonico(order, nil, handle)
	if err != nil || handle != nil {
		return handle, err
	}
	return runtimeHandleCanonicoRecienteConFallback(order.Agente, order.ProyectoID)
}

func resolverSesionParaOrden(order *RuntimeOrder) (*Sesion, error) {
	if order == nil {
		return nil, nil
	}
	if order.HandleID != nil {
		handle, err := resolverHandleParaOrden(order)
		if err != nil {
			return nil, err
		}
		if handle != nil && handle.SesionID != nil {
			sesion, err := sesionIfExists(handle.SesionID)
			if err != nil {
				return nil, err
			}
			if sesion != nil {
				return sesion, nil
			}
		}
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return nil, err
	}
	if runtime != nil && runtime.SesionID != nil {
		sesion, err := sesionIfExists(runtime.SesionID)
		if err != nil {
			return nil, err
		}
		if sesion != nil {
			return sesion, nil
		}
	}
	sesion, err := GetSesionActiva(order.Agente, order.ProyectoID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if sesion != nil {
		return sesion, nil
	}
	sesion, err = ObtenerUltimaSesion(order.Agente, order.ProyectoID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sesion, err
}

func resolverRuntimeParaOrden(order *RuntimeOrder) (*RuntimeInstance, error) {
	if order == nil {
		return nil, nil
	}
	var (
		handle  *RuntimeHandle
		runtime *RuntimeInstance
		err     error
	)
	if order.HandleID != nil {
		handle, err = GetRuntimeHandle(*order.HandleID)
		if err != nil {
			return nil, err
		}
	}
	runtime, err = runtimeInstanceIfExists(order.RuntimeID)
	if err != nil {
		return nil, err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderCanonico(order, runtime, handle)
	if err != nil {
		return nil, err
	}
	if runtime != nil {
		return runtime, nil
	}
	if handle != nil && handle.RuntimeID != nil {
		return runtimeInstanceIfExists(handle.RuntimeID)
	}
	return runtimePrincipalAgenteProyecto(order.Agente, order.ProyectoID)
}

func cancelarPendientesRuntimeAlDetener(order *RuntimeOrder) error {
	if order == nil || strings.TrimSpace(order.Agente) == "" {
		return nil
	}
	argsMailbox := []any{strings.TrimSpace(order.Agente)}
	sqlMailbox := `
		UPDATE runtime_mailbox
		SET estado='cancelado',
		    delivered_at=COALESCE(delivered_at, CURRENT_TIMESTAMP),
		    consumed_at=COALESCE(consumed_at, CURRENT_TIMESTAMP)
		WHERE to_agente=? AND estado IN ('pendiente','entregado')`
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		sqlMailbox += ` AND proyecto_id=?`
		argsMailbox = append(argsMailbox, *order.ProyectoID)
	}
	if _, err := DB.Exec(sqlMailbox, argsMailbox...); err != nil {
		return err
	}

	argsOrders := []any{strings.TrimSpace(order.Agente), order.ID}
	sqlOrders := `
		UPDATE runtime_orders
		SET estado='cancelada',
		    error_text=CASE
		        WHEN trim(COALESCE(error_text,''))='' THEN 'runtime_stop_cancelled_pending_work'
		        ELSE error_text
		    END,
		    finished_at=CURRENT_TIMESTAMP,
		    claimed_by='',
		    lease_token='',
		    lease_expires_at=NULL
		WHERE agente=?
		  AND id<>?
		  AND estado IN ('pendiente','tomada','ejecutando')
		  AND tipo NOT IN ('pause','stop')`
	if runtimeStopPreservaStartCoordinado(order) {
		sqlOrders += ` AND tipo <> 'start'`
	}
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		sqlOrders += ` AND proyecto_id=?`
		argsOrders = append(argsOrders, *order.ProyectoID)
	}
	if _, err := DB.Exec(sqlOrders, argsOrders...); err != nil {
		return err
	}
	return nil
}

func runtimeStopPreservaStartCoordinado(order *RuntimeOrder) bool {
	if order == nil || !strings.EqualFold(strings.TrimSpace(order.Tipo), "stop") {
		return false
	}
	payload := mapFromJSON(order.PayloadJSON)
	motivo := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "motivo", "")))
	switch motivo {
	case "esperar_recuperacion_runtime", "worker_atascado", "migrar a tmux":
		return true
	default:
		return strings.HasPrefix(motivo, "runtime_bootstrap_coordinated_restart:")
	}
}

func ejecutarRuntimeOrderRestart(order *RuntimeOrder) error {
	runtime, err := actualizarEstadoRuntime(order, "activo", "corriendo")
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('pausado','cerrado')`, order.Agente); err != nil {
		return err
	}
	runtimeHandleHotReset()
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
		"restarted":     true,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func actualizarEstadoRuntime(order *RuntimeOrder, logicalState, processState string) (*RuntimeInstance, error) {
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return nil, err
	}
	if runtime == nil {
		return nil, fmt.Errorf("no existe runtime activo para %s", order.Agente)
	}
	q := `UPDATE runtime_instances SET logical_state = ?, last_event_at = CURRENT_TIMESTAMP WHERE id = ?`
	args := []any{logicalState, runtime.ID}
	if processState != "" {
		q = `UPDATE runtime_instances SET logical_state = ?, process_state = ?, last_event_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []any{logicalState, processState, runtime.ID}
	}
	if _, err := DB.Exec(q, args...); err != nil {
		return nil, err
	}
	runtime.LogicalState = logicalState
	if processState != "" {
		runtime.ProcessState = processState
	}
	return runtime, nil
}

func controlarProcesoRuntime(order *RuntimeOrder, signaler func(controlruntime.ObjetivoProceso) (bool, int, error)) (bool, int, error) {
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return false, 0, err
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, 0, err
	}
	obj := objetivoProcesoDesdeHandleRuntimeOrden(order, handle, runtime)
	return signaler(obj)
}

func runtimeProcesoLocalYaNoVive(order *RuntimeOrder) (bool, int, error) {
	if order == nil {
		return false, 0, nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, 0, err
	}
	if known, yaNoVive, pid, err := runtimeProcesoLocalStructuredWorkerState(order); known || err != nil {
		return yaNoVive, pid, err
	}
	if handle != nil && RuntimeHandlePauseRequiresFreshStart(handle) {
		if runtimepolicy.RuntimeHandleTMUXSessionMissing(handle.MetadataJSON) {
			return true, 0, nil
		}
	}
	obj, err := objetivoProcesoDiagnosticoParaOrden(order)
	if err != nil {
		return false, 0, err
	}
	pid, ok, err := controlruntime.ResolverPID(obj)
	if err != nil || !ok {
		if err == nil && handle != nil && RuntimeHandlePauseRequiresFreshStart(handle) {
			return true, pid, nil
		}
		return false, pid, err
	}
	vivo, _, err := controlruntime.ProcesoVivo(obj)
	if err != nil {
		return false, pid, err
	}
	return !vivo, pid, nil
}

func objetivoProcesoDiagnosticoParaOrden(order *RuntimeOrder) (controlruntime.ObjetivoProceso, error) {
	obj, err := objetivoProcesoParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	var runtime *RuntimeInstance
	var handle *RuntimeHandle
	if order != nil {
		handle, err = resolverHandleParaOrden(order)
		if err != nil {
			return controlruntime.ObjetivoProceso{}, err
		}
	}
	runtime, err = resolverRuntimeParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	if runtime != nil && obj.PID == nil && runtime.PID != nil && !runtimeOrderBloqueaFallbackPID(order, runtime, handle) {
		obj.PID = runtime.PID
	}
	return obj, nil
}

func runtimeProcesoLocalStructuredWorkerState(order *RuntimeOrder) (bool, bool, int, error) {
	if order == nil {
		return false, false, 0, nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, false, 0, err
	}
	if handle == nil {
		return false, false, 0, nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false, false, 0, nil
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil {
		return false, false, 0, nil
	}
	if strings.EqualFold(strings.TrimSpace(view.Driver), "process_pty_cli") {
		return false, false, 0, nil
	}
	pid := view.ChildPID
	state := strings.ToLower(strings.TrimSpace(view.State))
	switch state {
	case "failed", "stopped", "exited", "closed", "stale":
		return true, true, pid, nil
	}
	if view.HeartbeatStale {
		return false, false, pid, nil
	}
	if !view.Alive {
		return true, true, pid, nil
	}
	return true, false, pid, nil
}

func cerrarSesionFantasmaParaStop(order *RuntimeOrder) (bool, error) {
	if order == nil {
		return false, nil
	}
	sesion, err := GetSesionAbierta(strings.TrimSpace(order.Agente), order.ProyectoID)
	if err == sql.ErrNoRows || sesion == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if SesionEsOperativa(sesion) {
		return false, nil
	}
	if err := cerrarSesionFantasmaActiva(sesion.ID); err != nil {
		return false, err
	}
	return true, nil
}

func ejecutarRuntimeOrderPause(order *RuntimeOrder) error {
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	signaler := controlruntime.PausarProceso
	pausedViaStop := false
	handleEstadoObjetivo := "pausado"
	if RuntimeHandlePauseRequiresFreshStart(handle) {
		signaler = controlruntime.DetenerProceso
		pausedViaStop = true
		handleEstadoObjetivo = "cerrado"
	}
	aplicado, pid, err := controlarProcesoRuntime(order, signaler)
	yaDetenido := false
	if err != nil || !aplicado {
		controlErr := err
		if pausedViaStop {
			yaDetenido, pid, err = runtimeProcesoLocalYaNoVive(order)
			if err != nil {
				return err
			}
			if yaDetenido {
				aplicado = false
			} else if controlErr != nil {
				return controlErr
			} else {
				return fmt.Errorf("pause sin control real para %s", order.Agente)
			}
		} else if controlErr != nil {
			return controlErr
		} else {
			return fmt.Errorf("pause sin control real para %s", order.Agente)
		}
	}
	if pausedViaStop && aplicado {
		if detenido, pidFinal, err := runtimeProcesoLocalYaNoVive(order); err != nil {
			return err
		} else if detenido {
			yaDetenido = true
			if pid == 0 {
				pid = pidFinal
			}
		}
	}
	runtime, err := actualizarEstadoRuntime(order, "pausado", "")
	if err != nil {
		return err
	}
	if handle != nil && handle.ID > 0 {
		if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, handleEstadoObjetivo, handle.ID); err != nil {
			return err
		}
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado='activo'`, handleEstadoObjetivo, order.Agente); err != nil {
		return err
	}
	runtimeHandleHotReset()
	checkpointID, err := asegurarCheckpointCambioContexto(order, "pause", "Checkpoint automático antes de pausa")
	if err != nil {
		return err
	}
	if err := actualizarEstadoSesionParaOrden(order, "pausada"); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"control_real":  aplicado,
		"pid":           pid,
	}
	if pausedViaStop {
		result["paused_via_stop"] = true
	}
	if yaDetenido {
		result["already_stopped"] = true
	}
	if checkpointID > 0 {
		result["checkpoint_id"] = checkpointID
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderResume(order *RuntimeOrder) error {
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	aplicado, pid, err := controlarProcesoRuntime(order, controlruntime.ContinuarProceso)
	if err != nil {
		return err
	}
	if !aplicado {
		return fmt.Errorf("resume sin control real para %s", order.Agente)
	}
	runtime, err := actualizarEstadoRuntime(order, "activo", "corriendo")
	if err != nil {
		return err
	}
	if handle != nil && handle.ID > 0 {
		if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, handle.ID); err != nil {
			return err
		}
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('pausado','fallido')`, order.Agente); err != nil {
		return err
	}
	runtimeHandleHotReset()
	if err := actualizarEstadoSesionParaOrden(order, "activa"); err != nil {
		return err
	}
	if err := reconciliarGobernanzaSesionParaOrden(order); err != nil {
		return err
	}
	if order.ProyectoID != nil {
		EmitirHookCicloVida(order.Agente, HookSessionResume, *order.ProyectoID, "runtime_order", order.ID, order.Agente, "resume")
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
		"control_real":  aplicado,
		"pid":           pid,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func reconciliarGobernanzaSesionParaOrden(order *RuntimeOrder) error {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil || sesion == nil {
		return err
	}
	agente, err := GetAgente(strings.TrimSpace(sesion.Agente))
	if err != nil || agente == nil {
		return err
	}
	proyectoID := sesion.ProyectoID
	if proyectoID == nil {
		proyectoID = order.ProyectoID
	}
	contexto, resumen := BuildGovernanceContextSummaryForContext(strings.TrimSpace(agente.Rol), proyectoID, strings.TrimSpace(sesion.Agente))
	if len(contexto) == 0 {
		return nil
	}
	resumePayload := AppendGovernanceCatalogPayload(strings.TrimSpace(sesion.ResumePayloadJSON), contexto)
	resumenContinuidad := strings.TrimSpace(sesion.ResumenContinuidad)
	if resumen != "" && !strings.Contains(resumenContinuidad, resumen) {
		if resumenContinuidad != "" {
			resumenContinuidad += "\n"
		}
		resumenContinuidad += resumen
	}
	if _, err := DB.Exec(`
		UPDATE sesiones
		SET resume_payload_json = ?, resumen_continuidad = ?
		WHERE id = ?`,
		resumePayload,
		resumenContinuidad,
		sesion.ID,
	); err != nil {
		return err
	}
	_ = enviarRefreshGobernanzaAgente("server", strings.TrimSpace(agente.Rol), strings.TrimSpace(sesion.Agente), proyectoID, "resume_reconcile", map[string]any{
		"runtime_order_id": order.ID,
	})
	return nil
}

func ejecutarRuntimeOrderStop(order *RuntimeOrder) error {
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	preserveExternalSession := runtimeHandlePreservesExternalSession(handle)
	signaler := controlruntime.DetenerProceso
	if preserveExternalSession {
		signaler = controlruntime.PausarProceso
	}
	aplicado, pid, err := controlarProcesoRuntime(order, signaler)
	yaDetenido := false
	sesionFantasmaCerrada := false
	if err != nil || !aplicado {
		controlErr := err
		yaDetenido, pid, err = runtimeProcesoLocalYaNoVive(order)
		if err != nil {
			return err
		}
		if !yaDetenido {
			if cerrada, closeErr := cerrarSesionFantasmaParaStop(order); closeErr != nil {
				return closeErr
			} else if cerrada {
				sesionFantasmaCerrada = true
				yaDetenido = true
			}
		}
		if !yaDetenido {
			if preserveExternalSession {
				aplicado = false
			} else {
				if controlErr != nil {
					return controlErr
				}
				return fmt.Errorf("stop sin control real para %s", order.Agente)
			}
		}
		aplicado = false
	}
	if sesionFantasmaCerrada {
		if err := MarcarRuntimeHandlesCerrados(order.Agente, order.ProyectoID); err != nil {
			return err
		}
		if err := cancelarPendientesRuntimeAlDetener(order); err != nil {
			return err
		}
		result := map[string]any{
			"ok":                     true,
			"control_real":           false,
			"already_stopped":        true,
			"phantom_session_closed": true,
			"pid":                    pid,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}
	checkpointID := int64(0)
	if aplicado {
		checkpointID, err = asegurarCheckpointCambioContexto(order, "stop", "Checkpoint automático antes de detener")
		if err != nil {
			return err
		}
	}
	logicalState := "cerrado"
	processState := "finalizado"
	handleState := "cerrado"
	if preserveExternalSession {
		logicalState = "pausado"
		processState = ""
		handleState = "pausado"
	}
	runtime, err := actualizarEstadoRuntime(order, logicalState, processState)
	if err != nil {
		return err
	}
	if handle != nil && handle.ID > 0 {
		if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, handleState, handle.ID); err != nil {
			return err
		}
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado','fallido')`, handleState, order.Agente); err != nil {
		return err
	}
	runtimeHandleHotReset()
	sesion, err := GetSesionAbierta(order.Agente, order.ProyectoID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if sesion != nil {
		if preserveExternalSession {
			if err := aparcarSesionActiva(order.Agente, order.ProyectoID); err != nil {
				return err
			}
		} else {
			if err := FinSesion(order.Agente); err != nil && !strings.Contains(err.Error(), "no tenía sesión activa") {
				return err
			}
		}
	}
	if err := cancelarPendientesRuntimeAlDetener(order); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
		"control_real":  aplicado,
		"pid":           pid,
	}
	if yaDetenido {
		result["already_stopped"] = true
	}
	if checkpointID > 0 {
		result["checkpoint_id"] = checkpointID
	}
	if preserveExternalSession {
		result["preserved_external_session"] = true
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func CrearRuntimeCheckpoint(cp *RuntimeCheckpoint) (int64, error) {
	if cp == nil || strings.TrimSpace(cp.Agente) == "" {
		return 0, sql.ErrNoRows
	}
	cp.CWD = rutaRuntimeCanonicaProyecto(cp.Agente, cp.ProyectoID, cp.CWD)
	if strings.TrimSpace(cp.PayloadJSON) == "" {
		cp.PayloadJSON = "{}"
	}
	if strings.TrimSpace(cp.CheckpointKind) == "" {
		cp.CheckpointKind = "manual"
	}
	return insertReturningID(`
		INSERT INTO runtime_checkpoints (
			agente, proyecto_id, sesion_id, runtime_id, checkpoint_kind, resumen,
			branch, cwd, payload_json, resume_strategy, source
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cp.Agente, cp.ProyectoID, cp.SesionID, cp.RuntimeID, cp.CheckpointKind, cp.Resumen,
		cp.Branch, cp.CWD, cp.PayloadJSON, cp.ResumeStrategy, cp.Source,
	)
}

func UltimoRuntimeCheckpoint(agente string, proyectoID *int64) (*RuntimeCheckpoint, error) {
	q := runtimeCheckpointSelectBase() + ` WHERE agente = ?`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY id DESC LIMIT 1`
	row := DB.QueryRow(q, args...)
	cp, err := scanRuntimeCheckpoint(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if cp != nil {
		cp.CWD = rutaRuntimeCanonicaProyecto(cp.Agente, cp.ProyectoID, cp.CWD)
	}
	return cp, err
}

func GetRuntimeCheckpoint(id int64) (*RuntimeCheckpoint, error) {
	row := DB.QueryRow(runtimeCheckpointSelectBase()+` WHERE id = ?`, id)
	cp, err := scanRuntimeCheckpoint(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if cp != nil {
		cp.CWD = rutaRuntimeCanonicaProyecto(cp.Agente, cp.ProyectoID, cp.CWD)
	}
	return cp, err
}

func GetRuntimeCheckpointBySource(source string) (*RuntimeCheckpoint, error) {
	row := DB.QueryRow(runtimeCheckpointSelectBase()+` WHERE source = ? ORDER BY id DESC LIMIT 1`, strings.TrimSpace(source))
	cp, err := scanRuntimeCheckpoint(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if cp != nil {
		cp.CWD = rutaRuntimeCanonicaProyecto(cp.Agente, cp.ProyectoID, cp.CWD)
	}
	return cp, err
}

func ListarRuntimeCheckpoints(filter FiltroRuntimeCheckpoints) ([]*RuntimeCheckpoint, error) {
	q := runtimeCheckpointSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.Agente != nil {
		q += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*filter.Agente))
	}
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.CheckpointKind != nil {
		q += ` AND checkpoint_kind = ?`
		args = append(args, strings.TrimSpace(*filter.CheckpointKind))
	}
	if filter.Source != nil {
		q += ` AND source = ?`
		args = append(args, strings.TrimSpace(*filter.Source))
	}
	q += ` ORDER BY id DESC`
	if filter.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeCheckpoint
	for rows.Next() {
		cp, err := scanRuntimeCheckpoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	normalizarRuntimeCheckpointCWDs(out)
	return out, nil
}

func ResumirRuntimeCheckpointsPorAgente() ([]*RuntimeCheckpointPanelSummary, error) {
	rows, err := DB.Query(`
		SELECT agg.total_count,
		       cp.id, cp.agente, cp.proyecto_id, cp.sesion_id, cp.runtime_id, cp.checkpoint_kind, cp.resumen,
		       cp.branch, cp.cwd, cp.payload_json, cp.resume_strategy, cp.source, cp.created_at
		FROM (
			SELECT agente, COUNT(*) AS total_count, MAX(id) AS last_id
			FROM runtime_checkpoints
			GROUP BY agente
		) agg
		JOIN runtime_checkpoints cp ON cp.id = agg.last_id
		ORDER BY agg.agente`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeCheckpointPanelSummary
	for rows.Next() {
		var total int
		cp, err := scanRuntimeCheckpointWithLeadingTotal(rows, &total)
		if err != nil {
			return nil, err
		}
		out = append(out, &RuntimeCheckpointPanelSummary{
			Agente: cp.Agente,
			Total:  total,
			Last:   cp,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, item := range out {
		if item == nil || item.Last == nil {
			continue
		}
		item.Last.CWD = rutaRuntimeCanonicaProyecto(item.Last.Agente, item.Last.ProyectoID, item.Last.CWD)
	}
	return out, nil
}

func normalizarRuntimeCheckpointCWDs(items []*RuntimeCheckpoint) {
	for _, cp := range items {
		if cp == nil {
			continue
		}
		cp.CWD = rutaRuntimeCanonicaProyecto(cp.Agente, cp.ProyectoID, cp.CWD)
	}
}
