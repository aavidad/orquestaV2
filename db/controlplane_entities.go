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
	"sort"
	"strconv"
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
	Limit      int
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

func runtimeHandleEsTMUXCanonico(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") {
		return true
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return true
	}
	return strings.TrimSpace(stringFromMap(meta, "tmux_session", "")) != "" ||
		strings.TrimSpace(stringFromMap(meta, "tmux_pane_id", "")) != ""
}

func runtimeHandleTMUXSessionRef(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	session := strings.TrimSpace(stringFromMap(meta, "tmux_session", ""))
	pane := strings.TrimSpace(stringFromMap(meta, "tmux_pane_id", ""))
	switch {
	case session != "" && pane != "":
		return session + "/" + pane
	case session != "":
		return session
	case pane != "":
		return pane
	case strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session"):
		return strings.TrimSpace(handle.HandleRef)
	default:
		return ""
	}
}

func runtimeHandleTMUXSessionRefObserved(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	if ref := runtimeHandleTMUXSessionRef(handle); ref != "" {
		return ref
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return ""
	}
	if ref := strings.TrimSpace(snap.RuntimeRef()); ref != "" {
		return ref
	}
	return strings.TrimSpace(snap.SessionRef())
}

func reconciliarTransporteSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.Transporte)
	}
	inferidoTransporte := strings.TrimSpace(inferido.Transporte)
	if existente == nil {
		return inferidoTransporte
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		return "tmux"
	}
	if inferidoTransporte != "" {
		return inferidoTransporte
	}
	return strings.TrimSpace(existente.Transporte)
}

func reconciliarHandleKindSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.HandleKind)
	}
	inferidoKind := strings.TrimSpace(inferido.HandleKind)
	if existente == nil {
		return inferidoKind
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		return "session"
	}
	existenteKind := strings.TrimSpace(existente.HandleKind)
	if inferidoKind == "process" {
		return "process"
	}
	if existenteKind != "" {
		return existenteKind
	}
	return inferidoKind
}

