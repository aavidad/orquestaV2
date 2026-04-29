package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"os"
	"strings"
	"time"
)

func boolFromAny(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.ToLower(val) == "true" || val == "1"
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	}
	return false
}

type RuntimeHandle struct {
	ID               int64      `json:"id"`
	Agente           string     `json:"agente"`
	ProyectoID       *int64     `json:"proyecto_id,omitempty"`
	SesionID         *int64     `json:"sesion_id,omitempty"`
	RuntimeID        *int64     `json:"runtime_id,omitempty"`
	Transporte       string     `json:"transporte"`
	HandleKind       string     `json:"handle_kind"`
	HandleRef        string     `json:"handle_ref"`
	Estado           string     `json:"estado"`
	LeaseToken       string     `json:"lease_token"`
	CapabilitiesJSON string     `json:"capabilities_json"`
	MetadataJSON     string     `json:"metadata_json"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type RuntimeOrder struct {
	ID             int64      `json:"id"`
	Agente         string     `json:"agente"`
	ProyectoID     *int64     `json:"proyecto_id,omitempty"`
	RuntimeID      *int64     `json:"runtime_id,omitempty"`
	HandleID       *int64     `json:"handle_id,omitempty"`
	Tipo           string     `json:"tipo"`
	PayloadJSON    string     `json:"payload_json"`
	ResultadoJSON  string     `json:"resultado_json"`
	ErrorText      string     `json:"error_text"`
	Estado         string     `json:"estado"`
	ClaimedBy      string     `json:"claimed_by"`
	LeaseToken     string     `json:"lease_token"`
	AttemptCount   int        `json:"attempt_count"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	AvailableAt    time.Time  `json:"available_at"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RuntimeMailboxMessage struct {
	ID             int64      `json:"id"`
	FromAgente     string     `json:"from_agente"`
	ToAgente       string     `json:"to_agente"`
	ProyectoID     *int64     `json:"proyecto_id,omitempty"`
	RuntimeOrderID *int64     `json:"runtime_order_id,omitempty"`
	Kind           string     `json:"kind"`
	PayloadJSON    string     `json:"payload_json"`
	Estado         string     `json:"estado"`
	CreatedAt      time.Time  `json:"created_at"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ConsumedAt     *time.Time `json:"consumed_at,omitempty"`
}

type RuntimeMailboxPanelSummary struct {
	Agente  string `json:"agente"`
	Total   int    `json:"total"`
	Pending int    `json:"pending"`
}

type RuntimeCheckpoint struct {
	ID             int64     `json:"id"`
	Agente         string    `json:"agente"`
	ProyectoID     *int64    `json:"proyecto_id,omitempty"`
	SesionID       *int64    `json:"sesion_id,omitempty"`
	RuntimeID      *int64    `json:"runtime_id,omitempty"`
	CheckpointKind string    `json:"checkpoint_kind"`
	Resumen        string    `json:"resumen"`
	Branch         string    `json:"branch"`
	CWD            string    `json:"cwd"`
	PayloadJSON    string    `json:"payload_json"`
	ResumeStrategy string    `json:"resume_strategy"`
	Source         string    `json:"source"`
	CreatedAt      time.Time `json:"created_at"`
}

type RuntimeCheckpointPanelSummary struct {
	Agente string             `json:"agente"`
	Total  int                `json:"total"`
	Last   *RuntimeCheckpoint `json:"last,omitempty"`
}

type FiltroPurgadoRuntimeHandles struct {
	Agente     *string
	ProyectoID *int64
	Estados    []string
}

type PurgaRuntimeHandlesResultado struct {
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
}

type FiltroPurgadoRuntimeOrders struct {
	Agente        *string
	ProyectoID    *int64
	Estados       []string
	Tipos         []string
	CreatedBefore *time.Time
}

type PurgaRuntimeOrdersResultado struct {
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
	Tipos      []string `json:"tipos,omitempty"`
}

type PurgaRuntimeHistoricoResultado struct {
	Handles  *PurgaRuntimeHandlesResultado   `json:"handles,omitempty"`
	Orders   *PurgaRuntimeOrdersResultado    `json:"orders,omitempty"`
	Runtimes *PurgaRuntimeInstancesResultado `json:"runtimes,omitempty"`
}

type PurgaRuntimeInstancesResultado struct {
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
}

type bootstrapRuntimeData struct {
	Order      *RuntimeOrder
	Mailbox    []*RuntimeMailboxMessage
	Checkpoint *RuntimeCheckpoint
	Consumidos int
}

