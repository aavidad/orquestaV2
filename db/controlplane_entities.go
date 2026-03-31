package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
	"os"
	"path/filepath"
	"regexp"
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
	Agente     *string
	ProyectoID *int64
	Estados    []string
	Tipos      []string
}

type PurgaRuntimeOrdersResultado struct {
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
	Tipos      []string `json:"tipos,omitempty"`
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

func reconciliarHandleKindSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if existente == nil {
		if inferido == nil {
			return ""
		}
		return strings.TrimSpace(inferido.HandleKind)
	}
	if strings.TrimSpace(existente.HandleKind) != "" {
		return strings.TrimSpace(existente.HandleKind)
	}
	if inferido == nil {
		return ""
	}
	return strings.TrimSpace(inferido.HandleKind)
}

func reconciliarHandleRefSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if existente != nil && strings.TrimSpace(existente.HandleRef) != "" {
		return strings.TrimSpace(existente.HandleRef)
	}
	if inferido == nil {
		return ""
	}
	return strings.TrimSpace(inferido.HandleRef)
}

func reconciliarMetadataSesion(existenteJSON, inferidoJSON string) string {
	existente := mapFromJSON(existenteJSON)
	if existente == nil {
		existente = map[string]any{}
	}
	inferido := mapFromJSON(inferidoJSON)
	if inferido == nil {
		inferido = map[string]any{}
	}
	for _, key := range []string{"herramienta", "external_session_id", "branch", "cwd"} {
		if val := strings.TrimSpace(stringFromMap(inferido, key, "")); val != "" {
			existente[key] = val
		}
	}
	data, _ := json.Marshal(existente)
	return string(data)
}