func reconciliarHandleRefSesion(existente *RuntimeHandle, inferido *RuntimeHandle) string {
	if inferido == nil {
		if existente == nil {
			return ""
		}
		return strings.TrimSpace(existente.HandleRef)
	}
	inferidoRef := strings.TrimSpace(inferido.HandleRef)
	if existente == nil {
		return inferidoRef
	}
	if runtimeHandleEsTMUXCanonico(existente) {
		if tmuxRef := runtimeHandleTMUXSessionRef(existente); tmuxRef != "" {
			return tmuxRef
		}
	}
	if strings.TrimSpace(inferido.HandleKind) == "process" && inferidoRef != "" {
		return inferidoRef
	}
	if existenteRef := strings.TrimSpace(existente.HandleRef); existenteRef != "" {
		return existenteRef
	}
	return inferidoRef
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
	if err == nil {
		runtimeHandleHotReset()
	}
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
	runtimeHandleHotReset()
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
	if filtro.CreatedBefore != nil && !filtro.CreatedBefore.IsZero() {
		query += ` AND created_at < ?`
		args = append(args, filtro.CreatedBefore.UTC())
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
	for _, chunk := range runtimeInt64Chunks(ids, 400) {
		argsDelete := int64SliceToAny(chunk)
		if _, err := tx.Exec(`UPDATE runtime_mailbox SET runtime_order_id = NULL WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(chunk))+`)`, argsDelete...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_orders WHERE id IN (`+runtimeSQLPlaceholders(len(chunk))+`)`, argsDelete...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
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

func PurgarRuntimeHistorico() (*PurgaRuntimeHistoricoResultado, error) {
	now := time.Now().UTC()
	handlesCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_handles_retention_minutes", "runtime_handles_retention_hours", 24)
	ordersCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_orders_retention_minutes", "runtime_orders_retention_hours", 72)
	runtimesCutoff := runtimeHistoricoRetentionCutoff(now, "runtime_instances_retention_minutes", "runtime_instances_retention_hours", 72)
	handleIDs, err := listarRuntimeHandlesPurgablesHistoricos([]string{"cerrado", "fallido"}, handlesCutoff)
	if err != nil {
		return nil, err
	}
	orderIDs, err := listarRuntimeOrdersPurgablesHistoricos([]string{"completada", "fallida", "expirada", "cancelada"}, ordersCutoff)
	if err != nil {
		return nil, err
	}
	runtimeIDs, err := listarRuntimeInstancesPurgablesHistoricos([]string{"cerrado", "bloqueado", "degradado"}, runtimesCutoff)
	if err != nil {
		return nil, err
	}
	if err := validarPurgadoRuntimeHandles(handleIDs); err != nil {
		return nil, err
	}
	if err := validarPurgadoRuntimeOrders(orderIDs); err != nil {
		return nil, err
	}
	if err := validarPurgadoRuntimeInstances(runtimeIDs); err != nil {
		return nil, err
	}
	result := &PurgaRuntimeHistoricoResultado{
		Handles:  &PurgaRuntimeHandlesResultado{Estados: []string{"cerrado", "fallido"}},
		Orders:   &PurgaRuntimeOrdersResultado{Estados: []string{"completada", "fallida", "expirada", "cancelada"}},
		Runtimes: &PurgaRuntimeInstancesResultado{Estados: []string{"cerrado", "bloqueado", "degradado"}},
	}
	if len(handleIDs) == 0 && len(orderIDs) == 0 && len(runtimeIDs) == 0 {
		return result, nil
	}
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	if len(handleIDs) > 0 {
		args := int64SliceToAny(handleIDs)
		if _, err := tx.Exec(`UPDATE runtime_orders SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE runtime_transcript SET handle_id = NULL WHERE handle_id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_handles WHERE id IN (`+runtimeSQLPlaceholders(len(handleIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Handles.Deleted = len(handleIDs)
		result.Handles.DeletedIDs = handleIDs
	}
	if len(orderIDs) > 0 {
		args := int64SliceToAny(orderIDs)
		if _, err := tx.Exec(`UPDATE runtime_mailbox SET runtime_order_id = NULL WHERE runtime_order_id IN (`+runtimeSQLPlaceholders(len(orderIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM runtime_orders WHERE id IN (`+runtimeSQLPlaceholders(len(orderIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Orders.Deleted = len(orderIDs)
		result.Orders.DeletedIDs = orderIDs
	}
	if len(runtimeIDs) > 0 {
		args := int64SliceToAny(runtimeIDs)
		if _, err := tx.Exec(`DELETE FROM runtime_instances WHERE id IN (`+runtimeSQLPlaceholders(len(runtimeIDs))+`)`, args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		result.Runtimes.Deleted = len(runtimeIDs)
		result.Runtimes.DeletedIDs = runtimeIDs
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func runtimeHistoricoRetentionCutoff(now time.Time, minutesKey, hoursKey string, fallbackHours int64) time.Time {
	now = now.UTC()
	if minutes := configInt64Fallback(minutesKey, -1); minutes >= 0 {
		cutoff := now.Add(-time.Duration(minutes) * time.Minute)
		if cutoff.Before(now) {
			return cutoff
		}
		return now
	}
	hours := configInt64Fallback(hoursKey, fallbackHours)
	if hours <= 0 {
		hours = fallbackHours
	}
	cutoff := now.Add(-time.Duration(hours) * time.Hour)
	if cutoff.Before(now) {
		return cutoff
	}
	return now.Add(-time.Duration(fallbackHours) * time.Hour)
}

func listarRuntimeHandlesPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_handles WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND COALESCE(last_seen_at, updated_at, created_at) < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listarRuntimeOrdersPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_orders WHERE estado IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND created_at < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listarRuntimeInstancesPurgablesHistoricos(estados []string, cutoff time.Time) ([]int64, error) {
	query := `SELECT id FROM runtime_instances WHERE logical_state IN (` + runtimeSQLPlaceholders(len(estados)) + `)
		AND COALESCE(last_heartbeat_at, last_event_at, updated_at, created_at) < ?
		ORDER BY id DESC`
	args := make([]any, 0, len(estados)+1)
	for _, estado := range estados {
		args = append(args, estado)
	}
	args = append(args, cutoff.UTC())
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
	var vivos []string
	for _, chunk := range runtimeInt64Chunks(ids, 400) {
		rows, err := DB.Query(`
			SELECT id, estado FROM runtime_orders
			WHERE id IN (`+runtimeSQLPlaceholders(len(chunk))+`)
			  AND estado IN ('pendiente','tomada','ejecutando')`,
			int64SliceToAny(chunk)...,
		)
		if err != nil {
			return err
		}
		for rows.Next() {
			var (
				id     int64
				estado string
			)
			if err := rows.Scan(&id, &estado); err != nil {
				rows.Close()
				return err
			}
			vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(estado)))
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime orders vivas: %s", strings.Join(vivos, ", "))
	}
	return nil
}

func runtimeInt64Chunks(ids []int64, chunkSize int) [][]int64 {
	if len(ids) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 400
	}
	out := make([][]int64, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

func validarPurgadoRuntimeInstances(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	args := int64SliceToAny(ids)
	rows, err := DB.Query(`
		SELECT id, logical_state FROM runtime_instances
		WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND logical_state NOT IN ('cerrado','bloqueado','degradado')`,
		args...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vivos []string
	for rows.Next() {
		var (
			id           int64
			logicalState string
		)
		if err := rows.Scan(&id, &logicalState); err != nil {
			return err
		}
		vivos = append(vivos, fmt.Sprintf("#%d(%s)", id, strings.TrimSpace(logicalState)))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(vivos) > 0 {
		return fmt.Errorf("no se pueden purgar runtime instances vivas: %s", strings.Join(vivos, ", "))
	}
	rows, err = DB.Query(`
		SELECT id, estado FROM runtime_handles
		WHERE runtime_id IN (`+runtimeSQLPlaceholders(len(ids))+`)
		  AND estado IN ('activo','pausado')`,
		args...,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	vivos = vivos[:0]
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
		return fmt.Errorf("no se pueden purgar runtime instances con handles vivos asociados: %s", strings.Join(vivos, ", "))
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
		h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE sesion_id = ? ORDER BY id DESC LIMIT 1`, sesionID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		runtimeHandleHotRemember(h)
		return reconciliarMetadataRuntimeHandleLeida(h)
	})
}

func GetRuntimeHandle(id int64) (*RuntimeHandle, error) {
	return consultarConReintentos(func() (*RuntimeHandle, error) {
		h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE id = ?`, id)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return reconciliarMetadataRuntimeHandleLeida(h)
	})
}

func ActualizarMetadataRuntimeHandle(id int64, metadataJSON string) error {
	if id <= 0 {
		return nil
	}
	_, err := consultarConReintentos(func() (struct{}, error) {
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, strings.TrimSpace(metadataJSON), id); err != nil {
			return struct{}{}, err
		}
		runtimeHandleHotReset()
		return struct{}{}, nil
	})
	return err
}

func GetRuntimeHandleActivoAgente(agente string) (*RuntimeHandle, error) {
	agente = strings.TrimSpace(agente)
	if cached := runtimeHandleHotLookup(agente, nil); cached != nil {
		validado, err := validarRuntimeHandleActivoSinFallback(cached)
		if err != nil {
			return nil, err
		}
		if validado != nil {
			return validado, nil
		}
	}
	handle, err := seleccionarRuntimeHandleActivo(agente, nil)
	if err == nil && handle != nil {
		runtimeHandleHotRemember(handle)
	}
	return handle, err
}

func GetRuntimeHandleActivoAgenteProyecto(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	agente = strings.TrimSpace(agente)
	if cached := runtimeHandleHotLookup(agente, proyectoID); cached != nil {
		validado, err := validarRuntimeHandleActivoSinFallback(cached)
		if err != nil {
			return nil, err
		}
		if validado != nil {
			return validado, nil
		}
	}
	handle, err := seleccionarRuntimeHandleActivo(agente, proyectoID)
	if err == nil && handle != nil {
		runtimeHandleHotRemember(handle)
	}
	return handle, err
}

func seleccionarRuntimeHandleActivo(agente string, proyectoID *int64) (*RuntimeHandle, error) {
	handles, err := listarRuntimeHandlesActivosCandidatos(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	validos := make([]*RuntimeHandle, 0, len(handles))
	for _, handle := range handles {
		validado, err := validarRuntimeHandleActivoSinFallback(handle)
		if err != nil {
			return nil, err
		}
		if validado != nil {
			validos = append(validos, validado)
		}
	}
	canonicos := make([]*RuntimeHandle, 0, len(validos))
	for _, handle := range validos {
		if runtimeHandleExcluidoDelActivoCanonico(handle) {
			continue
		}
		canonicos = append(canonicos, handle)
	}
	if len(canonicos) > 0 {
		preferido := elegirRuntimeHandleActivoPreferente(canonicos)
		if preferido != nil {
			if err := cerrarRuntimeHandlesActivosSuperseded(preferido, validos); err != nil {
				return nil, err
			}
			return GetRuntimeHandle(preferido.ID)
		}
	}
	return recuperarRuntimeHandleVivoAgenteProyecto(agente, proyectoID)
}

func runtimeHandleExcluidoDelActivoCanonico(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if runtimeHandleUsaLegacyProcessPTY(mapFromJSON(handle.MetadataJSON)) {
		return true
	}
	return false
}

func elegirRuntimeHandleActivoPreferente(handles []*RuntimeHandle) *RuntimeHandle {
	if len(handles) == 0 {
		return nil
	}
	now := time.Now().UTC()
	best := handles[0]
	bestInfo := runtimeHandleWorkerInfoFor(best, now)
	for _, candidate := range handles[1:] {
		candidateInfo := runtimeHandleWorkerInfoFor(candidate, now)
		if runtimeHandlePreferibleConInfo(candidate, candidateInfo, best, bestInfo, now) {
			best = candidate
			bestInfo = candidateInfo
		}
	}
	return best
}

func runtimeHandlePreferible(candidate, current *RuntimeHandle, now time.Time) bool {
	return runtimeHandlePreferibleConInfo(
		candidate,
		runtimeHandleWorkerInfoFor(candidate, now),
		current,
		runtimeHandleWorkerInfoFor(current, now),
		now,
	)
}

type runtimeHandleWorkerInfo struct {
	priority   int
	structured bool
	fresh      bool
}

func runtimeHandlePreferibleConInfo(candidate *RuntimeHandle, candidateInfo runtimeHandleWorkerInfo, current *RuntimeHandle, currentInfo runtimeHandleWorkerInfo, now time.Time) bool {
	if candidateInfo.priority != currentInfo.priority {
		return candidateInfo.priority > currentInfo.priority
	}
	if candidateInfo.fresh != currentInfo.fresh {
		return candidateInfo.fresh
	}
	if candidateInfo.structured != currentInfo.structured {
		return candidateInfo.structured
	}
	candidateActivo := strings.TrimSpace(candidate.Estado) == "activo"
	currentActivo := strings.TrimSpace(current.Estado) == "activo"
	if candidateActivo != currentActivo {
		return candidateActivo
	}
	candidateSeen := runtimeHandleRecency(candidate)
	currentSeen := runtimeHandleRecency(current)
	if !candidateSeen.Equal(currentSeen) {
		return candidateSeen.After(currentSeen)
	}
	return candidate.ID > current.ID
}

func runtimeHandleWorkerInfoFor(handle *RuntimeHandle, now time.Time) runtimeHandleWorkerInfo {
	if handle == nil {
		return runtimeHandleWorkerInfo{}
	}
	info := runtimeHandleWorkerInfo{}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return info
	}
	driver := strings.ToLower(strings.TrimSpace(snap.Driver()))
	if driver == "" {
		driver = strings.ToLower(strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "driver", "")))
	}
	if driver != "process_pty_cli" {
		info.structured = true
	}
	info.fresh = snap.Alive() && !snap.IsHeartbeatStale(now, time.Minute)
	switch driver {
	case "tmux_cli_session":
		if info.fresh {
			info.priority = 40
		} else {
			info.priority = 30
		}
	case "process_pty_cli":
		if info.fresh {
			info.priority = 0
		} else {
			info.priority = -10
		}
	default:
		if info.fresh {
			info.priority = 5
		} else {
			info.priority = 1
		}
	}
	return info
}

func runtimeHandleWorkerPriority(handle *RuntimeHandle, now time.Time) int {
	return runtimeHandleWorkerInfoFor(handle, now).priority
}

func runtimeHandleRecency(handle *RuntimeHandle) time.Time {
	if handle == nil {
		return time.Time{}
	}
	if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() {
		return handle.LastSeenAt.UTC()
	}
	if !handle.UpdatedAt.IsZero() {
		return handle.UpdatedAt.UTC()
	}
	return handle.CreatedAt.UTC()
}

func runtimeHandleTieneWorkerEstructurado(handle *RuntimeHandle) bool {
	return runtimeHandleWorkerInfoFor(handle, time.Now().UTC()).structured
}

func runtimeHandleTieneWorkerEstructuradoFresco(handle *RuntimeHandle, now time.Time) bool {
	return runtimeHandleWorkerInfoFor(handle, now).fresh
}

func cerrarRuntimeHandlesActivosSuperseded(preferido *RuntimeHandle, candidates []*RuntimeHandle) error {
	now := time.Now().UTC()
	if preferido == nil || !runtimeHandleWorkerInfoFor(preferido, now).fresh {
		return nil
	}
	for _, handle := range candidates {
		if handle == nil || handle.ID == preferido.ID {
			continue
		}
		if err := marcarRuntimeHandleSuperseded(handle); err != nil {
			return err
		}
	}
	return nil
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
	if runtimeHandleExcluidoDelActivoCanonico(handle) {
		return nil, nil
	}
	estadoPersistido := strings.ToLower(strings.TrimSpace(handle.Estado))
	if estadoPersistido == "pausado" || estadoPersistido == "cerrado" {
		return nil, nil
	}
	if !runtimeHandleSePuedeValidarLocalmente(handle) {
		return normalizarRuntimeHandleTMUXCanonico(handle)
	}
	if superseded, err := supersedeRuntimeHandleSiHaceFalta(handle); err != nil {
		return nil, err
	} else if superseded {
		return nil, nil
	}
	if fresh, estado, pid := runtimeHandleReviveStateFromStructuredWorker(handle, time.Now().UTC()); fresh {
		if estado == "pausado" {
			if _, err := revivirRuntimeHandle(handle, estado, pid); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return normalizarRuntimeHandleTMUXCanonico(handle)
	}
	if runtimeHandleConfiaEstadoReciente(handle, time.Now().UTC()) {
		return normalizarRuntimeHandleTMUXCanonico(handle)
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
	if superseded, err := supersedeRuntimeHandleSiHaceFalta(handle); err != nil {
		return nil, err
	} else if superseded {
		return nil, nil
	}
	if fresh, estado, pid := runtimeHandleReviveStateFromStructuredWorker(handle, time.Now().UTC()); fresh {
		return revivirRuntimeHandle(handle, estado, pid)
	}
	return refrescarRuntimeHandleSiSigueVivo(handle)
}

func refrescarRuntimeHandleSiSigueVivo(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	if fresh, estado, pid := runtimeHandleReviveStateFromStructuredWorker(handle, now); fresh {
		return revivirRuntimeHandle(handle, estado, pid)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandleUsaLegacyProcessPTY(meta) {
		return nil, nil
	}
	runtime, _ := runtimeHandleRuntime(handle)
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	vivo, _, err := controlruntime.ProcesoVivo(obj)
	if err != nil || !vivo {
		return nil, nil
	}
	estado := estadoReviveRuntimeHandle(handle)
	return revivirRuntimeHandle(handle, estado, obj.PID)
}

func revivirRuntimeHandle(handle *RuntimeHandle, estado string, pid *int64) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	normalizado, err := normalizarRuntimeHandleTMUXCanonico(handle)
	if err != nil {
		return nil, err
	}
	if normalizado != nil {
		handle = normalizado
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, estado, handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	if runtime, err := runtimeHandleRuntime(handle); err == nil && runtime != nil {
		processState := runtime.ProcessState
		if strings.TrimSpace(processState) == "" || strings.EqualFold(strings.TrimSpace(processState), "fallido") {
			processState = "running"
		}
		logicalState := runtime.LogicalState
		if strings.TrimSpace(logicalState) == "" || strings.EqualFold(strings.TrimSpace(logicalState), "fallido") {
			logicalState = "esperando_io"
		}
		if workerLogicalState, workerProcessState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle); ok {
			logicalState = workerLogicalState
			if strings.TrimSpace(workerProcessState) != "" {
				processState = workerProcessState
			}
		}
		if estado == "pausado" {
			processState = "stopped"
			logicalState = "pausado"
		}
		var runtimePID *int64
		if pid != nil && *pid > 0 {
			runtimePID = pid
		}
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET pid = COALESCE(pid, ?),
			    logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			runtimePID, logicalState, processState, runtime.ID,
		); err != nil {
			return nil, err
		}
	}
	return GetRuntimeHandle(handle.ID)
}

func runtimeHandleReviveStateFromStructuredWorker(handle *RuntimeHandle, now time.Time) (bool, string, *int64) {
	if handle == nil {
		return false, "", nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false, "", nil
	}
	view := snap.View(now, time.Minute)
	if view == nil {
		return false, "", nil
	}
	if strings.EqualFold(strings.TrimSpace(view.Driver), "process_pty_cli") && runtimeHandleUsaLegacyProcessPTY(meta) {
		return false, "", nil
	}
	if view.HeartbeatStale || !view.Alive {
		return false, "", nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "", "failed", "stopped", "exited", "closed", "stale":
		return false, "", nil
	}
	estado := estadoReviveRuntimeHandle(handle)
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "paused", "pausado":
		estado = "pausado"
	case "running":
		estado = "activo"
	}
	var pid *int64
	if view.ChildPID > 0 {
		childPID := int64(view.ChildPID)
		pid = &childPID
	}
	return true, estado, pid
}

func runtimeHandleConfiaEstadoReciente(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.ToLower(strings.TrimSpace(stringFromMap(meta, "driver", "")))
	if driver != "tmux_cli_session" && !strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") && !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return false
	}
	if runtimeHandleUsaLegacyCLITMUXPreferred(meta) {
		return false
	}
	recency := runtimeHandleRecency(handle)
	if recency.IsZero() || now.Sub(recency.UTC()) > time.Minute {
		return false
	}
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil {
		return false
	}
	if runtime == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "fallido", "cerrado", "finalizado":
		return false
	default:
		return true
	}
}

func supersedeRuntimeHandleSiHaceFalta(handle *RuntimeHandle) (bool, error) {
	if handle == nil {
		return false, nil
	}
	superseded, err := runtimeHandleDebeCederAWorkerEstructurado(handle)
	if err != nil {
		return false, err
	}
	if !superseded {
		return false, nil
	}
	if err := marcarRuntimeHandleSuperseded(handle); err != nil {
		return false, err
	}
	return true, nil
}

func runtimeHandleDebeCederAWorkerEstructurado(handle *RuntimeHandle) (bool, error) {
	if handle == nil {
		return false, nil
	}
	now := time.Now().UTC()
	currentInfo := runtimeHandleWorkerInfoFor(handle, now)
	candidatos, err := listarRuntimeHandlesActivosCandidatos(handle.Agente, handle.ProyectoID)
	if err != nil {
		return false, err
	}
	for _, candidato := range candidatos {
		if candidato == nil || candidato.ID == handle.ID {
			continue
		}
		candidateInfo := runtimeHandleWorkerInfoFor(candidato, now)
		if !candidateInfo.fresh {
			continue
		}
		if runtimeHandlePreferibleConInfo(candidato, candidateInfo, handle, currentInfo, now) {
			return true, nil
		}
	}
	return false, nil
}

func marcarRuntimeHandleSuperseded(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if err := detenerRuntimeHandleObsoleto(handle); err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE id = ?`, handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
	runtime, err := runtimeHandleRuntime(handle)
	if err != nil || runtime == nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='cerrado',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE id = ?`, runtime.ID); err != nil {
		return err
	}
	return nil
}

func runtimeHandleSePuedeValidarLocalmente(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	if runtimeHandleUsaLegacyProcessPTY(mapFromJSON(handle.MetadataJSON)) {
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
	if err := detenerRuntimeHandleObsoleto(handle); err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP
		WHERE id = ?`, handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
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

func detenerRuntimeHandleObsoleto(handle *RuntimeHandle) error {
	if handle == nil {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") ||
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "mcp_http") {
		return nil
	}
	runtime, _ := runtimeHandleRuntime(handle)
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	if detenido, err := controlruntime.DetenerSesionTMUXMetadata(obj.MetadataJSON); detenido || err != nil {
		return err
	}
	if runtimeHandleObsoletoRefiereProcesoActual(obj) {
		return nil
	}
	aplicado, pid, err := controlruntime.DetenerProceso(obj)
	if err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
		return err
	}
	if !aplicado {
		if resolvedPID, ok, resolveErr := controlruntime.ResolverPID(obj); resolveErr != nil && !errorDetenerSesionObsoletaIgnorable(resolveErr) {
			return resolveErr
		} else if ok && resolvedPID > 0 {
			if runtimeHandleObsoletoRefierePIDActual(int64(resolvedPID)) {
				return nil
			}
			fallbackPID := int64(resolvedPID)
			aplicado, pid, err = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: &fallbackPID})
			if err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
				return err
			}
		}
	}
	if pid > 0 {
		pid64 := int64(pid)
		for i := 0; i < 5; i++ {
			vivo, _, aliveErr := controlruntime.ProcesoVivo(controlruntime.ObjetivoProceso{PID: &pid64})
			if aliveErr != nil {
				if errorDetenerSesionObsoletaIgnorable(aliveErr) {
					return nil
				}
				return aliveErr
			}
			if !vivo {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

func runtimeHandleObsoletoRefiereProcesoActual(obj controlruntime.ObjetivoProceso) bool {
	meta := mapFromJSON(obj.MetadataJSON)
	looksLikeTMUX := strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") ||
		strings.TrimSpace(stringFromMap(meta, "tmux_session", "")) != "" ||
		strings.TrimSpace(stringFromMap(meta, "tmux_pane_id", "")) != ""
	if obj.PID != nil &&
		runtimeHandleObsoletoRefierePIDActual(*obj.PID) &&
		!looksLikeTMUX {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(obj.HandleKind), "process") {
		if pid, err := strconv.ParseInt(strings.TrimSpace(obj.HandleRef), 10, 64); err == nil && runtimeHandleObsoletoRefierePIDActual(pid) {
			return true
		}
	}
	return false
}

func runtimeHandleObsoletoRefierePIDActual(pid int64) bool {
	return pid > 0 && pid == int64(os.Getpid())
}

func errorDetenerSesionObsoletaIgnorable(err error) bool {
	if err == nil {
		return true
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case raw == "":
		return true
	case strings.Contains(raw, "can't find session"):
		return true
	case strings.Contains(raw, "no server running"):
		return true
	case strings.Contains(raw, "failed to connect to server"):
		return true
	case strings.Contains(raw, "no such process"):
		return true
	case strings.Contains(raw, "process already finished"):
		return true
	case strings.Contains(raw, "os: process already finished"):
		return true
	default:
		return false
	}
}

func estadoReviveRuntimeHandle(handle *RuntimeHandle) string {
	if handle == nil {
		return "activo"
	}
	if handle.SesionID != nil {
		if sesion, err := sesionIfExists(handle.SesionID); err == nil && sesion != nil {
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
	if handle == nil {
		return nil, nil
	}
	if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtime, err := GetRuntime(*handle.RuntimeID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if runtime != nil {
			return runtime, nil
		}
	}
	if handle.SesionID != nil && *handle.SesionID > 0 {
		runtime, err := GetRuntimeBySesionID(*handle.SesionID)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return runtime, err
	}
	return nil, nil
}

func ListarRuntimeHandles(agente *string) ([]*RuntimeHandle, error) {
	if agente != nil && strings.TrimSpace(*agente) != "" {
		if _, err := GetRuntimeHandleCanonicoRecienteAgente(strings.TrimSpace(*agente)); err != nil {
			return nil, err
		}
	}
	handles, err := listarRuntimeHandlesLeidos(agente)
	if err != nil {
		return nil, err
	}
	if err := reconciliarRuntimeHandlesInvalidosEnLista(handles); err != nil {
		return nil, err
	}
	if err := reconciliarRuntimeHandlesDuplicadosEnLista(handles); err != nil {
		return nil, err
	}
	return handles, nil
}

func ListarRuntimeHandlesPasivos(agente *string) ([]*RuntimeHandle, error) {
	return listarRuntimeHandlesLeidos(agente)
}

func listarRuntimeHandlesLeidos(agente *string) ([]*RuntimeHandle, error) {
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func reconciliarRuntimeHandlesDuplicadosEnLista(handles []*RuntimeHandle) error {
	if len(handles) < 2 {
		return nil
	}
	grupos := map[string][]*RuntimeHandle{}
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		estado := strings.TrimSpace(handle.Estado)
		if estado != "activo" && estado != "pausado" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(handle.Agente))
		grupos[key] = append(grupos[key], handle)
	}
	now := time.Now().UTC()
	for _, grupo := range grupos {
		if len(grupo) < 2 {
			continue
		}
		preferido := elegirRuntimeHandleActivoPreferente(grupo)
		if preferido == nil || !runtimeHandleWorkerInfoFor(preferido, now).fresh {
			continue
		}
		ids := make([]int64, 0, len(grupo)-1)
		for _, handle := range grupo {
			if handle == nil || handle.ID == preferido.ID {
				continue
			}
			ids = append(ids, handle.ID)
		}
		if len(ids) == 0 {
			continue
		}
		args := make([]any, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		if _, err := DB.Exec(`
			UPDATE runtime_handles
			SET estado='cerrado',
			    last_seen_at=CURRENT_TIMESTAMP
			WHERE id IN (`+runtimeSQLPlaceholders(len(ids))+`) AND estado IN ('activo','pausado')`, args...); err != nil {
			return err
		}
		runtimeHandleHotReset()
		for _, handle := range grupo {
			if handle == nil || handle.ID == preferido.ID {
				continue
			}
			handle.Estado = "cerrado"
			ts := now
			handle.LastSeenAt = &ts
		}
	}
	return nil
}

func reconciliarRuntimeHandlesInvalidosEnLista(handles []*RuntimeHandle) error {
	if len(handles) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		switch strings.TrimSpace(handle.Estado) {
		case "activo", "pausado":
		default:
			continue
		}
		if !runtimeHandleExternalSessionIncompatible(handle) {
			continue
		}
		if err := MarcarRuntimeHandleCanalRoto(handle, nil, "external_session_id incompatible with tmux premium runtime"); err != nil {
			return err
		}
		handle.Estado = "fallido"
		ts := now
		handle.LastSeenAt = &ts
	}
	return nil
}

func ListarRuntimeHandlesParaTranscript() ([]*RuntimeHandle, error) {
	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado IN ('activo','pausado','fallido')
		  AND metadata_json <> ''
		  AND metadata_json LIKE '%log_path%'
		ORDER BY id DESC`)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func ListarRuntimeHandlesParaPresupuesto() ([]*RuntimeHandle, error) {
	if snapshot, err := runtimeHandleHotSnapshotCanonico(); err == nil && len(snapshot) > 0 {
		out := make([]*RuntimeHandle, 0, len(snapshot))
		for _, handle := range snapshot {
			if handle == nil || strings.TrimSpace(handle.MetadataJSON) == "" {
				continue
			}
			compacted, compactErr := compactarMetadataRuntimeHandleEnMemoria(handle)
			if compactErr != nil {
				return nil, compactErr
			}
			out = append(out, compacted)
		}
		sort.SliceStable(out, func(i, j int) bool {
			left := preferTime(out[i].LastSeenAt, &out[i].UpdatedAt, &out[i].CreatedAt)
			right := preferTime(out[j].LastSeenAt, &out[j].UpdatedAt, &out[j].CreatedAt)
			switch {
			case left == nil && right == nil:
				return out[i].ID > out[j].ID
			case left == nil:
				return false
			case right == nil:
				return true
			case !left.Equal(*right):
				return left.After(*right)
			default:
				return out[i].ID > out[j].ID
			}
		})
		return out, nil
	}

	rows, err := DB.Query(runtimeHandleSelectBase() + `
		WHERE estado IN ('activo','pausado')
		  AND metadata_json <> ''
		ORDER BY id DESC`)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i, h := range out {
		out[i], err = compactarMetadataRuntimeHandleEnMemoria(h)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func getRuntimeHandleByQuery(query string, args ...any) (*RuntimeHandle, error) {
	row := DB.QueryRow(query, args...)
	return scanRuntimeHandle(row)
}

func getRuntimeHandleRaw(id int64) (*RuntimeHandle, error) {
	h, err := getRuntimeHandleByQuery(runtimeHandleSelectBase()+` WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func reconciliarMetadataRuntimeHandleLeida(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	refreshed, err := compactarMetadataHandleRuntimePersistida(handle)
	if err != nil {
		return nil, err
	}
	return normalizarRuntimeHandleTMUXCanonicoEnMemoria(refreshed), nil
}

func compactarMetadataRuntimeHandleEnMemoria(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil {
		return nil, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return handle, nil
	}
	before, _ := json.Marshal(meta)
	compactarMetadataRuntimeHandle(meta)
	after, _ := json.Marshal(meta)
	if string(before) == string(after) {
		return normalizarRuntimeHandleTMUXCanonicoEnMemoria(handle), nil
	}
	clone := *handle
	clone.MetadataJSON = string(after)
	return normalizarRuntimeHandleTMUXCanonicoEnMemoria(&clone), nil
}

func normalizarRuntimeHandleTMUXCanonicoEnMemoria(handle *RuntimeHandle) *RuntimeHandle {
	if handle == nil {
		return nil
	}
	if !runtimeHandleEsTMUXCanonico(handle) {
		return handle
	}
	canonicalRef := runtimeHandleTMUXSessionRefObserved(handle)
	if canonicalRef == "" {
		return handle
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		strings.TrimSpace(handle.HandleRef) == canonicalRef {
		return handle
	}
	clone := *handle
	clone.Transporte = "tmux"
	clone.HandleKind = "session"
	clone.HandleRef = canonicalRef
	return &clone
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
	id, err := res.LastInsertId()
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
		!runtimeHandleUsaLegacyProcessPTY(meta) {
		return true
	}
	return false
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

func ListarRuntimeOrders(filter FiltroRuntimeOrders) ([]*RuntimeOrder, error) {
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
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
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

func GetRuntimeMailbox(id int64) (*RuntimeMailboxMessage, error) {
	row := DB.QueryRow(runtimeMailboxSelectBase()+`
		WHERE id = ?
		LIMIT 1`, id)
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

func RearmarRuntimeMailboxPendiente(id int64) error {
	_, err := DB.Exec(`
		UPDATE runtime_mailbox
		SET estado='pendiente',
		    delivered_at=NULL,
		    consumed_at=NULL
		WHERE id = ?
		  AND estado='entregado'`, id)
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
		if superseded, err := supersedeRuntimeHandleSiHaceFalta(handle); err != nil {
			return 0, err
		} else if superseded {
			reconciled++
			continue
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
			runtimeHandleHotReset()
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
	candidates := make([]*RuntimeOrder, 0, 16)
	for _, estado := range []string{"tomada", "ejecutando"} {
		estado := estado
		orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
		if err != nil {
			return 0, err
		}
		for _, order := range orders {
			if order == nil {
				continue
			}
			if order.LeaseExpiresAt != nil && !order.LeaseExpiresAt.After(now) {
				candidates = append(candidates, order)
				continue
			}
			if runtimeOrderAttemptReference(order).After(cutoff) {
				continue
			}
			candidates = append(candidates, order)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	recovered := 0
	for _, item := range candidates {
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
	reconciled, err := reconciliarRuntimeOrdersPendientesMailboxConsumido()
	if err != nil {
		return 0, err
	}
	controlObsoletas, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		return reconciled, err
	}
	processed, err := procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDespachables())
	if err != nil {
		return reconciled + controlObsoletas + processed, err
	}
	promoted, err := procesarBootstrapRuntimeOrdersActivosBatch()
	if err != nil {
		return reconciled + controlObsoletas + processed + promoted, err
	}
	deferred, err := procesarRuntimeOrdersBatchTipos(runtimeOrderTiposDiferibles())
	return reconciled + controlObsoletas + processed + promoted + deferred, err
}
func ProcesarHigieneRuntimesAutonomosBatch() (int, error) {
	limit := configIntOrDefault("runtime_hygiene_autonomo_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	rows, err := DB.Query(runtimeHandleSelectBase()+`
		WHERE estado = 'activo'
		  AND metadata_json LIKE '%"tarea_id":%'
		ORDER BY COALESCE(last_seen_at, updated_at, created_at) ASC
		LIMIT ?`, limit)
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
		meta := mapFromJSON(handle.MetadataJSON)
		tareaID := int64FromMap(meta, "tarea_id")
		if tareaID <= 0 {
			continue
		}
		tarea, err := GetTarea(tareaID)
		if err != nil || tarea == nil {
			continue
		}
		if tarea.Estado == TareaCompletada || tarea.Estado == TareaCancelada {
			lastSeen := handle.LastSeenAt
			if lastSeen == nil {
				lastSeen = &handle.UpdatedAt
			}
			idleLimit := time.Duration(configIntOrDefault("runtime_autonomo_idle_timeout_seconds", 300)) * time.Second
			if time.Since(*lastSeen) > idleLimit {
				order := &RuntimeOrder{
					Agente:      strings.TrimSpace(handle.Agente),
					ProyectoID:  handle.ProyectoID,
					Tipo:        "stop",
					PayloadJSON: `{"reason": "higiene_autonoma:tarea_finalizada", "source": "server"}`,
					Estado:      "pending",
				}
				if _, err := EncolarRuntimeOrder(order); err != nil {
					return processed, err
				}
				Audit("server", "runtime_higiene_autonomo_stop_encolado", "handle", handle.ID, "agente: "+handle.Agente)
				processed++
			}
		}
	}
	return processed, nil
}

func ProcesarRuntimeSupervisionBatch() (int, error) {
	limit := configIntOrDefault("runtime_supervision_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	intervalSeconds := configIntOrDefault("runtime_supervision_interval_seconds", 60)
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	if intervalSeconds < 60 {
		intervalSeconds = 60
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
		runtime, err := runtimeHandleRuntime(handle)
		if err != nil {
			return processed, err
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
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
	if err != nil {
		return 0, err
	}
	tiposWanted := make(map[string]struct{}, len(tipos))
	for _, tipo := range tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo != "" {
			tiposWanted[tipo] = struct{}{}
		}
	}
	now := time.Now().UTC()
	ids := make([]int64, 0, limit)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	for _, order := range orders {
		if order == nil || !runtimeOrderPendingReady(order, now) {
			continue
		}
		if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; !ok {
			continue
		}
		ids = append(ids, order.ID)
		if len(ids) >= limit {
			break
		}
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

func reconciliarRuntimeOrdersPendientesMailboxConsumido() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, limit)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	for _, order := range orders {
		if order == nil || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		ids = append(ids, order.ID)
		if len(ids) >= limit {
			break
		}
	}

	processed := 0
	for _, id := range ids {
		order, err := GetRuntimeOrder(id)
		if err != nil {
			return processed, err
		}
		if order == nil || !strings.EqualFold(strings.TrimSpace(order.Estado), "pendiente") || strings.TrimSpace(order.Tipo) != "send_instruction" {
			continue
		}
		payload := mapFromJSON(order.PayloadJSON)
		if !runtimeOrderSendInstructionProvieneMailbox(payload) {
			continue
		}
		mailboxID := runtimeOrderSendInstructionMailboxID(payload)
		if mailboxID <= 0 {
			continue
		}
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return processed, err
		}
		if msg == nil {
			if err := retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "mailbox_missing_rearmed"); err != nil {
				return processed, err
			}
			processed++
			continue
		}
		switch strings.ToLower(strings.TrimSpace(msg.Estado)) {
		case "pendiente":
			continue
		case "entregado":
			if handled, err := reconciliarRuntimeOrderSendInstructionMailboxEntregado(order, payload, msg, time.Now().UTC()); err != nil {
				return processed, err
			} else if handled {
				processed++
				continue
			}
		}
		if err := completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "mailbox ya "+strings.TrimSpace(msg.Estado)); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func reconciliarRuntimeOrdersPendientesControlObsoletas() (int, error) {
	limit := configIntOrDefault("runtime_order_batch_size", 10)
	if limit <= 0 {
		limit = 10
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	ids := make([]int64, 0, limit)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID < orders[j].ID })
	for _, order := range orders {
		if order == nil || !runtimeOrderPendingReady(order, now) {
			continue
		}
		switch strings.TrimSpace(order.Tipo) {
		case "start", "resume", "stop":
			ids = append(ids, order.ID)
		default:
			continue
		}
		if len(ids) >= limit {
			break
		}
	}
	processed := 0
	for _, id := range ids {
		order, err := GetRuntimeOrder(id)
		if err != nil {
			return processed, err
		}
		if order == nil || !strings.EqualFold(strings.TrimSpace(order.Estado), "pendiente") {
			continue
		}
		runtime, handle, err := resolverDestinoRuntimeOrderCanonico(order, nil, nil)
		if err != nil {
			return processed, err
		}
		reason := ""
		if satisfied, satisfiedReason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle); satisfied {
			reason = satisfiedReason
		} else {
			if !runtimeOrderControlObsoletaPorWorkerRecuperado(order, runtime, handle) {
				continue
			}
			reason = "runtime_worker_recovered_after_order"
		}
		if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtime, handle); err != nil {
			return processed, err
		}
		resultado := map[string]any{
			"ok":       true,
			"obsoleta": true,
			"reason":   reason,
		}
		if runtime != nil && runtime.ID > 0 {
			resultado["runtime_id"] = runtime.ID
		}
		if handle != nil && handle.ID > 0 {
			resultado["handle_id"] = handle.ID
		}
		data, _ := json.Marshal(resultado)
		if err := MarcarRuntimeOrderEstado(order.ID, "completada", string(data), ""); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func runtimeOrderControlEstadoDeseadoSatisfecho(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (bool, string) {
	if order == nil {
		return false, ""
	}
	orderType := strings.ToLower(strings.TrimSpace(order.Tipo))
	now := time.Now().UTC()
	switch orderType {
	case "start", "resume":
		if handle != nil {
			switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
			case "pausado", "fallido", "cerrado", "fantasma":
				return false, ""
			}
			if runtimeHandleSesionTMUXAusente(handle) {
				return false, ""
			}
		}
		if runtime != nil {
			switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
			case "pausado", "stopped", "fallido", "degradado", "cerrado":
				return false, ""
			}
			switch strings.ToLower(strings.TrimSpace(runtime.ProcessState)) {
			case "fallido", "crashed", "exited", "remote_status_error":
				return false, ""
			}
		}
		if runtimeHandleActivoAPICompartido(handle, runtime) {
			return true, "runtime_handle_api_active"
		}
		if runtimeWorkerSnapshotSatisfaceControlActual(orderType, handle, runtimeWorkerSnapshot(handle, runtime), now) {
			return true, "runtime_worker_already_running"
		}
	case "stop":
		if runtimeWorkerSnapshotSatisfaceControlActual(orderType, handle, runtimeWorkerSnapshot(handle, runtime), now) {
			return true, "runtime_worker_already_stopped"
		}
		if handle == nil && runtime == nil {
			return true, "runtime_already_stopped"
		}
		if handle != nil {
			switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
			case "cerrado", "fallido":
				return true, "runtime_handle_already_closed"
			case "pausado":
				return false, ""
			}
		}
		if runtime != nil {
			switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
			case "cerrado", "stopped", "fallido":
				return true, "runtime_already_stopped"
			}
		}
	}
	return false, ""
}

func runtimeOrderPromoverEstadoObservadoSiSatisfecha(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if order == nil || runtime == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "start", "resume":
		return runtimePromoverEstadoObservadoDesdeHandle(handle, runtime)
	default:
		return nil
	}
}

func runtimePromoverEstadoObservadoDesdeHandle(handle *RuntimeHandle, runtime *RuntimeInstance) error {
	if runtime == nil {
		return nil
	}
	logicalState := strings.TrimSpace(runtime.LogicalState)
	processState := strings.TrimSpace(runtime.ProcessState)
	if workerLogicalState, workerProcessState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle); ok {
		logicalState = workerLogicalState
		if strings.TrimSpace(workerProcessState) != "" {
			processState = workerProcessState
		}
	} else if runtimeHandleActivoAPICompartido(handle, runtime) {
		if logicalState == "" || strings.EqualFold(logicalState, "esperando_io") || strings.EqualFold(logicalState, "disponible") {
			logicalState = "activo"
		}
		if processState == "" || strings.EqualFold(processState, "desconocido") {
			processState = "running"
		}
	}
	if logicalState == "" {
		return nil
	}
	if processState == "" {
		processState = "running"
	}
	_, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state = ?,
		    process_state = ?,
		    last_event_at = CURRENT_TIMESTAMP,
		    last_heartbeat_at = CURRENT_TIMESTAMP
		WHERE id = ?`, logicalState, processState, runtime.ID)
	return err
}

func runtimeHandleActivoAPICompartido(handle *RuntimeHandle, runtime *RuntimeInstance) bool {
	if handle == nil || runtime == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Estado), "activo") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
	case "", "disponible", "working", "esperando_io":
	default:
		return false
	}
	switch strings.ToLower(strings.TrimSpace(runtime.ProcessState)) {
	case "", "running", "desconocido":
	default:
		return false
	}
	return true
}

func runtimeHandleSesionTMUXAusente(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	known, exists := controlruntime.TMUXSessionExistsMetadata(strings.TrimSpace(handle.MetadataJSON))
	return known && !exists
}

func runtimeWorkerSnapshotSatisfaceControlActual(orderType string, handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, now time.Time) bool {
	if snap == nil {
		return false
	}
	if runtimeHandleSesionTMUXAusente(handle) {
		return strings.EqualFold(strings.TrimSpace(orderType), "stop")
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil {
		return false
	}
	state := strings.ToLower(strings.TrimSpace(view.State))
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "start", "resume":
		if view.HeartbeatStale || !view.Alive || strings.TrimSpace(view.ExitError) != "" {
			return false
		}
		return state == "ready" || state == "running" || state == "starting"
	case "stop":
		switch state {
		case "stopped", "failed", "exited", "closed", "stale":
			return true
		}
		if strings.TrimSpace(view.ExitError) != "" {
			return true
		}
		return !view.Alive && !view.HeartbeatStale
	default:
		return false
	}
}

func runtimeOrderControlObsoletaPorWorkerRecuperado(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) bool {
	if order == nil {
		return false
	}
	orderType := strings.TrimSpace(order.Tipo)
	switch orderType {
	case "start", "resume":
	default:
		return false
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		return false
	}
	if runtimeHandleSesionTMUXAusente(handle) {
		return false
	}
	if runtime != nil {
		state := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
		if state == "pausado" || state == "stopped" {
			return false
		}
	}
	recovery := runtimeWorkerRecoveryMoment(handle, runtime)
	if recovery.IsZero() {
		return false
	}
	baseline := order.CreatedAt.UTC()
	if order.StartedAt != nil && !order.StartedAt.IsZero() && order.StartedAt.UTC().After(baseline) {
		baseline = order.StartedAt.UTC()
	}
	if snap := runtimeWorkerSnapshot(handle, runtime); snap != nil {
		return runtimeWorkerSnapshotRecoveredAfterBaseline(orderType, snap, baseline)
	}
	return recovery.After(baseline)
}

func runtimeWorkerSnapshotRecoveredAfterBaseline(orderType string, snap *runtimeagente.WorkerSnapshot, baseline time.Time) bool {
	if snap == nil || !snap.Alive() || snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		return false
	}
	if strings.TrimSpace(snap.ExitError()) != "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(snap.EffectiveState())) {
	case "ready", "running":
	default:
		return false
	}
	candidates := []*time.Time{
		snap.ReadyTime(),
		snap.StartedTime(),
	}
	if strings.EqualFold(strings.TrimSpace(orderType), "resume") {
		candidates = append(candidates, snap.LastProgressTime(), snap.LastOutputTime())
	}
	for _, candidate := range candidates {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		if baseline.IsZero() || candidate.UTC().After(baseline) {
			return true
		}
	}
	return false
}

func runtimeWorkerRecoveryMoment(handle *RuntimeHandle, runtime *RuntimeInstance) time.Time {
	latest := time.Time{}
	if runtime != nil {
		if ts := runtimeMomentForRecovery(runtime); !ts.IsZero() && ts.After(latest) {
			latest = ts
		}
	}
	if handle != nil {
		if handle.LastSeenAt != nil && !handle.LastSeenAt.IsZero() && handle.LastSeenAt.UTC().After(latest) {
			latest = handle.LastSeenAt.UTC()
		}
		if !handle.UpdatedAt.IsZero() && handle.UpdatedAt.UTC().After(latest) {
			latest = handle.UpdatedAt.UTC()
		}
	}
	if snap := runtimeWorkerSnapshot(handle, runtime); snap != nil && snap.Alive() && !snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		for _, candidate := range []*time.Time{
			snap.ReadyTime(),
			snap.LastProgressTime(),
			snap.LastOutputTime(),
			snap.HeartbeatTime(),
			snap.UpdatedTime(),
			snap.StartedTime(),
		} {
			if candidate != nil && !candidate.IsZero() && candidate.UTC().After(latest) {
				latest = candidate.UTC()
			}
		}
	}
	return latest
}

func runtimeMomentForRecovery(runtime *RuntimeInstance) time.Time {
	if runtime == nil {
		return time.Time{}
	}
	for _, value := range []*time.Time{runtime.UltimaActividadAt, runtime.LastHeartbeatAt, runtime.LastEventAt} {
		if value != nil && !value.IsZero() {
			return value.UTC()
		}
	}
	if !runtime.UpdatedAt.IsZero() {
		return runtime.UpdatedAt.UTC()
	}
	return runtime.CreatedAt.UTC()
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
	estado := "pendiente"
	candidates, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	orders := make([]*RuntimeOrder, 0, limit)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	for _, order := range candidates {
		if order == nil || !runtimeOrderPendingReady(order, now) {
			continue
		}
		switch strings.TrimSpace(order.Tipo) {
		case "handoff", "resume":
			orders = append(orders, order)
		default:
			continue
		}
		if len(orders) >= limit {
			break
		}
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
	return runtimeHandleCanonicoRecienteConFallback(strings.TrimSpace(order.Agente), order.ProyectoID)
}

func existeRuntimeOrderAbiertaAgenteProyecto(agente string, proyectoID *int64, excludeID int64, tipos ...string) (bool, error) {
	if strings.TrimSpace(agente) == "" || len(tipos) == 0 {
		return false, nil
	}
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente: stringsPtrTrimmed(agente),
	})
	if err != nil {
		return false, err
	}
	tiposWanted := make(map[string]struct{}, len(tipos))
	for _, tipo := range tipos {
		tipo = strings.TrimSpace(tipo)
		if tipo != "" {
			tiposWanted[tipo] = struct{}{}
		}
	}
	for _, order := range orders {
		if order == nil || order.ID == excludeID {
			continue
		}
		if proyectoID != nil && order.ProyectoID != nil && *order.ProyectoID != *proyectoID {
			continue
		}
		if _, ok := tiposWanted[strings.TrimSpace(order.Tipo)]; ok {
			return true, nil
		}
	}
	return false, nil
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
		freshRuntime, err := runtimeInstanceIfExists(&runtime.ID)
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
	if repaired, restarted, err := controlruntime.EnsureTMUXMonitorFromMetadataJSON(handle.MetadataJSON); err != nil {
		return true, map[string]any{
			"process_alive": false,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	} else if strings.TrimSpace(repaired) != "" && strings.TrimSpace(repaired) != strings.TrimSpace(handle.MetadataJSON) {
		if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, repaired, handle.ID); err != nil {
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		runtimeHandleHotReset()
		handle.MetadataJSON = repaired
		if restarted {
			handle.Estado = "activo"
		}
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	estado, observed, err := controlruntime.ConsultarEstadoLocal(obj)
	if err != nil {
		return true, map[string]any{
			"process_alive": false,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if !observed {
		vivo, pid, err := controlruntime.ProcesoVivo(obj)
		if err != nil {
			return true, map[string]any{
				"process_alive": false,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		if pid <= 0 {
			return false, nil, nil
		}
		estado = &controlruntime.EstadoLocal{
			PID:  pid,
			Vivo: vivo,
		}
		observed = true
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
	if superseded, err := runtimeHandleDebeCederAWorkerEstructurado(handle); err != nil {
		return true, map[string]any{
			"process_alive": vivo,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	} else if superseded {
		if err := marcarRuntimeHandleSuperseded(handle); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":      vivo,
			"process_superseded": true,
		}, nil
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":  false,
			"observed_local": true,
			"process_error":  "runtime handle blocked by broken channel",
		}, nil
	}
	if err := aplicarEstadoLocalObservado(handle, estado); err != nil {
		return true, map[string]any{
			"process_alive": vivo,
			"process_error": strings.TrimSpace(err.Error()),
		}, err
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return true, map[string]any{
				"process_alive": vivo,
				"process_error": strings.TrimSpace(err.Error()),
			}, err
		}
		return true, map[string]any{
			"process_alive":  false,
			"observed_local": true,
			"process_error":  "runtime handle blocked by broken channel",
		}, nil
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
	runtimeHandleHotReset()

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
	if workerLogicalState, workerProcessState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle); ok {
		logicalState = workerLogicalState
		if strings.TrimSpace(workerProcessState) != "" {
			processState = workerProcessState
		}
	}
	if runtime != nil {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET pid = ?,
			    logical_state = ?,
			    process_state = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = CURRENT_TIMESTAMP
			WHERE id = ?`, pid64, logicalState, processState, runtime.ID); err != nil {
			return true, nil, err
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

func runtimeObservedLogicalStateFromStructuredWorker(handle *RuntimeHandle) (string, string, bool) {
	if handle == nil {
		return "", "", false
	}
	meta := mapFromJSON(strings.TrimSpace(handle.MetadataJSON))
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	if !strings.EqualFold(driver, "tmux_cli_session") && !strings.EqualFold(transport, "tmux") {
		return "", "", false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return "", "", false
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return "", "", false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running", "idle":
		return "activo", "running", true
	default:
		return "", "", false
	}
}

func aplicarEstadoLocalObservado(handle *RuntimeHandle, estado *controlruntime.EstadoLocal) error {
	if handle == nil || estado == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for k, v := range mapFromJSON(estado.MetadataJSON) {
		meta[k] = v
	}
	normalizarMetadataTranscriptObservada(meta)
	compactarMetadataRuntimeHandle(meta)
	meta["local_last_status_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	metaJSON, _ := json.Marshal(meta)
	caps := handle.CapabilitiesJSON
	if strings.TrimSpace(estado.CapabilitiesJSON) != "" {
		caps = strings.TrimSpace(estado.CapabilitiesJSON)
	}
	estadoHandle := strings.TrimSpace(handle.Estado)
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		estadoHandle = "fallido"
	} else if rawEstado := strings.TrimSpace(estado.HandleEstado); rawEstado != "" {
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
	return runtimeHandleHotResetOnSuccess(err)
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
		runtimeHandleHotReset()
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

func MarcarRuntimeHandleCanalRoto(handle *RuntimeHandle, runtime *RuntimeInstance, reason string) error {
	if handle == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		meta = map[string]any{}
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		meta["pty_last_broken_pipe_error"] = reason
	}
	meta["pty_last_broken_pipe_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	metaJSON, _ := json.Marshal(meta)
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado = 'fallido',
		    metadata_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, string(metaJSON), handle.ID); err != nil {
		return err
	}
	runtimeHandleHotReset()
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
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
	Audit("orquesta", "runtime_handle_broken_pipe", "runtime_handle", handle.ID, reason)
	return nil
}

func runtimeHandleBloqueaReactivacionPorCanalRoto(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	rawMeta := strings.ToLower(strings.TrimSpace(handle.MetadataJSON))
	if strings.Contains(rawMeta, "external_session_id incompatible with tmux premium runtime") {
		return true
	}
	reason := strings.ToLower(strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "pty_last_broken_pipe_error", "")))
	if reason == "" {
		return false
	}
	return strings.Contains(reason, "external_session_id incompatible")
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
		normalizarMetadataTranscriptObservada(meta)
		compactarMetadataRuntimeHandle(meta)
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
		runtimeHandleHotReset()
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

func normalizarMetadataTranscriptObservada(meta map[string]any) {
	if meta == nil {
		return
	}
	pending := compactarPendingTranscript(strings.TrimSpace(stringFromMap(meta, "transcript_log_pending", "")))
	if pending == "" {
		delete(meta, "transcript_log_pending")
	} else {
		meta["transcript_log_pending"] = pending
	}
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
		sesion, err := sesionIfExists(handle.SesionID)
		if err != nil {
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
		if known, exists := controlruntime.TMUXSessionExistsMetadata(handle.MetadataJSON); known {
			return !exists, 0, nil
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

func runtimeOrderStartDebeIgnorarBootstrap(payload map[string]any) bool {
	motivo := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "motivo", "")))
	return strings.Contains(motivo, "manual_rehabilitation") ||
		strings.Contains(motivo, "reanimacion_manual") ||
		strings.Contains(motivo, "rehabilitacion_manual")
}

func reutilizarSesionArranquePoolLocal(agente string, proyectoID *int64, conector *Conector, externalSessionID string, plan *runtimeagente.LaunchPlan, resume runtimeagente.ResumeContext, ultima *Sesion, host string, pid *int64) (*Sesion, bool, error) {
	if conector == nil || !strings.EqualFold(strings.TrimSpace(conector.Slug), "ollama_pool_local") || proyectoID == nil || *proyectoID <= 0 {
		return nil, false, nil
	}
	sesion, err := GetSesionActiva(strings.TrimSpace(agente), proyectoID)
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
	if err := GuardarSesionActiva(strings.TrimSpace(agente), proyectoID, SesionUpdate{
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
	actualizada, err := GetSesionActiva(strings.TrimSpace(agente), proyectoID)
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

func inyectarPromptArranque(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan, order *RuntimeOrder) error {
	if handle == nil || runtime == nil || plan == nil {
		return nil
	}
	if runtimeHandleOmitePromptArranqueInteractivo(handle, runtime, plan) {
		return nil
	}
	if plan.LaunchPromptEmbedded && plan.CanSendInput != nil && !*plan.CanSendInput {
		return nil
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	texto := controlruntime.NormalizarInstruccionProceso(obj, promptArranquePendiente(plan))
	if texto == "" {
		return nil
	}
	if plan.LaunchPromptDelayMS > 0 {
		time.Sleep(time.Duration(plan.LaunchPromptDelayMS) * time.Millisecond)
	}
	aplicado, _, err := controlruntime.EnviarInstruccionProceso(obj, texto)
	if err != nil {
		if runtimeOrderPromptArranqueDebeDiferirse(handle, err) {
			return encolarPromptArranqueEnMailbox(order, texto)
		}
		return err
	}
	if aplicado {
		return RegistrarRuntimeTranscriptInput(handle, runtime, texto)
	}
	return encolarPromptArranqueEnMailbox(order, texto)
}

func runtimeOrderPromptArranqueDebeDiferirse(handle *RuntimeHandle, runtimeErr error) bool {
	if handle == nil || runtimeErr == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	switch {
	case strings.Contains(raw, "requiere carpeta de confianza"),
		strings.Contains(raw, "requiere trust"),
		strings.Contains(raw, "untrusted folder"),
		strings.Contains(raw, "requiere autenticacion manual"),
		strings.Contains(raw, "bloqueado por cuota"),
		strings.Contains(raw, "tmux pane no listo para send-keys"):
		return true
	}
	reason, ok := runtimeOrderSendInstructionDiferirPorErrorTMUX(handle, runtimeErr)
	if !ok {
		return false
	}
	switch strings.TrimSpace(reason) {
	case "worker_not_ready", "worker_blocked_trust", "worker_blocked_auth", "worker_blocked_quota", "worker_starting":
		return true
	default:
		return false
	}
}

func encolarPromptArranqueEnMailbox(order *RuntimeOrder, texto string) error {
	if order == nil || strings.TrimSpace(texto) == "" {
		return nil
	}
	payloadJSON, _ := json.Marshal(map[string]any{
		"texto":       texto,
		"kind":        "instruction",
		"from_agente": "orquesta",
		"motivo":      "launch_prompt_start",
	})
	_, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    strings.TrimSpace(order.Agente),
		ProyectoID:  order.ProyectoID,
		Kind:        "instruction",
		PayloadJSON: string(payloadJSON),
	})
	return err
}

func runtimeHandleOmitePromptArranqueInteractivo(handle *RuntimeHandle, runtime *RuntimeInstance, plan *runtimeagente.LaunchPlan) bool {
	if handle == nil || runtime == nil || plan == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		rendered := strings.TrimSpace(stringFromMap(meta, "rendered_command", ""))
		if rendered == "" {
			rendered = runtimeagente.RenderCommand(plan)
		}
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(rendered) &&
			!strings.Contains(strings.ToLower(strings.TrimSpace(rendered)), "ollama") {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "api") {
		driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
		if strings.EqualFold(driver, "ollama_pool_local") {
			return true
		}
	}
	if strings.EqualFold(strings.TrimSpace(runtime.Connector), "ollama_pool_local") {
		return true
	}
	return false
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
	if err != nil || !aplicado {
		controlErr := err
		yaDetenido, pid, err = runtimeProcesoLocalYaNoVive(order)
		if err != nil {
			return err
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
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		sqlOrders += ` AND proyecto_id=?`
		argsOrders = append(argsOrders, *order.ProyectoID)
	}
	if _, err := DB.Exec(sqlOrders, argsOrders...); err != nil {
		return err
	}
	return nil
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

func ejecutarRuntimeOrderSendInstruction(order *RuntimeOrder) error {
	payload, err := runtimeOrderPayloadMap(order.PayloadJSON)
	if err != nil {
		return err
	}
	payloadDesdeMailbox := runtimeOrderSendInstructionProvieneMailbox(payload)

	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
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

func reconciliarRuntimeSendInstructionSessionResume(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if runtime != nil {
		logicalState := strings.ToLower(strings.TrimSpace(runtime.LogicalState))
		shouldReactivate := logicalState == "" ||
			logicalState == "pausado" ||
			logicalState == "disponible" ||
			logicalState == "degradado" ||
			logicalState == "esperando_io"
		if shouldReactivate {
			if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='activo',
		    process_state=CASE
		    	WHEN trim(COALESCE(process_state,''))='' OR lower(trim(process_state))='stopped' THEN 'corriendo'
		    	ELSE process_state
		    END,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id=?`, runtime.ID); err != nil {
				return err
			}
		}
	}
	if handle != nil && strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") {
		if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='activo',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, handle.ID); err != nil {
			return err
		}
		runtimeHandleHotReset()
	}
	if order != nil {
		if err := actualizarEstadoSesionParaOrden(order, "activa"); err != nil {
			return err
		}
	}
	return nil
}

func runtimeOrderSendInstructionDebeRetenerSinRuntimeActivo(order *RuntimeOrder, payload map[string]any) bool {
	if order == nil {
		return false
	}
	return runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(order.Agente), "")
}

func runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode string, payload map[string]any) string {
	reason := "runtime_handle_bootstrap_only_mailbox_only"
	switch runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) {
	case runtimeagente.MailboxDeliveryCoordinatedRestart:
		reason = "runtime_handle_coordinated_restart_mailbox_only"
	case runtimeagente.MailboxDeliverySessionResume:
		reason = "runtime_handle_session_resume_mailbox_only"
	}
	if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
		reason += ":" + kind
	}
	return reason
}

func runtimeOrderSendInstructionLongTextReason(handle *RuntimeHandle, payload map[string]any, texto string) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !runtimeHandleUsaCodexTTYInestable(meta) {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return ""
	}
	texto = strings.TrimSpace(texto)
	if len(texto) <= 32 {
		return ""
	}
	reason := "codex_long_text_mailbox_only"
	if kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")); kind != "" {
		reason += ":" + kind
	}
	return reason
}

func runtimeOrderSendInstructionDurableOnlyReason(handle *RuntimeHandle, payload map[string]any, texto, deliveryMode string) (string, bool) {
	if handle == nil {
		return "", false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandleUsaCodexTTYInestable(meta) && runtimeOrderSendInstructionEsGuidanceDurableServidor(payload) {
		return runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode, payload), true
	}
	if reason := runtimeOrderSendInstructionLongTextReason(handle, payload, texto); reason != "" {
		return reason, true
	}
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return "", false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "nudge":
		if runtimeHandleEsTMUXCanonico(handle) {
			return "", false
		}
		return runtimeOrderSendInstructionMailboxOnlyReason(deliveryMode, payload), true
	}
	return "", false
}

func runtimeOrderSendInstructionDebeIntentarSessionResume(handle *RuntimeHandle, runtime *RuntimeInstance, payload map[string]any, texto, externalSessionID, deliveryMode string, reboundToCanonical bool) bool {
	if handle == nil || RuntimeHandlePermiteSendInputInteractivo(handle) {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandleUsaCodexTTYInestable(meta) && runtimeOrderSendInstructionEsGuidanceDurableServidor(payload) {
		return false
	}
	if strings.TrimSpace(externalSessionID) == "" {
		detected, err := runtimeHandleEffectiveExternalSessionID(handle, runtime)
		if err == nil {
			externalSessionID = strings.TrimSpace(detected)
		}
	}
	externalSessionKnown := strings.TrimSpace(externalSessionID) != "" || runtimeHandleTieneExternalSessionID(handle, runtime)
	sessionResumeRequested := runtimeHandleSolicitaSessionResume(handle, deliveryMode)
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return externalSessionKnown && (reboundToCanonical || runtimeHandlePermiteSessionResumeDirecto(handle, sessionResumeRequested))
	}
	if reason := runtimeOrderSendInstructionLongTextReason(handle, payload, texto); reason != "" {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "", "instruction":
		return externalSessionKnown || sessionResumeRequested
	case "nudge":
		// TMUX premium may still need durable session_resume for incremental nudges
		// even when can_send_input stays false by contract.
		return runtimeHandleEsTMUXCanonico(handle) && externalSessionKnown
	case "autonomia", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return false
	}
	return sessionResumeRequested && externalSessionKnown
}

func runtimeOrderSendInstructionEsGuidanceDurableServidor(payload map[string]any) bool {
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false
	}
	switch strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) {
	case "nudge", "autonomia", "watchdog", MailboxKindGovernanceRefresh, MailboxKindSkillsRefresh:
		return true
	default:
		return false
	}
}

func runtimeHandlePermiteSessionResumeDirecto(handle *RuntimeHandle, sessionResumeRequested bool) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	for _, key := range []string{"supervisor_ref", "stdin_path", "stdin_raw_path"} {
		if strings.TrimSpace(stringFromMap(meta, key, "")) != "" {
			return true
		}
	}
	return sessionResumeRequested && !runtimeHandleUsaCodexTTYInestable(meta)
}

func runtimeHandleSolicitaSessionResume(handle *RuntimeHandle, deliveryMode string) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	explicitBootstrapOnly := false
	for _, raw := range []string{
		stringFromMap(caps, "mailbox_delivery_mode", ""),
		stringFromMap(meta, "mailbox_delivery_mode", ""),
	} {
		switch runtimeagente.NormalizeMailboxDeliveryMode(raw) {
		case runtimeagente.MailboxDeliverySessionResume:
			return true
		case runtimeagente.MailboxDeliveryBootstrapOnly:
			explicitBootstrapOnly = true
		}
	}
	if explicitBootstrapOnly {
		return false
	}
	if runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) == runtimeagente.MailboxDeliverySessionResume {
		return true
	}
	return false
}

func runtimeOrderSendInstructionDebeRetenerOrdenDirecta(handle *RuntimeHandle, runtime *RuntimeInstance, deliveryMode, externalSessionID string) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	requiresDurableRuntime := runtimeHandleUsaCodexTTYInestable(meta) ||
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") ||
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session")
	switch runtimeagente.NormalizeMailboxDeliveryMode(deliveryMode) {
	case runtimeagente.MailboxDeliveryBootstrapOnly:
		return requiresDurableRuntime
	case runtimeagente.MailboxDeliverySessionResume:
		return strings.TrimSpace(externalSessionID) != "" || runtimeHandleTieneExternalSessionID(handle, runtime)
	case runtimeagente.MailboxDeliveryCoordinatedRestart:
		return requiresDurableRuntime
	default:
		return false
	}
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
	return explicitScore > resolvedScore && runtimeHandleTieneContextoEntrega(explicitHandle) && !runtimeHandleTieneContextoEntrega(resolvedHandle)
}

func runtimeHandleTieneContextoEntrega(handle *RuntimeHandle) bool {
	return runtimeHandleContextoEntregaScore(handle) > 0
}

func runtimeHandleContextoEntregaScore(handle *RuntimeHandle) int {
	if handle == nil {
		return 0
	}
	score := 0
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	for _, key := range []string{
		"driver",
		"rendered_command",
		"wrapped_command",
		"working_dir",
		"external_session_id",
		"stdin_path",
		"supervisor_ref",
		"tmux_command",
		"tmux_session",
		"mailbox_delivery_mode",
	} {
		if strings.TrimSpace(stringFromMap(meta, key, "")) != "" {
			score++
		}
	}
	for _, key := range []string{"mailbox_delivery_mode"} {
		if _, ok := caps[key]; ok {
			score++
		}
	}
	return score
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

func intentarEntregarRuntimeOrderSendInstructionBootstrapTMUX(order *RuntimeOrder, payload map[string]any, handle *RuntimeHandle, runtime *RuntimeInstance, obj controlruntime.ObjetivoProceso) (bool, error) {
	if order == nil || handle == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, nil
	}
	if !runtimeHandleListaParaDispatchBootstrapTMUX(handle, time.Now().UTC()) {
		return false, nil
	}
	prompt, ok := runtimeOrderSendInstructionTMUXBootstrapPrompt(payload)
	if !ok {
		return false, nil
	}
	result, err := controlruntime.EnviarInstruccionTMUXVerificada(obj, prompt)
	if err != nil {
		return true, reencolarRuntimeOrderSendInstruction(order, payload, "tmux bootstrap dispatch error: "+strings.TrimSpace(err.Error()))
	}
	if !result.Attempted {
		return false, nil
	}
	if result.Delivered {
		if err := RegistrarRuntimeTranscriptInput(handle, runtime, prompt); err != nil {
			return true, err
		}
		if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
			if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
				return true, err
			}
		}
		reason := "tmux bootstrap dispatch delivered"
		if result.Reason != "" {
			reason += ":" + strings.TrimSpace(result.Reason)
		}
		return true, retenerRuntimeOrderSendInstructionNotificada(order, payload, reason, time.Now().UTC())
	}
	reason := "tmux bootstrap dispatch unconfirmed"
	if result.Reason != "" {
		reason += ":" + strings.TrimSpace(result.Reason)
	}
	return true, reencolarRuntimeOrderSendInstruction(order, payload, reason)
}