type HandoffPayload struct {
	AgenteOrigen       string `json:"agente_origen"`
	AgenteDestino      string `json:"agente_destino"`
	TareaID            *int64 `json:"tarea_id,omitempty"`
	Motivo             string `json:"motivo,omitempty"`
	ResumenContinuidad string `json:"resumen_continuidad,omitempty"`
	ExternalSessionID  string `json:"external_session_id,omitempty"`
}

type FiltroRuntimeOrders struct {
	Agente     *string
	ProyectoID *int64
	Estado     *string
	Tipos      []string
	Limit      int
}

type FiltroRuntimeMailbox struct {
	ToAgente   *string
	FromAgente *string
	ProyectoID *int64
	Estado     *string
	Limit      int
}

type FiltroRuntimeCheckpoints struct {
	Agente         *string
	ProyectoID     *int64
	CheckpointKind *string
	Source         *string
	Limit          int
}

func UpsertRuntimeHandleDesdeSesion(s *Sesion) error {
	if s == nil {
		return nil
	}
	runtime, err := GetRuntimeBySesionID(s.ID)
	if err != nil {
		return err
	}
	h := &RuntimeHandle{
		Agente:           s.Agente,
		ProyectoID:       s.ProyectoID,
		Transporte:       inferirTransporteSesion(s),
		HandleKind:       inferirHandleKindSesion(s),
		HandleRef:        inferirHandleRefSesion(s),
		Estado:           inferirEstadoHandleSesion(s),
		CapabilitiesJSON: inferirCapabilitiesSesion(s),
		MetadataJSON:     inferirMetadataSesion(s),
		LastSeenAt:       inferirLastSeenSesion(s),
	}
	h.SesionID = &s.ID
	if runtime != nil {
		h.RuntimeID = &runtime.ID
	}

	existente, err := GetRuntimeHandleBySesionID(s.ID)
	if err != nil {
		return err
	}
	if existente == nil {
		_, err = DB.Exec(`
			INSERT INTO runtime_handles (
				agente, proyecto_id, sesion_id, runtime_id, transporte, handle_kind, handle_ref,
				estado, capabilities_json, metadata_json, last_seen_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			h.Agente, h.ProyectoID, h.SesionID, h.RuntimeID, h.Transporte, h.HandleKind, h.HandleRef,
			h.Estado, h.CapabilitiesJSON, h.MetadataJSON, h.LastSeenAt,
		)
	} else {
		h.Transporte = reconciliarTransporteSesion(existente, h)
		h.HandleKind = reconciliarHandleKindSesion(existente, h)
		h.HandleRef = reconciliarHandleRefSesion(existente, h)
		h.CapabilitiesJSON = reconciliarCapabilitiesSesion(existente.CapabilitiesJSON, h.CapabilitiesJSON)
		h.MetadataJSON = reconciliarMetadataSesion(existente.MetadataJSON, h.MetadataJSON)
		_, err = DB.Exec(`
			UPDATE runtime_handles
			SET agente = ?,
			    proyecto_id = ?,
			    runtime_id = ?,
			    transporte = ?,
			    handle_kind = ?,
			    handle_ref = ?,
			    estado = ?,
			    capabilities_json = ?,
			    metadata_json = ?,
			    last_seen_at = ?
			WHERE id = ?`,
			h.Agente, h.ProyectoID, h.RuntimeID, h.Transporte, h.HandleKind, h.HandleRef,
			h.Estado, h.CapabilitiesJSON, h.MetadataJSON, h.LastSeenAt, existente.ID,
		)
	}
	if err != nil {
		return err
	}
	runtimeHandleHotReset()
	if h.Estado == "activo" || h.Estado == "pausado" {
		_, err = DB.Exec(`
			UPDATE runtime_handles
			SET estado = 'cerrado',
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE agente = ? AND id <> COALESCE((SELECT id FROM runtime_handles WHERE sesion_id = ?), -1)
			  AND estado IN ('activo','pausado')`,
			s.Agente, s.ID,
		)
	}
	if err == nil {
		runtimeHandleHotReset()
	}
	return err
}

func normalizarHandleRefSesionRuntimeLigado(sesionID int64) error {
	if sesionID <= 0 {
		return nil
	}
	handle, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil || handle == nil {
		return err
	}
	if handle.RuntimeID == nil || *handle.RuntimeID <= 0 {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return nil
	}
	if strings.TrimSpace(handle.HandleRef) != strings.TrimSpace(jsonNumber(sesionID)) {
		return nil
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET handle_ref = ?
		WHERE id = ?`,
		fmt.Sprintf("runtime:%d", *handle.RuntimeID), handle.ID,
	); err != nil {
		return err
	}
	runtimeHandleHotReset()
	return nil
}

func MarcarRuntimeHandlesCerradosPorAgente(agente string) error {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return err
	}
	if strings.TrimSpace(agente) == "" {
		return nil
	}
	_, err = DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE agente = ? AND estado IN ('activo','pausado')
		  AND (sesion_id IS NULL OR sesion_id NOT IN (
		      SELECT id FROM sesiones WHERE agente = ? AND activa = 1
		  ))`, agente, agente)
	if err == nil {
		runtimeHandleHotReset()
	}
	return err
}

func MarcarRuntimeHandlesCerrados(agente string, proyectoID *int64) error {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" && proyectoID == nil {
		return nil
	}
	query := `
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE estado IN ('activo','pausado')`
	args := make([]any, 0, 3)
	if agente != "" {
		query += ` AND agente = ?`
		args = append(args, agente)
	}
	if proyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	_, err = DB.Exec(query, args...)
	if err == nil {
		runtimeHandleHotReset()
	}
	return err
}

func runtimeHandleSelectBase() string {
	return `
		SELECT id, agente, proyecto_id, sesion_id, runtime_id, transporte, handle_kind, handle_ref,
		       estado, lease_token, capabilities_json, metadata_json, last_seen_at, created_at, updated_at
		FROM runtime_handles`
}

func runtimeOrderSelectBase() string {
	return `
		SELECT id, agente, proyecto_id, runtime_id, handle_id, tipo, payload_json, resultado_json,
		       error_text, estado, claimed_by, lease_token, attempt_count, lease_expires_at,
		       available_at, created_at, started_at, finished_at, updated_at
		FROM runtime_orders`
}

func runtimeMailboxSelectBase() string {
	return `
		SELECT id, from_agente, to_agente, proyecto_id, runtime_order_id, kind, payload_json,
		       estado, created_at, delivered_at, consumed_at
		FROM runtime_mailbox`
}

func runtimeCheckpointSelectBase() string {
	return `
		SELECT id, agente, proyecto_id, sesion_id, runtime_id, checkpoint_kind, resumen,
		       branch, cwd, payload_json, resume_strategy, source, created_at
		FROM runtime_checkpoints`
}

func ejecutarRuntimeOrderStart(order *RuntimeOrder) (err error) {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)

	proyectoRef := stringFromMap(payload, "proyecto", "")
	if strings.TrimSpace(proyectoRef) == "" && order.ProyectoID != nil {
		proyectoRef = jsonNumber(*order.ProyectoID)
	}
	if strings.TrimSpace(proyectoRef) == "" {
		return fmt.Errorf("start sin proyecto")
	}
	proyectoTarget, err := GetProyecto(strings.TrimSpace(proyectoRef))
	if err != nil {
		return err
	}
	runtimeActual, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}
	handleActual, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	if satisfied, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtimeActual, handleActual); satisfied {
		if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtimeActual, handleActual); err != nil {
			return err
		}
		resultado := map[string]any{
			"ok":       true,
			"obsoleta": true,
			"reason":   reason,
		}
		if runtimeActual != nil && runtimeActual.ID > 0 {
			resultado["runtime_id"] = runtimeActual.ID
		}
		if handleActual != nil && handleActual.ID > 0 {
			resultado["handle_id"] = handleActual.ID
		}
		data, _ := json.Marshal(resultado)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}
	if sesionActual, err := GetSesionAbierta(strings.TrimSpace(order.Agente), order.ProyectoID); err != nil && err != sql.ErrNoRows {
		return err
	} else if sesionActual != nil && !SesionEsOperativa(sesionActual) {
		if err := cerrarSesionFantasmaActiva(sesionActual.ID); err != nil {
			return err
		}
	}
	if sesionAjena, err := runtimeOrderStartSesionAbiertaAjena(order.Agente, &proyectoTarget.ID); err != nil {
		return err
	} else if sesionAjena != nil {
		proyectoAjenoID := int64(0)
		if sesionAjena.ProyectoID != nil {
			proyectoAjenoID = *sesionAjena.ProyectoID
		}
		reason := fmt.Sprintf("sesion_activa_en_otro_proyecto:%d", proyectoAjenoID)
		if !SesionEsOperativa(sesionAjena) {
			if err := cerrarSesionFantasmaActiva(sesionAjena.ID); err != nil {
				return err
			}
			reason = fmt.Sprintf("sesion_fantasma_en_otro_proyecto:%d", proyectoAjenoID)
		} else if err := asegurarStopSesionActivaAjena(order, sesionAjena); err != nil {
			return err
		}
		return reencolarRuntimeOrderControlAt(order, reason, time.Now().UTC().Add(runtimeOrderControlRetryDelay()))
	}

	skipBootstrap := runtimeOrderStartDebeIgnorarBootstrap(payload)
	agente, proyecto, conector, ultima, resume, bootstrap, plan, err := prepararStartRuntimeOrder(
		order.Agente,
		proyectoRef,
		order.ID,
		stringFromMap(payload, "conector", ""),
		stringFromMap(payload, "modelo", ""),
		stringFromMap(payload, "razonamiento", ""),
		stringFromMap(payload, "perfil", ""),
		int64PtrFromMap(payload, "tarea_id"),
		skipBootstrap,
	)
	if err != nil {
		return err
	}
	bootstrapClaimed := bootstrap != nil && bootstrap.Order != nil && bootstrap.Order.ID > 0
	defer func() {
		if err != nil && bootstrapClaimed {
			_ = revertBootstrapRuntimePreparation(bootstrap)
		}
	}()
	if disponible, op, err := ConectorDisponibleParaArranque(conector.ID); err != nil {
		return err
	} else if !disponible {
		detalle := "circuito abierto del conector"
		if op != nil && op.CooldownUntil != nil {
			detalle = fmt.Sprintf("%s hasta %s", detalle, op.CooldownUntil.UTC().Format(time.RFC3339))
		}
		return fmt.Errorf("%s %s (%s)", detalle, conector.Slug, strings.TrimSpace(op.Motivo))
	}
	arranque, err := controlruntime.ArrancarPlan(controlruntime.SolicitudArranque{
		Agente:       order.Agente,
		Proyecto:     proyecto.Slug,
		Plan:         plan,
		TimeoutReady: timeoutParaArranqueDurable(order),
	})
	if err != nil {
		return err
	}

	var pid64 *int64
	if arranque.PID > 0 {
		pid := int64(arranque.PID)
		pid64 = &pid
	}
	host, _ := os.Hostname()
	externalSessionID := strings.TrimSpace(arranque.ExternalSessionID)
	if externalSessionID == "" {
		externalSessionID = strings.TrimSpace(resume.ExternalSessionID)
	}
	sesion, reusedSession, err := reutilizarSesionArranquePoolLocal(order.Agente, &proyecto.ID, conector, externalSessionID, plan, resume, ultima, strings.TrimSpace(host), pid64)
	if err != nil {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{
			PID:          pid64,
			HandleKind:   strings.TrimSpace(arranque.HandleKind),
			HandleRef:    strings.TrimSpace(arranque.HandleRef),
			MetadataJSON: arranque.MetadataJSON,
		})
		return err
	}
	if sesion == nil {
		sesion, err = IniciarSesionContexto(SesionInicio{
			Agente:             order.Agente,
			ConectorID:         &conector.ID,
			ProyectoID:         &proyecto.ID,
			CWD:                plan.WorkingDir,
			Herramienta:        conector.Slug,
			ExternalSessionID:  externalSessionID,
			Host:               strings.TrimSpace(host),
			PID:                pid64,
			ResumePayloadJSON:  payloadJSONDesdePlanYResume(plan, resume),
			ResumenContinuidad: resumenContinuidadDesdeResume(ultima, resume),
			Branch:             branchDesdeResume(ultima, resume),
		})
	}
	if err != nil {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{
			PID:          pid64,
			HandleKind:   strings.TrimSpace(arranque.HandleKind),
			HandleRef:    strings.TrimSpace(arranque.HandleRef),
			MetadataJSON: arranque.MetadataJSON,
		})
		return err
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		return err
	}
	if handle != nil {
		if err := actualizarHandleRuntimeArranque(handle.ID, arranque, conector, plan, resume); err != nil {
			return err
		}
		handle, err = GetRuntimeHandle(handle.ID)
		if err != nil {
			return err
		}
		candidatosPrevios, err := listarRuntimeHandlesActivosCandidatos(order.Agente, order.ProyectoID)
		if err != nil {
			return err
		}
		for _, candidato := range candidatosPrevios {
			if candidato == nil || candidato.ID == handle.ID {
				continue
			}
			if err := marcarRuntimeHandleSuperseded(candidato); err != nil {
				return err
			}
		}
		runtimeHandleHotReset()
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil {
		return err
	}
	if runtime == nil {
		runtime, err = GetRuntimePrincipalAgenteProyecto(order.Agente, order.ProyectoID)
		if err != nil {
			return err
		}
	}
	if runtime == nil {
		runtime, err = GetRuntimeBySesionID(sesion.ID)
		if err != nil {
			return err
		}
	}
	if runtime == nil {
		return fmt.Errorf("no se pudo registrar runtime para la sesion %d", sesion.ID)
	}
	if reusedSession && strings.TrimSpace(runtime.LogicalState) == "" {
		runtime.LogicalState = "disponible"
	}
	if transporteArranque := strings.TrimSpace(inferirTransporteArranque(arranque)); transporteArranque != "api" && transporteArranque != "mcp_http" && transporteArranque != "otro" {
		if err := controlruntime.ActivarSupervisionOrquestada(controlruntime.ObjetivoProceso{
			PID:          pid64,
			HandleKind:   strings.TrimSpace(arranque.HandleKind),
			HandleRef:    strings.TrimSpace(arranque.HandleRef),
			MetadataJSON: strings.TrimSpace(arranque.MetadataJSON),
		}); err != nil {
			return err
		}
	}
	if handle != nil {
		if texto := construirTranscriptArranque(plan, resume, bootstrap); strings.TrimSpace(texto) != "" {
			if err := RegistrarRuntimeTranscriptSistema(handle, runtime, texto); err != nil {
				return err
			}
		}
		if err := inyectarPromptArranque(handle, runtime, plan, order); err != nil {
			return err
		}
	}
	if err := marcarBootstrapRuntimePreparationEjecutando(order, bootstrap, sesion, handle, runtime); err != nil {
		return err
	}
	if err := ackBootstrapRuntimeSinLeaseEnStart(order, bootstrap, sesion); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"agente":        agente.Nombre,
		"sesion_id":     sesion.ID,
		"handle_id":     runtimeHandleID(handle),
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
		"control_real":  true,
		"bootstrap":     bootstrapResumenJSON(bootstrap),
	}
	if len(bootstrap.Mailbox) > 0 {
		result["lease_state"] = "waiting_for_evidence"
		result["mailbox_count"] = len(bootstrap.Mailbox)
		result["mailbox_ids"] = runtimeMailboxIDs(bootstrap.Mailbox)
		result["ack_mode"] = "runtime_transcript_or_tick"
		result["continuidad"] = true
		result["order_tipo"] = "start"
	}
	if arranque.PID > 0 {
		result["pid"] = arranque.PID
	}
	if externalSessionID != "" {
		result["external_session_id"] = externalSessionID
	}
	if strings.TrimSpace(arranque.HandleKind) != "" {
		result["handle_kind"] = strings.TrimSpace(arranque.HandleKind)
	}
	if strings.TrimSpace(arranque.HandleRef) != "" {
		result["handle_ref"] = strings.TrimSpace(arranque.HandleRef)
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderSendInstruction(order *RuntimeOrder) error {
	payload, err := runtimeOrderPayloadMap(order.PayloadJSON)
	if err != nil {
		return err
	}
	if supersedida, err := reconciliarRuntimeOrderSendInstructionConMailboxActual(order, payload, time.Now().UTC()); err != nil {
		return err
	} else if supersedida {
		return nil
	}
	payloadDesdeMailbox := runtimeOrderSendInstructionProvieneMailbox(payload)

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	if canonico, err := CanonicalizeAgentName(toAgente); err == nil && strings.TrimSpace(canonico) != "" {
		toAgente = strings.TrimSpace(canonico)
		payload["to_agente"] = toAgente
	}
	fromAgente := stringFromMap(payload, "from_agente", "server")
	if strings.TrimSpace(fromAgente) != "" && !strings.EqualFold(strings.TrimSpace(fromAgente), "server") {
		if canonico, err := CanonicalizeAgentName(fromAgente); err == nil && strings.TrimSpace(canonico) != "" {
			fromAgente = strings.TrimSpace(canonico)
			payload["from_agente"] = fromAgente
		}
	}
	if strings.TrimSpace(toAgente) == "" {
		return fmt.Errorf("send_instruction sin agente destino")
	}
	texto := runtimeOrderSendInstructionTexto(payload)
	if payloadDesdeMailbox {
		mailboxID := runtimeOrderSendInstructionMailboxID(payload)
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return err
		}
		if msg != nil {
			if confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, msg); err != nil {
				return err
			} else if confirmed {
				if err := MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
					return err
				}
				if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
					return err
				}
				return completarRuntimeOrderSendInstructionEntregadaPorReceipt(order, payload, receiptSource, receiptAt)
			}
			switch strings.ToLower(strings.TrimSpace(msg.Estado)) {
			case "consumido", "cancelado", "expirado":
				return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "mailbox ya "+strings.TrimSpace(msg.Estado))
			case "entregado":
				handled, err := reconciliarRuntimeOrderSendInstructionMailboxEntregado(order, payload, msg, time.Now().UTC())
				if handled || err != nil {
					return err
				}
			}
		}
	}

	var runtime *RuntimeInstance
	runtime, err = runtimeInstanceIfExists(order.RuntimeID)
	if err != nil {
		return err
	}
	var handle *RuntimeHandle
	if order.HandleID != nil {
		handle, err = GetRuntimeHandle(*order.HandleID)
		if err != nil {
			return err
		}
	}
	originalHandleID := int64(0)
	if handle != nil {
		originalHandleID = handle.ID
	}
	originalRuntimeID := int64(0)
	if runtime != nil {
		originalRuntimeID = runtime.ID
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		return err
	}
	handle, externalSessionID, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	if err != nil {
		return err
	}
	if strings.TrimSpace(externalSessionID) == "" {
		externalSessionID = runtimeOrderSendInstructionExternalSessionID(payload)
		if strings.TrimSpace(externalSessionID) == "" && handle != nil {
			externalSessionID = strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "external_session_id", ""))
		}
	}
	handle, workingDir, err := SincronizarRuntimeHandleWorkingDir(handle, runtime, order.Agente, order.ProyectoID)
	if err != nil {
		return err
	}
	if err := actualizarRuntimeOrderDestino(order.ID, runtime, handle); err != nil {
		return err
	}
	if supersedida, err := completarRuntimeOrderSendInstructionSesionObsoleta(order, payload, handle, externalSessionID); err != nil {
		return err
	} else if supersedida {
		return nil
	}
	if supersedida, err := completarRuntimeOrderSendInstructionCubiertaPorBootstrap(order, payload, handle, runtime); err != nil {
		return err
	} else if supersedida {
		return nil
	}

	obj := objetivoProcesoSendInstruction(order, handle, runtime, workingDir)
	texto = controlruntime.NormalizarInstruccionProceso(obj, texto)
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("send_instruction sin texto aplicable")
	}

	if handle == nil {
		if runtimeOrderSendInstructionDebeRetenerSinRuntimeActivo(order, payload) {
			return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "sin runtime activo entregable")
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "sin runtime activo entregable")
	}

	deliveryMode := RuntimeHandleMailboxDeliveryMode(handle)
	durableDeliveryMode := runtimeHandleMailboxDeliveryModeExplicit(handle)
	if durableDeliveryMode == "" {
		durableDeliveryMode = deliveryMode
	}
	if reason, ok := runtimeOrderSendInstructionDurableOnlyReason(handle, payload, texto, deliveryMode); ok {
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}
	reboundToCanonical := (handle != nil && originalHandleID > 0 && handle.ID != originalHandleID) ||
		(runtime != nil && originalRuntimeID > 0 && runtime.ID != originalRuntimeID)
	attemptSessionResume := runtimeOrderSendInstructionDebeIntentarSessionResume(handle, runtime, payload, texto, externalSessionID, deliveryMode, reboundToCanonical)
	if !payloadDesdeMailbox && !attemptSessionResume {
		switch durableDeliveryMode {
		case runtimeagente.MailboxDeliveryBootstrapOnly:
			if runtimeOrderSendInstructionDebeRetenerOrdenDirecta(handle, runtime, deliveryMode, externalSessionID) {
				return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_bootstrap_only_mailbox_only")
			}
			return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_bootstrap_only_mailbox_only")
		case runtimeagente.MailboxDeliverySessionResume:
			if runtimeOrderSendInstructionDebeRetenerOrdenDirecta(handle, runtime, deliveryMode, externalSessionID) {
				return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_session_resume_mailbox_only")
			}
			return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_session_resume_mailbox_only")
		case runtimeagente.MailboxDeliveryCoordinatedRestart:
			if runtimeOrderSendInstructionDebeRetenerOrdenDirecta(handle, runtime, deliveryMode, externalSessionID) {
				return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_coordinated_restart_mailbox_only")
			}
			return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime_handle_coordinated_restart_mailbox_only")
		}
	}
	if payloadDesdeMailbox && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		reason := "runtime_handle_pausado_mailbox_only"
		if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
			reason += ":" + kind
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}

	aplicado := false
	pid := 0
	deliveryPath := ""
	if RuntimeHandlePermiteSendInputInteractivo(handle) {
		if ready, reason := runtimeHandleListaParaDispatchInteractivo(handle, time.Now().UTC()); !ready {
			if payloadDesdeMailbox {
				return reencolarRuntimeOrderSendInstruction(order, payload, reason)
			}
			return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
		}
		driver, transport := runtimeHandleDriverTransportObserved(handle)
		if strings.EqualFold(driver, "tmux_cli_session") || strings.EqualFold(transport, "tmux") {
			result, tmuxErr := controlruntime.EnviarInstruccionTMUXVerificada(obj, texto)
			if tmuxErr != nil {
				if reason, ok := runtimeOrderSendInstructionDiferirPorErrorTMUX(handle, tmuxErr); ok {
					if payloadDesdeMailbox {
						return reencolarRuntimeOrderSendInstruction(order, payload, reason)
					}
					return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
				}
				if payloadDesdeMailbox {
					return reencolarRuntimeOrderSendInstruction(order, payload, tmuxErr.Error())
				}
				return tmuxErr
			}
			aplicado = result.Delivered
			if aplicado {
				deliveryPath = "interactive_tmux"
			}
		} else {
			aplicado, pid, err = controlruntime.EnviarInstruccionProceso(obj, texto)
			if err != nil {
				if reason, ok := runtimeOrderSendInstructionDiferirPorErrorTMUX(handle, err); ok {
					if payloadDesdeMailbox {
						return reencolarRuntimeOrderSendInstruction(order, payload, reason)
					}
					return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
				}
				if payloadDesdeMailbox {
					return reencolarRuntimeOrderSendInstruction(order, payload, err.Error())
				}
				return err
			}
			if aplicado {
				deliveryPath = "interactive"
			}
		}
	}
	if payloadDesdeMailbox &&
		durableDeliveryMode == runtimeagente.MailboxDeliveryBootstrapOnly &&
		!attemptSessionResume &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) {
		handled, err := intentarEntregarRuntimeOrderSendInstructionBootstrapTMUX(order, payload, handle, runtime, obj)
		if handled || err != nil {
			return err
		}
	}
	if payloadDesdeMailbox &&
		durableDeliveryMode == runtimeagente.MailboxDeliveryBootstrapOnly &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		!attemptSessionResume {
		reason := "runtime_handle_bootstrap_only_mailbox_only"
		if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
			reason += ":" + kind
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}
	if payloadDesdeMailbox &&
		durableDeliveryMode == runtimeagente.MailboxDeliverySessionResume &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		!attemptSessionResume {
		reason := "runtime_handle_session_resume_mailbox_only"
		if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
			reason += ":" + kind
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}
	if !aplicado && attemptSessionResume {
		aplicado, pid, err = controlruntime.EnviarInstruccionSesionResume(obj, externalSessionID, texto)
		if err != nil {
			if handled, pauseErr := gestionarBackoffProveedorRuntimeOrderSendInstruction(order, payload, err); handled {
				return pauseErr
			}
			if runtimeOrderSendInstructionDebeRetenerNotificadaPorSessionResumeTimeout(order, handle, payload, err) {
				if err := reconciliarRuntimeSendInstructionSessionResume(order, runtime, handle); err != nil {
					return err
				}
				if err := RegistrarRuntimeTranscriptInput(handle, runtime, texto); err != nil {
					return err
				}
				if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
					if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
						return err
					}
				}
				return retenerRuntimeOrderSendInstructionNotificada(order, payload, "session_resume dispatch pending receipt", time.Now().UTC())
			}
			if runtimeOrderSendInstructionDebeDegradarseAMailboxPorError(order, payload, err) {
				return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, err.Error())
			}
			if payloadDesdeMailbox {
				return reencolarRuntimeOrderSendInstruction(order, payload, err.Error())
			}
			return err
		}
		if aplicado {
			deliveryPath = "session_resume"
		}
	}
	if aplicado {
		if deliveryPath == "session_resume" {
			if err := reconciliarRuntimeSendInstructionSessionResume(order, runtime, handle); err != nil {
				return err
			}
		}
		if err := RegistrarRuntimeTranscriptInput(handle, runtime, texto); err != nil {
			return err
		}
		if runtimeOrderSendInstructionDebeEsperarReceiptInteractivo(order, handle, payload) {
			mailboxID, err := asegurarRuntimeOrderSendInstructionMailboxPendiente(order, payload)
			if err != nil {
				return err
			}
			payload["mailbox_id"] = mailboxID
			if strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) == "" {
				payload["mailbox_kind"] = "instruction"
			}
			if strings.TrimSpace(stringFromMap(payload, "to_agente", "")) == "" {
				payload["to_agente"] = toAgente
			}
			if strings.TrimSpace(stringFromMap(payload, "from_agente", "")) == "" {
				payload["from_agente"] = fromAgente
			}
			if err := actualizarRuntimeOrderPayloadJSON(order.ID, payload); err != nil {
				return err
			}
			if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
				return err
			}
			return retenerRuntimeOrderSendInstructionNotificada(order, payload, "interactive dispatch pending receipt", time.Now().UTC())
		}
		if err := completarRuntimeMailboxDesdePayload(payload, order.ProyectoID); err != nil {
			return err
		}
		result := map[string]any{
			"ok":                  true,
			"to_agente":           toAgente,
			"from_agente":         fromAgente,
			"kind":                "instruction",
			"control_real":        true,
			"pid":                 pid,
			"mailbox_delivery":    deliveryMode,
			"delivery_path":       deliveryPath,
			"external_session_id": externalSessionID,
			"handle_id":           runtimeHandleID(handle),
		}
		if runtime != nil {
			result["runtime_id"] = runtime.ID
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	if payloadDesdeMailbox {
		if runtimeOrderSendInstructionDebeRetenerSinRuntimeActivo(order, payload) {
			return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
	}

	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return err
	}
	if existente != nil {
		payload["mailbox_id"] = existente.ID
		payload["mailbox_kind"] = "instruction"
		payload["to_agente"] = existente.ToAgente
		payload["from_agente"] = existente.FromAgente
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
	}

	id, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		ProyectoID:     order.ProyectoID,
		RuntimeOrderID: &order.ID,
		Kind:           "instruction",
		PayloadJSON:    order.PayloadJSON,
	})
	if err != nil {
		return err
	}
	payload["mailbox_id"] = id
	payload["mailbox_kind"] = "instruction"
	payload["to_agente"] = toAgente
	payload["from_agente"] = fromAgente
	if payloadDesdeMailbox {
		if runtimeOrderSendInstructionDebeRetenerSinRuntimeActivo(order, payload) {
			return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
		}
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
	}
	return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
}

func prepararStartRuntimeOrder(agenteRef, proyectoRef string, excludeOrderID int64, conectorRef, modelo, razonamiento, perfilTarea string, tareaID *int64, skipBootstrap bool) (*Agente, *Proyecto, *Conector, *Sesion, runtimeagente.ResumeContext, *bootstrapRuntimeData, *runtimeagente.LaunchPlan, error) {
	agente, proyecto, conector, ultima, resume, err := cargarContextoStartRuntime(agenteRef, proyectoRef, conectorRef)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	motivo, err := runtimeOrderMotivoStartRuntime(excludeOrderID)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	tareaID, err = resolverTareaIDStartRuntime(agenteRef, proyecto, tareaID, motivo)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	perfilTarea, modelo, razonamiento, err = resolverPerfilModeloStartRuntime(
		agenteRef,
		proyecto,
		conector,
		resume,
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	var bootstrap *bootstrapRuntimeData
	agenteRef, resume, bootstrap, err = prepararResumeStartRuntime(
		agenteRef,
		proyecto,
		conector,
		resume,
		perfilTarea,
		modelo,
		razonamiento,
		skipBootstrap,
		excludeOrderID,
	)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	plan, err := construirLaunchPlanStartRuntime(
		agente,
		proyecto,
		conector,
		agenteRef,
		perfilTarea,
		modelo,
		razonamiento,
		resume,
		tareaID,
	)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	return agente, proyecto, conector, ultima, resume, bootstrap, plan, nil
}

func resolverTareaIDStartRuntime(agenteRef string, proyecto *Proyecto, tareaID *int64, motivo string) (*int64, error) {
	if tareaID != nil && *tareaID > 0 {
		return tareaID, nil
	}
	if proyecto == nil || proyecto.ID <= 0 {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(motivo)) {
	case "premium_idle_autoassigned":
	default:
		return nil, nil
	}
	activaID, err := GetTareaActivaIDPorAgenteProyecto(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil {
		return nil, err
	}
	if activaID <= 0 {
		return nil, nil
	}
	id := activaID
	return &id, nil
}

func runtimeOrderMotivoStartRuntime(orderID int64) (string, error) {
	if orderID <= 0 {
		return "", nil
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if order == nil {
		return "", nil
	}
	payload, err := runtimeOrderPayloadMap(order.PayloadJSON)
	if err != nil {
		return "", err
	}
	return stringFromMap(payload, "motivo", ""), nil
}