func reconciliarCapabilitiesSesion(existenteJSON, inferidoJSON string) string {
	existente := mapFromJSON(existenteJSON)
	if existente == nil {
		existente = map[string]any{}
	}
	inferido := mapFromJSON(inferidoJSON)
	if inferido == nil {
		inferido = map[string]any{}
	}
	for key, value := range inferido {
		if _, ok := existente[key]; ok {
			continue
		}
		existente[key] = value
	}
	data, _ := json.Marshal(existente)
	return string(data)
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

func PurgarRuntimeHandlesInactivos(filtro FiltroPurgadoRuntimeHandles) (*PurgaRuntimeHandlesResultado, error) {
	if filtro.Agente == nil && filtro.ProyectoID == nil {
		return nil, fmt.Errorf("debes indicar agente o proyecto para purgar runtime handles")
	}
	estados, err := normalizarEstadosPurgadoRuntimeHandles(filtro.Estados)
	if err != nil {
		return nil, err
	}
	query := `SELECT id FROM runtime_handles WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)`
	args := make([]any, 0, len(estados)+2)
	for _, estado := range estados {
		args = append(args, estado)
	}
	if filtro.Agente != nil && strings.TrimSpace(*filtro.Agente) != "" {
		query += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	if filtro.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	query += ` ORDER BY id DESC`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &PurgaRuntimeHandlesResultado{Estados: estados}, nil
	}
	if err := validarPurgadoRuntimeHandles(ids); err != nil {
		return nil, err
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	argsDelete := int64SliceToAny(ids)
	if _, err := tx.Exec(`UPDATE runtime_orders SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE runtime_transcript SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM runtime_handles WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &PurgaRuntimeHandlesResultado{
		Deleted:    len(ids),
		DeletedIDs: ids,
		Estados:    estados,
	}, nil
}

func PurgarRuntimeOrdersTerminales(filtro FiltroPurgadoRuntimeOrders) (*PurgaRuntimeOrdersResultado, error) {
	if filtro.Agente == nil && filtro.ProyectoID == nil {
		return nil, fmt.Errorf("debes indicar agente o proyecto para purgar runtime orders")
	}
	estados, err := normalizarEstadosPurgadoRuntimeOrders(filtro.Estados)
	if err != nil {
		return nil, err
	}
	tipos, err := normalizarTiposPurgadoRuntimeOrders(filtro.Tipos)
	if err != nil {
		return nil, err
	}
	query := `SELECT id FROM runtime_orders WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)`
	args := make([]any, 0, len(estados)+4)
	for _, estado := range estados {
		args = append(args, estado)
	}
	if filtro.Agente != nil && strings.TrimSpace(*filtro.Agente) != "" {
		query += ` AND agente = ?`
		args = append(args, strings.TrimSpace(*filtro.Agente))
	}
	if filtro.ProyectoID != nil {
		query += ` AND proyecto_id = ?`
		args = append(args, *filtro.ProyectoID)
	}
	if len(tipos) > 0 {
		query += ` AND tipo IN (` + runtimeSQLPlaceholders(len(tipos)) + `)`
		for _, tipo := range tipos {
			args = append(args, tipo)
		}
	}
	query += ` ORDER BY id DESC`

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &PurgaRuntimeOrdersResultado{Estados: estados, Tipos: tipos}, nil
	}
	if err := validarPurgadoRuntimeOrders(ids); err != nil {
		return nil, err
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	argsDelete := int64SliceToAny(ids)
	if _, err := tx.Exec(`UPDATE runtime_mailbox SET runtime_order_id = NULL WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM runtime_orders WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)`, argsDelete...); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &PurgaRuntimeOrdersResultado{
		Deleted:    len(ids),
		DeletedIDs: ids,
		Estados:    estados,
		Tipos:      tipos,
	}, nil
}

func normalizarEstadosPurgadoRuntimeHandles(estados []string) ([]string, error) {
	if len(estados) == 0 {
		return []string{"cerrado", "fallido"}, nil
	}
	seen := make(map[string]struct{}, len(estados))
	out := make([]string, 0, len(estados))
	for _, raw := range estados {
		estado := strings.ToLower(strings.TrimSpace(raw))
		switch estado {
		case "cerrado", "fallido":
		case "activo", "pausado":
			return nil, fmt.Errorf("no se permite purgar handles en estado %q; deten el agente primero", estado)
		default:
			return nil, fmt.Errorf("estado de runtime handle no soportado: %s", strings.TrimSpace(raw))
		}
		if _, ok := seen[estado]; ok {
			continue
		}
		seen[estado] = struct{}{}
		out = append(out, estado)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("debes indicar al menos un estado purgable")
	}
	return out, nil
}

func normalizarEstadosPurgadoRuntimeOrders(estados []string) ([]string, error) {
	if len(estados) == 0 {
		return []string{"completada", "fallida", "expirada", "cancelada"}, nil
	}
	seen := make(map[string]struct{}, len(estados))
	out := make([]string, 0, len(estados))
	for _, raw := range estados {
		estado := strings.ToLower(strings.TrimSpace(raw))
		switch estado {
		case "completada", "fallida", "expirada", "cancelada":
		case "pendiente", "tomada", "ejecutando":
			return nil, fmt.Errorf("no se permite purgar runtime orders en estado %q; espera a que terminen o cancelalas primero", estado)
		default:
			return nil, fmt.Errorf("estado de runtime order no soportado: %s", strings.TrimSpace(raw))
		}
		if _, ok := seen[estado]; ok {
			continue
		}
		seen[estado] = struct{}{}
		out = append(out, estado)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("debes indicar al menos un estado purgable")
	}
	return out, nil
}

func normalizarTiposPurgadoRuntimeOrders(tipos []string) ([]string, error) {
	if len(tipos) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(tipos))
	out := make([]string, 0, len(tipos))
	for _, raw := range tipos {
		tipo := strings.TrimSpace(raw)
		if tipo == "" {
			continue
		}
		if !runtimeOrderTipoDespachable(tipo) && tipo != "handoff" {
			return nil, fmt.Errorf("tipo de runtime order no soportado para purga: %s", tipo)
		}
		if _, ok := seen[tipo]; ok {
			continue
		}
		seen[tipo] = struct{}{}
		out = append(out, tipo)
	}
	return out, nil
}

func validarPurgadoRuntimeHandles(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := DB.Query(`
		SELECT id, estado FROM runtime_orders
		WHERE handle_id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND estado IN ('pendiente','tomada','ejecutando')`,
		int64SliceToAny(ids)...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vivos []string
	for rows.Next() {
		var (
			id     int64
			estado string
		)
		if err := rows.Scan(&id, &estado); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar handles con runtime orders vivas asociadas: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func validarPurgadoRuntimeOrders(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := DB.Query(`
		SELECT id, estado FROM runtime_orders
		WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND estado IN ('pendiente','tomada','ejecutando')`,
		int64SliceToAny(ids)...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vivos []string
	for rows.Next() {
		var (
			id     int64
			estado string
		)
		if err := rows.Scan(&id, &estado); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime orders vivas: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func int64SliceToAny(ids []int64) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}

func runtimeSQLPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func GetRuntimeHandleBySesionID(sesionID int64) (*RuntimeHandle, error) {
	return consultarConReintentos(func() (*RuntimeHandle, error) {
		row := DB.QueryRow(runtimeHandleSelectBase()+` WHERE sesion_id = ? ORDER BY id DESC LIMIT 1`, sesionID)
		h, err := scanRuntimeHandle(row)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return h, err
	})
}

func GetRuntimeHandle(id int64) (*RuntimeHandle, error) {
	return consultarConReintentos(func() (*RuntimeHandle, error) {
		row := DB.QueryRow(runtimeHandleSelectBase()+` WHERE id = ?`, id)
		h, err := scanRuntimeHandle(row)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return h, err
	})
}

func GetRuntimeHandleActivoAgente(agente string) (*RuntimeHandle, error) {
	return seleccionarRuntimeHandleActivo(strings.TrimSpace(agente), nil)
}

func GetRuntimeHandleActivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	return seleccionarRuntimeHandleActivo(strings.TrimSpace(agente), proyectoID)
}

func seleccionarRuntimeHandleActivo(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	handles, err := listarRuntimeHandlesActivosCandidatos(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		validado, err := validarRuntimeHandleActivoSinFallback(handle)
		if err != nil {
			return nil, err
		}
		if validado != nil {
			return validado, nil
		}
	}
	return recuperarRuntimeHandleVivoAgenteProyecto(agente, proyectoID)
}

func listarRuntimeHandlesActivosCandidatos(agente string, proyectoID *int64) ([]*RuntimeHandle, error) {
	q := runtimeHandleSelectBase() + `
		WHERE agente = ? AND estado IN ('activo','pausado')`
	args := []any{strings.TrimSpace(agente)}
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	q += ` ORDER BY COALESCE(last_seen_at, updated_at, created_at) DESC, id DESC`
	return consultarConReintentos(func() ([]*RuntimeHandle, error) {
		rows, err := DB.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var handles []*RuntimeHandle
		for rows.Next() {
			handle, err := scanRuntimeHandle(rows)
			if err != nil {
				return nil, err
			}
			handles = append(handles, handle)
		}
		return handles, rows.Err()
	})
}

func validarRuntimeHandleActivo(handle *RuntimeHandle, agente string, proyectoID *int64) (*RuntimeHandle, error) {
	validado, err := validarRuntimeHandleActivoSinFallback(handle)
	if err != nil || validado != nil {
		return validado, err
	}
	return recuperarRuntimeHandleVivoAgenteProyecto(agente, proyectoID)
}

func validarRuntimeHandleActivoSinFallback(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	if !runtimeHandleSePuedeValidarLocalmente(handle) {
		return handle, nil
	}
	revived, err := refrescarRuntimeHandleSiSigueVivo(handle)
	if err != nil {
		return nil, err
	}
	if revived != nil {
		return revived, nil
	}
	if err := marcarRuntimeHandleFantasma(handle); err != nil {
		return nil, err
	}
	return nil, nil
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
	handle, err := consultarConReintentos(func() (*RuntimeHandle, error) {
		item, err := scanRuntimeHandle(DB.QueryRow(q, args...))
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return item, err
	})
	if err != nil || handle == nil {
		return handle, err
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

func runtimeHandleSePuedeValidarLocalmente(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return true
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil || runtime.PID == nil {
		return false
	}
	return *runtime.PID > 0
}

func marcarRuntimeHandleFantasma(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
		WHERE id = ?`, handle.ID); err != nil {
		return err
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE id = ?`, runtime.ID); err != nil {
		return err
	}
	return nil
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
	payloadJSON, err := normalizarRuntimeOrderPayloadJSON(order.PayloadJSON)
	if err != nil {
		return 0, err
	}
	order.PayloadJSON = payloadJSON
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
	handleOrigen, err := GetRuntimeHandleActivoAgente(origen)
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
		if err := asegurarEntregaBootstrapHandoffDestino(existenteID, proyectoID, destinoHandle); err != nil {
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
	if err := asegurarEntregaBootstrapHandoffDestino(orderID, proyectoID, destinoHandle); err != nil {
		return orderID, err
	}
	return orderID, nil
}

func asegurarEntregaBootstrapHandoffDestino(orderID int64, proyectoID *int64, destinoHandle *RuntimeHandle) error {
	if orderID <= 0 || !runtimeHandlePermiteReinicioBootstrapHandoff(destinoHandle) {
		return nil
	}
	targetProjectID := proyectoID
	if targetProjectID == nil && destinoHandle != nil && destinoHandle.ProyectoID != nil {
		targetProjectID = destinoHandle.ProyectoID
	}
	if targetProjectID == nil || *targetProjectID <= 0 {
		return nil
	}
	if abierta, err := existeRuntimeOrderAbiertaDestino(*targetProjectID, strings.TrimSpace(destinoHandle.Agente), "stop", "start", "restart", "resume"); err != nil {
		return err
	} else if abierta {
		return nil
	}
	proyecto, err := GetProyecto(strconv.FormatInt(*targetProjectID, 10))
	if err != nil {
		return err
	}
	motivo := fmt.Sprintf("handoff_bootstrap:%d", orderID)
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
		Agente:      strings.TrimSpace(destinoHandle.Agente),
		ProyectoID:  targetProjectID,
		Tipo:        "stop",
		PayloadJSON: string(stopPayload),
		HandleID:    &destinoHandle.ID,
	}
	if destinoHandle.RuntimeID != nil && *destinoHandle.RuntimeID > 0 {
		stopOrder.RuntimeID = destinoHandle.RuntimeID
	}
	stopOrderID, err := EncolarRuntimeOrder(stopOrder)
	if err != nil {
		return err
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
		Agente:      strings.TrimSpace(destinoHandle.Agente),
		ProyectoID:  targetProjectID,
		Tipo:        "start",
		PayloadJSON: string(startPayload),
	})
	if err != nil {
		return err
	}
	Audit("orquesta", "handoff_destino_reinicio_bootstrap", "runtime_order", orderID,
		fmt.Sprintf("agente=%s proyecto=%s stop_order_id=%d start_order_id=%d", strings.TrimSpace(destinoHandle.Agente), strings.TrimSpace(proyecto.Slug), stopOrderID, startOrderID))
	return nil
}

func runtimeHandlePermiteReinicioBootstrapHandoff(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.Transporte) != "cli" || strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	if runtimeHandlePreservesExternalSession(handle) {
		return false
	}
	return true
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
		})
		if err != nil {
			return false, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			for _, tipo := range tipos {
				if strings.TrimSpace(order.Tipo) == strings.TrimSpace(tipo) {
					return true, nil
				}
			}
		}
	}
	return false, nil
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

func PeekNextBootstrapRuntimeOrder(agente string, proyectoID *int64) (*RuntimeOrder, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, nil
	}
	id, err := nextBootstrapRuntimeOrderID(agente, proyectoID)
	if err != nil || id == 0 {
		return nil, err
	}
	return GetRuntimeOrder(id)
}

func nextBootstrapRuntimeOrderID(agente string, proyectoID *int64) (int64, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, nil
	}
	tipos := runtimeOrderTiposBootstrap()
	placeholders, tipoArgs := runtimeOrderPlaceholders(tipos)
	argsBase := make([]any, 0, len(tipoArgs)+3)
	argsBase = append(argsBase, agente)
	argsBase = append(argsBase, tipoArgs...)
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
		return 0, nil
	}
	return id, err
}

func GetRuntimeOrder(id int64) (*RuntimeOrder, error) {
	return consultarConReintentos(func() (*RuntimeOrder, error) {
		row := DB.QueryRow(runtimeOrderSelectBase()+` WHERE id = ?`, id)
		return scanRuntimeOrder(row)
	})
}

func ListarRuntimeOrders(filter FiltroRuntimeOrders) ([]*RuntimeOrder, error) {
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
	})
}

func MarcarRuntimeOrderEstado(id int64, estado, resultadoJSON, errorText string) error {
	if strings.TrimSpace(resultadoJSON) == "" {
		resultadoJSON = "{}"
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
		return err
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
		return err
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
		return err
	default:
		_, err := DB.Exec(`
			UPDATE runtime_orders
			SET estado = ?,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`, estado, id)
		return err
	}
}

func MarcarRuntimeOrderEjecutando(order *RuntimeOrder) error {
	if order == nil {
		return sql.ErrNoRows
	}
	resultadoJSON := strings.TrimSpace(order.ResultadoJSON)
	if resultadoJSON == "" {
		resultadoJSON = "{}"
	}
	leaseUntil := runtimeOrderLeaseDeadlineForType(order.Tipo, time.Now().UTC())
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='ejecutando',
		    resultado_json = ?,
		    error_text = '',
		    lease_expires_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, resultadoJSON, leaseUntil, order.ID)
	return err
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
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if runtimeMailboxKindSupersedible(strings.TrimSpace(msg.Kind)) {
		if _, err := ConsumirRuntimeMailboxPendienteSupersedido(strings.TrimSpace(msg.ToAgente), msg.ProyectoID, strings.TrimSpace(msg.Kind), id); err != nil {
			return 0, err
		}
	}
	return id, nil
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

func ConsumirRuntimeMailboxPendienteSupersedido(toAgente string, proyectoID *int64, kind string, keepID int64) (int64, error) {
	toAgente = strings.TrimSpace(toAgente)
	kind = strings.TrimSpace(kind)
	if toAgente == "" || kind == "" || keepID <= 0 {
		return 0, nil
	}
	args := []any{toAgente}
	q := `
		UPDATE runtime_mailbox
		SET estado='consumido',
		    delivered_at=COALESCE(delivered_at, CURRENT_TIMESTAMP),
		    consumed_at=CURRENT_TIMESTAMP
		WHERE estado='pendiente'
		  AND to_agente = ?`
	if family := runtimeMailboxSupersedeFamily(kind); family != "" {
		q += ` AND kind IN (` + runtimeMailboxFamilyPlaceholders(family) + `)`
		args = append(args, runtimeMailboxFamilyKinds(family)...)
	} else {
		q += ` AND kind = ?`
		args = append(args, kind)
	}
	q += ` AND id < ?`
	args = append(args, keepID)
	if proyectoID != nil {
		q += ` AND proyecto_id = ?`
		args = append(args, *proyectoID)
	}
	res, err := DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

func runtimeMailboxSupersedeFamily(kind string) string {
	switch strings.TrimSpace(kind) {
	case "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return "guidance"
	default:
		return ""
	}
}

func runtimeMailboxFamilyKinds(family string) []any {
	switch strings.TrimSpace(family) {
	case "guidance":
		return []any{"autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh}
	default:
		return nil
	}
}

func runtimeMailboxFamilyPlaceholders(family string) string {
	kinds := runtimeMailboxFamilyKinds(family)
	if len(kinds) == 0 {
		return "?"
	}
	return runtimeSQLPlaceholders(len(kinds))
}

func runtimeMailboxKindSupersedible(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "instruction", "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
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
		res, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
			WHERE id = ?
			  AND estado IN ('activo','pausado')`, id)
		if err != nil {
			return 0, err
		}
		rows, _ := res.RowsAffected()
		if rows > 0 {
			reconciled++
		}
	}
	return reconciled, nil
}