func runtimeHandleListaParaDispatchBootstrapTMUX(handle *RuntimeHandle, now time.Time) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return false
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false
	}
	ready, _ := snap.ReadyForTextDispatch(now, time.Minute)
	return ready
}

func runtimeHandleListaParaDispatchInteractivo(handle *RuntimeHandle, now time.Time) (bool, string) {
	if handle == nil {
		return false, "worker_missing"
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	isTMUX := strings.EqualFold(driver, "tmux_cli_session") || strings.EqualFold(transport, "tmux")
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		if isTMUX {
			return false, "worker_snapshot_missing"
		}
		return true, ""
	}
	view := snap.View(now, time.Minute)
	if view == nil {
		if isTMUX {
			return false, "worker_view_missing"
		}
		return true, ""
	}
	if !strings.EqualFold(strings.TrimSpace(view.Driver), "tmux_cli_session") &&
		!strings.EqualFold(strings.TrimSpace(view.Transport), "tmux") {
		return true, ""
	}
	return snap.ReadyForTextDispatch(now, time.Minute)
}

func runtimeOrderSendInstructionDiferirPorErrorTMUX(handle *RuntimeHandle, runtimeErr error) (string, bool) {
	if handle == nil || runtimeErr == nil {
		return "", false
	}
	ready, reason := runtimeHandleListaParaDispatchInteractivo(handle, time.Now().UTC())
	if !ready && strings.TrimSpace(reason) != "" {
		return reason, true
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	switch {
	case strings.Contains(raw, "tmux pane no listo para send-keys"):
		return "worker_not_ready", true
	case strings.Contains(raw, "tmux pane requiere carpeta de confianza"),
		strings.Contains(raw, "tmux pane requiere trust"),
		strings.Contains(raw, "untrusted folder"):
		return "worker_blocked_trust", true
	case strings.Contains(raw, "tmux pane requiere autenticacion manual"):
		return "worker_blocked_auth", true
	case strings.Contains(raw, "tmux pane bloqueado por cuota"):
		return "worker_blocked_quota", true
	default:
		return "", false
	}
}

func runtimeOrderSendInstructionTMUXBootstrapPrompt(payload map[string]any) (string, bool) {
	if !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return "", false
	}
	texto := strings.TrimSpace(stringFromMap(payload, "texto", ""))
	if texto == "" {
		return "", false
	}
	texto = strings.Join(strings.Fields(strings.ReplaceAll(texto, "\r", " ")), " ")
	if texto == "" || len(texto) > 160 {
		return "", false
	}
	return texto, true
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

func runtimeOrderSendInstructionReceiptEvidence(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage) (bool, string, time.Time, error) {
	if order == nil || msg == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, "", time.Time{}, nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, "", time.Time{}, err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return false, "", time.Time{}, err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil || handle == nil {
		return false, "", time.Time{}, err
	}
	baseline := runtimeOrderSendInstructionReceiptBaseline(order, msg)
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		if runtimeOrderSendInstructionPermiteReceiptTranscriptPatch(payload) {
			confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionTranscriptPatchEvidence(runtime, handle, baseline)
			if err != nil {
				return false, "", time.Time{}, err
			}
			if confirmed {
				return true, receiptSource, receiptAt, nil
			}
		}
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false, "", time.Time{}, err
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionTMUXPanePatchEvidence(handle, snap, baseline)
		if err != nil {
			return false, "", time.Time{}, err
		}
		if confirmed {
			return true, receiptSource, receiptAt, nil
		}
	}
	if runtimeOrderSendInstructionPermiteReceiptLastProgress(payload) {
		if progress := snap.LastProgressTime(); progress != nil {
			progressAt := progress.UTC()
			if progressAt.After(baseline) {
				return true, "last_progress", progressAt, nil
			}
		}
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		return false, "", time.Time{}, nil
	}
	if confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionPremiumActivityEvidence(runtime, handle, snap, payload, baseline); err != nil {
		return false, "", time.Time{}, err
	} else if confirmed {
		return true, receiptSource, receiptAt, nil
	}
	if output := snap.LastOutputTime(); output != nil && runtimeOrderSendInstructionPermiteReceiptLastOutput(handle, snap, payload) {
		outputAt := output.UTC()
		if outputAt.After(baseline) {
			return true, "last_output", outputAt, nil
		}
	}
	return false, "", time.Time{}, nil
}

