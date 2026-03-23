package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/runtimeobs"
	"strings"
	"time"
)

type RuntimeInstance struct {
	ID                int64      `json:"id"`
	Agente            string     `json:"agente"`
	ProyectoID        *int64     `json:"proyecto_id,omitempty"`
	ProyectoSlug      string     `json:"proyecto_slug,omitempty"`
	ProyectoNombre    string     `json:"proyecto_nombre,omitempty"`
	SesionID          *int64     `json:"sesion_id,omitempty"`
	ParentRuntimeID   *int64     `json:"parent_runtime_id,omitempty"`
	Provider          string     `json:"provider"`
	Connector         string     `json:"connector"`
	ExternalSessionID string     `json:"external_session_id"`
	LogicalState      string     `json:"logical_state"`
	ProcessState      string     `json:"process_state"`
	PID               *int64     `json:"pid,omitempty"`
	PPID              *int64     `json:"ppid,omitempty"`
	ChildCount        int        `json:"child_count"`
	ThreadCount       int        `json:"thread_count"`
	Model             string     `json:"model"`
	Reasoning         string     `json:"reasoning"`
	TaskProfile       string     `json:"task_profile"`
	CWD               string     `json:"cwd"`
	Branch            string     `json:"branch"`
	LastEventAt       *time.Time `json:"last_event_at,omitempty"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty"`
	UltimaActividadAt *time.Time `json:"ultima_actividad_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type RuntimeEvent struct {
	ID          int64     `json:"id"`
	RuntimeID   int64     `json:"runtime_id"`
	Kind        string    `json:"kind"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	PayloadJSON string    `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type RuntimeTelemetrySample struct {
	ID           int64     `json:"id"`
	RuntimeID    int64     `json:"runtime_id"`
	CPUPct       float64   `json:"cpu_pct"`
	MemBytes     int64     `json:"mem_bytes"`
	RSSBytes     int64     `json:"rss_bytes"`
	OpenFDs      int64     `json:"open_fds"`
	ChildCount   int       `json:"child_count"`
	ThreadCount  int       `json:"thread_count"`
	LogicalState string    `json:"logical_state"`
	Source       string    `json:"source"`
	SampleJSON   string    `json:"sample_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type RuntimeTreeNode struct {
	Runtime *RuntimeInstance   `json:"runtime"`
	Hijos   []*RuntimeTreeNode `json:"hijos"`
}

type FiltroRuntimes struct {
	Agente     *string
	ProyectoID *int64
	Activos    *bool
}

func UpsertRuntimeDesdeSesion(s *Sesion) error {
	if s == nil {
		return nil
	}
	provider := providerDesdeSesion(s)
	connector := connectorDesdeSesion(s)
	logicalState := logicalStateDesdeSesion(s)
	processState := "desconocido"
	if s.Fin != nil || !s.Activa {
		processState = "finalizado"
	} else if s.PID != nil && *s.PID > 0 {
		processState = "vivo"
	}

	existente, err := GetRuntimeBySesionID(s.ID)
	if err != nil {
		return err
	}
	if existente == nil {
		_, err = DB.Exec(`
			INSERT INTO runtime_instances (
				agente, proyecto_id, sesion_id, provider, connector, external_session_id,
				logical_state, process_state, pid, cwd, branch, last_event_at, last_heartbeat_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,?)`,
			s.Agente, s.ProyectoID, s.ID, provider, connector, s.ExternalSessionID,
			logicalState, processState, s.PID, s.CWD, s.Branch, s.HeartbeatAt,
		)
	} else {
		_, err = DB.Exec(`
			UPDATE runtime_instances
			SET agente = ?,
			    proyecto_id = ?,
			    provider = ?,
			    connector = ?,
			    external_session_id = ?,
			    logical_state = ?,
			    process_state = ?,
			    pid = ?,
			    cwd = ?,
			    branch = ?,
			    last_event_at = CURRENT_TIMESTAMP,
			    last_heartbeat_at = ?
			WHERE id = ?`,
			s.Agente, s.ProyectoID, provider, connector, s.ExternalSessionID,
			logicalState, processState, s.PID, s.CWD, s.Branch, s.HeartbeatAt, existente.ID,
		)
	}
	if err != nil {
		return err
	}
	runtime, err := GetRuntimeBySesionID(s.ID)
	if err != nil || runtime == nil {
		return err
	}
	if _, err := RegistrarMuestraGenericProcess(runtime.ID, runtime); err != nil {
		return insertarMuestraRuntime(runtime.ID, runtime.LogicalState, "sesion_sync")
	}
	return nil
}

