package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	ID            int64      `json:"id"`
	Agente        string     `json:"agente"`
	ProyectoID    *int64     `json:"proyecto_id,omitempty"`
	RuntimeID     *int64     `json:"runtime_id,omitempty"`
	HandleID      *int64     `json:"handle_id,omitempty"`
	Tipo          string     `json:"tipo"`
	PayloadJSON   string     `json:"payload_json"`
	ResultadoJSON string     `json:"resultado_json"`
	ErrorText     string     `json:"error_text"`
	Estado        string     `json:"estado"`
	AvailableAt   time.Time  `json:"available_at"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
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
}

type FiltroRuntimeMailbox struct {
	ToAgente   *string
	FromAgente *string
	ProyectoID *int64
	Estado     *string
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
	return err
}

func MarcarRuntimeHandlesCerradosPorAgente(agente string) error {
	if strings.TrimSpace(agente) == "" {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE agente = ? AND estado IN ('activo','pausado')
		  AND (sesion_id IS NULL OR sesion_id NOT IN (
		      SELECT id FROM sesiones WHERE agente = ? AND activa = 1
		  ))`, strings.TrimSpace(agente), strings.TrimSpace(agente))
	return err
}

func GetRuntimeHandleBySesionID(sesionID int64) (*RuntimeHandle, error) {
	row := DB.QueryRow(runtimeHandleSelectBase()+` WHERE sesion_id = ? ORDER BY id DESC LIMIT 1`, sesionID)
	h, err := scanRuntimeHandle(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func GetRuntimeHandle(id int64) (*RuntimeHandle, error) {
	row := DB.QueryRow(runtimeHandleSelectBase()+` WHERE id = ?`, id)
	h, err := scanRuntimeHandle(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func GetRuntimeHandleActivoAgente(agente string) (*RuntimeHandle, error) {
	row := DB.QueryRow(runtimeHandleSelectBase()+`
		WHERE agente = ? AND estado IN ('activo','pausado')
		ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC
		LIMIT 1`, strings.TrimSpace(agente))
	h, err := scanRuntimeHandle(row)
	if err == sql.ErrNoRows {
		return recuperarRuntimeHandleVivoAgenteProyecto(strings.TrimSpace(agente), nil)
	}
	return h, err
}

func GetRuntimeHandleActivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	q := runtimeHandleSelectBase() + `
		WHERE agente = ? AND estado IN ('activo','pausado')`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC LIMIT 1`
	h, err := scanRuntimeHandle(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return recuperarRuntimeHandleVivoAgenteProyecto(strings.TrimSpace(agente), proyectoID)
	}
	return h, err
}

func recuperarRuntimeHandleVivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	q := runtimeHandleSelectBase() + `
		WHERE agente = ? AND estado = 'fallido'`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC LIMIT 1`
	handle, err := scanRuntimeHandle(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return refrescarRuntimeHandleSiSigueVivo(handle)
}

func refrescarRuntimeHandleSiSigueVivo(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	obj := controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}
	if runtime, err := runtimeHandleRuntime(handle); err == nil && runtime != nil && runtime.PID != nil && *runtime.PID > 0 {
		obj.PID = runtime.PID
	}
	vivo, _, err := controlruntime.ProcesoVivo(obj)
	if err != nil || !vivo {
		return nil, nil
	}
	estado := estadoReviveRuntimeHandle(handle)
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estado, handle.ID); err != nil {
		return nil, err
	}
	if runtime, err := runtimeHandleRuntime(handle); err == nil && runtime != nil {
		processState := runtime.ProcessState
		if strings.TrimSpace(processState) == "" || strings.EqualFold(strings.TrimSpace(processState), "fallido") {
			processState = "running"
		}
		logicalState := runtime.LogicalState
		if strings.TrimSpace(logicalState) == "" || strings.EqualFold(strings.TrimSpace(logicalState), "fallido") {
			logicalState = "esperando_io"
		}
		if estado == "pausado" {
			processState = "stopped"
			logicalState = "pausado"
		}
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET pid = COALESCE(pid, ?),
			    logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			obj.PID, logicalState, processState, runtime.ID,
		); err != nil {
			return nil, err
		}
	}
	return GetRuntimeHandle(handle.ID)
}

func estadoReviveRuntimeHandle(handle *RuntimeHandle) string {
	if handle == nil {
		return "activo"
	}
	if handle.SesionID != nil {
		if sesion, err := GetSesionByID(*handle.SesionID); err == nil && sesion != nil {
			switch strings.TrimSpace(sesion.Estado) {
			case "pausada":
				return "pausado"
			case "activa":
				return "activo"
			}
		}
	}
	return "activo"
}

func runtimeHandleRuntime(handle *RuntimeHandle) (*RuntimeInstance, error) {
	if handle == nil || handle.RuntimeID == nil || *handle.RuntimeID <= 0 {
		return nil, nil
	}
	return GetRuntime(*handle.RuntimeID)
}

func ListarRuntimeHandles(agente *string) ([]*RuntimeHandle, error) {
	q := runtimeHandleSelectBase() + ` WHERE 1=1`
	var args []any
	if agente != nil {
		q += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*agente))
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeHandle
	for rows.Next() {
		h, err := scanRuntimeHandle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func EncolarRuntimeOrder(order *RuntimeOrder) (int64, error) {
	if order == nil || strings.TrimSpace(order.Agente) == "" || strings.TrimSpace(order.Tipo) == "" {
		return 0, sql.ErrNoRows
	}
	if strings.TrimSpace(order.PayloadJSON) == "" {
		order.PayloadJSON = "{}"
	}
	if strings.TrimSpace(order.ResultadoJSON) == "" {
		order.ResultadoJSON = "{}"
	}
	res, err := DB.Exec(`
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
	return res.LastInsertId()
}

func CrearHandoffAgenteVivo(origen, destino string, tareaID *int64, motivo, resumenContinuidad, externalSessionID string) (int64, error) {
	origen = strings.TrimSpace(origen)
	destino = strings.TrimSpace(destino)
	if origen == "" || destino == "" {
		return 0, fmt.Errorf("agente origen y destino son obligatorios")
	}
	if origen == destino {
		return 0, fmt.Errorf("origen y destino deben ser distintos")
	}

	sesionOrigen, err := GetSesionActiva(origen, nil)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("el agente origen %s no tiene sesión activa", origen)
	}
	if err != nil {
		return 0, err
	}
	handleOrigen, err := GetRuntimeHandleActivoAgente(origen)
	if err != nil {
		return 0, err
	}
	if handleOrigen == nil {
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
	destinoHandle, err = GetRuntimeHandleActivoAgente(destino)
	if err != nil {
		return 0, err
	}
	if destinoHandle != nil {
		if destinoHandle.ProyectoID != nil {
			proyectoID = destinoHandle.ProyectoID
		}
		if destinoHandle.RuntimeID != nil {
			destinoRuntime, err = GetRuntime(*destinoHandle.RuntimeID)
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
			`UPDATE tareas SET notas = notas || char(10) || ? || ' [' || datetime('now') || ' orquesta]' WHERE id=?`,
			nota, *tareaID,
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

	res, err := tx.Exec(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, runtime_id, handle_id, tipo, payload_json,
			resultado_json, error_text, estado, available_at
		) VALUES (?,?,?,?,?,?,'{}','', 'pendiente', CURRENT_TIMESTAMP)`,
		destino, proyectoID, runtimeID, handleID, "handoff", string(payloadJSON),
	)
	if err != nil {
		return 0, err
	}
	orderID, _ := res.LastInsertId()

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if err := marcarOrigenHandoffPausado(origen, sesionOrigen, handleOrigen); err != nil {
		return 0, err
	}
	detalle := fmt.Sprintf("%s→%s sesion_origen=%d", origen, destino, sesionOrigen.ID)
	Audit(origen, "handoff_agente_vivo", "runtime_order", orderID, detalle)
	return orderID, nil
}

func ClaimNextRuntimeOrder(agente string) (*RuntimeOrder, error) {
	for {
		var id int64
		err := DB.QueryRow(`
			SELECT id
			FROM runtime_orders
			WHERE agente = ?
			  AND estado = 'pendiente'
			  AND available_at <= CURRENT_TIMESTAMP
			ORDER BY id
			LIMIT 1`, strings.TrimSpace(agente)).Scan(&id)
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

func ClaimNextBootstrapRuntimeOrder(agente string, proyectoID *int64) (*RuntimeOrder, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	tipos := runtimeOrderTiposBootstrap()
	placeholders, tipoArgs := runtimeOrderPlaceholders(tipos)
	argsBase := make([]any, 0, len(tipoArgs)+3)
	argsBase = append(argsBase, agente)
	argsBase = append(argsBase, tipoArgs...)

	for {
		q := `
			SELECT id
			FROM runtime_orders
			WHERE agente = ?
			  AND estado = 'pendiente'
			  AND available_at <= CURRENT_TIMESTAMP
			  AND tipo IN (` + placeholders + `)`
		args := append([]any(nil), argsBase...)
		if proyectoID != nil {
			q += `
			  AND (proyecto_id = ? OR proyecto_id IS NULL)
			ORDER BY
			  CASE
			    WHEN proyecto_id = ? THEN 0
			    WHEN proyecto_id IS NULL THEN 1
			    ELSE 2
			  END,
			  CASE tipo
			    WHEN 'handoff' THEN 0
			    WHEN 'resume' THEN 1
			    WHEN 'start' THEN 2
			    ELSE 9
			  END,
			  id
			LIMIT 1`
			args = append(args, *proyectoID, *proyectoID)
		} else {
			q += `
			ORDER BY
			  CASE
			    WHEN proyecto_id IS NULL THEN 0
			    ELSE 1
			  END,
			  CASE tipo
			    WHEN 'handoff' THEN 0
			    WHEN 'resume' THEN 1
			    WHEN 'start' THEN 2
			    ELSE 9
			  END,
			  id
			LIMIT 1`
		}

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

func GetRuntimeOrder(id int64) (*RuntimeOrder, error) {
	row := DB.QueryRow(runtimeOrderSelectBase()+` WHERE id = ?`, id)
	return scanRuntimeOrder(row)
}

func ListarRuntimeOrders(filter FiltroRuntimeOrders) ([]*RuntimeOrder, error) {
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
	q += ` ORDER BY id DESC`
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
}

func MarcarRuntimeOrderEstado(id int64, estado, resultadoJSON, errorText string) error {
	if strings.TrimSpace(resultadoJSON) == "" {
		resultadoJSON = "{}"
	}
	switch strings.TrimSpace(estado) {
	case "ejecutando":
		_, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando' WHERE id = ?`, id)
		return err
	case "completada", "fallida", "expirada", "cancelada":
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = ?, resultado_json = ?, error_text = ?, finished_at = CURRENT_TIMESTAMP
			WHERE id = ?`, estado, resultadoJSON, errorText, id)
		return err
	default:
		_, err := DB.Exec(`UPDATE runtime_orders SET estado = ? WHERE id = ?`, estado, id)
		return err
	}
}

func CrearRuntimeCheckpoint(cp *RuntimeCheckpoint) (int64, error) {
	if cp == nil || strings.TrimSpace(cp.Agente) == "" {
		return 0, sql.ErrNoRows
	}
	if strings.TrimSpace(cp.PayloadJSON) == "" {
		cp.PayloadJSON = "{}"
	}
	if strings.TrimSpace(cp.CheckpointKind) == "" {
		cp.CheckpointKind = "manual"
	}
	res, err := DB.Exec(`
		INSERT INTO runtime_checkpoints (
			agente, proyecto_id, sesion_id, runtime_id, checkpoint_kind, resumen,
			branch, cwd, payload_json, resume_strategy, source
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		cp.Agente, cp.ProyectoID, cp.SesionID, cp.RuntimeID, cp.CheckpointKind, cp.Resumen,
		cp.Branch, cp.CWD, cp.PayloadJSON, cp.ResumeStrategy, cp.Source,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
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
	return cp, err
}

func GetRuntimeCheckpoint(id int64) (*RuntimeCheckpoint, error) {
	row := DB.QueryRow(runtimeCheckpointSelectBase()+` WHERE id = ?`, id)
	cp, err := scanRuntimeCheckpoint(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cp, err
}

func GetRuntimeCheckpointBySource(source string) (*RuntimeCheckpoint, error) {
	row := DB.QueryRow(runtimeCheckpointSelectBase()+` WHERE source = ? ORDER BY id DESC LIMIT 1`, strings.TrimSpace(source))
	cp, err := scanRuntimeCheckpoint(row)
	if err == sql.ErrNoRows {
		return nil, nil
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
	return out, rows.Err()
}

func EnviarRuntimeMailbox(msg *RuntimeMailboxMessage) (int64, error) {
	if msg == nil || strings.TrimSpace(msg.FromAgente) == "" || strings.TrimSpace(msg.ToAgente) == "" || strings.TrimSpace(msg.Kind) == "" {
		return 0, sql.ErrNoRows
	}
	if strings.TrimSpace(msg.PayloadJSON) == "" {
		msg.PayloadJSON = "{}"
	}
	res, err := DB.Exec(`
		INSERT INTO runtime_mailbox (
			from_agente, to_agente, proyecto_id, runtime_order_id, kind, payload_json, estado
		) VALUES (?,?,?,?,?,?,?)`,
		msg.FromAgente, msg.ToAgente, msg.ProyectoID, msg.RuntimeOrderID, msg.Kind, msg.PayloadJSON, defaultMailboxEstado(msg.Estado),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListarRuntimeMailbox(filter FiltroRuntimeMailbox) ([]*RuntimeMailboxMessage, error) {
	q := runtimeMailboxSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.ToAgente != nil {
		q += ` AND to_agente = ?`
		args = append(args, strings.TrimSpace(*filter.ToAgente))
	}
	if filter.FromAgente != nil {
		q += ` AND from_agente = ?`
		args = append(args, strings.TrimSpace(*filter.FromAgente))
	}
	if filter.ProyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Estado != nil {
		q += ` AND estado = ?`
		args = append(args, strings.TrimSpace(*filter.Estado))
	}
	q += ` ORDER BY id DESC`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeMailboxMessage
	for rows.Next() {
		msg, err := scanRuntimeMailbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func GetRuntimeMailboxByRuntimeOrderID(runtimeOrderID int64) (*RuntimeMailboxMessage, error) {
	row := DB.QueryRow(runtimeMailboxSelectBase()+`
		WHERE runtime_order_id = ?
		ORDER BY id DESC
		LIMIT 1`, runtimeOrderID)
	msg, err := scanRuntimeMailbox(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return msg, err
}

func MarcarRuntimeMailboxEntregado(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='entregado', delivered_at=CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

func MarcarRuntimeMailboxConsumido(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='consumido', consumed_at=CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

func ReconciliarRuntimeHandlesStale() (int, error) {
	staleSeconds := configIntOrDefault("runtime_handle_stale_seconds", 120)
	if staleSeconds <= 0 {
		staleSeconds = 120
	}
	cutoff := time.Now().UTC().Add(-time.Duration(staleSeconds) * time.Second)
	rows, err := DB.Query(`
		SELECT id
		FROM runtime_handles
		WHERE estado IN ('activo','pausado')
		  AND COALESCE(last_seen_at, created_at) <= ?`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	reconciled := 0
	for _, id := range ids {
		handle, err := GetRuntimeHandle(id)
		if err != nil {
			return 0, err
		}
		if revived, err := refrescarRuntimeHandleSiSigueVivo(handle); err != nil {
			return 0, err
		} else if revived != nil {
			continue
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
			WHERE id = ?`, id); err != nil {
			return 0, err
		}
		reconciled++
	}
	return reconciled, nil
}

func ReconciliarRuntimeOrdersStale() (int, error) {
	staleSeconds := configIntOrDefault("runtime_order_stale_seconds", 120)
	if staleSeconds <= 0 {
		staleSeconds = 120
	}
	cutoff := time.Now().UTC().Add(-time.Duration(staleSeconds) * time.Second)
	rows, err := DB.Query(`
		SELECT id, tipo
		FROM runtime_orders
		WHERE estado IN ('tomada','ejecutando')
		  AND COALESCE(started_at, updated_at, created_at) <= ?
		ORDER BY id`, cutoff)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type staleOrder struct {
		ID   int64
		Tipo string
	}
	var orders []staleOrder
	for rows.Next() {
		var item staleOrder
		if err := rows.Scan(&item.ID, &item.Tipo); err != nil {
			return 0, err
		}
		orders = append(orders, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	recovered := 0
	for _, item := range orders {
		if runtimeOrderTipoDespachable(item.Tipo) {
			if _, err := DB.Exec(`
				UPDATE runtime_orders
				SET estado = 'pendiente',
				    started_at = NULL,
				    finished_at = NULL,
				    available_at = CURRENT_TIMESTAMP,
				    error_text = CASE
				        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'reencolada tras stale del control plane'
				        ELSE error_text || CHAR(10) || 'reencolada tras stale del control plane'
				    END
				WHERE id = ?`, item.ID); err != nil {
				return recovered, err
			}
		} else {
			if _, err := DB.Exec(`
				UPDATE runtime_orders
				SET estado = 'expirada',
				    finished_at = CURRENT_TIMESTAMP,
				    error_text = CASE
				        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'orden expirada por stale sin dispatcher compatible'
				        ELSE error_text || CHAR(10) || 'orden expirada por stale sin dispatcher compatible'
				    END
				WHERE id = ?`, item.ID); err != nil {
				return recovered, err
			}
		}
		recovered++
	}
	return recovered, nil
}

func ProcesarRuntimeOrdersBatch() (int, error) {
	return procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDespachables())
}

func ProcesarRuntimeOrdersBasicasBatch() (int, error) {
	return procesarRuntimeOrdersBatchTipos(runtimeOrderTiposBasicos())
}

func procesarRuntimeOrdersBatchTipos(tipos []string) (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	placeholders, args := runtimeOrderPlaceholders(tipos)
	args = append(args, limit)
	rows, err := DB.Query(`
		SELECT id
		FROM runtime_orders
		WHERE estado = 'pendiente'
		  AND available_at <= CURRENT_TIMESTAMP
		  AND tipo IN (`+placeholders+`)
		ORDER BY id
		LIMIT ?`, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	for _, id := range ids {
		order, err := claimRuntimeOrderByID(id)
		if err != nil {
			return processed, err
		}
		if order == nil {
			continue
		}
		if err := MarcarRuntimeOrderEstado(order.ID, "ejecutando", order.ResultadoJSON, ""); err != nil {
			return processed, err
		}
		if err := ejecutarRuntimeOrderBasica(order); err != nil {
			_ = MarcarRuntimeOrderEstado(order.ID, "fallida", `{"ok":false}`, err.Error())
			processed++
			continue
		}
		processed++
	}
	return processed, nil
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
		       error_text, estado, available_at, created_at, started_at, finished_at, updated_at
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

func ejecutarRuntimeOrderBasica(order *RuntimeOrder) error {
	switch strings.TrimSpace(order.Tipo) {
	case "sync_status":
		return ejecutarRuntimeOrderSyncStatus(order)
	case "checkpoint":
		return ejecutarRuntimeOrderCheckpoint(order)
	case "nudge":
		return ejecutarRuntimeOrderNudge(order)
	case "discordia":
		return ejecutarRuntimeOrderDiscordia(order)
	case "start":
		return ejecutarRuntimeOrderStart(order)
	case "pause":
		return ejecutarRuntimeOrderPause(order)
	case "resume":
		return ejecutarRuntimeOrderResume(order)
	case "stop":
		return ejecutarRuntimeOrderStop(order)
	case "restart":
		return ejecutarRuntimeOrderRestart(order)
	case "send_instruction":
		return ejecutarRuntimeOrderSendInstruction(order)
	case "handoff":
		return ejecutarRuntimeOrderHandoff(order)
	default:
		return fmt.Errorf("tipo de orden aún no soportado por el dispatcher básico: %s", order.Tipo)
	}
}

func ejecutarRuntimeOrderSyncStatus(order *RuntimeOrder) error {
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}
	result := map[string]any{
		"ok":              true,
		"handle_id":       nil,
		"runtime_id":      nil,
		"handle":          nil,
		"runtime":         nil,
		"sin_handle":      handle == nil,
		"sin_runtime":     runtime == nil,
		"observed_remote": false,
	}
	if handle != nil {
		estadoRemoto, observed, err := controlruntime.ConsultarEstadoRemoto(controlruntime.ObjetivoProceso{
			HandleKind:   handle.HandleKind,
			HandleRef:    handle.HandleRef,
			MetadataJSON: handle.MetadataJSON,
		})
		if err != nil {
			failures, degraded, obsErr := registrarFalloObservacionRemota(handle, runtime, err)
			if obsErr != nil {
				return obsErr
			}
			result["ok"] = false
			result["observed_remote_error"] = true
			result["remote_sync_failures"] = failures
			result["remote_degraded"] = degraded
			result["remote_error"] = strings.TrimSpace(err.Error())
			if handle.ID > 0 {
				handle, err = GetRuntimeHandle(handle.ID)
				if err != nil {
					return err
				}
			}
			if runtime != nil && runtime.ID > 0 {
				runtime, err = GetRuntime(runtime.ID)
				if err != nil {
					return err
				}
			}
			data, _ := json.Marshal(result)
			return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
		}
		if observed && estadoRemoto != nil {
			if err := aplicarEstadoRemotoObservado(handle, runtime, estadoRemoto); err != nil {
				return err
			}
			result["observed_remote"] = true
			if handle.ID > 0 {
				handle, err = GetRuntimeHandle(handle.ID)
				if err != nil {
					return err
				}
			}
			if runtime != nil && runtime.ID > 0 {
				runtime, err = GetRuntime(runtime.ID)
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(estadoRemoto.RawJSON) != "" {
				result["remote_status"] = json.RawMessage(estadoRemoto.RawJSON)
			}
		}
	}
	if handle != nil {
		result["handle_id"] = handle.ID
		result["handle"] = map[string]any{
			"estado":      handle.Estado,
			"transporte":  handle.Transporte,
			"handle_kind": handle.HandleKind,
			"handle_ref":  handle.HandleRef,
		}
	}
	if runtime != nil {
		result["runtime_id"] = runtime.ID
		result["runtime"] = map[string]any{
			"logical_state": runtime.LogicalState,
			"process_state": runtime.ProcessState,
			"connector":     runtime.Connector,
			"pid":           runtime.PID,
		}
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func aplicarEstadoRemotoObservado(handle *RuntimeHandle, runtime *RuntimeInstance, estado *controlruntime.EstadoRemoto) error {
	if estado == nil {
		return nil
	}
	observedAt := time.Now().UTC().Format(time.RFC3339)
	conector, err := resolverConectorRuntime(handle, runtime)
	if err != nil {
		return err
	}
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		meta["remote_last_status_at"] = observedAt
		meta["remote_sync_failures"] = 0
		delete(meta, "remote_last_error")
		delete(meta, "remote_last_error_at")
		if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
			meta["remote_handle_state"] = rawEstado
		}
		for k, v := range mapFromJSON(estado.MetadataJSON) {
			meta[k] = v
		}
		metaJSON, _ := json.Marshal(meta)
		caps := handle.CapabilitiesJSON
		if strings.TrimSpace(estado.CapabilitiesJSON) != "" {
			caps = strings.TrimSpace(estado.CapabilitiesJSON)
		}
		estadoHandle := strings.TrimSpace(handle.Estado)
		if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
			estadoHandle = normalizarEstadoHandleObservado(rawEstado, estadoHandle)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado = ?,
			    metadata_json = ?,
			    capabilities_json = ?,
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			estadoHandle, string(metaJSON), caps, handle.ID,
		); err != nil {
			return err
		}
	}
	if runtime != nil {
		logical := strings.TrimSpace(runtime.LogicalState)
		if strings.TrimSpace(estado.LogicalState) != "" {
			logical = strings.TrimSpace(estado.LogicalState)
		}
		process := strings.TrimSpace(runtime.ProcessState)
		if strings.TrimSpace(estado.ProcessState) != "" {
			process = strings.TrimSpace(estado.ProcessState)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			logical, process, runtime.ID,
		); err != nil {
			return err
		}
	}
	if conector != nil {
		if err := RegistrarExitoConector(conector.ID); err != nil {
			return err
		}
	}
	return nil
}

func registrarFalloObservacionRemota(handle *RuntimeHandle, runtime *RuntimeInstance, remoteErr error) (int, bool, error) {
	if handle == nil {
		return 0, false, nil
	}
	conector, err := resolverConectorRuntime(handle, runtime)
	if err != nil {
		return 0, false, err
	}
	meta := mapFromJSON(handle.MetadataJSON)
	failures := intFromMap(meta, "remote_sync_failures") + 1
	meta["remote_sync_failures"] = failures
	meta["remote_last_error"] = strings.TrimSpace(remoteErr.Error())
	meta["remote_last_error_at"] = time.Now().UTC().Format(time.RFC3339)
	threshold := configIntOrDefault("runtime_remote_sync_failure_threshold", 3)
	if threshold <= 0 {
		threshold = 3
	}
	degraded := failures >= threshold
	handleState := strings.TrimSpace(handle.Estado)
	if degraded {
		handleState = "fallido"
	}
	metaJSON, _ := json.Marshal(meta)
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    metadata_json = ?
		WHERE id = ?`,
		handleState, string(metaJSON), handle.ID,
	); err != nil {
		return failures, degraded, err
	}
	if runtime != nil && degraded {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			"degradado", "remote_status_error", runtime.ID,
		); err != nil {
			return failures, degraded, err
		}
	}
	if conector != nil {
		if _, err := RegistrarFalloConector(conector.ID, remoteErr.Error()); err != nil {
			return failures, degraded, err
		}
	}
	return failures, degraded, nil
}

func resolverConectorRuntime(handle *RuntimeHandle, runtime *RuntimeInstance) (*Conector, error) {
	slug := ""
	if runtime != nil {
		slug = strings.TrimSpace(runtime.Connector)
	}
	if slug == "" && handle != nil {
		slug = stringFromMap(mapFromJSON(handle.MetadataJSON), "conector", "")
	}
	if slug == "" && handle != nil && handle.SesionID != nil {
		sesion, err := GetSesionByID(*handle.SesionID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if sesion != nil {
			slug = strings.TrimSpace(sesion.ConectorSlug)
		}
	}
	if slug == "" {
		return nil, nil
	}
	conector, err := GetConector(slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return conector, err
}

func normalizarEstadoHandleObservado(observado, actual string) string {
	switch strings.ToLower(strings.TrimSpace(observado)) {
	case "activo", "active", "running", "online", "ready", "idle", "esperando", "degradado", "warning":
		return "activo"
	case "pausado", "paused", "suspended":
		return "pausado"
	case "cerrado", "closed", "stopped", "finished", "finalizado":
		return "cerrado"
	case "fallido", "failed", "error", "crashed":
		return "fallido"
	}
	actual = strings.TrimSpace(actual)
	if actual == "" {
		return "activo"
	}
	return actual
}

func ejecutarRuntimeOrderCheckpoint(order *RuntimeOrder) error {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return err
	}
	if sesion == nil {
		return fmt.Errorf("no existe sesión activa ni reciente para %s", order.Agente)
	}

	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)
	checkpointKind := stringFromMap(payload, "checkpoint_kind", "manual")
	resumen := stringFromMap(payload, "resumen", sesion.ResumenContinuidad)
	resumeStrategy := stringFromMap(payload, "resume_strategy", "resumen_y_payload")
	source := fmt.Sprintf("runtime_order:%d", order.ID)

	existente, err := GetRuntimeCheckpointBySource(source)
	if err != nil {
		return err
	}
	if existente != nil {
		result := map[string]any{
			"ok":            true,
			"checkpoint_id": existente.ID,
			"sesion_id":     sesion.ID,
			"reused":        true,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	cp := &RuntimeCheckpoint{
		Agente:         order.Agente,
		ProyectoID:     order.ProyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: checkpointKind,
		Resumen:        resumen,
		Branch:         sesion.Branch,
		CWD:            sesion.CWD,
		PayloadJSON:    order.PayloadJSON,
		ResumeStrategy: resumeStrategy,
		Source:         source,
	}
	if runtime != nil {
		cp.RuntimeID = &runtime.ID
	}
	id, err := CrearRuntimeCheckpoint(cp)
	if err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"checkpoint_id": id,
		"sesion_id":     sesion.ID,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderNudge(order *RuntimeOrder) error {
	return ejecutarRuntimeOrderMailboxSimple(order, "nudge", "nudge sin agente destino")
}

func ejecutarRuntimeOrderDiscordia(order *RuntimeOrder) error {
	return ejecutarRuntimeOrderMailboxSimple(order, "discordia", "discordia sin supervisor destino")
}

func ejecutarRuntimeOrderMailboxSimple(order *RuntimeOrder, defaultKind, missingTargetError string) error {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	kind := stringFromMap(payload, "kind", defaultKind)
	if strings.TrimSpace(toAgente) == "" {
		return fmt.Errorf("%s", missingTargetError)
	}

	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return err
	}
	if existente != nil {
		result := map[string]any{
			"ok":          true,
			"mailbox_id":  existente.ID,
			"to_agente":   existente.ToAgente,
			"from_agente": existente.FromAgente,
			"kind":        existente.Kind,
			"reused":      true,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	id, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		ProyectoID:     order.ProyectoID,
		RuntimeOrderID: &order.ID,
		Kind:           kind,
		PayloadJSON:    order.PayloadJSON,
	})
	if err != nil {
		return err
	}

	result := map[string]any{
		"ok":          true,
		"mailbox_id":  id,
		"to_agente":   toAgente,
		"from_agente": fromAgente,
		"kind":        kind,
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
	obj := controlruntime.ObjetivoProceso{}
	if handle != nil {
		obj.HandleKind = handle.HandleKind
		obj.HandleRef = handle.HandleRef
		obj.MetadataJSON = handle.MetadataJSON
	}
	if runtime != nil && (handle == nil || strings.TrimSpace(handle.HandleKind) == "" || strings.TrimSpace(handle.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return signaler(obj)
}

func ejecutarRuntimeOrderStart(order *RuntimeOrder) error {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)

	proyectoRef := stringFromMap(payload, "proyecto", "")
	if strings.TrimSpace(proyectoRef) == "" && order.ProyectoID != nil {
		proyectoRef = jsonNumber(*order.ProyectoID)
	}
	if strings.TrimSpace(proyectoRef) == "" {
		return fmt.Errorf("start sin proyecto")
	}

	agente, proyecto, conector, ultima, resume, bootstrap, plan, err := prepararStartRuntimeOrder(
		order.Agente,
		proyectoRef,
		order.ID,
		stringFromMap(payload, "conector", ""),
		stringFromMap(payload, "modelo", ""),
		stringFromMap(payload, "razonamiento", ""),
		stringFromMap(payload, "perfil", ""),
	)
	if err != nil {
		return err
	}
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
		Agente:   order.Agente,
		Proyecto: proyecto.Slug,
		Plan:     plan,
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
	sesion, err := IniciarSesionContexto(SesionInicio{
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
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		return err
	}
	if runtime == nil {
		return fmt.Errorf("no se pudo registrar runtime para la sesion %d", sesion.ID)
	}
	if handle != nil {
		if texto := construirTranscriptArranque(plan, resume, bootstrap); strings.TrimSpace(texto) != "" {
			if err := RegistrarRuntimeTranscriptSistema(handle, runtime, texto); err != nil {
				return err
			}
		}
		if err := inyectarContinuidadArranque(handle, runtime, plan, order); err != nil {
			return err
		}
	}
	if err := registrarEvidenciaHandoffReanudado(order.Agente, sesion.ID, bootstrap); err != nil {
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

func inyectarContinuidadArranque(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan, order *RuntimeOrder) error {
	if handle == nil || runtime == nil || plan == nil {
		return nil
	}
	obj := controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}
	if runtime.PID != nil && *runtime.PID > 0 {
		obj.PID = runtime.PID
	}
	texto := controlruntime.NormalizarInstruccionProceso(obj, strings.TrimSpace(plan.ContinuityPrompt))
	if texto == "" {
		return nil
	}
	aplicado, _, err := controlruntime.EnviarInstruccionProceso(obj, texto)
	if err != nil {
		return err
	}
	if aplicado {
		return RegistrarRuntimeTranscriptInput(handle, runtime, texto)
	}
	payloadJSON, _ := json.Marshal(map[string]any{
		"texto":       texto,
		"kind":        "instruction",
		"from_agente": "orquesta",
		"motivo":      "continuity_prompt_start",
	})
	_, err = EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    strings.TrimSpace(order.Agente),
		ProyectoID:  order.ProyectoID,
		Kind:        "instruction",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func ejecutarRuntimeOrderPause(order *RuntimeOrder) error {
	aplicado, pid, err := controlarProcesoRuntime(order, controlruntime.PausarProceso)
	if err != nil {
		return err
	}
	if !aplicado {
		return fmt.Errorf("pause sin control real para %s", order.Agente)
	}
	runtime, err := actualizarEstadoRuntime(order, "pausado", "")
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='pausado', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado='activo'`, order.Agente); err != nil {
		return err
	}
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
	if checkpointID > 0 {
		result["checkpoint_id"] = checkpointID
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderResume(order *RuntimeOrder) error {
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
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('pausado','fallido')`, order.Agente); err != nil {
		return err
	}
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
	if err != nil {
		return err
	}
	if !aplicado {
		return fmt.Errorf("stop sin control real para %s", order.Agente)
	}
	checkpointID, err := asegurarCheckpointCambioContexto(order, "stop", "Checkpoint automático antes de detener")
	if err != nil {
		return err
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
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado')`, handleState, order.Agente); err != nil {
		return err
	}
	sesion, err := GetSesionActiva(order.Agente, order.ProyectoID)
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
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
		"control_real":  aplicado,
		"pid":           pid,
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

func ejecutarRuntimeOrderSendInstruction(order *RuntimeOrder) error {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	if strings.TrimSpace(toAgente) == "" {
		return fmt.Errorf("send_instruction sin agente destino")
	}
	texto := stringFromMap(payload, "texto", stringFromMap(payload, "instruction", ""))

	obj, err := objetivoProcesoParaOrden(order)
	if err != nil {
		return err
	}
	texto = controlruntime.NormalizarInstruccionProceso(obj, texto)
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("send_instruction sin texto aplicable")
	}

	aplicado, pid, err := controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		return controlruntime.EnviarInstruccionProceso(obj, texto)
	})
	if err != nil {
		return err
	}
	if aplicado {
		handle, _ := resolverHandleParaOrden(order)
		runtime, _ := resolverRuntimeParaOrden(order)
		if err := RegistrarRuntimeTranscriptInput(handle, runtime, texto); err != nil {
			return err
		}
		result := map[string]any{
			"ok":           true,
			"to_agente":    toAgente,
			"from_agente":  fromAgente,
			"kind":         "instruction",
			"control_real": true,
			"pid":          pid,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
	}

	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return err
	}
	if existente != nil {
		result := map[string]any{
			"ok":          true,
			"mailbox_id":  existente.ID,
			"to_agente":   existente.ToAgente,
			"from_agente": existente.FromAgente,
			"reused":      true,
		}
		data, _ := json.Marshal(result)
		return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
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
	result := map[string]any{
		"ok":          true,
		"mailbox_id":  id,
		"to_agente":   toAgente,
		"from_agente": fromAgente,
		"kind":        "instruction",
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func objetivoProcesoParaOrden(order *RuntimeOrder) (controlruntime.ObjetivoProceso, error) {
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return controlruntime.ObjetivoProceso{}, err
	}
	obj := controlruntime.ObjetivoProceso{}
	if handle != nil {
		obj.HandleKind = handle.HandleKind
		obj.HandleRef = handle.HandleRef
		obj.MetadataJSON = handle.MetadataJSON
	}
	if runtime != nil && (handle == nil || strings.TrimSpace(handle.HandleKind) == "" || strings.TrimSpace(handle.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return obj, nil
}

func runtimeHandleID(handle *RuntimeHandle) any {
	if handle == nil {
		return nil
	}
	return handle.ID
}

func payloadJSONDesdePlan(plan *runtimeagente.LaunchPlan) string {
	if plan == nil {
		return "{}"
	}
	data, err := json.Marshal(map[string]any{
		"modo":              strings.TrimSpace(plan.Modo),
		"native_resume":     plan.NativeResume,
		"continuity_prompt": strings.TrimSpace(plan.ContinuityPrompt),
	})
	if err != nil {
		return "{}"
	}
	return string(data)
}

func payloadJSONDesdePlanYResume(plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext) string {
	base := payloadJSONDesdePlan(plan)
	if strings.TrimSpace(resume.ResumePayloadJSON) != "" {
		return strings.TrimSpace(resume.ResumePayloadJSON)
	}
	return base
}

func construirTranscriptArranque(plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext, bootstrap *bootstrapRuntimeData) string {
	parts := []string{"Orquesta ha arrancado o reanudado este agente bajo control plane."}
	if plan != nil {
		if strings.TrimSpace(plan.Modo) != "" {
			parts = append(parts, fmt.Sprintf("Modo: %s.", strings.TrimSpace(plan.Modo)))
		}
		if strings.TrimSpace(plan.WorkingDir) != "" {
			parts = append(parts, fmt.Sprintf("Directorio: %s.", strings.TrimSpace(plan.WorkingDir)))
		}
		if strings.TrimSpace(plan.ContinuityPrompt) != "" {
			parts = append(parts, "Prompt de continuidad: "+strings.TrimSpace(plan.ContinuityPrompt))
		}
	}
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		parts = append(parts, "Resumen de continuidad: "+strings.TrimSpace(resume.ResumenContinuidad))
	}
	if bootstrap != nil && bootstrap.Checkpoint != nil && bootstrap.Checkpoint.ID > 0 {
		parts = append(parts, fmt.Sprintf("Checkpoint de referencia: #%d.", bootstrap.Checkpoint.ID))
	}
	return strings.Join(parts, " ")
}

func branchDesdeResume(ultima *Sesion, resume runtimeagente.ResumeContext) string {
	if strings.TrimSpace(resume.Branch) != "" {
		return strings.TrimSpace(resume.Branch)
	}
	if ultima == nil {
		return ""
	}
	return strings.TrimSpace(ultima.Branch)
}

func resumenContinuidadDesdeResume(ultima *Sesion, resume runtimeagente.ResumeContext) string {
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		return strings.TrimSpace(resume.ResumenContinuidad)
	}
	if ultima == nil {
		return ""
	}
	return strings.TrimSpace(ultima.ResumenContinuidad)
}

func actualizarHandleRuntimeArranque(handleID int64, arranque *controlruntime.ProcesoArrancado, conector *Conector, plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext) error {
	if handleID == 0 || arranque == nil {
		return nil
	}
	meta := mapFromJSON(arranque.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	if strings.TrimSpace(arranque.StdinPath) != "" {
		meta["stdin_path"] = strings.TrimSpace(arranque.StdinPath)
	}
	if strings.TrimSpace(arranque.LogPath) != "" {
		meta["log_path"] = strings.TrimSpace(arranque.LogPath)
	}
	if strings.TrimSpace(arranque.WorkingDir) != "" {
		meta["working_dir"] = strings.TrimSpace(arranque.WorkingDir)
	}
	if strings.TrimSpace(arranque.WrappedCommand) != "" {
		meta["wrapped_command"] = strings.TrimSpace(arranque.WrappedCommand)
	}
	if strings.TrimSpace(arranque.RenderedCommand) != "" {
		meta["rendered_command"] = strings.TrimSpace(arranque.RenderedCommand)
	}
	if conector != nil {
		meta["conector"] = strings.TrimSpace(conector.Slug)
	}
	if plan != nil {
		meta["modo_plan"] = strings.TrimSpace(plan.Modo)
		meta["continuity_prompt"] = strings.TrimSpace(plan.ContinuityPrompt)
	}
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		meta["resumen_continuidad"] = strings.TrimSpace(resume.ResumenContinuidad)
	}
	metaJSON, _ := json.Marshal(meta)
	caps := mapFromJSON(arranque.CapabilitiesJSON)
	if caps == nil {
		caps = map[string]any{}
	}
	if len(caps) == 0 {
		caps = map[string]any{
			"can_send_input":       true,
			"can_checkpoint":       true,
			"can_resume":           true,
			"can_capture_pid":      arranque.PID > 0,
			"can_track_continuity": true,
			"can_pause":            arranque.PID > 0,
			"can_stop":             true,
		}
	}
	capsJSON, _ := json.Marshal(caps)
	handleKind := strings.TrimSpace(arranque.HandleKind)
	if handleKind == "" {
		if arranque.PID > 0 {
			handleKind = "process"
		} else {
			handleKind = "session"
		}
	}
	handleRef := strings.TrimSpace(arranque.HandleRef)
	if handleRef == "" {
		if arranque.PID > 0 {
			handleRef = jsonNumber(int64(arranque.PID))
		} else if strings.TrimSpace(arranque.ExternalSessionID) != "" {
			handleRef = strings.TrimSpace(arranque.ExternalSessionID)
		}
	}
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET handle_kind = ?,
		    handle_ref = ?,
		    metadata_json = ?,
		    capabilities_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, handleKind, handleRef, string(metaJSON), string(capsJSON), handleID)
	return err
}

func prepararStartRuntimeOrder(agenteRef, proyectoRef string, excludeOrderID int64, conectorRef, modelo, razonamiento, perfilTarea string) (*Agente, *Proyecto, *Conector, *Sesion, runtimeagente.ResumeContext, *bootstrapRuntimeData, *runtimeagente.LaunchPlan, error) {
	agente, err := GetAgente(strings.TrimSpace(agenteRef))
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	proyecto, err := GetProyecto(strings.TrimSpace(proyectoRef))
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	ultima, err := ObtenerUltimaSesion(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	conector, err := resolverConectorRuntimeOrder(conectorRef, ultima)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	resume := runtimeagente.ResumeContext{}
	if ultima != nil {
		resume = runtimeagente.ResumeContext{
			ExternalSessionID:  strings.TrimSpace(ultima.ExternalSessionID),
			ResumePayloadJSON:  strings.TrimSpace(ultima.ResumePayloadJSON),
			ResumenContinuidad: strings.TrimSpace(ultima.ResumenContinuidad),
			Branch:             strings.TrimSpace(ultima.Branch),
			CWD:                strings.TrimSpace(ultima.CWD),
		}
	}
	resume, bootstrap, err := prepararResumeBootstrapRuntime(strings.TrimSpace(agenteRef), proyecto, resume, excludeOrderID)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	plan, err := runtimeagente.DefaultRegistry().Prepare(runtimeagente.LaunchRequest{
		Agente:       strings.TrimSpace(agente.Nombre),
		Rol:          strings.TrimSpace(agente.Rol),
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		ProyectoRuta: strings.TrimSpace(proyecto.RutaAbs),
		Modelo:       strings.TrimSpace(modelo),
		Razonamiento: strings.TrimSpace(razonamiento),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		Conector: runtimeagente.ConnectorConfig{
			Slug:         strings.TrimSpace(conector.Slug),
			Nombre:       strings.TrimSpace(conector.Nombre),
			Transporte:   strings.TrimSpace(conector.Transporte),
			Comando:      strings.TrimSpace(conector.Comando),
			ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
			EnvJSON:      strings.TrimSpace(conector.EnvJSON),
			MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
			Activo:       conector.Activo,
		},
		Resume: resume,
	})
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	return agente, proyecto, conector, ultima, resume, bootstrap, plan, nil
}

func resolverConectorRuntimeOrder(conectorRef string, ultima *Sesion) (*Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 {
			ref = jsonNumber(*ultima.ConectorID)
		}
	}
	if ref == "" {
		ref = "codex-cli"
	}
	return GetConector(ref)
}

func enriquecerResumeConGobernanzaDB(resume *runtimeagente.ResumeContext, rol, agente string, proyecto *Proyecto) {
	if resume == nil || strings.TrimSpace(rol) == "" {
		return
	}
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" {
		return
	}
	var proyectoID *int64
	if proyecto != nil {
		proyectoID = &proyecto.ID
	}
	contexto, resumen := BuildGovernanceContextSummaryForContext(strings.TrimSpace(rol), proyectoID, strings.TrimSpace(agente))
	if len(contexto) == 0 {
		return
	}
	if payload := AppendGovernanceCatalogPayload(resume.ResumePayloadJSON, contexto); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen != "" && !strings.Contains(resume.ResumenContinuidad, resumen) {
		if strings.TrimSpace(resume.ResumenContinuidad) == "" {
			resume.ResumenContinuidad = resumen
		} else {
			resume.ResumenContinuidad += ". " + resumen
		}
	}
}

func prepararResumeBootstrapRuntime(agente string, proyecto *Proyecto, resume runtimeagente.ResumeContext, excludeOrderID int64) (runtimeagente.ResumeContext, *bootstrapRuntimeData, error) {
	if proyecto == nil {
		return resume, nil, nil
	}
	order, err := claimBootstrapRuntimeOrderParaStart(strings.TrimSpace(agente), &proyecto.ID, excludeOrderID)
	if err != nil {
		return resume, nil, err
	}
	estadoPendiente := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrRuntime(strings.TrimSpace(agente)),
		ProyectoID: &proyecto.ID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		return resume, nil, err
	}
	checkpoint, err := UltimoRuntimeCheckpoint(strings.TrimSpace(agente), &proyecto.ID)
	if err != nil {
		return resume, nil, err
	}
	bootstrap := &bootstrapRuntimeData{
		Order:      order,
		Mailbox:    mailbox,
		Checkpoint: checkpoint,
	}

	if checkpoint != nil {
		if strings.TrimSpace(resume.Branch) == "" {
			resume.Branch = strings.TrimSpace(checkpoint.Branch)
		}
		if strings.TrimSpace(resume.CWD) == "" {
			resume.CWD = strings.TrimSpace(checkpoint.CWD)
		}
	}
	if strings.TrimSpace(resume.CWD) == "" {
		resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
	}
	if order != nil && order.Tipo == "handoff" {
		var payload HandoffPayload
		if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err == nil {
			if strings.TrimSpace(resume.ExternalSessionID) == "" {
				resume.ExternalSessionID = strings.TrimSpace(payload.ExternalSessionID)
			}
			if strings.TrimSpace(resume.ResumenContinuidad) == "" {
				resume.ResumenContinuidad = strings.TrimSpace(payload.ResumenContinuidad)
			}
		}
	}
	if payload := construirResumePayloadBootstrapDB(resume.ResumePayloadJSON, order, mailbox, checkpoint); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen := construirResumenBootstrapDB(resume.ResumenContinuidad, order, mailbox, checkpoint); resumen != "" {
		resume.ResumenContinuidad = resumen
	}
	enriquecerResumeConContextoProyectoDB(&resume, strings.TrimSpace(agente), proyecto)
	if infoAgente, err := GetAgente(strings.TrimSpace(agente)); err == nil && infoAgente != nil {
		enriquecerResumeConGobernanzaDB(&resume, strings.TrimSpace(infoAgente.Rol), strings.TrimSpace(agente), proyecto)
	}
	for _, msg := range mailbox {
		if msg == nil || msg.ID == 0 {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(msg.ID); err != nil {
			return resume, bootstrap, err
		}
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return resume, bootstrap, err
		}
		bootstrap.Consumidos++
	}
	if order != nil {
		resultado, _ := json.Marshal(map[string]any{
			"ok":             true,
			"bootstrap":      true,
			"mailbox_count":  len(mailbox),
			"checkpoint_id":  checkpointIDOrZeroDB(checkpoint),
			"continuidad":    strings.TrimSpace(resume.ResumenContinuidad) != "",
			"resume_payload": strings.TrimSpace(resume.ResumePayloadJSON) != "",
		})
		if err := MarcarRuntimeOrderEstado(order.ID, "completada", string(resultado), ""); err != nil {
			return resume, bootstrap, err
		}
	}
	return resume, bootstrap, nil
}

func enriquecerResumeConContextoProyectoDB(resume *runtimeagente.ResumeContext, agente string, proyecto *Proyecto) {
	if resume == nil || proyecto == nil {
		return
	}
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" {
		return
	}
	contexto, resumen := BuildProjectContextSummary(agente, proyecto)
	if len(contexto) == 0 {
		return
	}
	if payload := AppendProjectContextPayload(resume.ResumePayloadJSON, contexto); payload != "" {
		resume.ResumePayloadJSON = payload
	}
	if resumen != "" && !strings.Contains(resume.ResumenContinuidad, resumen) {
		if strings.TrimSpace(resume.ResumenContinuidad) == "" {
			resume.ResumenContinuidad = resumen
		} else {
			resume.ResumenContinuidad += ". " + resumen
		}
	}
}

func claimBootstrapRuntimeOrderParaStart(agente string, proyectoID *int64, excludeOrderID int64) (*RuntimeOrder, error) {
	tipos := []string{"handoff", "resume"}
	placeholders, tipoArgs := runtimeOrderPlaceholders(tipos)
	args := make([]any, 0, len(tipoArgs)+4)
	args = append(args, strings.TrimSpace(agente))
	args = append(args, tipoArgs...)
	q := runtimeOrderSelectBase() + `
		WHERE agente = ?
		  AND estado = 'pendiente'
		  AND available_at <= CURRENT_TIMESTAMP
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
	order, err := scanRuntimeOrder(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return order, err
}

func construirResumePayloadBootstrapDB(prev string, order *RuntimeOrder, mailbox []*RuntimeMailboxMessage, checkpoint *RuntimeCheckpoint) string {
	prev = strings.TrimSpace(prev)
	envelope := map[string]any{}
	if prev != "" {
		var parsed any
		if err := json.Unmarshal([]byte(prev), &parsed); err == nil {
			envelope["resume_previo"] = parsed
		} else {
			envelope["resume_previo_raw"] = prev
		}
	}
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
			"resumen":         checkpoint.Resumen,
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
	data, err := json.Marshal(envelope)
	if err != nil {
		return prev
	}
	return string(data)
}

func construirResumenBootstrapDB(prev string, order *RuntimeOrder, mailbox []*RuntimeMailboxMessage, checkpoint *RuntimeCheckpoint) string {
	partes := make([]string, 0, 4)
	prev = strings.TrimSpace(prev)
	if prev != "" {
		partes = append(partes, prev)
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
	if checkpoint != nil && strings.TrimSpace(checkpoint.Resumen) != "" {
		partes = append(partes, "Checkpoint: "+strings.TrimSpace(checkpoint.Resumen))
	}
	if len(mailbox) > 0 {
		partes = append(partes, fmt.Sprintf("Mailbox: %d mensaje(s) inyectados", len(mailbox)))
	}
	return strings.Join(partes, ". ")
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
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &destino, ProyectoID: proyectoID, Estado: &estado})
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
		if strings.TrimSpace(payload.AgenteOrigen) != strings.TrimSpace(origen) || strings.TrimSpace(payload.AgenteDestino) != strings.TrimSpace(destino) {
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

func registrarEvidenciaHandoffReanudado(destino string, sesionID int64, bootstrap *bootstrapRuntimeData) error {
	if bootstrap == nil || bootstrap.Order == nil || bootstrap.Order.Tipo != "handoff" {
		return nil
	}
	var payload HandoffPayload
	if err := json.Unmarshal([]byte(bootstrap.Order.PayloadJSON), &payload); err != nil {
		return nil
	}
	if payload.TareaID != nil && *payload.TareaID > 0 {
		res, err := DB.Exec(`UPDATE tareas SET estado='en_progreso' WHERE id=? AND agente=? AND estado='asignada'`, *payload.TareaID, strings.TrimSpace(destino))
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			nota := fmt.Sprintf("handoff completado por %s en sesion #%d", strings.TrimSpace(destino), sesionID)
			if _, err := DB.Exec(
				`UPDATE tareas SET notas = notas || char(10) || ? || ' [' || datetime('now') || ' orquesta]' WHERE id=?`,
				nota, *payload.TareaID,
			); err != nil {
				return err
			}
		}
	}
	detalle := fmt.Sprintf("origen=%s destino=%s sesion_destino=%d order=%d", payload.AgenteOrigen, payload.AgenteDestino, sesionID, bootstrap.Order.ID)
	Audit(strings.TrimSpace(destino), "handoff_reanudado", "runtime_order", bootstrap.Order.ID, detalle)
	return nil
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
		if runtime, err := GetRuntimeBySesionID(sesionOrigen.ID); err == nil && runtime != nil {
			if _, err := DB.Exec(`
				UPDATE runtime_instances
				SET logical_state='pausado',
				    process_state=CASE WHEN pid IS NOT NULL THEN 'pausado' ELSE process_state END,
				    last_event_at=CURRENT_TIMESTAMP
				WHERE id=?`, runtime.ID); err != nil {
				return err
			}
		} else if err != nil {
			return err
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
	}
	detalle := fmt.Sprintf("origen=%s control_real=%t pid=%d", strings.TrimSpace(origen), aplicado, pid)
	var handleID int64
	if handleOrigen != nil {
		handleID = handleOrigen.ID
	}
	Audit(strings.TrimSpace(origen), "handoff_origen_pausado", "runtime_handle", handleID, detalle)
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

func claimRuntimeOrderByID(id int64) (*RuntimeOrder, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(`
		UPDATE runtime_orders
		SET estado = 'tomada', started_at = CURRENT_TIMESTAMP
		WHERE id = ? AND estado = 'pendiente' AND available_at <= CURRENT_TIMESTAMP`, id)
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
	return GetRuntimeOrder(id)
}

func resolverHandleParaOrden(order *RuntimeOrder) (*RuntimeHandle, error) {
	if order == nil {
		return nil, nil
	}
	if order.HandleID != nil {
		return GetRuntimeHandle(*order.HandleID)
	}
	if order.ProyectoID != nil {
		return GetRuntimeHandleActivoAgenteProyecto(order.Agente, order.ProyectoID)
	}
	return GetRuntimeHandleActivoAgente(order.Agente)
}

func resolverSesionParaOrden(order *RuntimeOrder) (*Sesion, error) {
	if order == nil {
		return nil, nil
	}
	if order.HandleID != nil {
		handle, err := GetRuntimeHandle(*order.HandleID)
		if err != nil {
			return nil, err
		}
		if handle != nil && handle.SesionID != nil {
			return GetSesionByID(*handle.SesionID)
		}
	}
	if order.RuntimeID != nil {
		runtime, err := GetRuntime(*order.RuntimeID)
		if err != nil {
			return nil, err
		}
		if runtime != nil && runtime.SesionID != nil {
			return GetSesionByID(*runtime.SesionID)
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
	if order.RuntimeID != nil {
		return GetRuntime(*order.RuntimeID)
	}
	if order.HandleID != nil {
		handle, err := GetRuntimeHandle(*order.HandleID)
		if err != nil || handle == nil {
			return nil, err
		}
		if handle.RuntimeID != nil {
			return GetRuntime(*handle.RuntimeID)
		}
		if handle.SesionID != nil {
			return GetRuntimeBySesionID(*handle.SesionID)
		}
	}
	sesion, err := GetSesionActiva(order.Agente, order.ProyectoID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if sesion != nil {
		runtime, err := GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime != nil {
			return runtime, err
		}
	}
	sesion, err = ObtenerUltimaSesion(order.Agente, order.ProyectoID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if sesion != nil {
		runtime, err := GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime != nil {
			return runtime, err
		}
	}
	return runtimePrincipalAgenteProyecto(order.Agente, order.ProyectoID)
}

func runtimePrincipalAgenteProyecto(agente string, proyectoID *int64) (*RuntimeInstance, error) {
	q := runtimeSelectBase() + ` WHERE r.agente = ?`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND r.proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY ` + runtimeActividadExpr("r") + ` DESC, r.id DESC LIMIT 1`
	runtime, err := escanearRuntime(DB.QueryRow(q, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func runtimeOrderTiposBasicos() []string {
	return []string{"sync_status", "checkpoint", "nudge", "discordia"}
}

func runtimeOrderTiposDespachables() []string {
	return []string{
		"sync_status",
		"checkpoint",
		"nudge",
		"discordia",
		"start",
		"pause",
		"resume",
		"stop",
		"restart",
		"send_instruction",
		"handoff",
	}
}

func runtimeOrderTiposBootstrap() []string {
	return []string{"handoff", "resume", "start"}
}

func runtimeOrderTipoDespachable(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "sync_status", "checkpoint", "nudge", "discordia", "start", "pause", "resume", "stop", "restart", "send_instruction", "handoff":
		return true
	default:
		return false
	}
}

func runtimeOrderPlaceholders(tipos []string) (string, []any) {
	placeholders := make([]string, 0, len(tipos))
	args := make([]any, 0, len(tipos))
	for _, tipo := range tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, tipo)
	}
	if len(placeholders) == 0 {
		return "''", args
	}
	return strings.Join(placeholders, ","), args
}

func scanRuntimeHandle(s scanner) (*RuntimeHandle, error) {
	var h RuntimeHandle
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var runtimeID sql.NullInt64
	var lastSeen sql.NullTime
	err := s.Scan(
		&h.ID, &h.Agente, &proyectoID, &sesionID, &runtimeID, &h.Transporte, &h.HandleKind, &h.HandleRef,
		&h.Estado, &h.LeaseToken, &h.CapabilitiesJSON, &h.MetadataJSON, &lastSeen, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		h.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		h.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		h.RuntimeID = &runtimeID.Int64
	}
	if lastSeen.Valid {
		h.LastSeenAt = &lastSeen.Time
	}
	return &h, nil
}

func scanRuntimeOrder(s scanner) (*RuntimeOrder, error) {
	var o RuntimeOrder
	var proyectoID sql.NullInt64
	var runtimeID sql.NullInt64
	var handleID sql.NullInt64
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	err := s.Scan(
		&o.ID, &o.Agente, &proyectoID, &runtimeID, &handleID, &o.Tipo, &o.PayloadJSON, &o.ResultadoJSON,
		&o.ErrorText, &o.Estado, &o.AvailableAt, &o.CreatedAt, &startedAt, &finishedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		o.ProyectoID = &proyectoID.Int64
	}
	if runtimeID.Valid {
		o.RuntimeID = &runtimeID.Int64
	}
	if handleID.Valid {
		o.HandleID = &handleID.Int64
	}
	if startedAt.Valid {
		o.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		o.FinishedAt = &finishedAt.Time
	}
	return &o, nil
}

func scanRuntimeMailbox(s scanner) (*RuntimeMailboxMessage, error) {
	var msg RuntimeMailboxMessage
	var proyectoID sql.NullInt64
	var orderID sql.NullInt64
	var deliveredAt sql.NullTime
	var consumedAt sql.NullTime
	err := s.Scan(
		&msg.ID, &msg.FromAgente, &msg.ToAgente, &proyectoID, &orderID, &msg.Kind, &msg.PayloadJSON,
		&msg.Estado, &msg.CreatedAt, &deliveredAt, &consumedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		msg.ProyectoID = &proyectoID.Int64
	}
	if orderID.Valid {
		msg.RuntimeOrderID = &orderID.Int64
	}
	if deliveredAt.Valid {
		msg.DeliveredAt = &deliveredAt.Time
	}
	if consumedAt.Valid {
		msg.ConsumedAt = &consumedAt.Time
	}
	return &msg, nil
}

func scanRuntimeCheckpoint(s scanner) (*RuntimeCheckpoint, error) {
	var cp RuntimeCheckpoint
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var runtimeID sql.NullInt64
	err := s.Scan(
		&cp.ID, &cp.Agente, &proyectoID, &sesionID, &runtimeID, &cp.CheckpointKind, &cp.Resumen,
		&cp.Branch, &cp.CWD, &cp.PayloadJSON, &cp.ResumeStrategy, &cp.Source, &cp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		cp.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		cp.SesionID = &sesionID.Int64
	}
	if runtimeID.Valid {
		cp.RuntimeID = &runtimeID.Int64
	}
	return &cp, nil
}

func inferirTransporteSesion(s *Sesion) string {
	if s == nil {
		return "cli"
	}
	if strings.TrimSpace(s.ConectorSlug) != "" {
		if conector, err := GetConector(s.ConectorSlug); err == nil && strings.TrimSpace(conector.Transporte) != "" {
			return conector.Transporte
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(s.Herramienta)), "mcp") {
		return "mcp_stdio"
	}
	return "cli"
}

func inferirHandleKindSesion(s *Sesion) string {
	if s == nil {
		return "session"
	}
	if s.PID != nil && *s.PID > 0 {
		return "process"
	}
	if strings.TrimSpace(s.ExternalSessionID) != "" {
		return "session"
	}
	return "session"
}

func inferirHandleRefSesion(s *Sesion) string {
	if s == nil {
		return ""
	}
	if s.PID != nil && *s.PID > 0 {
		return strings.TrimSpace(jsonNumber(*s.PID))
	}
	if strings.TrimSpace(s.ExternalSessionID) != "" {
		return strings.TrimSpace(s.ExternalSessionID)
	}
	return strings.TrimSpace(jsonNumber(s.ID))
}

func inferirEstadoHandleSesion(s *Sesion) string {
	if s == nil {
		return "fallido"
	}
	if s.Fin != nil || !s.Activa || s.Estado == "cerrada" || s.Estado == "fallida" {
		return "cerrado"
	}
	if s.Estado == "pausada" {
		return "pausado"
	}
	return "activo"
}

func inferirCapabilitiesSesion(s *Sesion) string {
	caps := map[string]any{
		"can_send_input":       true,
		"can_checkpoint":       true,
		"can_resume":           strings.TrimSpace(s.ExternalSessionID) != "",
		"can_capture_pid":      s.PID != nil && *s.PID > 0,
		"can_track_continuity": true,
	}
	data, _ := json.Marshal(caps)
	return string(data)
}

func inferirMetadataSesion(s *Sesion) string {
	if s == nil {
		return "{}"
	}
	meta := map[string]any{
		"herramienta":         strings.TrimSpace(s.Herramienta),
		"external_session_id": strings.TrimSpace(s.ExternalSessionID),
		"branch":              strings.TrimSpace(s.Branch),
		"cwd":                 strings.TrimSpace(s.CWD),
	}
	data, _ := json.Marshal(meta)
	return string(data)
}

func defaultRuntimeOrderEstado(v string) string {
	switch strings.TrimSpace(v) {
	case "tomada", "ejecutando", "completada", "fallida", "expirada", "cancelada":
		return strings.TrimSpace(v)
	default:
		return "pendiente"
	}
}

func defaultMailboxEstado(v string) string {
	switch strings.TrimSpace(v) {
	case "entregado", "consumido", "expirado", "cancelado":
		return strings.TrimSpace(v)
	default:
		return "pendiente"
	}
}

func mapFromJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func boolFromMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "si", "sí", "on":
			return true
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	}
	return false
}

func intFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

func stringFromMap(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	raw, ok := m[key]
	if !ok {
		return fallback
	}
	v, ok := raw.(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func runtimeHandlePreservesExternalSession(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if boolFromMap(meta, "preserve_external_session_on_stop") || boolFromMap(meta, "requires_human_reauth") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "auth_mode", "")), "oauth") {
		return true
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if _, ok := caps["can_stop_without_reauth"]; ok && !boolFromMap(caps, "can_stop_without_reauth") {
		return true
	}
	return false
}

func asegurarCheckpointCambioContexto(order *RuntimeOrder, suffix, fallbackSummary string) (int64, error) {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return 0, err
	}
	if sesion == nil {
		return 0, nil
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return 0, err
	}
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(order.PayloadJSON), &payload)
	source := fmt.Sprintf("runtime_order:%d:%s", order.ID, strings.TrimSpace(suffix))
	existente, err := GetRuntimeCheckpointBySource(source)
	if err != nil {
		return 0, err
	}
	if existente != nil {
		return existente.ID, nil
	}
	resumen := stringFromMap(payload, "resumen", "")
	if strings.TrimSpace(resumen) == "" {
		resumen = stringFromMap(payload, "motivo", "")
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = strings.TrimSpace(sesion.ResumenContinuidad)
	}
	if strings.TrimSpace(resumen) == "" {
		resumen = fallbackSummary
	}
	cp := &RuntimeCheckpoint{
		Agente:         order.Agente,
		ProyectoID:     order.ProyectoID,
		SesionID:       &sesion.ID,
		CheckpointKind: suffix,
		Resumen:        resumen,
		Branch:         sesion.Branch,
		CWD:            sesion.CWD,
		PayloadJSON:    order.PayloadJSON,
		ResumeStrategy: "resumen_y_payload",
		Source:         source,
	}
	if runtime != nil {
		cp.RuntimeID = &runtime.ID
	}
	return CrearRuntimeCheckpoint(cp)
}

func aparcarSesionActiva(agente string, proyectoID *int64) error {
	q := `UPDATE sesiones SET activa=0, estado='pausada', heartbeat_at=CURRENT_TIMESTAMP WHERE agente=? AND activa=1`
	args := []any{agente}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	if _, err := DB.Exec(q, args...); err != nil {
		return err
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=0, estado_sesion='disponible' WHERE nombre=?`, agente); err != nil {
		return err
	}
	Audit(agente, "aparcar_sesion", "sesion", 0, "")
	if proyectoID != nil {
		EmitirHookCicloVida(agente, HookSessionPark, *proyectoID, "sesion", 0, agente, "aparcar_sesion")
	}
	return nil
}

func AparcarSesionActiva(agente string, proyectoID *int64) error {
	if err := aparcarSesionActiva(agente, proyectoID); err != nil {
		return err
	}
	return MarcarRuntimeHandlesCerradosPorAgente(strings.TrimSpace(agente))
}

func actualizarEstadoSesionParaOrden(order *RuntimeOrder, estado string) error {
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return err
	}
	if sesion == nil || sesion.ID == 0 {
		return nil
	}
	if _, err := DB.Exec(`UPDATE sesiones SET estado=? WHERE id=?`, strings.TrimSpace(estado), sesion.ID); err != nil {
		return err
	}
	return nil
}

func preferTime(values ...*time.Time) *time.Time {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func timePtr(t time.Time) *time.Time { return &t }

func inferirLastSeenSesion(s *Sesion) *time.Time {
	if s == nil {
		return nil
	}
	now := time.Now().UTC()
	if estado := inferirEstadoHandleSesion(s); estado == "activo" || estado == "pausado" {
		return &now
	}
	return preferTime(s.HeartbeatAt, timePtr(s.Inicio), &now)
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func jsonNumber[T ~int64](v T) string {
	data, _ := json.Marshal(v)
	return string(data)
}