func runtimeOrderSendInstructionPermiteReceiptLastOutput(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, payload map[string]any) bool {
	if handle == nil || snap == nil {
		return false
	}
	if runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle, payload) {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return false
	}
	if RuntimeHandleMailboxDeliveryMode(handle) != runtimeagente.MailboxDeliveryInteractive {
		return true
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "running":
		return true
	default:
		return false
	}
}

func runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle *RuntimeHandle, payload map[string]any) bool {
	if handle == nil {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return runtimeHandleEsTMUXCanonico(handle)
	}
	if !runtimeHandleEsTMUXCanonico(handle) {
		return false
	}
	if runtimeagente.NormalizeMailboxDeliveryMode(RuntimeHandleMailboxDeliveryMode(handle)) != runtimeagente.MailboxDeliverySessionResume {
		return false
	}
	mailboxKind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", ""))
	if strings.EqualFold(mailboxKind, "nudge") {
		return true
	}
	return strings.EqualFold(mailboxKind, "autonomia") &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "accion", "")), "continuar_trabajo")
}

func runtimeOrderSendInstructionPremiumActivityEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, payload map[string]any, baseline time.Time) (bool, string, time.Time, error) {
	if handle == nil || snap == nil {
		return false, "", time.Time{}, nil
	}
	usaPipelinePremium := runtimeOrderSendInstructionEsPipelinePremium(payload)
	usaTMUXPaneEvidence := runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle, payload)
	if !usaPipelinePremium && !usaTMUXPaneEvidence {
		return false, "", time.Time{}, nil
	}
	view := snap.View(time.Now().UTC(), time.Minute)
	if view == nil || view.HeartbeatStale || !view.Alive {
		return false, "", time.Time{}, nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "ready", "idle", "running":
	default:
		return false, "", time.Time{}, nil
	}
	driver := strings.ToLower(strings.TrimSpace(view.Driver))
	transport := strings.ToLower(strings.TrimSpace(view.Transport))
	if driver == "process_pty_cli" || transport == "pty_broker" || transport == "cli" {
		if !usaPipelinePremium {
			return false, "", time.Time{}, nil
		}
		for _, candidate := range []*time.Time{
			snap.LastProgressTime(),
			snap.LastOutputTime(),
		} {
			if candidate == nil || candidate.IsZero() {
				continue
			}
			receiptAt := candidate.UTC()
			if runtimeOrderSendInstructionReceiptAtOrAfter(receiptAt, baseline) {
				return true, "worker_activity", receiptAt, nil
			}
		}
		return false, "", time.Time{}, nil
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) && !usaTMUXPaneEvidence {
		return false, "", time.Time{}, nil
	}
	known, captured, err := controlruntime.CaptureTMUXPaneMetadata(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return false, "", time.Time{}, err
	}
	if !known || !runtimeTMUXPaneShowsInteractiveWork(captured) {
		return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
	}
	if usaPipelinePremium {
		return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
	}
	for _, candidate := range []*time.Time{
		snap.LastProgressTime(),
		snap.LastOutputTime(),
	} {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		receiptAt := candidate.UTC()
		if runtimeOrderSendInstructionReceiptAtOrAfter(receiptAt, baseline) {
			return true, "tmux_pane_activity", receiptAt, nil
		}
	}
	return runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime, handle, baseline)
}

func runtimeTMUXPaneShowsInteractiveWork(captured string) bool {
	normalized := strings.ToLower(strings.TrimSpace(normalizarTextoTranscript(captured)))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "esc to interrupt") ||
		strings.Contains(normalized, "background terminal running") ||
		strings.Contains(normalized, "shift+tab to accept edits")
}

func runtimeOrderSendInstructionTMUXTranscriptActivityEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeTMUXInteractiveActivity(entry) {
			continue
		}
		return true, "tmux_transcript_activity", entry.CreatedAt.UTC(), nil
	}
	return false, "", time.Time{}, nil
}

func runtimeTranscriptLooksLikeTMUXInteractiveActivity(entry *RuntimeTranscriptEntry) bool {
	if entry == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(entry.Stream), "pty_out") {
		return false
	}
	normalized := strings.TrimSpace(strings.ToLower(entry.NormalizedText))
	if normalized == "" {
		normalized = strings.TrimSpace(strings.ToLower(normalizarTextoTranscript(entry.Text)))
	}
	if normalized == "" {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(entry.Classification)) {
	case "runtime_panic", "runtime_failure_signal":
		return false
	}
	markers := []string{
		"working",
		"ran ",
		"explored",
		"edited ",
		"waited for background terminal",
		"aplique un slice",
		"apliqué un slice",
		"gofmt",
		"go test",
		"git diff",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
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

func runtimeOrderSendInstructionReceiptBlocked(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage, now time.Time) (bool, string, error) {
	if order == nil || msg == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, "", nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, "", err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return false, "", err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		return false, "", err
	}
	if handle == nil {
		return true, "worker_missing_after_notified_dispatch", nil
	}
	if runtimeOrderSendInstructionReceiptBloqueoIgnoraSnapshot(handle) {
		return false, "", nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil {
		return false, "", err
	}
	if snap == nil {
		return true, "worker_snapshot_missing_after_notified_dispatch", nil
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil {
		return true, "worker_view_missing_after_notified_dispatch", nil
	}
	if view.HeartbeatStale {
		return true, "worker_heartbeat_stale_after_notified_dispatch", nil
	}
	if !view.Alive {
		return true, "worker_not_alive_after_notified_dispatch", nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "failed", "stopped", "exited", "closed", "stale":
		return true, "worker_" + strings.ToLower(strings.TrimSpace(view.State)) + "_after_notified_dispatch", nil
	default:
		return false, "", nil
	}
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

func runtimeOrderSendInstructionTranscriptPatchEvidence(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeMicroprogramacionPatch(entry) {
			continue
		}
		return true, "transcript_patch", entry.CreatedAt.UTC(), nil
	}
	windowText := strings.Builder{}
	var latestAt time.Time
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if strings.TrimSpace(entry.Text) == "" {
			continue
		}
		if windowText.Len() > 0 {
			windowText.WriteString("\n")
		}
		windowText.WriteString(entry.Text)
		if entry.CreatedAt.UTC().After(latestAt) {
			latestAt = entry.CreatedAt.UTC()
		}
	}
	if latestAt.IsZero() {
		return false, "", time.Time{}, nil
	}
	if runtimeMicroprogramacionPatchDetected(windowText.String(), "") {
		return true, "transcript_patch", latestAt, nil
	}
	return false, "", time.Time{}, nil
}

func runtimeOrderSendInstructionTMUXPanePatchEvidence(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot, baseline time.Time) (bool, string, time.Time, error) {
	if handle == nil || snap == nil {
		return false, "", time.Time{}, nil
	}
	known, captured, err := controlruntime.CaptureTMUXPaneMetadata(strings.TrimSpace(handle.MetadataJSON))
	if err != nil {
		return false, "", time.Time{}, err
	}
	if !known || !runtimeMicroprogramacionPatchDetected(captured, "") {
		return false, "", time.Time{}, nil
	}
	for _, candidate := range []*time.Time{
		snap.LastOutputTime(),
		snap.LastProgressTime(),
		snap.HeartbeatTime(),
		snap.UpdatedTime(),
		snap.ReadyTime(),
	} {
		if candidate == nil || candidate.IsZero() {
			continue
		}
		receiptAt := candidate.UTC()
		if baseline.IsZero() || receiptAt.After(baseline) {
			return true, "tmux_pane_patch", receiptAt, nil
		}
	}
	return true, "tmux_pane_patch", time.Now().UTC(), nil
}

func runtimeOrderSendInstructionTranscriptBlockedByAgent(runtime *RuntimeInstance, handle *RuntimeHandle, baseline time.Time) (bool, string, time.Time, error) {
	if runtime == nil || handle == nil {
		return false, "", time.Time{}, nil
	}
	filter := FiltroRuntimeTranscript{
		RuntimeID: &runtime.ID,
		HandleID:  &handle.ID,
		Limit:     64,
	}
	entries, err := ListarRuntimeTranscript(filter)
	if err != nil {
		return false, "", time.Time{}, err
	}
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]
		if entry == nil || !runtimeOrderSendInstructionReceiptAtOrAfter(entry.CreatedAt.UTC(), baseline) {
			continue
		}
		if !runtimeTranscriptLooksLikeMicroprogramacionBlockedResponse(entry.Text, entry.NormalizedText) {
			continue
		}
		return true, strings.TrimSpace(entry.Text), entry.CreatedAt.UTC(), nil
	}
	return false, "", time.Time{}, nil
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