func MarcarRuntimesCerradosPorAgente(agente string) error {
	if strings.TrimSpace(agente) == "" {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='cerrado',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE agente = ? AND logical_state <> 'cerrado'`, agente)
	return err
}

func GetRuntime(id int64) (*RuntimeInstance, error) {
	row := DB.QueryRow(runtimeSelectBase()+` WHERE r.id = ?`, id)
	return escanearRuntime(row)
}

func GetRuntimeBySesionID(sesionID int64) (*RuntimeInstance, error) {
	row := DB.QueryRow(runtimeSelectBase()+` WHERE r.sesion_id = ? ORDER BY r.id DESC LIMIT 1`, sesionID)
	runtime, err := escanearRuntime(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func ListarRuntimes(filter FiltroRuntimes) ([]*RuntimeInstance, error) {
	q := runtimeSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.Agente != nil {
		q += ` AND r.agente = ?`
		args = append(args, *filter.Agente)
	}
	if filter.ProyectoID != nil {
		q += ` AND r.proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Activos != nil {
		if *filter.Activos {
			q += ` AND r.logical_state <> 'cerrado'`
		} else {
			q += ` AND r.logical_state = 'cerrado'`
		}
	}
	q += ` ORDER BY COALESCE(r.parent_runtime_id, r.id), r.id`
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeInstance
	for rows.Next() {
		runtime, err := escanearRuntime(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, runtime)
	}
	return out, rows.Err()
}

func ConstruirArbolRuntimes(filter FiltroRuntimes) ([]*RuntimeTreeNode, error) {
	runtimes, err := ListarRuntimes(filter)
	if err != nil {
		return nil, err
	}
	nodes := make(map[int64]*RuntimeTreeNode, len(runtimes))
	roots := make([]*RuntimeTreeNode, 0, len(runtimes))
	for _, runtime := range runtimes {
		nodes[runtime.ID] = &RuntimeTreeNode{Runtime: runtime}
	}
	for _, runtime := range runtimes {
		node := nodes[runtime.ID]
		if runtime.ParentRuntimeID != nil {
			if parent, ok := nodes[*runtime.ParentRuntimeID]; ok {
				parent.Hijos = append(parent.Hijos, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	return roots, nil
}

func ListarMuestrasRuntime(runtimeID int64, limit int) ([]*RuntimeTelemetrySample, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := DB.Query(`
		SELECT id, runtime_id, cpu_pct, mem_bytes, rss_bytes, open_fds, child_count, thread_count,
		       logical_state, source, sample_json, created_at
		FROM runtime_telemetry_samples
		WHERE runtime_id = ?
		ORDER BY id DESC
		LIMIT ?`, runtimeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RuntimeTelemetrySample
	for rows.Next() {
		var sample RuntimeTelemetrySample
		if err := rows.Scan(
			&sample.ID, &sample.RuntimeID, &sample.CPUPct, &sample.MemBytes, &sample.RSSBytes,
			&sample.OpenFDs, &sample.ChildCount, &sample.ThreadCount, &sample.LogicalState,
			&sample.Source, &sample.SampleJSON, &sample.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &sample)
	}
	return out, rows.Err()
}

func UltimaMuestraRuntime(runtimeID int64) (*RuntimeTelemetrySample, error) {
	row := DB.QueryRow(`
		SELECT id, runtime_id, cpu_pct, mem_bytes, rss_bytes, open_fds,
		       child_count, thread_count, logical_state, source, sample_json, created_at
		FROM runtime_telemetry_samples
		WHERE runtime_id = ?
		ORDER BY id DESC
		LIMIT 1`, runtimeID)
	sample, err := escanearRuntimeTelemetrySample(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sample, err
}

func NormalizarMuestraRuntime(sample *RuntimeTelemetrySample) (*runtimeobs.Sample, error) {
	if sample == nil {
		return nil, fmt.Errorf("runtime telemetry sample nula")
	}
	return runtimeobs.Normalize(sample.Source, json.RawMessage(sample.SampleJSON))
}

func NormalizarUltimaMuestraRuntime(runtimeID int64) (*runtimeobs.Sample, error) {
	sample, err := UltimaMuestraRuntime(runtimeID)
	if err != nil {
		return nil, err
	}
	if sample == nil {
		return nil, nil
	}
	return NormalizarMuestraRuntime(sample)
}

func RegistrarRuntimeEvent(e *RuntimeEvent) (int64, error) {
	if e == nil {
		return 0, fmt.Errorf("runtime event nulo")
	}
	if e.RuntimeID <= 0 {
		return 0, fmt.Errorf("runtime_id es obligatorio")
	}
	if strings.TrimSpace(e.Kind) == "" {
		return 0, fmt.Errorf("kind es obligatorio")
	}
	if strings.TrimSpace(e.Level) == "" {
		e.Level = "info"
	}
	if strings.TrimSpace(e.PayloadJSON) == "" {
		e.PayloadJSON = "{}"
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO runtime_events (runtime_id, kind, level, message, payload_json)
		VALUES (?,?,?,?,?)`,
		e.RuntimeID, e.Kind, e.Level, e.Message, e.PayloadJSON,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(
		`UPDATE runtime_instances SET last_event_at = CURRENT_TIMESTAMP WHERE id = ?`,
		e.RuntimeID,
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func RegistrarRuntimeTelemetrySample(sample *RuntimeTelemetrySample) (int64, error) {
	if sample == nil {
		return 0, fmt.Errorf("runtime telemetry sample nula")
	}
	if sample.RuntimeID <= 0 {
		return 0, fmt.Errorf("runtime_id es obligatorio")
	}
	if strings.TrimSpace(sample.Source) == "" {
		return 0, fmt.Errorf("source es obligatorio")
	}
	if strings.TrimSpace(sample.SampleJSON) == "" {
		sample.SampleJSON = sampleJSONConFuente(sample.Source)
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	id, err := registrarRuntimeTelemetrySampleTx(tx, sample)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`
		UPDATE runtime_instances
		SET last_heartbeat_at = CURRENT_TIMESTAMP,
		    child_count = ?,
		    thread_count = ?,
		    logical_state = COALESCE(NULLIF(?, ''), logical_state)
		WHERE id = ?`,
		sample.ChildCount, sample.ThreadCount, sample.LogicalState, sample.RuntimeID,
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	sample.ID = id
	return id, nil
}

func insertarMuestraRuntime(runtimeID int64, logicalState, source string) error {
	_, err := RegistrarRuntimeTelemetrySample(&RuntimeTelemetrySample{
		RuntimeID:    runtimeID,
		LogicalState: logicalState,
		Source:       source,
		SampleJSON:   sampleJSONConFuente(source),
	})
	return err
}

func RuntimePrincipalAgente(agente string) (*RuntimeInstance, error) {
	row := DB.QueryRow(runtimeSelectBase()+`
		WHERE r.agente = ?
		ORDER BY `+runtimeActividadExpr("r")+` DESC, r.id DESC
		LIMIT 1`, strings.TrimSpace(agente))
	runtime, err := escanearRuntime(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func ListarRuntimesConUltimaActividad() ([]*RuntimeInstance, error) {
	rows, err := DB.Query(`
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch, r.last_event_at,
		       r.last_heartbeat_at,
		       ` + runtimeActividadExpr("r") + ` AS ultima_actividad_at,
		       r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id
		ORDER BY ultima_actividad_at DESC, r.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*RuntimeInstance
	for rows.Next() {
		runtime, err := escanearRuntimeConActividad(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, runtime)
	}
	return out, rows.Err()
}

func runtimeSelectBase() string {
	return `
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug,''), COALESCE(p.nombre,''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch, r.last_event_at,
		       r.last_heartbeat_at, r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id`
}

func escanearRuntime(s scanner) (*RuntimeInstance, error) {
	var runtime RuntimeInstance
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var parentID sql.NullInt64
	var pid sql.NullInt64
	var ppid sql.NullInt64
	var lastEvent sql.NullTime
	var lastHeartbeat sql.NullTime
	if err := s.Scan(
		&runtime.ID, &runtime.Agente, &proyectoID, &runtime.ProyectoSlug, &runtime.ProyectoNombre,
		&sesionID, &parentID, &runtime.Provider, &runtime.Connector, &runtime.ExternalSessionID,
		&runtime.LogicalState, &runtime.ProcessState, &pid, &ppid, &runtime.ChildCount, &runtime.ThreadCount,
		&runtime.Model, &runtime.Reasoning, &runtime.TaskProfile, &runtime.CWD, &runtime.Branch,
		&lastEvent, &lastHeartbeat, &runtime.CreatedAt, &runtime.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		runtime.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		runtime.SesionID = &sesionID.Int64
	}
	if parentID.Valid {
		runtime.ParentRuntimeID = &parentID.Int64
	}
	if pid.Valid {
		runtime.PID = &pid.Int64
	}
	if ppid.Valid {
		runtime.PPID = &ppid.Int64
	}
	if lastEvent.Valid {
		runtime.LastEventAt = &lastEvent.Time
	}
	if lastHeartbeat.Valid {
		runtime.LastHeartbeatAt = &lastHeartbeat.Time
	}
	return &runtime, nil
}

func escanearRuntimeConActividad(s scanner) (*RuntimeInstance, error) {
	var runtime RuntimeInstance
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var parentID sql.NullInt64
	var pid sql.NullInt64
	var ppid sql.NullInt64
	var lastEvent sql.NullTime
	var lastHeartbeat sql.NullTime
	var ultimaActividad sql.NullString
	if err := s.Scan(
		&runtime.ID, &runtime.Agente, &proyectoID, &runtime.ProyectoSlug, &runtime.ProyectoNombre,
		&sesionID, &parentID, &runtime.Provider, &runtime.Connector, &runtime.ExternalSessionID,
		&runtime.LogicalState, &runtime.ProcessState, &pid, &ppid, &runtime.ChildCount, &runtime.ThreadCount,
		&runtime.Model, &runtime.Reasoning, &runtime.TaskProfile, &runtime.CWD, &runtime.Branch,
		&lastEvent, &lastHeartbeat, &ultimaActividad, &runtime.CreatedAt, &runtime.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		runtime.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		runtime.SesionID = &sesionID.Int64
	}
	if parentID.Valid {
		runtime.ParentRuntimeID = &parentID.Int64
	}
	if pid.Valid {
		runtime.PID = &pid.Int64
	}
	if ppid.Valid {
		runtime.PPID = &ppid.Int64
	}
	if lastEvent.Valid {
		runtime.LastEventAt = &lastEvent.Time
	}
	if lastHeartbeat.Valid {
		runtime.LastHeartbeatAt = &lastHeartbeat.Time
	}
	if ultimaActividad.Valid {
		if parsed, ok := parseSQLiteTimestamp(ultimaActividad.String); ok {
			runtime.UltimaActividadAt = &parsed
		}
	}
	return &runtime, nil
}

func escanearRuntimeTelemetrySample(s scanner) (*RuntimeTelemetrySample, error) {
	var sample RuntimeTelemetrySample
	if err := s.Scan(
		&sample.ID, &sample.RuntimeID, &sample.CPUPct, &sample.MemBytes, &sample.RSSBytes, &sample.OpenFDs,
		&sample.ChildCount, &sample.ThreadCount, &sample.LogicalState, &sample.Source, &sample.SampleJSON, &sample.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &sample, nil
}

func sampleJSONConFuente(source string) string {
	sampleJSON, _ := json.Marshal(map[string]any{"source": source})
	return string(sampleJSON)
}

func parseSQLiteTimestamp(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.UTC); err == nil {
		return parsed, true
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, v); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func connectorDesdeSesion(s *Sesion) string {
	connector := strings.TrimSpace(s.ConectorSlug)
	if connector == "" {
		connector = strings.TrimSpace(s.Herramienta)
	}
	return connector
}

func providerDesdeSesion(s *Sesion) string {
	ref := strings.ToLower(connectorDesdeSesion(s))
	switch {
	case strings.Contains(ref, "claude"):
		return "anthropic"
	case strings.Contains(ref, "codex"):
		return "openai"
	case strings.Contains(ref, "gemini"):
		return "google"
	default:
		return ref
	}
}

func logicalStateDesdeSesion(s *Sesion) string {
	switch strings.ToLower(strings.TrimSpace(s.Estado)) {
	case "pausada":
		return "pausado"
	case "cerrada":
		return "cerrado"
	case "fallida":
		return "bloqueado"
	default:
		return "disponible"
	}
}

func registrarRuntimeTelemetrySampleTx(tx interface {
	Exec(query string, args ...any) (sql.Result, error)
}, sample *RuntimeTelemetrySample) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO runtime_telemetry_samples (
			runtime_id, cpu_pct, mem_bytes, rss_bytes, open_fds, child_count, thread_count,
			logical_state, source, sample_json
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		sample.RuntimeID, sample.CPUPct, sample.MemBytes, sample.RSSBytes, sample.OpenFDs,
		sample.ChildCount, sample.ThreadCount, sample.LogicalState, sample.Source, sample.SampleJSON,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func runtimeActividadExpr(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "r"
	}
	return `
		CASE
			WHEN ` + alias + `.last_event_at IS NULL THEN COALESCE(` + alias + `.last_heartbeat_at, ` + alias + `.updated_at, ` + alias + `.created_at)
			WHEN ` + alias + `.last_heartbeat_at IS NULL THEN COALESCE(` + alias + `.last_event_at, ` + alias + `.updated_at, ` + alias + `.created_at)
			WHEN ` + alias + `.last_event_at >= ` + alias + `.last_heartbeat_at THEN ` + alias + `.last_event_at
			ELSE ` + alias + `.last_heartbeat_at
		END`
}
