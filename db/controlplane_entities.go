package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		return nil, nil
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
		return nil, nil
	}
	return h, err
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
	for _, id := range ids {
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
			WHERE id = ?`, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
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
		if runtimeOrderTipoBasico(item.Tipo) {
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
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	tipos := runtimeOrderTiposBasicos()
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
		"ok":          true,
		"handle_id":   nil,
		"runtime_id":  nil,
		"handle":      nil,
		"runtime":     nil,
		"sin_handle":  handle == nil,
		"sin_runtime": runtime == nil,
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

func ejecutarRuntimeOrderStart(order *RuntimeOrder) error {
	runtime, err := actualizarEstadoRuntime(order, "activo", "corriendo")
	if err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderPause(order *RuntimeOrder) error {
	runtime, err := actualizarEstadoRuntime(order, "pausado", "")
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='pausado', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado='activo'`, order.Agente); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderResume(order *RuntimeOrder) error {
	runtime, err := actualizarEstadoRuntime(order, "activo", "corriendo")
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='activo', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado='pausado'`, order.Agente); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func ejecutarRuntimeOrderStop(order *RuntimeOrder) error {
	runtime, err := actualizarEstadoRuntime(order, "cerrado", "finalizado")
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado='cerrado', last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado')`, order.Agente); err != nil {
		return err
	}
	result := map[string]any{
		"ok":            true,
		"runtime_id":    runtime.ID,
		"logical_state": runtime.LogicalState,
		"process_state": runtime.ProcessState,
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

func runtimeOrderTiposBootstrap() []string {
	return []string{"handoff", "resume", "start"}
}

func runtimeOrderTipoBasico(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "sync_status", "checkpoint", "nudge", "discordia":
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