func runtimeOrderSendInstructionReceiptTimedOut(order *RuntimeOrder, msg *RuntimeMailboxMessage, now time.Time) bool {
	notifiedAt := runtimeOrderSendInstructionNotifiedAt(order, msg)
	if notifiedAt.IsZero() {
		return false
	}
	return now.UTC().Sub(notifiedAt) > runtimeOrderSendInstructionReceiptTimeout()
}

func reconciliarRuntimeOrderSendInstructionMailboxEntregado(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage, now time.Time) (bool, error) {
	if order == nil || msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Estado), "entregado") {
		return false, nil
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		baseline := runtimeOrderSendInstructionNotifiedAt(order, msg)
		if baseline.IsZero() {
			baseline = msg.CreatedAt.UTC()
		}
		handle, err := resolverHandleParaOrden(order)
		if err != nil {
			return true, err
		}
		runtime, err := resolverRuntimeParaOrden(order)
		if err != nil {
			return true, err
		}
		runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
		if err != nil {
			return true, err
		}
		blockedByAgent, blockedDetail, blockedAt, err := runtimeOrderSendInstructionTranscriptBlockedByAgent(runtime, handle, baseline)
		if err != nil {
			return true, err
		}
		if blockedByAgent {
			if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
				return true, err
			}
			if err := fallarRuntimeOrderSendInstructionPorBloqueoAgente(order, payload, blockedDetail, blockedAt); err != nil {
				return true, err
			}
			return true, nil
		}
	}
	confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, msg)
	if err != nil {
		return true, err
	}
	if confirmed {
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return true, err
		}
		if err := completarRuntimeOrderSendInstructionEntregadaPorReceipt(order, payload, receiptSource, receiptAt); err != nil {
			return true, err
		}
		return true, nil
	}
	blocked, blockedReason, err := runtimeOrderSendInstructionReceiptBlocked(order, payload, msg, now)
	if err != nil {
		return true, err
	}
	if blocked {
		runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
		if err := RearmarRuntimeMailboxPendiente(msg.ID); err != nil {
			return true, err
		}
		if err := reencolarRuntimeOrderSendInstruction(order, payload, blockedReason); err != nil {
			return true, err
		}
		return true, nil
	}
	if runtimeOrderSendInstructionReceiptTimedOut(order, msg, now) {
		runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
		if err := RearmarRuntimeMailboxPendiente(msg.ID); err != nil {
			return true, err
		}
		if err := reencolarRuntimeOrderSendInstruction(order, payload, "delivery receipt timeout"); err != nil {
			return true, err
		}
		return true, nil
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, payload, "mailbox notificado pendiente de recibo", runtimeOrderSendInstructionNotifiedAt(order, msg)); err != nil {
		return true, err
	}
	return true, nil
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

func actualizarRuntimeOrderPayloadJSON(orderID int64, payload map[string]any) error {
	if orderID <= 0 || payload == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET payload_json = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		string(raw),
		orderID,
	); err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(orderID)
}

func objetivoProcesoSendInstruction(order *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance, workingDir string) controlruntime.ObjetivoProceso {
	obj := controlruntime.ObjetivoProceso{}
	if handle != nil {
		obj.HandleKind = handle.HandleKind
		obj.HandleRef = handle.HandleRef
		obj.MetadataJSON = handle.MetadataJSON
	}
	if strings.TrimSpace(workingDir) != "" || (order != nil && order.ID > 0) {
		meta := mapFromJSON(obj.MetadataJSON)
		if meta == nil {
			meta = map[string]any{}
		}
		meta["working_dir"] = strings.TrimSpace(workingDir)
		if order != nil && order.ID > 0 {
			meta["runtime_order_id"] = order.ID
		}
		if metaJSON, marshalErr := json.Marshal(meta); marshalErr == nil {
			obj.MetadataJSON = string(metaJSON)
		}
	}
	if runtime != nil &&
		!runtimeOrderBloqueaFallbackPID(order, runtime, handle) &&
		(handle == nil || strings.TrimSpace(handle.HandleKind) == "" || strings.TrimSpace(handle.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return obj
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

func runtimeMailboxReusableForPendingSendInstruction(msg *RuntimeMailboxMessage) bool {
	if msg == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(msg.Estado)) {
	case "", "pendiente", "entregado":
		return true
	default:
		return false
	}
}

func runtimeOrderSendInstructionMailboxPayloadJSON(order *RuntimeOrder, payload map[string]any) string {
	if payload == nil {
		return strings.TrimSpace(order.PayloadJSON)
	}
	sanitized := make(map[string]any, len(payload))
	for key, value := range payload {
		switch strings.TrimSpace(key) {
		case "mailbox_id":
			continue
		default:
			sanitized[key] = value
		}
	}
	if len(sanitized) == 0 {
		return strings.TrimSpace(order.PayloadJSON)
	}
	raw, err := json.Marshal(sanitized)
	if err != nil {
		return strings.TrimSpace(order.PayloadJSON)
	}
	return string(raw)
}

func asegurarRuntimeOrderSendInstructionMailboxPendiente(order *RuntimeOrder, payload map[string]any) (int64, error) {
	if order == nil {
		return 0, nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return 0, err
		}
		if runtimeMailboxReusableForPendingSendInstruction(msg) {
			return mailboxID, nil
		}
		delete(payload, "mailbox_id")
	}
	existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
	if err != nil {
		return 0, err
	}
	if runtimeMailboxReusableForPendingSendInstruction(existente) {
		return existente.ID, nil
	}
	toAgente := stringFromMap(payload, "to_agente", order.Agente)
	fromAgente := stringFromMap(payload, "from_agente", "server")
	kind := strings.TrimSpace(stringFromMap(payload, "mailbox_kind", ""))
	if kind == "" {
		kind = "instruction"
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		ProyectoID:     order.ProyectoID,
		RuntimeOrderID: &order.ID,
		Kind:           kind,
		PayloadJSON:    runtimeOrderSendInstructionMailboxPayloadJSON(order, payload),
	})
	if err != nil {
		return 0, err
	}
	return msgID, nil
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

func runtimeOrderSendInstructionDebeRetenerNotificadaPorSessionResumeTimeout(order *RuntimeOrder, handle *RuntimeHandle, payload map[string]any, runtimeErr error) bool {
	if order == nil || handle == nil || runtimeErr == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(runtimeErr.Error()))
	if !strings.Contains(raw, "session_resume timeout") {
		return false
	}
	if !runtimeHandleUsaCodexTTYInestable(mapFromJSON(handle.MetadataJSON)) {
		return false
	}
	if runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return true
	}
	return runtimeHandleEsTMUXCanonico(handle) &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")), "nudge")
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
		"dispatch_state":       "pending",
		"ok":                   false,
		"deferred":             true,
		"mailbox_id":           runtimeOrderSendInstructionMailboxID(payload),
		"delivery_state":       "queued",
		"delivery_notified_at": "",
		"delivery_receipt_at":  "",
		"receipt_source":       "",
		"retry_after":          nextAttempt.Format(time.RFC3339Nano),
		"deferred_reason":      reason,
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
		WHERE id = ?
		  AND estado NOT IN ('completada','fallida','cancelada','expirada')`,
		resultado,
		reason,
		reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}

func retenerRuntimeOrderSendInstructionDiferidaAMailbox(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox retenido hasta runtime entregable"
	}
	if payload == nil {
		payload = map[string]any{}
	}
	mailboxID, err := asegurarRuntimeOrderSendInstructionMailboxPendiente(order, payload)
	if err != nil {
		return err
	}
	payload["mailbox_id"] = mailboxID
	if strings.TrimSpace(stringFromMap(payload, "mailbox_kind", "")) == "" {
		payload["mailbox_kind"] = "instruction"
	}
	if strings.TrimSpace(stringFromMap(payload, "to_agente", "")) == "" && strings.TrimSpace(order.Agente) != "" {
		payload["to_agente"] = strings.TrimSpace(order.Agente)
	}
	updatedPayloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	nextAttempt := time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":  "pending",
		"ok":              false,
		"mailbox_id":      mailboxID,
		"deferred":        true,
		"deferred_reason": reason,
		"mailbox_only":    true,
		"delivery_state":  "queued",
		"retry_after":     nextAttempt.Format(time.RFC3339Nano),
	})
	_, err = DB.Exec(`
		UPDATE runtime_orders
		SET payload_json = ?,
		    estado = 'pendiente',
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
		string(updatedPayloadJSON),
		resultado,
		reason,
		reason,
		nextAttempt,
		order.ID,
	)
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}

func retenerRuntimeOrderSendInstructionNotificada(order *RuntimeOrder, payload map[string]any, reason string, notifiedAt time.Time) error {
	if order == nil {
		return nil
	}
	actual, err := GetRuntimeOrder(order.ID)
	if err == nil && runtimeOrderYaTieneEntregaValida(actual) {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox notificado pendiente de recibo"
	}
	if notifiedAt.IsZero() {
		notifiedAt = time.Now().UTC()
	}
	nextAttempt := time.Now().UTC().Add(runtimeOrderSendInstructionRetryDelay())
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":       "notified",
		"ok":                   false,
		"mailbox_id":           runtimeOrderSendInstructionMailboxID(payload),
		"deferred":             true,
		"deferred_reason":      reason,
		"mailbox_only":         true,
		"delivery_state":       "notified",
		"delivery_notified_at": notifiedAt.Format(time.RFC3339Nano),
		"delivery_receipt_at":  "",
		"receipt_source":       "",
		"retry_after":          nextAttempt.Format(time.RFC3339Nano),
	})
	_, err = DB.Exec(`
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
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(order.ID)
}

func runtimeOrderYaTieneEntregaValida(order *RuntimeOrder) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(order.Estado)) {
	case "completada", "fallida", "cancelada", "expirada":
		return true
	}
	result := mapFromJSON(order.ResultadoJSON)
	if result == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(result, "delivery_state", "")), "delivered") {
		return true
	}
	if strings.TrimSpace(stringFromMap(result, "receipt_source", "")) != "" {
		return true
	}
	return false
}

func completarRuntimeOrderSendInstructionEntregadaPorReceipt(order *RuntimeOrder, payload map[string]any, receiptSource string, receiptAt time.Time) error {
	if order == nil {
		return nil
	}
	if receiptAt.IsZero() {
		receiptAt = time.Now().UTC()
	}
	receiptSource = strings.TrimSpace(receiptSource)
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":      "delivered",
		"ok":                  true,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"mailbox_only":        true,
		"delivery_state":      "delivered",
		"delivery_receipt_at": receiptAt.Format(time.RFC3339Nano),
		"receipt_source":      receiptSource,
	})
	if err := MarcarRuntimeOrderEstado(order.ID, "completada", resultado, ""); err != nil {
		return err
	}
	runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
	return nil
}

func fallarRuntimeOrderSendInstructionPorBloqueoAgente(order *RuntimeOrder, payload map[string]any, detalle string, blockedAt time.Time) error {
	if order == nil {
		return nil
	}
	detalle = strings.TrimSpace(detalle)
	if detalle == "" {
		detalle = "BLOQUEO: el agente no pudo ejecutar la microtarea"
	}
	if blockedAt.IsZero() {
		blockedAt = time.Now().UTC()
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":      "failed",
		"ok":                  false,
		"mailbox_id":          runtimeOrderSendInstructionMailboxID(payload),
		"delivery_state":      "blocked",
		"blocked":             true,
		"blocked_reason":      detalle,
		"delivery_receipt_at": blockedAt.Format(time.RFC3339Nano),
		"receipt_source":      "transcript_blocked",
	})
	if err := MarcarRuntimeOrderEstado(order.ID, "fallida", resultado, detalle); err != nil {
		return err
	}
	runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
	return nil
}

func runtimeOrderSendInstructionAutoStopLocalOllama(order *RuntimeOrder, payload map[string]any) {
	if order == nil || !runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		return
	}
	if !runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(order.Agente), "") {
		return
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	if handle == nil {
		return
	}
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	if _, _, err := controlruntime.DetenerProceso(obj); err != nil && !errorDetenerSesionObsoletaIgnorable(err) {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	if runtime != nil && runtime.ID > 0 {
		if _, err := DB.Exec(`
			UPDATE runtime_instances
			SET logical_state='cerrado',
			    process_state='finalizado',
			    last_event_at=CURRENT_TIMESTAMP
			WHERE id=?`, runtime.ID); err != nil {
			Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
			return
		}
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET estado='cerrado',
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE agente=? AND estado IN ('activo','pausado','fallido')`, order.Agente); err != nil {
		Audit("orquesta", "runtime_send_instruction_autostop_error", "runtime_order", order.ID, err.Error())
		return
	}
	runtimeHandleHotReset()
}

func completarRuntimeOrderSendInstructionDiferidaAMailbox(order *RuntimeOrder, payload map[string]any, reason string) error {
	if order == nil {
		return nil
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) || runtimeOrderSendInstructionEsPipelinePremium(payload) {
		return retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, reason)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mailbox retenido hasta runtime entregable"
	}
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 {
		existente, err := GetRuntimeMailboxByRuntimeOrderID(order.ID)
		if err != nil {
			return err
		}
		if existente != nil {
			mailboxID = existente.ID
		} else {
			toAgente := stringFromMap(payload, "to_agente", order.Agente)
			fromAgente := stringFromMap(payload, "from_agente", "server")
			msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
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
			mailboxID = msgID
		}
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"dispatch_state":  "delivered",
		"ok":              true,
		"mailbox_id":      mailboxID,
		"deferred":        true,
		"deferred_reason": reason,
		"mailbox_only":    true,
		"delivery_state":  "delivered",
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
	if err := registrarPresupuestoSesionProviderBackoff(order, runtimeErr, delay, motivo); err != nil {
		return true, err
	}
	detalle := fmt.Sprintf("agente en enfriamiento por %s", motivo)
	if runtimeOrderSendInstructionProvieneMailbox(payload) {
		if mailboxID := runtimeOrderSendInstructionMailboxID(payload); mailboxID > 0 {
			msg, err := GetRuntimeMailbox(mailboxID)
			if err != nil {
				return true, err
			}
			if msg != nil && !strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente") {
				return true, completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, detalle+"; mailbox ya "+strings.TrimSpace(msg.Estado))
			}
		}
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

func registrarPresupuestoSesionProviderBackoff(order *RuntimeOrder, runtimeErr error, delay time.Duration, motivo string) error {
	if order == nil || runtimeErr == nil || strings.TrimSpace(order.Agente) == "" {
		return nil
	}
	sesionID, modelSlug, err := resolverSesionYModeloRuntimeOrder(order)
	if err != nil || sesionID <= 0 {
		return err
	}
	now := time.Now().UTC()
	if delay <= 0 {
		delay = 60 * time.Minute
	}
	resetAt := now.Add(delay)
	remainingZero := int64(0)
	raw := strings.TrimSpace(runtimeErr.Error())
	snapshot := map[string]any{
		"source":         "runtime_order_send_instruction",
		"runtime_order":  order.ID,
		"agente":         strings.TrimSpace(order.Agente),
		"motivo":         strings.TrimSpace(motivo),
		"window_kind":    "provider",
		"window_start":   now.Format(time.RFC3339Nano),
		"reset_at":       resetAt.Format(time.RFC3339Nano),
		"remaining_zero": true,
		"error":          raw,
	}
	for key, value := range identidadObservadaDesdeErrorProveedor(raw) {
		snapshot[key] = value
	}
	for key, value := range identidadObservadaDesdeOrdenRuntime(order) {
		if _, exists := snapshot[key]; !exists {
			snapshot[key] = value
		}
	}
	rawJSON, _ := json.Marshal(snapshot)
	id, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:          sesionID,
		ModelSlug:         strings.TrimSpace(modelSlug),
		WindowKind:        "provider",
		WindowStartedAt:   &now,
		ResetAt:           &resetAt,
		RemainingMessages: &remainingZero,
		BudgetSource:      "provider_backoff",
		RawSnapshotJSON:   string(rawJSON),
		CheckedAt:         now,
	})
	if err != nil {
		return err
	}
	Audit("orquesta", "registrar_presupuesto_provider_backoff", "presupuesto_sesion", id,
		fmt.Sprintf("runtime_order=%d agente=%s reset_at=%s motivo=%s",
			order.ID, strings.TrimSpace(order.Agente), resetAt.Format(time.RFC3339), strings.TrimSpace(motivo)))
	return nil
}

func resolverSesionYModeloRuntimeOrder(order *RuntimeOrder) (int64, string, error) {
	if order == nil {
		return 0, "", nil
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return 0, "", err
	}
	model := ""
	if runtime != nil {
		model = strings.TrimSpace(runtime.Model)
	}
	sesion, err := resolverSesionParaOrden(order)
	if err != nil {
		return 0, "", err
	}
	if sesion != nil {
		return sesion.ID, model, nil
	}
	return 0, model, nil
}