func ReconciliarRuntimeOrdersStale() (int, error) {
	now := time.Now().UTC()
	staleSeconds := configIntOrDefault("runtime_order_stale_seconds", 120)
	if staleSeconds <= 0 {
		staleSeconds = 120
	}
	cutoff := now.Add(-time.Duration(staleSeconds) * time.Second)
	rows, err := DB.Query(`
		SELECT id, tipo
		FROM runtime_orders
		WHERE estado IN ('tomada','ejecutando')
		  AND (
		        (lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
		     OR (lease_expires_at IS NULL AND COALESCE(started_at, updated_at, created_at) <= ?)
		  )
		ORDER BY id`, now, cutoff)
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
		applied, err := reconciliarRuntimeOrderStale(item.ID, item.Tipo, now, cutoff)
		if err != nil {
			return recovered, err
		}
		if applied {
			recovered++
		}
	}
	return recovered, nil
}

func ProcesarRuntimeOrdersBatch() (int, error) {
	processed, err := procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDespachables())
	if err != nil {
		return processed, err
	}
	promoted, err := procesarBootstrapRuntimeOrdersActivosBatch()
	return processed + promoted, err
}

func ProcesarRuntimeSupervisionBatch() (int, error) {
	limit := configIntOrDefault("runtime_supervision_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	intervalSeconds := configIntOrDefault("runtime_supervision_interval_seconds", 30)
	if intervalSeconds <= 0 {
		intervalSeconds = 30
	}
	cutoff := time.Now().UTC().Add(-time.Duration(intervalSeconds) * time.Second)

	rows, err := DB.Query(runtimeHandleSelectBase()+`
		WHERE estado IN ('activo','pausado')
		  AND COALESCE(last_seen_at, updated_at, created_at) <= ?
		ORDER BY COALESCE(last_seen_at, updated_at, created_at), id
		LIMIT ?`, cutoff, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	handles := make([]*RuntimeHandle, 0, limit)
	for rows.Next() {
		handle, err := scanRuntimeHandle(rows)
		if err != nil {
			return 0, err
		}
		if handle != nil {
			handles = append(handles, handle)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaAgenteProyecto(strings.TrimSpace(handle.Agente), handle.ProyectoID, 0,
			"sync_status", "start", "resume", "pause", "stop", "restart", "handoff"); err != nil {
			return processed, err
		} else if abierta {
			continue
		}
		var runtime *RuntimeInstance
		if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
			runtime, err = GetRuntime(*handle.RuntimeID)
			if err != nil {
				return processed, err
			}
		} else if handle.SesionID != nil && *handle.SesionID > 0 {
			runtime, err = GetRuntimeBySesionID(*handle.SesionID)
			if err != nil {
				return processed, err
			}
		}
		if _, err := supervisarRuntimeHandle(handle, runtime, "runtime_supervision"); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
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
		if err := MarcarRuntimeOrderEjecutando(order); err != nil {
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

func procesarBootstrapRuntimeOrdersActivosBatch() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	placeholders, args := runtimeOrderPlaceholders([]string{"handoff", "resume"})
	args = append(args, limit)
	rows, err := DB.Query(runtimeOrderSelectBase()+`
		WHERE estado = 'pendiente'
		  AND available_at <= CURRENT_TIMESTAMP
		  AND tipo IN (`+placeholders+`)
		ORDER BY id
		LIMIT ?`, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var orders []*RuntimeOrder
	for rows.Next() {
		order, err := scanRuntimeOrder(rows)
		if err != nil {
			return 0, err
		}
		if order != nil {
			orders = append(orders, order)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	processed := 0
	for _, order := range orders {
		handle, err := runtimeHandleActivoParaBootstrapOrder(order)
		if err != nil {
			return processed, err
		}
		if handle == nil {
			continue
		}
		if abierta, err := existeRuntimeOrderAbiertaAgenteProyecto(order.Agente, order.ProyectoID, order.ID, "stop", "start", "restart"); err != nil {
			return processed, err
		} else if abierta {
			continue
		}
		motivo := fmt.Sprintf("runtime_bootstrap_coordinated_restart:%s:%d", strings.TrimSpace(order.Tipo), order.ID)
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

func runtimeHandleActivoParaBootstrapOrder(order *RuntimeOrder) (*RuntimeHandle, error) {
	if order == nil {
		return nil, nil
	}
	if order.ProyectoID != nil {
		return GetRuntimeHandleActivoAgenteProyecto(strings.TrimSpace(order.Agente), order.ProyectoID)
	}
	return GetRuntimeHandleActivoAgente(strings.TrimSpace(order.Agente))
}

func existeRuntimeOrderAbiertaAgenteProyecto(agente string, proyectoID *int64, excludeID int64, tipos ...string) (bool, error) {
	if strings.TrimSpace(agente) == "" || len(tipos) == 0 {
		return false, nil
	}
	placeholders, args := runtimeOrderPlaceholders(tipos)
	args = append([]any{strings.TrimSpace(agente), excludeID}, args...)
	q := `
		SELECT COUNT(*)
		FROM runtime_orders
		WHERE agente = ?
		  AND id <> ?
		  AND estado IN ('pendiente','tomada','ejecutando')
		  AND tipo IN (` + placeholders + `)`
	if proyectoID != nil {
		q += ` AND (proyecto_id = ? OR proyecto_id IS NULL)`
		args = append(args, *proyectoID)
	}
	var n int
	if err := DB.QueryRow(q, args...).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
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
		sesion, err := GetSesionByID(*handle.SesionID)
		if err != nil && err != sql.ErrNoRows {
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
	if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
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
	result, err := supervisarRuntimeHandle(handle, runtime, "runtime_sync_status")
	if err != nil {
		return err
	}
	data, _ := json.Marshal(result)
	return MarcarRuntimeOrderEstado(order.ID, "completada", string(data), "")
}

func supervisarRuntimeHandle(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (map[string]any, error) {
	result := map[string]any{
		"ok":               true,
		"handle_id":        nil,
		"runtime_id":       nil,
		"handle":           nil,
		"runtime":          nil,
		"sin_handle":       handle == nil,
		"sin_runtime":      runtime == nil,
		"observed_remote":  false,
		"observed_process": false,
	}
	if handle != nil {
		estadoRemoto, observedRemote, err := controlruntime.ConsultarEstadoRemoto(controlruntime.ObjetivoProceso{
			HandleKind:   handle.HandleKind,
			HandleRef:    handle.HandleRef,
			MetadataJSON: handle.MetadataJSON,
		})
		if err != nil {
			failures, degraded, obsErr := registrarFalloObservacionRemota(handle, runtime, err)
			if obsErr != nil {
				return nil, obsErr
			}
			result["ok"] = false
			result["observed_remote_error"] = true
			result["remote_sync_failures"] = failures
			result["remote_degraded"] = degraded
			result["remote_error"] = strings.TrimSpace(err.Error())
		} else if observedRemote && estadoRemoto != nil {
			if err := aplicarEstadoRemotoObservado(handle, runtime, estadoRemoto); err != nil {
				return nil, err
			}
			result["observed_remote"] = true
			if strings.TrimSpace(estadoRemoto.RawJSON) != "" {
				result["remote_status"] = json.RawMessage(estadoRemoto.RawJSON)
			}
			if err := marcarSesionHeartbeatSupervisada(handle, runtime, nil, strings.TrimSpace(estadoRemoto.LogicalState)); err != nil {
				return nil, err
			}
			if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, source); err != nil {
				return nil, err
			}
		} else if observedProcess, processResult, err := observarProcesoLocalRuntime(handle, runtime, source); err != nil {
			return nil, err
		} else if observedProcess {
			result["observed_process"] = true
			for k, v := range processResult {
				result[k] = v
			}
		}
		if handle.ID > 0 {
			handle, err = GetRuntimeHandle(handle.ID)
			if err != nil {
				return nil, err
			}
		}
	}
	if runtime != nil && runtime.ID > 0 {
		freshRuntime, err := GetRuntime(runtime.ID)
		if err != nil {
			return nil, err
		}
		if freshRuntime != nil {
			runtime = freshRuntime
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
	return result, nil
}

func observarProcesoLocalRuntime(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (bool, map[string]any, error) {
	if handle == nil {
		return false, nil, nil
	}
	obj := controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}
	if runtime != nil && runtime.PID != nil && *runtime.PID > 0 {
		obj.PID = runtime.PID
	}
	estado, observed, err := controlruntime.ConsultarEstadoLocal(obj)
	if err != nil {
		return true, map[string]any{
			"process_alive": false,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if !observed {
		return false, nil, nil
	}
	if estado == nil {
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}
	pid := estado.PID
	vivo := estado.Vivo
	if !vivo && pid == 0 {
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}
	if err := aplicarEstadoLocalObservado(handle, estado); err != nil {
		return true, map[string]any{
			"process_alive": vivo,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if !vivo {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, nil, err
		}
		return true, map[string]any{
			"process_alive": false,
		}, nil
	}

	estadoHandle := strings.TrimSpace(handle.Estado)
	if estadoHandle == "" {
		estadoHandle = "activo"
	}
	if estadoHandle != "pausado" {
		estadoHandle = "activo"
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estadoHandle, handle.ID); err != nil {
		return true, nil, err
	}

	pid64 := int64(pid)
	logicalState := ""
	processState := ""
	if runtime != nil {
		runtimeSnapshot := *runtime
		runtimeSnapshot.PID = &pid64
		if sample, err := RegistrarMuestraGenericProcess(runtime.ID, &runtimeSnapshot); err == nil && sample != nil {
			logicalState = strings.TrimSpace(sample.LogicalState)
			if normalized, normErr := NormalizarMuestraRuntime(sample); normErr == nil && normalized != nil {
				processState = strings.TrimSpace(normalized.State)
			}
		} else {
			if _, updErr := DB.Exec(`
				UPDATE runtime_instances
				SET pid = ?,
				    process_state = ?,
				    last_event_at = CURRENT_TIMESTAMP,
				    last_heartbeat_at = CURRENT_TIMESTAMP
				WHERE id = ?`, pid64, "running", runtime.ID); updErr != nil {
				return true, nil, updErr
			}
		}
	}
	if logicalState == "" {
		if estadoHandle == "pausado" {
			logicalState = "pausado"
		} else {
			logicalState = "disponible"
		}
	}
	if processState == "" {
		if estadoHandle == "pausado" {
			processState = "stopped"
		} else {
			processState = "running"
		}
	}
	if handle != nil {
		syncedHandle, _, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
		if err != nil {
			return true, nil, err
		}
		if syncedHandle != nil {
			handle = syncedHandle
		}
	}
	if err := marcarSesionHeartbeatSupervisada(handle, runtime, &pid64, logicalState); err != nil {
		return true, nil, err
	}
	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, source); err != nil {
		return true, nil, err
	}
	return true, map[string]any{
		"process_alive":  true,
		"process_pid":    pid64,
		"process_state":  processState,
		"logical_state":  logicalState,
		"observed_local": true,
	}, nil
}

func aplicarEstadoLocalObservado(handle *RuntimeHandle, estado *controlruntime.EstadoLocal) error {
	if handle == nil || estado == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for k, v := range mapFromJSON(estado.MetadataJSON) {
		meta[k] = v
	}
	meta["local_last_status_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	metaJSON, _ := json.Marshal(meta)
	caps := handle.CapabilitiesJSON
	if strings.TrimSpace(estado.CapabilitiesJSON) != "" {
		caps = strings.TrimSpace(estado.CapabilitiesJSON)
	}
	estadoHandle := strings.TrimSpace(handle.Estado)
	if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
		estadoHandle = normalizarEstadoHandleObservado(rawEstado, estadoHandle)
	}
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    metadata_json = ?,
		    capabilities_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		estadoHandle, string(metaJSON), caps, handle.ID,
	)
	return err
}

func marcarProcesoLocalNoDisponible(handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if handle != nil && handle.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado = 'fallido',
			    last_seen_at = CURRENT_TIMESTAMP
			WHERE id = ?`, handle.ID); err != nil {
			return err
		}
	}
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state = 'degradado',
			    process_state = 'missing',
			    last_event_at = CURRENT_TIMESTAMP
			WHERE id = ?`, runtime.ID); err != nil {
			return err
		}
	}
	return nil
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
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
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

func marcarSesionHeartbeatSupervisada(handle *RuntimeHandle, runtime *RuntimeInstance, pid *int64, logicalState string) error {
	sesionID := int64(0)
	agente := ""
	if handle != nil {
		agente = strings.TrimSpace(handle.Agente)
		if handle.SesionID != nil {
			sesionID = *handle.SesionID
		}
	}
	if runtime != nil {
		if agente == "" {
			agente = strings.TrimSpace(runtime.Agente)
		}
		if sesionID <= 0 && runtime.SesionID != nil {
			sesionID = *runtime.SesionID
		}
	}
	if sesionID <= 0 {
		return nil
	}
	estado := "activa"
	switch strings.ToLower(strings.TrimSpace(logicalState)) {
	case "pausado":
		estado = "pausada"
	}
	args := []any{estado, sesionID}
	q := `UPDATE sesiones SET estado = ?, heartbeat_at = CURRENT_TIMESTAMP`
	if pid != nil && *pid > 0 {
		q += `, pid = ?`
		args = []any{estado, *pid, sesionID}
	}
	q += ` WHERE id = ? AND activa = 1`
	if _, err := DB.Exec(q, args...); err != nil {
		return err
	}
	if agente != "" {
		SetEstadoSesion(agente, strings.TrimSpace(logicalState))
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

func runtimeProcesoLocalYaNoVive(order *RuntimeOrder) (bool, int, error) {
	if order == nil {
		return false, 0, nil
	}
	obj, err := objetivoProcesoParaOrden(order)
	if err != nil {
		return false, 0, err
	}
	pid, ok, err := controlruntime.ResolverPID(obj)
	if err != nil || !ok {
		return false, pid, err
	}
	vivo, _, err := controlruntime.ProcesoVivo(obj)
	if err != nil {
		return false, pid, err
	}
	return !vivo, pid, nil
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
	if err := controlruntime.ActivarSupervisionOrquestada(controlruntime.ObjetivoProceso{
		PID:          pid64,
		HandleKind:   strings.TrimSpace(arranque.HandleKind),
		HandleRef:    strings.TrimSpace(arranque.HandleRef),
		MetadataJSON: strings.TrimSpace(arranque.MetadataJSON),
	}); err != nil {
		return err
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

func ackBootstrapRuntimeSinLeaseEnStart(startOrder *RuntimeOrder, bootstrap *bootstrapRuntimeData, sesion *Sesion) error {
	if bootstrap == nil || bootstrap.Order != nil {
		return nil
	}
	mailboxIDs := runtimeMailboxIDs(bootstrap.Mailbox)
	if len(mailboxIDs) == 0 {
		return nil
	}
	if err := marcarRuntimeMailboxConsumidoPorIDs(mailboxIDs); err != nil {
		return err
	}
	bootstrap.Consumidos = len(mailboxIDs)
	if startOrder != nil && startOrder.ID > 0 {
		var sesionID int64
		if sesion != nil {
			sesionID = sesion.ID
		}
		Audit("orquesta", "runtime_bootstrap_start_ack", "runtime_order", startOrder.ID,
			fmt.Sprintf("sesion_id=%d mailbox_ids=%v ack_mode=start_success", sesionID, mailboxIDs))
	}
	return nil
}

func inyectarPromptArranque(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan, order *RuntimeOrder) error {
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
	texto := controlruntime.NormalizarInstruccionProceso(obj, promptArranquePendiente(plan))
	if texto == "" {
		return nil
	}
	if plan.LaunchPromptDelayMS > 0 {
		time.Sleep(time.Duration(plan.LaunchPromptDelayMS) * time.Millisecond)
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
		"motivo":      "launch_prompt_start",
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
	yaDetenido := false
	if err != nil || !aplicado {
		controlErr := err
		yaDetenido, pid, err = runtimeProcesoLocalYaNoVive(order)
		if err != nil {
			return err
		}
		if !yaDetenido {
			if controlErr != nil {
				return controlErr
			}
			return fmt.Errorf("stop sin control real para %s", order.Agente)
		}
		aplicado = false
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
	if _, err := DB.Exec(`
		UPDATE runtime_handles SET estado=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado')`, handleState, order.Agente); err != nil {
		return err
	}
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
	payload, err := runtimeOrderPayloadMap(order.PayloadJSON)
	if err != nil {
		return err
	}

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	if strings.TrimSpace(toAgente) == "" {
		return fmt.Errorf("send_instruction sin agente destino")
	}
	texto := stringFromMap(payload, "texto", stringFromMap(payload, "instruction", ""))

	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return err
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		return err
	}
	handle, externalSessionID, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	if err != nil {
		return err
	}
	handle, workingDir, err := SincronizarRuntimeHandleWorkingDir(handle, runtime, order.Agente, order.ProyectoID)
	if err != nil {
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

	obj := controlruntime.ObjetivoProceso{}
	if handle != nil {
		obj.HandleKind = handle.HandleKind
		obj.HandleRef = handle.HandleRef
		obj.MetadataJSON = handle.MetadataJSON
	}
	if strings.TrimSpace(workingDir) != "" {
		meta := mapFromJSON(obj.MetadataJSON)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["working_dir"] = strings.TrimSpace(workingDir)
		if metaJSON, marshalErr := json.Marshal(meta); marshalErr == nil {
			obj.MetadataJSON = string(metaJSON)
		}
	}
	if runtime != nil && (handle == nil || strings.TrimSpace(handle.HandleKind) == "" || strings.TrimSpace(handle.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	texto = controlruntime.NormalizarInstruccionProceso(obj, texto)
	if strings.TrimSpace(texto) == "" {
		return fmt.Errorf("send_instruction sin texto aplicable")
	}

	if handle == nil {
		if runtimeOrderSendInstructionProvieneMailbox(payload) {
			return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "sin runtime activo entregable")
		}
		return fmt.Errorf("send_instruction sin runtime handle entregable")
	}

	aplicado := false
	pid := 0
	deliveryPath := ""
	deliveryMode := RuntimeHandleMailboxDeliveryMode(handle)
	if RuntimeHandlePermiteSendInputInteractivo(handle) || RuntimeHandlePermiteEntregaCalienteSupervisada(handle) {
		aplicado, pid, err = controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
			return controlruntime.EnviarInstruccionProceso(obj, texto)
		})
		if err != nil {
			if runtimeOrderSendInstructionProvieneMailbox(payload) {
				return reencolarRuntimeOrderSendInstruction(order, payload, err.Error())
			}
			return err
		}
		if aplicado {
			if RuntimeHandlePermiteEntregaCalienteSupervisada(handle) && !RuntimeHandlePermiteSendInputInteractivo(handle) {
				deliveryPath = "supervisor_local"
			} else {
				deliveryPath = "interactive"
			}
		}
	}
	if !aplicado && deliveryMode == runtimeagente.MailboxDeliverySessionResume {
		aplicado, pid, err = controlruntime.EnviarInstruccionSesionResume(obj, externalSessionID, texto)
		if err != nil {
			if handled, pauseErr := gestionarBackoffProveedorRuntimeOrderSendInstruction(order, payload, err); handled {
				return pauseErr
			}
			if runtimeOrderSendInstructionDebeDegradarseAMailboxPorError(order, payload, err) {
				return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, err.Error())
			}
			if runtimeOrderSendInstructionProvieneMailbox(payload) {
				return reencolarRuntimeOrderSendInstruction(order, payload, err.Error())
			}
			return err
		}
		if aplicado {
			deliveryPath = "session_resume"
		}
	}
	if aplicado {
		if err := RegistrarRuntimeTranscriptInput(handle, runtime, texto); err != nil {
			return err
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

	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		return completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata")
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

func resolverDestinoRuntimeOrderSendInstruction(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (*RuntimeInstance, *RuntimeHandle, error) {
	if order == nil {
		return runtime, handle, nil
	}
	if handle != nil {
		validado, err := validarRuntimeHandleActivo(handle, order.Agente, order.ProyectoID)
		if err != nil {
			return nil, nil, err
		}
		if validado != nil {
			if validado.RuntimeID != nil && *validado.RuntimeID > 0 {
				runtime, err = GetRuntime(*validado.RuntimeID)
				if err != nil {
					return nil, nil, err
				}
			} else {
				runtime = nil
			}
			if handle.ID != validado.ID {
				if err := actualizarRuntimeOrderDestino(order.ID, runtime, validado); err != nil {
					return nil, nil, err
				}
			}
			return runtime, validado, nil
		}
		handle = nil
		runtime = nil
	}

	var (
		candidato *RuntimeHandle
		err       error
	)
	if order.ProyectoID != nil {
		candidato, err = GetRuntimeHandleActivoAgenteProyecto(order.Agente, order.ProyectoID)
	} else {
		candidato, err = GetRuntimeHandleActivoAgente(order.Agente)
	}
	if err != nil || candidato == nil {
		return runtime, handle, err
	}
	if handle != nil && handle.ID == candidato.ID {
		return runtime, handle, nil
	}
	runtime, err = runtimeHandleRuntime(candidato)
	if err != nil {
		return nil, nil, err
	}
	if err := actualizarRuntimeOrderDestino(order.ID, runtime, candidato); err != nil {
		return nil, nil, err
	}
	return runtime, candidato, nil
}

func actualizarRuntimeOrderDestino(orderID int64, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if orderID <= 0 || handle == nil || handle.ID <= 0 {
		return nil
	}
	var runtimeID any
	if runtime != nil && runtime.ID > 0 {
		runtimeID = runtime.ID
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
	return err
}

func runtimeOrderSendInstructionProvieneMailbox(payload map[string]any) bool {
	return runtimeOrderSendInstructionMailboxID(payload) > 0
}

func runtimeOrderSendInstructionMailboxID(payload map[string]any) int64 {
	if payload == nil {
		return 0
	}
	return int64FromAny(payload["mailbox_id"])
}

func runtimeOrderSendInstructionExternalSessionID(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(stringFromMap(payload, "external_session_id", ""))
}

func runtimeOrderSendInstructionRetryDelay() time.Duration {
	seconds := configIntOrDefault("runtime_send_instruction_retry_seconds", 15)
	if seconds <= 0 {
		seconds = 15
	}
	return time.Duration(seconds) * time.Second
}

func runtimeOrderSendInstructionDebeDegradarseAMailboxPorError(order *RuntimeOrder, payload map[string]any, runtimeErr error) bool {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) || runtimeErr == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	if !strings.Contains(raw, "session_resume timeout") {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "autonomia", "nudge", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func reencolarRuntimeOrderSendInstruction(order *RuntimeOrder, payload map[string]any, reason string) error {
	return reencolarRuntimeOrderSendInstructionAt(order, payload, reason, time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay()))
}

func reencolarRuntimeOrderSendInstructionAt(order *RuntimeOrder, payload map[string]any, reason string, nextAttempt time.Time) error {
	if order == nil || order.ID <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "runtime no entregable todavía"
	}
	if nextAttempt.IsZero() {
		nextAttempt = time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":              false,
		"deferred":        true,
		"mailbox_id":      runtimeOrderSendInstructionMailboxID(payload),
		"retry_after":     nextAttempt.Format(time.RFC3339Nano),
		"deferred_reason": reason,
	})
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado = 'pendiente',
		    resultado_json = ?,
		    error_text = CASE
		        WHEN TRIM(COALESCE(error_text, '')) = '' THEN ?
		        ELSE error_text || CHAR(10) || ?
		    END,
		    started_at = NULL,
		    finished_at = NULL,
		    claimed_by = '',
		    lease_token = '',
		    lease_expires_at = NULL,
		    available_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		resultado,
		reason,
		reason,
		nextAttempt,
		order.ID,
	)
	return err
}

func completarRuntimeOrderSendInstructionDiferidaAMailbox(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox retenido hasta runtime entregable"
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":              true,
		"mailbox_id":      runtimeOrderSendInstructionMailboxID(payload),
		"deferred":        true,
		"deferred_reason": reason,
		"mailbox_only":    true,
	})
	return MarcarRuntimeOrderEstado(order.ID, "completada", resultado, "")
}

func gestionarBackoffProveedorRuntimeOrderSendInstruction(order *RuntimeOrder, payload map[string]any, runtimeErr error) (bool, error) {
	delay, motivo, ok := runtimeOrderProviderBackoff(runtimeErr)
	if !ok {
		return false, nil
	}
	minutos := int((delay + time.Minute - 1) / time.Minute)
	if minutos < 1 {
		minutos = 1
	}
	motivoPausa := "Auto-pausa por agotamiento: " + motivo
	if err := PausarAgente(strings.TrimSpace(order.Agente), minutos, motivoPausa); err != nil {
		return true, err
	}
	detalle := fmt.Sprintf("agente en enfriamiento por %s", motivo)
	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		return true, reencolarRuntimeOrderSendInstructionAt(order, payload, detalle, time.Now().UTC().Add(delay))
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":               false,
		"provider_backoff": true,
		"pause_minutes":    minutos,
		"pause_reason":     motivo,
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "fallida", resultado, detalle)
}

func completarRuntimeOrderSendInstructionSesionObsoleta(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, externalSessionID string) (bool, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	payloadExternal := runtimeOrderSendInstructionExternalSessionID(payload)
	actualExternal := strings.TrimSpace(externalSessionID)
	if payloadExternal == "" || actualExternal == "" || payloadExternal == actualExternal {
		return false, nil
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                  true,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"superseded":          true,
		"superseded_reason":   "external_session_id_changed",
		"external_session_id": actualExternal,
		"handle_id":           runtimeHandleID(handle),
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "completada", resultado, "")
}

func completarRuntimeOrderSendInstructionCubiertaPorBootstrap(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		return false, nil
	}
	covered, bootstrapOrderID, startOrderID, err := RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
	if err != nil || !covered {
		return false, err
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                 true,
		"mailbox_id":         mailboxID,
		"superseded":         true,
		"superseded_reason":  "covered_by_bootstrap_lease",
		"bootstrap_order_id": bootstrapOrderID,
		"start_order_id":     startOrderID,
		"handle_id":          runtimeHandleID(handle),
		"runtime_id":         runtimeInstanceID(runtime),
	})
	return true, MarcarRuntimeOrderEstado(order.ID, "completada", resultado, "")
}

func runtimeOrderProviderBackoff(err error) (time.Duration, string, bool) {
	if err == nil {
		return 0, "", false
	}
	raw := strings.TrimSpace(err.Error())
	if raw == "" {
		return 0, "", false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "hit your usage limit") &&
		!strings.Contains(lower, "usage limit") &&
		!strings.Contains(lower, "rate limit") &&
		!strings.Contains(lower, "purchase more credits") &&
		!strings.Contains(lower, "try again at") {
		return 0, "", false
	}
	motivo := "cuota de proveedor"
	switch {
	case strings.Contains(lower, "rate limit"):
		motivo = "rate limit de proveedor"
	case strings.Contains(lower, "usage limit"), strings.Contains(lower, "purchase more credits"):
		motivo = "usage limit de proveedor"
	}
	delay := 60 * time.Minute
	if parsed, ok := runtimeOrderProviderRetryDelay(lower, time.Now()); ok && parsed > 0 {
		delay = parsed
	}
	return delay, motivo, true
}

func runtimeOrderProviderRetryDelay(lower string, now time.Time) (time.Duration, bool) {
	re := regexp.MustCompile(`try again at\s+(\d{1,2}):(\d{2})\s*([ap]m)`)
	match := re.FindStringSubmatch(strings.ToLower(lower))
	if len(match) != 4 {
		return 0, false
	}
	hour, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, false
	}
	ampm := match[3]
	if hour == 12 {
		hour = 0
	}
	if ampm == "pm" {
		hour += 12
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	return target.Sub(now), true
}

func completarRuntimeMailboxDesdePayload(payload map[string]any, proyectoID *int64) error {
	if payload == nil {
		return nil
	}
	mailboxID := int64FromAny(payload["mailbox_id"])
	if mailboxID <= 0 {
		return nil
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		return err
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		return err
	}
	kind := strings.TrimSpace(stringFromAny(payload["mailbox_kind"]))
	toAgente := strings.TrimSpace(stringFromAny(payload["to_agente"]))
	if runtimeMailboxKindSupersedible(kind) && toAgente != "" {
		_, err := ConsumirRuntimeMailboxPendienteSupersedido(toAgente, proyectoID, kind, mailboxID)
		return err
	}
	return nil
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

func runtimeInstanceID(runtime *RuntimeInstance) any {
	if runtime == nil {
		return nil
	}
	return runtime.ID
}

func payloadJSONDesdePlan(plan *runtimeagente.LaunchPlan) string {
	if plan == nil {
		return "{}"
	}
	data, err := json.Marshal(map[string]any{
		"modo":                   strings.TrimSpace(plan.Modo),
		"native_resume":          plan.NativeResume,
		"continuity_prompt":      strings.TrimSpace(plan.ContinuityPrompt),
		"bootstrap_prompt":       strings.TrimSpace(plan.BootstrapPrompt),
		"launch_prompt_embedded": plan.LaunchPromptEmbedded,
		"launch_prompt_mode":     strings.TrimSpace(plan.LaunchPromptMode),
		"launch_prompt_delay_ms": plan.LaunchPromptDelayMS,
	})
	if err != nil {
		return "{}"
	}
	return string(data)
}

func payloadJSONDesdePlanYResume(plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext) string {
	_ = plan
	return strings.TrimSpace(resume.ResumePayloadJSON)
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
		if strings.TrimSpace(plan.BootstrapPrompt) != "" {
			parts = append(parts, "Bootstrap inicial: "+strings.TrimSpace(plan.BootstrapPrompt))
		}
		if strings.TrimSpace(plan.ContinuityPrompt) != "" {
			parts = append(parts, "Prompt de continuidad: "+strings.TrimSpace(plan.ContinuityPrompt))
		}
		if plan.LaunchPromptEmbedded {
			parts = append(parts, "El prompt inicial quedó embebido en el comando de arranque.")
		} else if strings.TrimSpace(promptArranquePendiente(plan)) != "" {
			parts = append(parts, "El prompt inicial quedó programado para inyección tras arranque.")
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
	if strings.TrimSpace(arranque.ExternalSessionID) != "" {
		meta["external_session_id"] = strings.TrimSpace(arranque.ExternalSessionID)
	}
	if conector != nil {
		meta["conector"] = strings.TrimSpace(conector.Slug)
	}
	if plan != nil {
		meta["modo_plan"] = strings.TrimSpace(plan.Modo)
		meta["continuity_prompt"] = strings.TrimSpace(plan.ContinuityPrompt)
		meta["bootstrap_prompt"] = strings.TrimSpace(plan.BootstrapPrompt)
		meta["launch_prompt_embedded"] = plan.LaunchPromptEmbedded
		meta["launch_prompt_mode"] = strings.TrimSpace(plan.LaunchPromptMode)
		meta["launch_prompt_delay_ms"] = plan.LaunchPromptDelayMS
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
	proyecto = ProyectoConRutaEfectiva(proyecto, "")
	ultima, err := ObtenerUltimaSesion(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	conector, err := resolverConectorRuntimeOrder(conectorRef, ultima)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	perfilTarea, modelo, razonamiento, err = ResolverPerfilEjecucionLanzamiento(
		strings.TrimSpace(proyecto.Slug),
		perfilTarea,
		modelo,
		razonamiento,
	)
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
		Agente:       nombreAgenteRuntime(strings.TrimSpace(agenteRef), strings.TrimSpace(agente.Nombre)),
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
	plan.BootstrapPrompt, err = BuildLaunchBootstrapPromptForContext(agente, proyecto, plan)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	if err := runtimeagente.ApplyLaunchPromptMetadata(plan, strings.TrimSpace(conector.MetadataJSON)); err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	return agente, proyecto, conector, ultima, resume, bootstrap, plan, nil
}

func promptArranquePendiente(plan *runtimeagente.LaunchPlan) string {
	if plan == nil || plan.LaunchPromptEmbedded {
		return ""
	}
	return strings.TrimSpace(runtimeagente.LaunchPromptText(plan))
}

func nombreAgenteRuntime(preferido, almacenado string) string {
	preferido = strings.TrimSpace(preferido)
	almacenado = strings.TrimSpace(almacenado)
	if preferido != "" && strings.EqualFold(preferido, almacenado) {
		return preferido
	}
	return almacenado
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
	resume = SanitizeResumeContextForProject(resume, proyecto)
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
	resume.CWD = RutaTrabajoPreferidaAgenteProyecto(strings.TrimSpace(agente), proyecto, strings.TrimSpace(resume.CWD))
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
	return resume, bootstrap, nil
}

func marcarBootstrapRuntimePreparationEjecutando(startOrder *RuntimeOrder, bootstrap *bootstrapRuntimeData, sesion *Sesion, handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if bootstrap == nil || bootstrap.Order == nil || bootstrap.Order.ID <= 0 {
		return nil
	}
	resultado := map[string]any{
		"ok":            true,
		"bootstrap":     true,
		"lease_state":   "waiting_for_evidence",
		"checkpoint_id": checkpointIDOrZeroDB(bootstrap.Checkpoint),
		"mailbox_count": len(bootstrap.Mailbox),
		"mailbox_ids":   runtimeMailboxIDs(bootstrap.Mailbox),
		"order_tipo":    strings.TrimSpace(bootstrap.Order.Tipo),
		"ack_mode":      "runtime_transcript_or_tick",
		"continuidad":   bootstrap.Checkpoint != nil || len(bootstrap.Mailbox) > 0,
	}
	if startOrder != nil && startOrder.ID > 0 {
		resultado["start_order_id"] = startOrder.ID
	}
	if sesion != nil && sesion.ID > 0 {
		resultado["sesion_id"] = sesion.ID
	}
	if handle != nil && handle.ID > 0 {
		resultado["handle_id"] = handle.ID
	}
	if runtime != nil && runtime.ID > 0 {
		resultado["runtime_id"] = runtime.ID
	}
	return MarcarRuntimeOrderEstado(bootstrap.Order.ID, "ejecutando", mergeRuntimeOrderResultJSON(bootstrap.Order.ResultadoJSON, resultado), "")
}

func revertBootstrapRuntimePreparation(bootstrap *bootstrapRuntimeData) error {
	if bootstrap == nil || bootstrap.Order == nil || bootstrap.Order.ID <= 0 {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='pendiente',
		    started_at=NULL,
		    claimed_by='',
		    lease_token='',
		    lease_expires_at=NULL,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id = ?
		  AND estado IN ('tomada','ejecutando')`, bootstrap.Order.ID)
	return err
}

func marcarRuntimeMailboxConsumidoPorIDs(ids []int64) error {
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(id); err != nil {
			return err
		}
		if err := MarcarRuntimeMailboxConsumido(id); err != nil {
			return err
		}
	}
	return nil
}

func runtimeMailboxIDs(mailbox []*RuntimeMailboxMessage) []int64 {
	if len(mailbox) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(mailbox))
	for _, msg := range mailbox {
		if msg == nil || msg.ID <= 0 {
			continue
		}
		ids = append(ids, msg.ID)
	}
	return ids
}

func mergeRuntimeOrderResultJSON(prev string, extras map[string]any) string {
	base := map[string]any{}
	prev = strings.TrimSpace(prev)
	if prev != "" {
		_ = json.Unmarshal([]byte(prev), &base)
	}
	for k, v := range extras {
		base[k] = v
	}
	data, err := json.Marshal(base)
	if err != nil {
		return prev
	}
	return string(data)
}

func int64FromAny(raw any) int64 {
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func int64SliceFromAny(raw any) []int64 {
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, item := range values {
		if id := int64FromAny(item); id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func runtimeBootstrapLeaseFromOrder(order *RuntimeOrder) (startOrderID int64, mailboxIDs []int64, sesionID int64) {
	if order == nil {
		return 0, nil, 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	startOrderID = int64FromAny(result["start_order_id"])
	mailboxIDs = int64SliceFromAny(result["mailbox_ids"])
	sesionID = int64FromAny(result["sesion_id"])
	return startOrderID, mailboxIDs, sesionID
}

func AckBootstrapRuntimeLease(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 && len(mailboxIDs) == 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if err := marcarRuntimeMailboxConsumidoPorIDs(mailboxIDs); err != nil {
		return err
	}
	if bootstrapOrderID > 0 {
		order, err := GetRuntimeOrder(bootstrapOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "completada" && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":          true,
				"bootstrap":   true,
				"acked_by":    ackSource,
				"lease_state": "acked",
				"sesion_id":   sesionID,
				"mailbox_ids": mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
				return err
			}
			if err := registrarEvidenciaHandoffReanudadoPorOrden(order, sesionID); err != nil {
				return err
			}
		}
	}
	if startOrderID > 0 {
		order, err := GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado == "ejecutando" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":                 true,
				"acked_by":           ackSource,
				"sesion_id":          sesionID,
				"bootstrap_order_id": bootstrapOrderID,
				"bootstrap_mailbox":  mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func AckBootstrapRuntimeLeaseByEvidence(handle *RuntimeHandle, runtime *RuntimeInstance, ackSource string) error {
	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil || order == nil {
		return err
	}
	return AckBootstrapRuntimeLease(startOrderID, order.ID, mailboxIDs, sesionID, ackSource)
}

func RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, int64, int64, error) {
	if mailboxID <= 0 {
		return false, 0, 0, nil
	}
	order, startOrderID, mailboxIDs, _, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil || order == nil {
		return false, 0, 0, err
	}
	for _, id := range mailboxIDs {
		if id == mailboxID {
			return true, order.ID, startOrderID, nil
		}
	}
	return false, 0, 0, nil
}

func resolverBootstrapRuntimeLeasePendiente(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	agente := ""
	var proyectoID *int64
	var sesionID int64
	if handle != nil {
		agente = strings.TrimSpace(handle.Agente)
		proyectoID = handle.ProyectoID
		if handle.SesionID != nil {
			sesionID = *handle.SesionID
		}
	}
	if runtime != nil {
		if agente == "" {
			agente = strings.TrimSpace(runtime.Agente)
		}
		if proyectoID == nil {
			proyectoID = runtime.ProyectoID
		}
		if sesionID <= 0 && runtime.SesionID != nil {
			sesionID = *runtime.SesionID
		}
	}
	if agente == "" {
		return nil, 0, nil, 0, nil
	}

	candidates := make([]*RuntimeOrder, 0, 4)
	for _, estado := range []string{"ejecutando", "tomada"} {
		estado := estado
		orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return nil, 0, nil, 0, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			switch strings.TrimSpace(order.Tipo) {
			case "handoff", "resume":
				candidates = append(candidates, order)
			}
		}
	}
	for _, order := range candidates {
		startOrderID, mailboxIDs, orderSesionID := runtimeBootstrapLeaseFromOrder(order)
		if sesionID > 0 && orderSesionID > 0 && orderSesionID != sesionID {
			continue
		}
		if sesionID <= 0 {
			sesionID = orderSesionID
		}
		return order, startOrderID, mailboxIDs, sesionID, nil
	}
	return nil, 0, nil, sesionID, nil
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
				`UPDATE tareas SET notas = notas || char(10) || ? || ' [' || datetime('now') || ' orquesta]' WHERE id=?`,
				nota, *payload.TareaID,
			); err != nil {
				return err
			}
		}
	}
	detalle := fmt.Sprintf("origen=%s destino=%s sesion_destino=%d order=%d", payload.AgenteOrigen, payload.AgenteDestino, sesionID, order.ID)
	Audit(strings.TrimSpace(destino), "handoff_reanudado", "runtime_order", order.ID, detalle)
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
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
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

func claimRuntimeOrderByID(id int64) (*RuntimeOrder, error) {
	now := time.Now().UTC()
	current, err := GetRuntimeOrder(id)
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
		WHERE id = ? AND estado = 'pendiente' AND available_at <= CURRENT_TIMESTAMP`,
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

func reconciliarRuntimeOrderStale(id int64, tipo string, now, cutoff time.Time) (bool, error) {
	var (
		res sql.Result
		err error
	)
	if runtimeOrderTipoDespachable(tipo) {
		res, err = DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'pendiente',
			    started_at = NULL,
			    finished_at = NULL,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL,
			    available_at = CURRENT_TIMESTAMP,
			    error_text = CASE
			        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'reencolada tras stale del control plane'
			        ELSE error_text || CHAR(10) || 'reencolada tras stale del control plane'
			    END
			WHERE id = ?
			  AND estado IN ('tomada','ejecutando')
			  AND (
			        (lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
			     OR (lease_expires_at IS NULL AND COALESCE(started_at, updated_at, created_at) <= ?)
			  )`, id, now, cutoff)
	} else {
		res, err = DB.Exec(`
			UPDATE runtime_orders
			SET estado = 'expirada',
			    finished_at = CURRENT_TIMESTAMP,
			    claimed_by = '',
			    lease_token = '',
			    lease_expires_at = NULL,
			    error_text = CASE
			        WHEN TRIM(COALESCE(error_text, '')) = '' THEN 'orden expirada por stale sin dispatcher compatible'
			        ELSE error_text || CHAR(10) || 'orden expirada por stale sin dispatcher compatible'
			    END
			WHERE id = ?
			  AND estado IN ('tomada','ejecutando')
			  AND (
			        (lease_expires_at IS NOT NULL AND lease_expires_at <= ?)
			     OR (lease_expires_at IS NULL AND COALESCE(started_at, updated_at, created_at) <= ?)
			  )`, id, now, cutoff)
	}
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
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
	var leaseExpires sql.NullTime
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	err := s.Scan(
		&o.ID, &o.Agente, &proyectoID, &runtimeID, &handleID, &o.Tipo, &o.PayloadJSON, &o.ResultadoJSON,
		&o.ErrorText, &o.Estado, &o.ClaimedBy, &o.LeaseToken, &o.AttemptCount, &leaseExpires,
		&o.AvailableAt, &o.CreatedAt, &startedAt, &finishedAt, &o.UpdatedAt,
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
	if leaseExpires.Valid {
		o.LeaseExpiresAt = &leaseExpires.Time
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

func RuntimeHandleMailboxDeliveryMode(handle *RuntimeHandle) string {
	if handle == nil {
		return runtimeagente.MailboxDeliveryInteractive
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if RuntimeHandlePermiteEntregaCalienteSupervisada(handle) {
		if mode := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(caps, "mailbox_delivery_mode", "")); mode != "" {
			return mode
		}
		if mode := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(meta, "mailbox_delivery_mode", "")); mode != "" {
			return mode
		}
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if runtimeHandleUsaCodexTTYInestable(meta) && runtimeHandleTieneExternalSessionID(handle, nil) {
		return runtimeagente.MailboxDeliverySessionResume
	}
	if runtimeHandleUsaCodexTTYInestable(meta) &&
		!boolFromMap(meta, "mailbox_restart_safe") &&
		!boolFromMap(caps, "mailbox_restart_safe") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if mode := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(caps, "mailbox_delivery_mode", "")); mode != "" {
		return mode
	}
	if mode := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(meta, "mailbox_delivery_mode", "")); mode != "" {
		return mode
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	return runtimeagente.MailboxDeliveryInteractive
}

func runtimeHandleTieneExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) bool {
	if handle == nil {
		return false
	}
	if ext := strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "external_session_id", "")); ext != "" {
		return true
	}
	if runtime != nil && strings.TrimSpace(runtime.ExternalSessionID) != "" {
		return true
	}
	if handle.SesionID != nil {
		sesion, err := GetSesionByID(*handle.SesionID)
		if err == nil && sesion != nil && strings.TrimSpace(sesion.ExternalSessionID) != "" {
			return true
		}
	}
	return false
}

func RuntimeHandlePermiteSendInputInteractivo(handle *RuntimeHandle) bool {
	if handle == nil {
		return true
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if _, ok := caps["can_send_input"]; ok && !boolFromMap(caps, "can_send_input") {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if _, ok := meta["can_send_input"]; ok && !boolFromMap(meta, "can_send_input") {
		return false
	}
	if runtimeHandleUsaCodexTTYInestable(meta) {
		return false
	}
	if runtimeHandleLocalProcesoSinCanalInteractivo(handle, meta) {
		return false
	}
	return true
}

func RuntimeHandlePermiteEntregaCalienteSupervisada(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli") {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "stdin_path", "")) == "" {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "supervisor_ref", "")) == "" {
		return false
	}
	if runtimeHandleUsaCodexTTYInestable(meta) {
		return true
	}
	return true
}

func runtimeHandleUsaCodexTTYInestable(meta map[string]any) bool {
	for _, candidate := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
		stringFromMap(meta, "herramienta", ""),
		stringFromMap(meta, "conector", ""),
	} {
		text := strings.ToLower(strings.TrimSpace(candidate))
		if text == "" {
			continue
		}
		first := text
		if fields := strings.Fields(text); len(fields) > 0 {
			first = fields[0]
		}
		base := filepath.Base(first)
		if strings.Contains(text, "codex-perfil") ||
			strings.Contains(text, "/codex") ||
			base == "codex" ||
			base == "codex-cli" ||
			strings.HasPrefix(text, "codex ") ||
			strings.Contains(text, "codex-cli") {
			return true
		}
	}
	return false
}

func runtimeHandleLocalProcesoSinCanalInteractivo(handle *RuntimeHandle, meta map[string]any) bool {
	if handle == nil {
		return false
	}
	if strings.TrimSpace(handle.HandleKind) != "process" {
		return false
	}
	if strings.TrimSpace(stringFromMap(meta, "stdin_path", "")) != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") {
		return true
	}
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	if strings.EqualFold(driver, "process_pty_cli") {
		return true
	}
	if strings.TrimSpace(stringFromMap(meta, "supervisor_ref", "")) != "" {
		return true
	}
	if _, ok := meta["supervisor_owner_pid"]; ok {
		return true
	}
	if strings.TrimSpace(stringFromMap(meta, "supervision_mode", "")) != "" {
		return true
	}
	return false
}

func SincronizarRuntimeHandleExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeHandle, string, error) {
	if handle == nil {
		return nil, "", nil
	}
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
			return nil, "", err
		}
	}
	externalSessionID, err := runtimeHandleEffectiveExternalSessionID(handle, runtime)
	if err != nil || strings.TrimSpace(externalSessionID) == "" {
		return handle, strings.TrimSpace(externalSessionID), err
	}

	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	if strings.TrimSpace(stringFromMap(meta, "external_session_id", "")) != externalSessionID {
		meta["external_session_id"] = externalSessionID
		metaJSON, _ := json.Marshal(meta)
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), handle.ID); err != nil {
			return nil, "", err
		}
	}
	if runtime != nil && runtime.ID > 0 && strings.TrimSpace(runtime.ExternalSessionID) != externalSessionID {
		if _, err := DB.Exec(`UPDATE runtime_instances SET external_session_id=?, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, externalSessionID, runtime.ID); err != nil {
			return nil, "", err
		}
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		if _, err := DB.Exec(`UPDATE sesiones SET external_session_id=? WHERE id=?`, externalSessionID, *handle.SesionID); err != nil {
			return nil, "", err
		}
	}
	fresh, err := GetRuntimeHandle(handle.ID)
	if err != nil {
		return nil, "", err
	}
	return fresh, externalSessionID, nil
}

func SincronizarRuntimeHandleWorkingDir(handle *RuntimeHandle, runtime *RuntimeInstance, agente string, proyectoID *int64) (*RuntimeHandle, string, error) {
	current := ""
	if handle != nil {
		current = strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "working_dir", ""))
	}
	if current == "" && runtime != nil {
		current = strings.TrimSpace(runtime.CWD)
	}
	if proyectoID == nil && runtime != nil {
		proyectoID = runtime.ProyectoID
	}
	if proyectoID == nil || *proyectoID <= 0 {
		return handle, current, nil
	}
	proyecto, err := GetProyecto(jsonNumber(*proyectoID))
	if err != nil || proyecto == nil {
		return handle, current, err
	}
	preferred := RutaTrabajoPreferidaAgenteProyecto(strings.TrimSpace(agente), proyecto, current)
	if preferred == "" || preferred == current {
		return handle, preferred, nil
	}
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["working_dir"] = preferred
		metaJSON, _ := json.Marshal(meta)
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(metaJSON), handle.ID); err != nil {
			return nil, "", err
		}
	}
	if runtime != nil && runtime.ID > 0 && strings.TrimSpace(runtime.CWD) != preferred {
		if _, err := DB.Exec(`UPDATE runtime_instances SET cwd=?, last_event_at=CURRENT_TIMESTAMP WHERE id=?`, preferred, runtime.ID); err != nil {
			return nil, "", err
		}
	}
	if handle != nil && handle.SesionID != nil && *handle.SesionID > 0 {
		if _, err := DB.Exec(`UPDATE sesiones SET cwd=? WHERE id=?`, preferred, *handle.SesionID); err != nil {
			return nil, "", err
		}
	}
	if handle == nil {
		return nil, preferred, nil
	}
	fresh, err := GetRuntimeHandle(handle.ID)
	if err != nil {
		return nil, "", err
	}
	return fresh, preferred, nil
}

func runtimeHandleEffectiveExternalSessionID(handle *RuntimeHandle, runtime *RuntimeInstance) (string, error) {
	if handle == nil {
		return "", nil
	}
	if ext := strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "external_session_id", "")); ext != "" {
		return ext, nil
	}
	if runtime != nil && strings.TrimSpace(runtime.ExternalSessionID) != "" {
		return strings.TrimSpace(runtime.ExternalSessionID), nil
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		sesion, err := GetSesionByID(*handle.SesionID)
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		if sesion != nil && strings.TrimSpace(sesion.ExternalSessionID) != "" {
			return strings.TrimSpace(sesion.ExternalSessionID), nil
		}
	}
	if runtime != nil && runtime.ID > 0 {
		if sample, err := NormalizarUltimaMuestraRuntime(runtime.ID); err == nil && sample != nil && strings.TrimSpace(sample.SessionRef) != "" {
			return strings.TrimSpace(sample.SessionRef), nil
		}
	}
	obj := controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}
	if runtime != nil && runtime.PID != nil && *runtime.PID > 0 {
		obj.PID = runtime.PID
	}
	return controlruntime.DetectExternalSessionID(obj)
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