func identidadObservadaDesdeErrorProveedor(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]any{}
	emailRe := regexp.MustCompile(`(?i)[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}`)
	if email := strings.TrimSpace(emailRe.FindString(raw)); email != "" {
		out["account_email"] = email
	}
	userRe := regexp.MustCompile(`(?im)^(?:perfil activo|active profile|logged in as|usuario|user|username|login)\s*:\s*([^\r\n]+)\s*$`)
	if match := userRe.FindStringSubmatch(raw); len(match) == 2 {
		if user := strings.TrimSpace(match[1]); user != "" {
			out["account_user"] = user
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func identidadObservadaDesdeOrdenRuntime(order *RuntimeOrder) map[string]any {
	if order == nil {
		return nil
	}
	candidates := make([]*RuntimeHandle, 0, 2)
	if order.HandleID != nil && *order.HandleID > 0 {
		handle, err := GetRuntimeHandle(*order.HandleID)
		if err == nil && handle != nil {
			candidates = append(candidates, handle)
		}
	}
	handle, err := resolverHandleParaOrden(order)
	if err == nil && handle != nil {
		if len(candidates) == 0 || candidates[0].ID != handle.ID {
			candidates = append(candidates, handle)
		}
	}
	for _, candidate := range candidates {
		if observed := identidadObservadaDesdeHandleRuntime(candidate); observed != nil {
			return observed
		}
	}
	return nil
}

func identidadObservadaDesdeHandleRuntime(handle *RuntimeHandle) map[string]any {
	if handle == nil {
		return nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return nil
	}
	rendered := strings.TrimSpace(stringFromMap(meta, "rendered_command", ""))
	if rendered == "" {
		return nil
	}
	re := regexp.MustCompile(`(?i)codex-perfil'\s+'([^']+)'`)
	match := re.FindStringSubmatch(rendered)
	if len(match) != 2 || strings.TrimSpace(match[1]) == "" {
		return nil
	}
	return map[string]any{"account_user": strings.TrimSpace(match[1])}
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
	return objetivoProcesoDesdeHandleRuntimeOrden(order, handle, runtime), nil
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
	if plan == nil {
		return strings.TrimSpace(resume.ResumePayloadJSON)
	}
	return MergeResumePayloadPerfilEjecucion(
		resume.ResumePayloadJSON,
		strings.TrimSpace(plan.PerfilTarea),
		strings.TrimSpace(plan.Modelo),
		strings.TrimSpace(plan.Razonamiento),
	)
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
		asignarResumenPromptHandle(meta, "continuity_prompt", plan.ContinuityPrompt)
		asignarResumenPromptHandle(meta, "bootstrap_prompt", plan.BootstrapPrompt)
		meta["launch_prompt_embedded"] = plan.LaunchPromptEmbedded
		meta["launch_prompt_mode"] = strings.TrimSpace(plan.LaunchPromptMode)
		meta["launch_prompt_delay_ms"] = plan.LaunchPromptDelayMS
	}
	if strings.TrimSpace(resume.ResumenContinuidad) != "" {
		meta["resumen_continuidad"] = strings.TrimSpace(resume.ResumenContinuidad)
	}
	compactarMetadataRuntimeHandle(meta)
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
	transporte := inferirTransporteArranque(arranque)
	_, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte = ?,
		    handle_kind = ?,
		    handle_ref = ?,
		    metadata_json = ?,
		    capabilities_json = ?,
		    last_seen_at = CURRENT_TIMESTAMP
		WHERE id = ?`, transporte, handleKind, handleRef, string(metaJSON), string(capsJSON), handleID)
	return runtimeHandleHotResetOnSuccess(err)
}

func inferirTransporteArranque(arranque *controlruntime.ProcesoArrancado) string {
	if arranque == nil {
		return ""
	}
	meta := mapFromJSON(arranque.MetadataJSON)
	for _, candidate := range []string{
		stringFromMap(meta, "transport", ""),
		stringFromMap(meta, "transporte", ""),
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	switch strings.TrimSpace(stringFromMap(meta, "driver", "")) {
	case "tmux_cli_session":
		return "tmux"
	case "process_pty_cli":
		return "cli"
	case "remote_http", "ollama_pool_local":
		return "api"
	}
	if strings.TrimSpace(arranque.HandleKind) == "session" {
		return "tmux"
	}
	return "cli"
}

func prepararStartRuntimeOrder(agenteRef, proyectoRef string, excludeOrderID int64, conectorRef, modelo, razonamiento, perfilTarea string, tareaID *int64, skipBootstrap bool) (*Agente, *Proyecto, *Conector, *Sesion, runtimeagente.ResumeContext, *bootstrapRuntimeData, *runtimeagente.LaunchPlan, error) {
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
	conector, err := resolverConectorRuntimeOrder(strings.TrimSpace(agenteRef), strings.TrimSpace(proyecto.Slug), conectorRef, ultima)
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
	perfilSolicitado := strings.TrimSpace(perfilTarea)
	modeloSolicitado := strings.TrimSpace(modelo)
	razonamientoSolicitado := strings.TrimSpace(razonamiento)
	perfilPersistido, modeloPersistido, razonamientoPersistido := ResumePayloadPerfilEjecucion(resume.ResumePayloadJSON)
	if perfilSolicitado == "" {
		perfilSolicitado = perfilPersistido
		perfilTarea = perfilPersistido
	}
	modeloPersistidoCompatible := runtimeagente.ModeloCompatibleConConector(runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}, modeloPersistido)
	if modeloSolicitado == "" && modeloPersistidoCompatible {
		modeloSolicitado = modeloPersistido
		modelo = modeloPersistido
	}
	if razonamientoSolicitado == "" && modeloPersistidoCompatible {
		razonamientoSolicitado = razonamientoPersistido
		razonamiento = razonamientoPersistido
	}
	perfilTarea, modelo, razonamiento, err = ResolverPerfilEjecucionLanzamiento(
		&agenteRef,
		strings.TrimSpace(proyecto.Slug),
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
		runtimeagente.ConnectorConfig{
			Slug:         strings.TrimSpace(conector.Slug),
			Nombre:       strings.TrimSpace(conector.Nombre),
			Transporte:   strings.TrimSpace(conector.Transporte),
			Comando:      strings.TrimSpace(conector.Comando),
			ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
			EnvJSON:      strings.TrimSpace(conector.EnvJSON),
			MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
			Activo:       conector.Activo,
		},
		perfilSolicitado,
		modeloSolicitado,
		razonamientoSolicitado,
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	conectorRuntime := runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}
	if !runtimeagente.ModeloCompatibleConConector(conectorRuntime, modelo) {
		modelo = ""
		if modeloSolicitado == "" {
			perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
				conectorRuntime,
				perfilSolicitado,
				"",
				razonamientoSolicitado,
				perfilTarea,
				"",
				razonamiento,
			)
			if err != nil {
				return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
			}
		}
	}
	var bootstrap *bootstrapRuntimeData
	if skipBootstrap {
		resume = runtimeagente.ResumeContext{
			Branch: strings.TrimSpace(resume.Branch),
			CWD:    strings.TrimSpace(resume.CWD),
		}
		resume = SanitizeResumeContextForProject(resume, proyecto)
		resume.CWD = RutaTrabajoPreferidaAgenteProyecto(strings.TrimSpace(agenteRef), proyecto, strings.TrimSpace(resume.CWD))
		if strings.TrimSpace(resume.CWD) == "" {
			resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
		}
	} else {
		resume, bootstrap, err = prepararResumeBootstrapRuntime(strings.TrimSpace(agenteRef), proyecto, resume, excludeOrderID)
		if err != nil {
			return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
		}
	}
	resume.ResumePayloadJSON = MergeResumePayloadPerfilEjecucion(
		resume.ResumePayloadJSON,
		strings.TrimSpace(perfilTarea),
		strings.TrimSpace(modelo),
		strings.TrimSpace(razonamiento),
	)
	resume = runtimeagente.SanitizarResumeParaConector(runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}, resume)
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		(strings.TrimSpace(perfilTarea) != "" || strings.TrimSpace(modelo) != "" || strings.TrimSpace(razonamiento) != "") {
		resume.ResumePayloadJSON = MergeResumePayloadPerfilEjecucion(
			"",
			strings.TrimSpace(perfilTarea),
			strings.TrimSpace(modelo),
			strings.TrimSpace(razonamiento),
		)
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
		Resume:  resume,
		TareaID: tareaID,
	})
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	plan.BootstrapPrompt, err = BuildLaunchBootstrapPromptForContext(agente, proyecto, plan)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, err
	}
	metadataJSONNormalizada, err := runtimeagente.NormalizeConnectorMetadataJSON(runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	})
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, nil, nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	if err := runtimeagente.ApplyLaunchPromptMetadata(plan, metadataJSONNormalizada); err != nil {
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

func resolverConectorRuntimeOrder(agenteNombre, proyectoSlug, conectorRef string, ultima *Sesion) (*Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" &&
			runtimeagente.ConectorCompatibleConAgente(agenteNombre, strings.TrimSpace(ultima.ConectorSlug), "") {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 &&
			runtimeagente.ConectorCompatibleConAgente(agenteNombre, "", strings.TrimSpace(ultima.Herramienta)) {
			ref = jsonNumber(*ultima.ConectorID)
		}
	}
	if ref == "" {
		if runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(agenteNombre), "") {
			if preferido, err := resolverConectorPoolLocalCompartido(agenteNombre, proyectoSlug, ""); err == nil && strings.TrimSpace(preferido) != "" {
				ref = preferido
			}
		}
	}
	if ref == "" {
		ref = runtimeagente.ConectorPorDefectoAgente(agenteNombre)
	}
	return GetConector(ref)
}

func resolverConectorPoolLocalCompartido(agenteNombre, proyectoSlug, perfilTarea string) (string, error) {
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		AgenteNombre: strPtrRuntime(strings.TrimSpace(agenteNombre)),
	})
	if err != nil || resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return "", err
	}
	pool, err := GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil || pool == nil {
		return "", err
	}
	meta := mapFromJSON(strings.TrimSpace(pool.MetadataJSON))
	if !strings.EqualFold(strings.TrimSpace(pool.Runtime), "ollama") {
		return "", nil
	}
	if strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "conector_canonico", "")), "ollama_pool_local") {
		return "ollama_pool_local", nil
	}
	return "", nil
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
	if err != nil {
		return err
	}
	return runtimeOrdersHotIndexSyncByID(bootstrap.Order.ID)
}

func timeoutParaArranqueDurable(order *RuntimeOrder) time.Duration {
	if order == nil {
		return 0
	}
	payload := mapFromJSON(order.PayloadJSON)
	por := strings.ToLower(strings.TrimSpace(stringFromMap(payload, "por", "")))
	if por == "orquesta" || por == "sistema" || por == "autonomia" {
		// En orchestración autónoma, no bloqueamos el runner esperando el Ready del prompt.
		// El buzón ya estará lleno y el agente lo consumirá en cuanto esté listo.
		return 1500 * time.Millisecond
	}
	return 0 // Usa el default del conector (usualmente 60s o configurable via env)
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

func compactarMetadataRuntimeHandle(meta map[string]any) {
	if meta == nil {
		return
	}
	for _, key := range []string{"bootstrap_prompt", "continuity_prompt"} {
		asignarResumenPromptHandle(meta, key, stringFromMap(meta, key, ""))
	}
	asignarResumenPromptHandle(meta, "resumen_continuidad", stringFromMap(meta, "resumen_continuidad", ""))
	for _, key := range []string{"rendered_command", "wrapped_command"} {
		compactarComandoMetadataHandle(meta, key)
	}
}

func asignarResumenPromptHandle(meta map[string]any, key, raw string) {
	if meta == nil {
		return
	}
	raw = strings.TrimSpace(raw)
	delete(meta, key)
	delete(meta, key+"_summary")
	delete(meta, key+"_len")
	if raw == "" {
		return
	}
	meta[key+"_summary"] = resumirPromptHandle(raw)
	meta[key+"_len"] = len([]rune(raw))
}

func resumirPromptHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	first := raw
	if idx := strings.IndexByte(first, '\n'); idx >= 0 {
		first = first[:idx]
	}
	first = strings.Join(strings.Fields(strings.TrimSpace(first)), " ")
	runes := []rune(first)
	if len(runes) > 160 {
		first = strings.TrimSpace(string(runes[:160])) + "..."
	}
	return first
}

func compactarComandoMetadataHandle(meta map[string]any, key string) {
	if meta == nil {
		return
	}
	raw := strings.TrimSpace(stringFromMap(meta, key, ""))
	if raw == "" {
		delete(meta, key+"_len")
		return
	}
	delete(meta, key+"_len")
	compactado := resumirComandoRuntimeHandle(raw)
	if compactado == raw {
		return
	}
	meta[key] = compactado
	meta[key+"_len"] = len([]rune(raw))
}

func resumirComandoRuntimeHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tokens, err := splitShellQuotedCommandRuntimeHandle(raw)
	if err != nil || len(tokens) == 0 {
		return raw
	}
	base := filepath.Base(strings.TrimSpace(tokens[0]))
	switch base {
	case "codex-perfil":
		if len(tokens) >= 2 {
			return joinShellQuotedTokens(tokens[:2])
		}
		return joinShellQuotedTokens(tokens[:1])
	case "codex":
		return joinShellQuotedTokens(tokens[:1])
	case "script":
		keep := minInt(len(tokens), 4)
		out := append([]string{}, tokens[:keep]...)
		if len(tokens) > keep {
			last := strings.TrimSpace(tokens[len(tokens)-1])
			if last != "" && last != tokens[keep-1] {
				out = append(out, "<omitted>", last)
			}
		}
		return joinShellQuotedTokens(out)
	default:
		if len(raw) <= 240 {
			return raw
		}
		return strings.TrimSpace(string([]rune(raw)[:240])) + "..."
	}
}

func joinShellQuotedTokens(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, shellQuoteRuntimeHandle(token))
	}
	return strings.Join(out, " ")
}

func shellQuoteRuntimeHandle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(raw, "'", `'\''`) + "'"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func splitShellQuotedCommandRuntimeHandle(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	var token strings.Builder
	inSingle := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case inSingle && ch == '\'':
			inSingle = false
		case !inSingle && ch == '\'':
			inSingle = true
		case !inSingle && (ch == ' ' || ch == '\t' || ch == '\n'):
			if token.Len() > 0 {
				out = append(out, token.String())
				token.Reset()
			}
		case ch == '\\' && i+1 < len(raw):
			i++
			token.WriteByte(raw[i])
		default:
			token.WriteByte(ch)
		}
	}
	if inSingle {
		return nil, fmt.Errorf("runtime_handle command con comillas sin cerrar")
	}
	if token.Len() > 0 {
		out = append(out, token.String())
	}
	return out, nil
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

func runtimeBootstrapLeaseMailboxIDs(order, startOrder *RuntimeOrder) []int64 {
	_, mailboxIDs, _ := runtimeBootstrapLeaseFromOrder(order)
	if len(mailboxIDs) > 0 {
		return mailboxIDs
	}
	if startOrder == nil {
		return nil
	}
	_, mailboxIDs, _ = runtimeBootstrapLeaseFromOrder(startOrder)
	return mailboxIDs
}

func runtimeBootstrapLeaseSesionID(order, startOrder *RuntimeOrder) int64 {
	if order == nil {
		return 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	if sesionID := int64FromAny(result["sesion_id"]); sesionID > 0 {
		return sesionID
	}
	if startOrder == nil {
		return 0
	}
	return int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["sesion_id"])
}

func runtimeBootstrapLeaseAffinityScore(order, startOrder *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance, sesionID int64) int {
	if order == nil {
		return 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	score := 0
	if handle != nil && handle.ID > 0 {
		switch {
		case order.HandleID != nil && *order.HandleID == handle.ID:
			score += 8
		case int64FromAny(result["handle_id"]) == handle.ID:
			score += 8
		case startOrder != nil && startOrder.HandleID != nil && *startOrder.HandleID == handle.ID:
			score += 8
		case startOrder != nil && int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["handle_id"]) == handle.ID:
			score += 8
		}
	}
	if runtime != nil && runtime.ID > 0 {
		switch {
		case order.RuntimeID != nil && *order.RuntimeID == runtime.ID:
			score += 4
		case int64FromAny(result["runtime_id"]) == runtime.ID:
			score += 4
		case startOrder != nil && startOrder.RuntimeID != nil && *startOrder.RuntimeID == runtime.ID:
			score += 4
		case startOrder != nil && int64FromAny(mapFromJSON(startOrder.ResultadoJSON)["runtime_id"]) == runtime.ID:
			score += 4
		}
	}
	if sesionID > 0 && runtimeBootstrapLeaseSesionID(order, startOrder) == sesionID {
		score += 2
	}
	return score
}

func runtimeBootstrapLeaseLinkedToStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	if order == nil || startOrder == nil {
		return false
	}
	if strings.TrimSpace(startOrder.Tipo) != "start" {
		return false
	}
	result := mapFromJSON(order.ResultadoJSON)
	return int64FromAny(result["start_order_id"]) == startOrder.ID
}

func runtimeBootstrapLeaseResumeDerivadoPrefiereStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	return order != nil &&
		strings.EqualFold(strings.TrimSpace(order.Tipo), "resume") &&
		runtimeBootstrapLeaseLinkedToStart(order, startOrder)
}

func runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(order *RuntimeOrder, startOrder *RuntimeOrder) bool {
	return order != nil &&
		!strings.EqualFold(strings.TrimSpace(order.Tipo), "resume") &&
		runtimeBootstrapLeaseLinkedToStart(order, startOrder)
}

func runtimeBootstrapLeaseMailboxIDsVigentes(mailboxIDs []int64) ([]int64, error) {
	if len(mailboxIDs) == 0 {
		return nil, nil
	}
	out := make([]int64, 0, len(mailboxIDs))
	for _, mailboxID := range mailboxIDs {
		if mailboxID <= 0 {
			continue
		}
		msg, err := GetRuntimeMailbox(mailboxID)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}
		switch strings.TrimSpace(msg.Estado) {
		case "pendiente", "entregado":
			out = append(out, mailboxID)
		}
	}
	return out, nil
}

func marcarRuntimeMailboxEntregadoPorIDs(ids []int64) error {
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if err := MarcarRuntimeMailboxEntregado(id); err != nil {
			return err
		}
	}
	return nil
}

func AckBootstrapRuntimeLease(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 && len(mailboxIDs) == 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if err := marcarBootstrapRuntimeLeaseEntregado(startOrderID, bootstrapOrderID, mailboxIDs, sesionID, ackSource, time.Now().UTC()); err != nil {
		return err
	}
	if err := marcarRuntimeMailboxConsumidoPorIDs(mailboxIDs); err != nil {
		return err
	}
	if bootstrapOrderID > 0 {
		order, err := GetRuntimeOrder(bootstrapOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":          true,
				"bootstrap":   true,
				"acked_by":    ackSource,
				"lease_state": "acked",
				"sesion_id":   sesionID,
				"mailbox_ids": mailboxIDs,
			})
			targetState := strings.TrimSpace(order.Estado)
			targetError := order.ErrorText
			if targetState == "" || targetState == "ejecutando" || targetState == "pendiente" || targetState == "tomada" {
				targetState = "completada"
				targetError = ""
			}
			if err := MarcarRuntimeOrderEstado(order.ID, targetState, resultado, targetError); err != nil {
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
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"ok":                 true,
				"acked_by":           ackSource,
				"lease_state":        "acked",
				"sesion_id":          sesionID,
				"bootstrap_order_id": bootstrapOrderID,
				"bootstrap_mailbox":  mailboxIDs,
			})
			targetState := strings.TrimSpace(order.Estado)
			targetError := order.ErrorText
			if targetState == "" || targetState == "ejecutando" || targetState == "pendiente" || targetState == "tomada" {
				targetState = "completada"
				targetError = ""
			}
			if err := MarcarRuntimeOrderEstado(order.ID, targetState, resultado, targetError); err != nil {
				return err
			}
		}
	}
	return nil
}

func marcarBootstrapRuntimeLeaseEntregado(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string, deliveredAt time.Time) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 && len(mailboxIDs) == 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if deliveredAt.IsZero() {
		deliveredAt = time.Now().UTC()
	}
	if err := marcarRuntimeMailboxEntregadoPorIDs(mailboxIDs); err != nil {
		return err
	}
	return marcarBootstrapRuntimeLeaseObservada(startOrderID, bootstrapOrderID, mailboxIDs, sesionID, ackSource, deliveredAt)
}

func marcarBootstrapRuntimeLeaseObservada(startOrderID, bootstrapOrderID int64, mailboxIDs []int64, sesionID int64, ackSource string, deliveredAt time.Time) error {
	if startOrderID <= 0 && bootstrapOrderID <= 0 {
		return nil
	}
	ackSource = strings.TrimSpace(ackSource)
	if ackSource == "" {
		ackSource = "agente_tick"
	}
	if deliveredAt.IsZero() {
		deliveredAt = time.Now().UTC()
	}
	if bootstrapOrderID > 0 {
		order, err := GetRuntimeOrder(bootstrapOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"bootstrap":    true,
				"delivered_by": ackSource,
				"delivered_at": deliveredAt.Format(time.RFC3339Nano),
				"lease_state":  "delivered",
				"sesion_id":    sesionID,
				"mailbox_ids":  mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, order.Estado, resultado, order.ErrorText); err != nil {
				return err
			}
		}
	}
	if startOrderID > 0 {
		order, err := GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
		if order != nil && order.Estado != "fallida" && order.Estado != "cancelada" && order.Estado != "expirada" {
			resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
				"delivered_by":       ackSource,
				"delivered_at":       deliveredAt.Format(time.RFC3339Nano),
				"lease_state":        "delivered",
				"sesion_id":          sesionID,
				"bootstrap_order_id": bootstrapOrderID,
				"bootstrap_mailbox":  mailboxIDs,
			})
			if err := MarcarRuntimeOrderEstado(order.ID, order.Estado, resultado, order.ErrorText); err != nil {
				return err
			}
		}
	}
	return nil
}

type bootstrapLeaseEvidence struct {
	delivered   bool
	deliveredAt time.Time
	consumed    bool
}

func bootstrapRuntimeLeaseEvidence(handle *RuntimeHandle, runtime *RuntimeInstance, order, startOrder *RuntimeOrder) bootstrapLeaseEvidence {
	snap := runtimeWorkerSnapshot(handle, runtime)
	if snap == nil || !snap.Alive() || snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		return bootstrapLeaseEvidence{}
	}
	baseline := runtimeBootstrapLeaseBaseline(order, startOrder)
	state := strings.ToLower(strings.TrimSpace(snap.EffectiveState()))
	deliveredAt := time.Time{}
	if readyAt := snap.ReadyTime(); readyAt != nil {
		deliveredAt = readyAt.UTC()
	} else if hb := snap.HeartbeatTime(); hb != nil {
		deliveredAt = hb.UTC()
	} else if updated := snap.UpdatedTime(); updated != nil {
		deliveredAt = updated.UTC()
	} else if state == "ready" || state == "running" || state == "starting" {
		if hb := snap.HeartbeatTime(); hb != nil {
			deliveredAt = hb.UTC()
		} else if updated := snap.UpdatedTime(); updated != nil {
			deliveredAt = updated.UTC()
		}
	}
	if deliveredAt.IsZero() {
		return bootstrapLeaseEvidence{}
	}
	if !baseline.IsZero() && deliveredAt.Before(baseline) {
		return bootstrapLeaseEvidence{}
	}
	evidence := bootstrapLeaseEvidence{
		delivered:   true,
		deliveredAt: deliveredAt,
	}
	if progressAt := snap.LastProgressTime(); progressAt != nil {
		progress := progressAt.UTC()
		if baseline.IsZero() || progress.After(baseline) {
			evidence.consumed = true
			return evidence
		}
	}
	if workerSnapshotAllowsBootstrapTMUXConsumeFromOutput(handle, snap) {
		if outputAt := snap.LastOutputTime(); outputAt != nil {
			output := outputAt.UTC()
			if baseline.IsZero() || output.After(baseline) {
				evidence.consumed = true
				return evidence
			}
		}
	}
	return evidence
}

func workerSnapshotAllowsBootstrapTMUXConsumeFromOutput(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot) bool {
	if snap == nil {
		return false
	}
	deliveryMode := runtimeagente.NormalizeMailboxDeliveryMode(snap.MailboxDeliveryMode())
	if deliveryMode == "" {
		deliveryMode = runtimeagente.NormalizeMailboxDeliveryMode(RuntimeHandleMailboxDeliveryMode(handle))
	}
	if deliveryMode != runtimeagente.MailboxDeliveryBootstrapOnly {
		return false
	}
	driver := strings.TrimSpace(snap.Driver())
	if driver == "" && handle != nil {
		driver = strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "driver", ""))
	}
	if !strings.EqualFold(driver, "tmux_cli_session") {
		return false
	}
	transport := strings.TrimSpace(snap.Transport())
	if transport == "" && handle != nil {
		transport = strings.TrimSpace(handle.Transporte)
	}
	if !strings.EqualFold(transport, "tmux") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(snap.EffectiveState())) {
	case "ready", "running":
		return true
	default:
		return false
	}
}

func runtimeWorkerSnapshot(handle *RuntimeHandle, runtime *RuntimeInstance) *runtimeagente.WorkerSnapshot {
	if handle == nil {
		return nil
	}
	metaJSON := strings.TrimSpace(handle.MetadataJSON)
	if metaJSON == "" {
		return nil
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(metaJSON); err == nil && snap != nil {
		return snap
	}
	return nil
}

func runtimeBootstrapLeaseBaseline(order, startOrder *RuntimeOrder) time.Time {
	if runtimeBootstrapLeaseLinkedToStart(order, startOrder) {
		return runtimeBootstrapLeaseBaselineStartOrder(startOrder)
	}
	latest := time.Time{}
	for _, candidate := range []time.Time{
		func() time.Time {
			if startOrder != nil && startOrder.StartedAt != nil {
				return startOrder.StartedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if order != nil && order.StartedAt != nil {
				return order.StartedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if startOrder != nil {
				return startOrder.CreatedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if order != nil {
				return order.CreatedAt.UTC()
			}
			return time.Time{}
		}(),
	} {
		if candidate.IsZero() {
			continue
		}
		if latest.IsZero() || candidate.After(latest) {
			latest = candidate
		}
	}
	return latest
}

func runtimeBootstrapLeaseBaselineStartOrder(startOrder *RuntimeOrder) time.Time {
	if startOrder == nil {
		return time.Time{}
	}
	if startOrder.StartedAt != nil && !startOrder.StartedAt.IsZero() {
		return startOrder.StartedAt.UTC()
	}
	if !startOrder.CreatedAt.IsZero() {
		return startOrder.CreatedAt.UTC()
	}
	return time.Time{}
}

func AckBootstrapRuntimeLeaseByEvidence(handle *RuntimeHandle, runtime *RuntimeInstance, ackSource string) error {
	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil || order == nil {
		return err
	}
	var startOrder *RuntimeOrder
	if startOrderID > 0 {
		startOrder, err = GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
	}
	evidence := bootstrapRuntimeLeaseEvidence(handle, runtime, order, startOrder)
	if !evidence.delivered {
		return nil
	}
	if err := runtimePromoverEstadoObservadoDesdeHandle(handle, runtime); err != nil {
		return err
	}
	if evidence.consumed {
		if err := marcarBootstrapRuntimeLeaseEntregado(startOrderID, order.ID, mailboxIDs, sesionID, ackSource, evidence.deliveredAt); err != nil {
			return err
		}
		return AckBootstrapRuntimeLease(startOrderID, order.ID, mailboxIDs, sesionID, ackSource)
	}
	if err := marcarBootstrapRuntimeLeaseObservada(startOrderID, order.ID, mailboxIDs, sesionID, ackSource, evidence.deliveredAt); err != nil {
		return err
	}
	return nil
}

func RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, int64, int64, error) {
	if mailboxID <= 0 {
		return false, 0, 0, nil
	}
	order, startOrderID, mailboxIDs, _, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxID, handle, runtime)
	if err != nil || order == nil {
		return false, 0, 0, err
	}
	if !runtimeBootstrapLeaseStateBlocksMailbox(mapFromJSON(order.ResultadoJSON)) {
		return false, 0, 0, nil
	}
	for _, id := range mailboxIDs {
		if id == mailboxID {
			return true, order.ID, startOrderID, nil
		}
	}
	return false, 0, 0, nil
}

func RuntimeMailboxEntregadoPorBootstrapObservado(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, int64, int64, error) {
	if mailboxID <= 0 {
		return false, 0, 0, nil
	}
	order, startOrderID, mailboxIDs, _, err := resolverBootstrapRuntimeLeaseObservadaParaMailbox(mailboxID, handle, runtime)
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

func runtimeBootstrapLeaseTieneReceiptUtil(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if strings.TrimSpace(stringFromMap(result, "receipt_source", "")) != "" {
		return true
	}
	if strings.TrimSpace(stringFromMap(result, "delivery_receipt_at", "")) != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(stringFromMap(result, "delivery_state", "")), "delivered")
}

func runtimeBootstrapLeaseTieneCoberturaDeclarada(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if len(int64SliceFromAny(result["mailbox_ids"])) > 0 {
		return true
	}
	return int64FromAny(result["start_order_id"]) > 0
}

func runtimeBootstrapLeaseStateBlocksMailbox(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if runtimeBootstrapLeaseTieneReceiptUtil(result) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(stringFromMap(result, "lease_state", "")), "waiting_for_evidence")
}

func runtimeOrderMantieneBootstrapLeasePendiente(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if !runtimeBootstrapLeaseTieneCoberturaDeclarada(result) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(stringFromMap(result, "lease_state", ""))) {
	case "waiting_for_evidence", "delivered":
		return true
	default:
		return false
	}
}

func resolverBootstrapRuntimeLeaseObservada(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(0, handle, runtime, true)
}

func resolverBootstrapRuntimeLeaseObservadaParaMailbox(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(mailboxID, handle, runtime, true)
}

func resolverBootstrapRuntimeLeasePendiente(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(0, handle, runtime, false)
}

func resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(mailboxID, handle, runtime, false)
}

func resolverBootstrapRuntimeLeaseConFiltro(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance, observed bool) (*RuntimeOrder, int64, []int64, int64, error) {
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

	targetSesionID := sesionID
	candidates, err := listarRuntimeOrdersBootstrapLeaseCandidatas(agente, proyectoID, observed)
	if err != nil {
		return nil, 0, nil, 0, err
	}
	var (
		selectedOrder        *RuntimeOrder
		selectedStartOrderID int64
		selectedMailboxIDs   []int64
		selectedSesionID     int64
		selectedScore        int
	)
	for _, order := range candidates {
		startOrderID, mailboxIDs, orderSesionID := runtimeBootstrapLeaseFromOrder(order)
		var startOrder *RuntimeOrder
		if startOrderID > 0 && startOrderID != order.ID {
			startOrder, err = GetRuntimeOrder(startOrderID)
			if err != nil {
				return nil, 0, nil, 0, err
			}
			if orderSesionID <= 0 {
				orderSesionID = runtimeBootstrapLeaseSesionID(order, startOrder)
			}
		}
		mailboxIDs = runtimeBootstrapLeaseMailboxIDs(order, startOrder)
		vigentes, vigentesErr := runtimeBootstrapLeaseMailboxIDsVigentes(mailboxIDs)
		if vigentesErr != nil {
			return nil, 0, nil, 0, vigentesErr
		}
		mailboxIDs = vigentes
		if len(mailboxIDs) == 0 {
			continue
		}
		if mailboxID > 0 {
			match := false
			for _, candidateMailboxID := range mailboxIDs {
				if candidateMailboxID == mailboxID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if targetSesionID > 0 && orderSesionID > 0 && orderSesionID != targetSesionID {
			continue
		}
		orderMatchesTargetSession := targetSesionID > 0 && orderSesionID == targetSesionID
		score := runtimeBootstrapLeaseAffinityScore(order, startOrder, handle, runtime, targetSesionID)
		if selectedOrder == nil {
			selectedOrder = order
			selectedStartOrderID = startOrderID
			selectedMailboxIDs = mailboxIDs
			selectedSesionID = orderSesionID
			selectedScore = score
			continue
		}
		selectedMatchesTargetSession := targetSesionID > 0 && selectedSesionID == targetSesionID
		if orderMatchesTargetSession != selectedMatchesTargetSession {
			if !orderMatchesTargetSession {
				continue
			}
			selectedOrder = order
			selectedStartOrderID = startOrderID
			selectedMailboxIDs = mailboxIDs
			selectedSesionID = orderSesionID
			selectedScore = score
			continue
		}
		if score < selectedScore {
			continue
		}
		if score == selectedScore {
			switch {
			case runtimeBootstrapLeaseResumeDerivadoPrefiereStart(order, selectedOrder):
				continue
			case runtimeBootstrapLeaseResumeDerivadoPrefiereStart(selectedOrder, order):
				// Prefiere la start fuente frente al resume derivado.
			case runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(order, selectedOrder):
				// Prefiere la bootstrap fuente cuando está enlazada a la start derivada.
			case runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(selectedOrder, order):
				continue
			default:
				if order.ID < selectedOrder.ID {
					continue
				}
			}
		}
		selectedOrder = order
		selectedStartOrderID = startOrderID
		selectedMailboxIDs = mailboxIDs
		selectedSesionID = orderSesionID
		selectedScore = score
	}
	if selectedOrder == nil {
		return nil, 0, nil, sesionID, nil
	}
	if targetSesionID <= 0 {
		targetSesionID = selectedSesionID
	}
	return selectedOrder, selectedStartOrderID, selectedMailboxIDs, targetSesionID, nil
}

func listarRuntimeOrdersBootstrapLeaseCandidatas(agente string, proyectoID *int64, observed bool) ([]*RuntimeOrder, error) {
	candidates := make([]*RuntimeOrder, 0, 8)
	for _, estado := range []string{"ejecutando", "tomada", "pendiente", "completada"} {
		estado := estado
		orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
			Agente:     &agente,
			ProyectoID: proyectoID,
			Estado:     &estado,
		})
		if err != nil {
			return nil, err
		}
		for _, order := range orders {
			if runtimeOrderCalificaComoBootstrapLeaseCandidata(order, observed) {
				candidates = append(candidates, order)
			}
		}
	}
	return candidates, nil
}

func runtimeOrderCalificaComoBootstrapLeaseCandidata(order *RuntimeOrder, observed bool) bool {
	if order == nil {
		return false
	}
	switch strings.TrimSpace(order.Tipo) {
	case "handoff", "resume", "start":
	default:
		return false
	}
	res := mapFromJSON(order.ResultadoJSON)
	switch strings.TrimSpace(order.Estado) {
	case "pendiente":
		if observed && runtimeBootstrapLeaseTieneReceiptUtil(res) && runtimeBootstrapLeaseTieneCoberturaDeclarada(res) {
			return true
		}
		if !runtimeOrderMantieneBootstrapLeasePendiente(res) &&
			!boolFromAny(res["deferred"]) &&
			stringFromMap(res, "estado_dispatch", "") == "" {
			return false
		}
	case "completada":
		leaseState := strings.ToLower(strings.TrimSpace(stringFromMap(res, "lease_state", "")))
		if leaseState != "waiting_for_evidence" && leaseState != "delivered" {
			return false
		}
		if !runtimeBootstrapLeaseTieneCoberturaDeclarada(res) {
			return false
		}
	}
	return runtimeBootstrapLeaseTieneReceiptUtil(res) == observed
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
	if err := runtimeOrdersHotIndexSyncByID(id); err != nil {
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
	if handle != nil {
		runtime, err := runtimeHandleRuntime(handle)
		if err != nil || runtime != nil {
			return runtime, err
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

func runtimeInstanceIfExists(runtimeID *int64) (*RuntimeInstance, error) {
	if runtimeID == nil || *runtimeID <= 0 {
		return nil, nil
	}
	runtime, err := GetRuntime(*runtimeID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func sesionIfExists(sesionID *int64) (*Sesion, error) {
	if sesionID == nil || *sesionID <= 0 {
		return nil, nil
	}
	if DB == nil {
		return nil, nil
	}
	sesion, err := GetSesionByID(*sesionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sesion, err
}

func resolverDestinoRuntimeOrderCanonico(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) (*RuntimeInstance, *RuntimeHandle, error) {
	if order == nil {
		return runtime, handle, nil
	}
	if handle != nil {
		validado, err := validarRuntimeHandleActivo(handle, order.Agente, order.ProyectoID)
		if err != nil {
			return nil, nil, err
		}
		if validado != nil && !runtimeHandleExcluidoDelActivoCanonico(validado) {
			handle = validado
			runtime, err = runtimeHandleRuntime(validado)
			if err != nil {
				return nil, nil, err
			}
			if err := refrescarRuntimeOrderDestinoCanonico(order, runtime, validado); err != nil {
				return nil, nil, err
			}
			return runtime, validado, nil
		}
		if runtimeOrderConservaHandleExplicitoParaControl(order, handle) {
			if runtime == nil {
				runtime, err = runtimeHandleRuntime(handle)
				if err != nil {
					return nil, nil, err
				}
			}
			return runtime, handle, nil
		}
		handle = nil
		runtime = nil
	}

	var (
		candidato *RuntimeHandle
		err       error
	)
	candidato, err = runtimeHandleCanonicoRecienteConFallback(order.Agente, order.ProyectoID)
	if err != nil || candidato == nil {
		return runtime, handle, err
	}
	runtime, err = runtimeHandleRuntime(candidato)
	if err != nil {
		return nil, nil, err
	}
	if err := refrescarRuntimeOrderDestinoCanonico(order, runtime, candidato); err != nil {
		return nil, nil, err
	}
	return runtime, candidato, nil
}

func runtimeOrderConservaHandleExplicitoParaControl(order *RuntimeOrder, handle *RuntimeHandle) bool {
	if order == nil || handle == nil || order.HandleID == nil || *order.HandleID != handle.ID {
		return false
	}
	if runtimeHandleExcluidoDelActivoCanonico(handle) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "pause", "stop":
		return true
	case "resume":
		return strings.EqualFold(strings.TrimSpace(handle.Estado), "pausado") || RuntimeHandlePauseRequiresFreshStart(handle)
	default:
		return false
	}
}

func refrescarRuntimeOrderDestinoCanonico(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) error {
	if order == nil || handle == nil {
		return nil
	}
	runtimeID := int64(0)
	if runtime != nil && runtime.ID > 0 {
		runtimeID = runtime.ID
	} else if resolvedRuntime, err := runtimeHandleRuntime(handle); err != nil {
		return err
	} else if resolvedRuntime != nil && resolvedRuntime.ID > 0 {
		runtimeID = resolvedRuntime.ID
	} else if handle.RuntimeID != nil && *handle.RuntimeID > 0 {
		runtimeID = *handle.RuntimeID
	}
	changed := order.HandleID == nil || *order.HandleID != handle.ID
	if !changed {
		switch {
		case runtimeID == 0 && order.RuntimeID != nil && *order.RuntimeID > 0:
			changed = true
		case runtimeID > 0 && (order.RuntimeID == nil || *order.RuntimeID != runtimeID):
			changed = true
		}
	}
	if changed && order.ID > 0 {
		if err := actualizarRuntimeOrderDestino(order.ID, runtime, handle); err != nil {
			return err
		}
	}
	order.HandleID = &handle.ID
	if runtimeID > 0 {
		order.RuntimeID = &runtimeID
	} else {
		order.RuntimeID = nil
	}
	return nil
}

func runtimePrincipalAgenteProyecto(agente string, proyectoID *int64) (*RuntimeInstance, error) {
	handle, err := runtimeHandleCanonicoRecienteConFallback(agente, proyectoID)
	if err != nil {
		return nil, err
	}
	if handle != nil {
		runtime, err := runtimeHandleRuntime(handle)
		if err != nil {
			return nil, err
		}
		if runtime != nil {
			return runtime, nil
		}
	}
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
	}
}

func runtimeOrderTiposDiferibles() []string {
	return []string{"send_instruction"}
}

func runtimeOrderTiposBootstrap() []string {
	return []string{"handoff", "resume", "start"}
}

func runtimeOrderTipoBootstrap(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "handoff", "resume", "start":
		return true
	default:
		return false
	}
}

func runtimeOrderTipoDespachable(tipo string) bool {
	switch strings.TrimSpace(tipo) {
	case "sync_status", "checkpoint", "nudge", "discordia", "start", "pause", "resume", "stop", "restart", "send_instruction", "handoff":
		return true
	default:
		return false
	}
}

func runtimeOrderBootstrapStaleCoveredByFollowup(order *RuntimeOrder) (bool, string, string, error) {
	if order == nil {
		return false, "", "", nil
	}
	tipo := strings.TrimSpace(order.Tipo)
	if tipo != "handoff" && tipo != "resume" && tipo != "start" {
		return false, "", "", nil
	}
	result := mapFromJSON(order.ResultadoJSON)
	if len(result) == 0 {
		return false, "", "", nil
	}
	startOrderID := int64FromAny(result["start_order_id"])
	if startOrderID <= 0 || startOrderID == order.ID {
		return false, "", "", nil
	}
	startOrder, err := GetRuntimeOrder(startOrderID)
	if err != nil {
		return false, "", "", err
	}
	if startOrder == nil {
		return false, "", "", nil
	}
	if !startOrder.CreatedAt.IsZero() && !order.CreatedAt.IsZero() && startOrder.CreatedAt.Before(order.CreatedAt) {
		return false, "", "", nil
	}
	resultado := mergeRuntimeOrderResultJSON(order.ResultadoJSON, map[string]any{
		"ok":                true,
		"obsoleta":          true,
		"superseded":        true,
		"superseded_reason": "bootstrap_followup_order_exists",
		"start_order_id":    startOrderID,
	})
	return true, resultado, "bootstrap cubierto por start_order_id más reciente", nil
}

func reconciliarRuntimeOrderStale(id int64, tipo string, now, cutoff time.Time) (bool, error) {
	order, err := GetRuntimeOrder(id)
	if err != nil {
		return false, err
	}
	if order != nil {
		switch strings.TrimSpace(order.Tipo) {
		case "start", "resume", "stop":
			runtime, handle, err := resolverDestinoRuntimeOrderCanonico(order, nil, nil)
			if err != nil {
				return false, err
			}
			reason := ""
			if satisfied, satisfiedReason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle); satisfied {
				reason = satisfiedReason
			} else if runtimeOrderControlObsoletaPorWorkerRecuperado(order, runtime, handle) {
				reason = "runtime_worker_recovered_after_order"
			}
			if strings.TrimSpace(reason) != "" {
				if err := runtimeOrderPromoverEstadoObservadoSiSatisfecha(order, runtime, handle); err != nil {
					return false, err
				}
				resultado := map[string]any{
					"ok":       true,
					"obsoleta": true,
					"reason":   reason,
				}
				if runtime != nil && runtime.ID > 0 {
					resultado["runtime_id"] = runtime.ID
				}
				if handle != nil && handle.ID > 0 {
					resultado["handle_id"] = handle.ID
				}
				data, _ := json.Marshal(resultado)
				if err := MarcarRuntimeOrderEstado(id, "completada", string(data), ""); err != nil {
					return false, err
				}
				return true, nil
			}
		}
	}
	if covered, resultado, detalle, coveredErr := runtimeOrderBootstrapStaleCoveredByFollowup(order); coveredErr != nil {
		return false, coveredErr
	} else if covered {
		if err := MarcarRuntimeOrderEstado(id, "completada", resultado, detalle); err != nil {
			return false, err
		}
		return true, nil
	}
	var (
		res     sql.Result
		execErr error
	)
	if runtimeOrderTipoDespachable(tipo) {
		res, execErr = DB.Exec(`
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
			     OR COALESCE(started_at, updated_at, created_at) <= ?
			  )`, id, now, cutoff)
	} else {
		res, execErr = DB.Exec(`
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
			     OR COALESCE(started_at, updated_at, created_at) <= ?
			  )`, id, now, cutoff)
	}
	if execErr != nil {
		return false, execErr
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
	for key, value := range extraerCapacidadesResumeSesion(s) {
		caps[key] = value
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
	for key, value := range extraerMetadataResumeSesion(s) {
		meta[key] = value
	}
	data, _ := json.Marshal(meta)
	return string(data)
}

func extraerMetadataResumeSesion(s *Sesion) map[string]any {
	if s == nil {
		return nil
	}
	envelope := ParseResumePayloadEnvelope(strings.TrimSpace(s.ResumePayloadJSON))
	if len(envelope) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, key := range []string{
		"driver",
		"transport",
		"endpoint",
		"launch_path",
		"resume_path",
		"input_path",
		"status_path",
		"pause_path",
		"continue_path",
		"stop_path",
		"auth_header",
		"auth_token_env",
		"auth_mode",
		"mailbox_delivery_mode",
		"pool_compartido",
		"can_send_input",
	} {
		if value, ok := envelope[key]; ok {
			out[key] = value
		}
	}
	if perfil, ok := envelope["perfil_ejecucion"].(map[string]any); ok {
		for _, key := range []string{
			"driver",
			"transport",
			"endpoint",
			"launch_path",
			"resume_path",
			"input_path",
			"status_path",
			"pause_path",
			"continue_path",
			"stop_path",
			"mailbox_delivery_mode",
			"pool_slug",
			"modelo",
			"perfil_tarea",
			"razonamiento",
			"can_send_input",
		} {
			if value, ok := perfil[key]; ok {
				out[key] = value
			}
		}
	}
	return out
}

func extraerCapacidadesResumeSesion(s *Sesion) map[string]any {
	meta := extraerMetadataResumeSesion(s)
	if len(meta) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, key := range []string{
		"can_send_input",
		"can_pause",
		"can_stop",
		"can_resume",
		"can_track_continuity",
		"can_stop_without_reauth",
		"mailbox_delivery_mode",
	} {
		if value, ok := meta[key]; ok {
			out[key] = value
		}
	}
	return out
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

func int64PtrFromMap(m map[string]any, key string) *int64 {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case int:
		val := int64(v)
		return &val
	case int64:
		return &v
	case float64:
		val := int64(v)
		return &val
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return &n
		}
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &n
		}
	}
	return nil
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

func RuntimeHandlePauseRequiresFreshStart(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") {
		return false
	}
	if runtimeHandlePreservesExternalSession(handle) {
		return false
	}
	if RuntimeHandleMailboxDeliveryMode(handle) == runtimeagente.MailboxDeliveryBootstrapOnly {
		return true
	}
	for _, raw := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
		stringFromMap(meta, "command", ""),
	} {
		lower := strings.ToLower(strings.TrimSpace(raw))
		if strings.HasPrefix(lower, "ollama run ") || strings.HasPrefix(lower, "ollama serve ") {
			return true
		}
	}
	return false
}

func runtimeHandleMailboxDeliveryModeExplicit(handle *RuntimeHandle) string {
	if handle == nil {
		return ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	for _, raw := range []string{
		stringFromMap(caps, "mailbox_delivery_mode", ""),
		stringFromMap(meta, "mailbox_delivery_mode", ""),
	} {
		if mode := runtimeagente.NormalizeMailboxDeliveryMode(raw); mode != "" {
			return mode
		}
	}
	return ""
}

func RuntimeHandleMailboxDeliveryMode(handle *RuntimeHandle) string {
	if handle == nil {
		return runtimeagente.MailboxDeliveryInteractive
	}
	meta := mapFromJSON(handle.MetadataJSON)
	caps := mapFromJSON(handle.CapabilitiesJSON)
	legacyTMUXPreferredCLI := runtimeHandleUsaLegacyCLITMUXPreferred(meta)
	externalSessionReady := runtimeHandleTieneExternalSessionID(handle, nil)
	localCLIBrokerNoInteractive := runtimeHandleUsaTMUXPreferredCLI(meta) && !RuntimeHandlePermiteSendInputInteractivo(handle)
	tmuxSessionResumeReady := strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "tmux_cli_session") &&
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		externalSessionReady
	processSessionResumeReady := runtimeHandleUsaCodexTTYInestable(meta) &&
		strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli") &&
		strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) &&
		externalSessionReady
	normalizeMode := func(raw string) string {
		mode := runtimeagente.NormalizeMailboxDeliveryMode(raw)
		if mode == "" {
			return ""
		}
		if mode == runtimeagente.MailboxDeliverySessionResume && !externalSessionReady {
			if runtimeHandlePuedeInteractuarAntesDeSessionResume(handle, meta) {
				return runtimeagente.MailboxDeliveryInteractive
			}
			return runtimeagente.MailboxDeliveryBootstrapOnly
		}
		if legacyTMUXPreferredCLI && mode == runtimeagente.MailboxDeliveryInteractive {
			return runtimeagente.MailboxDeliveryBootstrapOnly
		}
		return mode
	}
	explicitMode := normalizeMode(runtimeHandleMailboxDeliveryModeExplicit(handle))
	if explicitMode != "" {
		if explicitMode == runtimeagente.MailboxDeliveryBootstrapOnly && (processSessionResumeReady || tmuxSessionResumeReady) {
			return runtimeagente.MailboxDeliverySessionResume
		}
		return explicitMode
	}
	if tmuxSessionResumeReady || processSessionResumeReady {
		return runtimeagente.MailboxDeliverySessionResume
	}
	if localCLIBrokerNoInteractive {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if runtimeHandleUsaCodexTTYInestable(meta) &&
		!boolFromMap(meta, "mailbox_restart_safe") &&
		!boolFromMap(caps, "mailbox_restart_safe") &&
		!RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	if !RuntimeHandlePermiteSendInputInteractivo(handle) {
		return runtimeagente.MailboxDeliveryBootstrapOnly
	}
	return runtimeagente.MailboxDeliveryInteractive
}

func runtimeHandleUsaTMUXPreferredCLI(meta map[string]any) bool {
	if runtimeHandleUsaLegacyCLITMUXPreferred(meta) {
		return true
	}
	if meta == nil {
		return false
	}
	for _, candidate := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
		stringFromMap(meta, "herramienta", ""),
		stringFromMap(meta, "conector", ""),
		stringFromMap(meta, "profile_status_wrapper", ""),
	} {
		if runtimeOrderUsaCLITMUXPreferred(candidate) {
			return true
		}
	}
	return false
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
		sesion, err := sesionIfExists(handle.SesionID)
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
	meta := mapFromJSON(handle.MetadataJSON)
	if runtimeHandlePuedeInteractuarAntesDeSessionResume(handle, meta) {
		return true
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if _, ok := caps["can_send_input"]; ok && !boolFromMap(caps, "can_send_input") {
		return false
	}
	if _, ok := meta["can_send_input"]; ok && !boolFromMap(meta, "can_send_input") {
		return false
	}
	if runtimeHandleUsaLegacyCLITMUXPreferred(meta) {
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

func runtimeHandlePuedeInteractuarAntesDeSessionResume(handle *RuntimeHandle, meta map[string]any) bool {
	if handle == nil || !runtimeHandleUsaCodexTTYInestable(meta) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli") {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "cli") || !strings.EqualFold(strings.TrimSpace(handle.HandleKind), "process") {
		return false
	}
	if runtimeHandleTieneExternalSessionID(handle, nil) || strings.TrimSpace(stringFromMap(meta, "external_session_id", "")) != "" {
		return false
	}
	for _, key := range []string{"supervisor_ref", "stdin_path", "stdin_raw_path"} {
		if strings.TrimSpace(stringFromMap(meta, key, "")) != "" {
			return true
		}
	}
	return false
}

func runtimeHandleUsaCodexTTYInestable(meta map[string]any) bool {
	if runtimeHandleUsaLegacyCLITMUXPreferred(meta) {
		return true
	}
	if meta == nil {
		return false
	}
	for _, candidate := range []string{
		stringFromMap(meta, "herramienta", ""),
		stringFromMap(meta, "conector", ""),
		stringFromMap(meta, "profile_status_wrapper", ""),
	} {
		if runtimeHandleLooksLikeCodexCLIRef(candidate) {
			return true
		}
	}
	for _, candidate := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
	} {
		if runtimeHandleLooksLikeCodexCommand(candidate) {
			return true
		}
	}
	return false
}

func runtimeHandleLooksLikeCodexCLIRef(ref string) bool {
	lower := strings.ToLower(strings.TrimSpace(ref))
	if lower == "" {
		return false
	}
	switch lower {
	case "codex", "codex-cli":
		return true
	}
	base := filepath.Base(strings.Trim(lower, "'\""))
	return strings.HasPrefix(base, "codex-perfil")
}

func runtimeHandleLooksLikeCodexCommand(rendered string) bool {
	lower := strings.ToLower(strings.TrimSpace(rendered))
	if lower == "" {
		return false
	}
	if lower == "codex" || strings.HasPrefix(lower, "codex ") || strings.Contains(lower, "codex-perfil") {
		return true
	}
	first := lower
	if fields := strings.Fields(lower); len(fields) > 0 {
		first = fields[0]
	}
	first = strings.Trim(first, "'\"")
	base := filepath.Base(first)
	return base == "codex" || base == "codex-cli" || strings.HasPrefix(base, "codex-perfil")
}

func runtimeOrderBloqueaFallbackPID(order *RuntimeOrder, runtime *RuntimeInstance, handle *RuntimeHandle) bool {
	if handle != nil {
		meta := mapFromJSON(handle.MetadataJSON)
		driver, transport := runtimeHandleDriverTransportObserved(handle)
		if strings.EqualFold(transport, "tmux") ||
			strings.EqualFold(driver, "tmux_cli_session") ||
			runtimeHandleUsaLegacyCLITMUXPreferred(meta) {
			return true
		}
	}
	if runtime != nil && runtimeOrderUsaCLITMUXPreferred(runtime.Connector) {
		return true
	}
	sesion, err := resolverSesionParaOrden(order)
	if err == nil && sesion != nil && runtimeOrderUsaCLITMUXPreferred(connectorDesdeSesion(sesion), sesion.Herramienta) {
		return true
	}
	return false
}

func runtimeOrderUsaCLITMUXPreferred(refs ...string) bool {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(ref) {
			return true
		}
		lower := strings.ToLower(ref)
		if strings.Contains(lower, "codex") || strings.Contains(lower, "claude") || strings.Contains(lower, "gemini") {
			return true
		}
	}
	return false
}

func runtimeHandleBloqueaFallbackPID(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	driver, transport := runtimeHandleDriverTransportObserved(handle)
	meta := mapFromJSON(handle.MetadataJSON)
	if strings.EqualFold(transport, "tmux") ||
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") ||
		strings.EqualFold(driver, "tmux_cli_session") ||
		runtimeHandleUsaLegacyProcessPTY(meta) {
		return true
	}
	return false
}

func objetivoProcesoDesdeHandleRuntimeOrden(order *RuntimeOrder, handle *RuntimeHandle, runtime *RuntimeInstance) controlruntime.ObjetivoProceso {
	obj := controlruntime.ObjetivoProceso{}
	blockedPID := runtimeHandleBloqueaFallbackPID(handle)
	if handle != nil {
		driver, transport := runtimeHandleDriverTransportObserved(handle)
		if strings.EqualFold(transport, "tmux") || strings.EqualFold(driver, "tmux_cli_session") {
			obj.HandleKind = "session"
			obj.HandleRef = runtimeHandleTMUXSessionRefObserved(handle)
		} else {
			obj.HandleKind = handle.HandleKind
			obj.HandleRef = handle.HandleRef
		}
		obj.MetadataJSON = handle.MetadataJSON
	}
	if runtime != nil &&
		runtime.PID != nil &&
		*runtime.PID > 0 &&
		!blockedPID &&
		!runtimeOrderBloqueaFallbackPID(order, runtime, handle) &&
		(handle == nil || strings.TrimSpace(obj.HandleKind) == "" || strings.TrimSpace(obj.HandleKind) == "process") {
		obj.PID = runtime.PID
	}
	return obj
}

func objetivoProcesoDesdeHandleRuntime(handle *RuntimeHandle, runtime *RuntimeInstance) controlruntime.ObjetivoProceso {
	return objetivoProcesoDesdeHandleRuntimeOrden(nil, handle, runtime)
}

func runtimeHandleDriverTransportObserved(handle *RuntimeHandle) (string, string) {
	if handle == nil {
		return "", ""
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(handle.Transporte)
	if transport == "" {
		transport = strings.TrimSpace(stringFromMap(meta, "transport", ""))
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON); err == nil && snap != nil {
		if value := strings.TrimSpace(snap.Driver()); value != "" {
			driver = value
		}
		if value := strings.TrimSpace(snap.Transport()); value != "" {
			transport = value
		}
	}
	if transport == "" && strings.EqualFold(driver, "tmux_cli_session") {
		transport = "tmux"
	}
	return strings.ToLower(driver), strings.ToLower(transport)
}

func runtimeHandleUsaLegacyCLITMUXPreferred(meta map[string]any) bool {
	if meta == nil {
		return false
	}
	if !runtimeHandleUsaLegacyProcessPTY(meta) {
		return false
	}
	for _, candidate := range []string{
		stringFromMap(meta, "rendered_command", ""),
		stringFromMap(meta, "wrapped_command", ""),
		stringFromMap(meta, "herramienta", ""),
		stringFromMap(meta, "conector", ""),
		stringFromMap(meta, "profile_status_wrapper", ""),
	} {
		if !controlruntime.RenderedCommandLooksLikeTMUXPreferredCLI(candidate) {
			continue
		}
		return true
	}
	return false
}

func runtimeHandleUsaLegacyProcessPTY(meta map[string]any) bool {
	if meta == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(stringFromMap(meta, "driver", "")), "process_pty_cli")
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
		runtimeHandleHotReset()
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

func SincronizarRuntimeHandleSupervisado(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (*RuntimeHandle, *RuntimeInstance, string, error) {
	if handle == nil {
		return nil, runtime, "", nil
	}
	if runtime == nil {
		var err error
		runtime, err = runtimeHandleRuntime(handle)
		if err != nil {
			return nil, nil, "", err
		}
	}
	if runtimeHandleBloqueaReactivacionPorCanalRoto(handle) {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return nil, nil, "", err
		}
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
		return fresh, runtime, "", nil
	}
	if observed, _, err := observarProcesoLocalRuntime(handle, runtime, source); err != nil {
		return nil, nil, "", err
	} else if observed {
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		handle = fresh
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		} else if handle != nil {
			if refreshedRuntime, err := runtimeHandleRuntime(handle); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
	}
	if known, exists := controlruntime.TMUXSessionExistsMetadata(handle.MetadataJSON); known && !exists {
		if err := marcarProcesoLocalNoDisponible(handle, runtime); err != nil {
			return nil, nil, "", err
		}
		fresh, err := GetRuntimeHandle(handle.ID)
		if err != nil {
			return nil, nil, "", err
		}
		handle = fresh
		if runtime != nil && runtime.ID > 0 {
			if refreshedRuntime, err := GetRuntime(runtime.ID); err == nil && refreshedRuntime != nil {
				runtime = refreshedRuntime
			}
		}
	}
	handle, externalSessionID, err := SincronizarRuntimeHandleExternalSessionID(handle, runtime)
	if err != nil {
		return nil, nil, "", err
	}
	handle, err = normalizarRuntimeHandleTMUXCanonico(handle)
	if err != nil {
		return nil, nil, "", err
	}
	handle, err = compactarMetadataHandleRuntimePersistida(handle)
	if err != nil {
		return nil, nil, "", err
	}
	return handle, runtime, externalSessionID, nil
}

func normalizarRuntimeHandleTMUXCanonico(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil || handle.ID <= 0 {
		return handle, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.TrimSpace(stringFromMap(meta, "driver", ""))
	transport := strings.TrimSpace(stringFromMap(meta, "transport", ""))
	snap, snapErr := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if snapErr == nil && snap != nil {
		if driver == "" {
			driver = strings.TrimSpace(snap.Driver())
		}
		if transport == "" {
			transport = strings.TrimSpace(snap.Transport())
		}
	}
	if !strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		!strings.EqualFold(driver, "tmux_cli_session") &&
		!strings.EqualFold(transport, "tmux") &&
		!runtimeHandleEsTMUXCanonico(handle) {
		return handle, nil
	}
	canonicalRef := runtimeHandleTMUXSessionRefObserved(handle)
	if canonicalRef == "" {
		if snapErr != nil {
			return handle, nil
		}
		if snap != nil {
			canonicalRef = strings.TrimSpace(snap.RuntimeRef())
		}
	}
	if canonicalRef == "" {
		return handle, nil
	}
	if strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") &&
		strings.TrimSpace(handle.HandleRef) == canonicalRef {
		return handle, nil
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session',
		    handle_ref=?,
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`, canonicalRef, handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	return getRuntimeHandleRaw(handle.ID)
}

func compactarMetadataHandleRuntimePersistida(handle *RuntimeHandle) (*RuntimeHandle, error) {
	if handle == nil || handle.ID <= 0 {
		return handle, nil
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if meta == nil {
		return handle, nil
	}
	before, _ := json.Marshal(meta)
	compactarMetadataRuntimeHandle(meta)
	after, _ := json.Marshal(meta)
	if string(before) == string(after) {
		return handle, nil
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, string(after), handle.ID); err != nil {
		return nil, err
	}
	runtimeHandleHotReset()
	return getRuntimeHandleRaw(handle.ID)
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
		runtimeHandleHotReset()
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
		sesion, err := sesionIfExists(handle.SesionID)
		if err != nil {
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
	obj := objetivoProcesoDesdeHandleRuntime(handle, runtime)
	return controlruntime.DetectExternalSessionID(obj)
}

func runtimeHandleExternalSessionIncompatible(handle *RuntimeHandle) bool {
	if handle == nil {
		return false
	}
	meta := mapFromJSON(handle.MetadataJSON)
	driver := strings.ToLower(strings.TrimSpace(stringFromMap(meta, "driver", "")))
	if driver != "tmux_cli_session" &&
		!strings.EqualFold(strings.TrimSpace(handle.Transporte), "tmux") &&
		!strings.EqualFold(strings.TrimSpace(handle.HandleKind), "session") {
		return false
	}
	externalSessionID := strings.ToLower(strings.TrimSpace(stringFromMap(meta, "external_session_id", "")))
	if externalSessionID == "" {
		if effective, err := runtimeHandleEffectiveExternalSessionID(handle, nil); err == nil {
			externalSessionID = strings.ToLower(strings.TrimSpace(effective))
		}
	}
	if !strings.HasPrefix(externalSessionID, "ollama-pool-") {
		return false
	}
	commandHints := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(handle.HandleRef),
		strings.TrimSpace(stringFromMap(meta, "rendered_command", "")),
		strings.TrimSpace(stringFromMap(meta, "wrapped_command", "")),
		strings.TrimSpace(stringFromMap(meta, "command", "")),
		strings.TrimSpace(stringFromMap(meta, "herramienta", "")),
		strings.TrimSpace(stringFromMap(meta, "connector", "")),
		strings.TrimSpace(stringFromMap(meta, "conector", "")),
	}, " "))
	if strings.Contains(commandHints, "ollama run ") || strings.Contains(commandHints, "ollama serve") || strings.Contains(commandHints, "ollama_pool_local") {
		return false
	}
	for _, token := range []string{"claude", "gemini", "codex"} {
		if strings.Contains(commandHints, token) {
			return true
		}
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
