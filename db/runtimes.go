package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"orquesta/internal/observabilidadruntime"
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
	ID          int64          `json:"id"`
	RuntimeID   int64          `json:"runtime_id"`
	Kind        string         `json:"kind"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	PayloadJSON string         `json:"payload_json"`
	Payload     map[string]any `json:"payload,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
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

type FiltroRuntimeEvents struct {
	RuntimeID  *int64
	Agente     *string
	ProyectoID *int64
	Kind       *string
	Limit      int
}

type FiltroRuntimes struct {
	Agente     *string
	ProyectoID *int64
	Activos    *bool
}

func RegistrarRuntimeInstance(r *RuntimeInstance) (int64, error) {
	if r == nil {
		return 0, fmt.Errorf("runtime instance nula")
	}
	if strings.TrimSpace(r.Agente) == "" {
		return 0, fmt.Errorf("agente es obligatorio")
	}
	normalizarRuntimeInstance(r)

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	id, err := upsertRuntimeInstanceTx(tx, r)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit(r.Agente, "registrar_runtime_instance", "runtime_instance", id, r.Provider)
	return id, nil
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
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return err
	}
	if strings.TrimSpace(agente) == "" {
		return nil
	}
	_, err = DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='cerrado',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE agente = ? AND logical_state <> 'cerrado'`, agente)
	return err
}

func MarcarRuntimeCerrado(id int64) error {
	if id <= 0 {
		return nil
	}
	_, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='cerrado',
		    process_state='finalizado',
		    last_event_at=CURRENT_TIMESTAMP
		WHERE id = ? AND logical_state <> 'cerrado'`, id)
	return err
}

func GetRuntime(id int64) (*RuntimeInstance, error) {
	return consultarConReintentos(func() (*RuntimeInstance, error) {
		row := DB.QueryRow(runtimeSelectBase()+` WHERE r.id = ?`, id)
		runtime, err := escanearRuntime(row)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return runtime, err
	})
}

func GetRuntimeBySesionID(sesionID int64) (*RuntimeInstance, error) {
	return consultarConReintentos(func() (*RuntimeInstance, error) {
		row := DB.QueryRow(runtimeSelectBase()+` WHERE r.sesion_id = ? ORDER BY r.id DESC LIMIT 1`, sesionID)
		runtime, err := escanearRuntime(row)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return runtime, err
	})
}

func ListarRuntimes(filter FiltroRuntimes) ([]*RuntimeInstance, error) {
	q := runtimeSelectBase() + ` WHERE 1=1`
	var args []any
	if filter.Agente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agente)
		if err != nil {
			return nil, err
		}
		q += ` AND r.agente = ?`
		args = append(args, agenteCanonico)
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

func NormalizarMuestraRuntime(sample *RuntimeTelemetrySample) (*observabilidadruntime.Sample, error) {
	if sample == nil {
		return nil, fmt.Errorf("runtime telemetry sample nula")
	}
	return observabilidadruntime.Normalize(sample.Source, json.RawMessage(sample.SampleJSON))
}

func NormalizarUltimaMuestraRuntime(runtimeID int64) (*observabilidadruntime.Sample, error) {
	sample, err := UltimaMuestraRuntime(runtimeID)
	if err != nil {
		return nil, err
	}
	if sample == nil {
		return nil, nil
	}
	return NormalizarMuestraRuntime(sample)
}

func ListarRuntimeEvents(filter FiltroRuntimeEvents) ([]*RuntimeEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	rowsQuery := `
		SELECT e.id, e.runtime_id, e.kind, e.level, e.message, e.payload_json, e.created_at
		FROM runtime_events e
		INNER JOIN runtime_instances r ON r.id = e.runtime_id
		WHERE 1=1`
	args := make([]any, 0, 5)
	if filter.RuntimeID != nil {
		rowsQuery += ` AND e.runtime_id = ?`
		args = append(args, *filter.RuntimeID)
	}
	if filter.Agente != nil {
		agenteCanonico, err := CanonicalizeAgentName(*filter.Agente)
		if err != nil {
			return nil, err
		}
		rowsQuery += ` AND r.agente = ?`
		args = append(args, agenteCanonico)
	}
	if filter.ProyectoID != nil {
		rowsQuery += ` AND r.proyecto_id = ?`
		args = append(args, *filter.ProyectoID)
	}
	if filter.Kind != nil {
		rowsQuery += ` AND e.kind = ?`
		args = append(args, strings.TrimSpace(*filter.Kind))
	}
	rowsQuery += ` ORDER BY e.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := DB.Query(rowsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*RuntimeEvent
	for rows.Next() {
		var item RuntimeEvent
		if err := rows.Scan(
			&item.ID,
			&item.RuntimeID,
			&item.Kind,
			&item.Level,
			&item.Message,
			&item.PayloadJSON,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.PayloadJSON) != "" {
			payload := map[string]any{}
			if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err == nil && len(payload) > 0 {
				item.Payload = payload
			}
		}
		out = append(out, &item)
	}
	return out, rows.Err()
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

	id, err := insertReturningIDWith(tx, `
		INSERT INTO runtime_events (runtime_id, kind, level, message, payload_json)
		VALUES (?,?,?,?,?)`,
		e.RuntimeID, e.Kind, e.Level, e.Message, e.PayloadJSON,
	)
	if err != nil {
		return 0, err
	}
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
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return nil, err
	}
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

func RuntimeInstancePorSesionID(sesionID int64) (*RuntimeInstance, error) {
	row := DB.QueryRow(runtimeSelectBase()+` WHERE r.sesion_id = ?`, sesionID)
	runtime, err := escanearRuntime(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
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
		if parsed, ok := parseLegacyDBTimestamp(ultimaActividad.String); ok {
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

func parseLegacyDBTimestamp(v string) (time.Time, bool) {
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
	QueryRow(query string, args ...any) *sql.Row
}, sample *RuntimeTelemetrySample) (int64, error) {
	id, err := insertReturningIDWith(tx, `
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

func upsertRuntimeInstanceTx(tx *Tx, r *RuntimeInstance) (int64, error) {
	existente, err := runtimeInstancePorSesionTx(tx, r.SesionID)
	if err != nil {
		return 0, err
	}
	if r.ID <= 0 && existente != nil {
		r.ID = existente.ID
	}

	if r.ID > 0 {
		if err := actualizarRuntimeInstanceTx(tx, r); err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	id, err := insertReturningIDWith(tx, `
		INSERT INTO runtime_instances (
			agente, proyecto_id, sesion_id, parent_runtime_id, provider, connector, external_session_id,
			logical_state, process_state, pid, ppid, child_count, thread_count, model, reasoning,
			task_profile, cwd, branch, last_event_at, last_heartbeat_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.Agente, runtimeNullableInt64(r.ProyectoID), runtimeNullableInt64(r.SesionID), runtimeNullableInt64(r.ParentRuntimeID),
		r.Provider, r.Connector, r.ExternalSessionID, r.LogicalState, r.ProcessState,
		runtimeNullableInt64(r.PID), runtimeNullableInt64(r.PPID), r.ChildCount, r.ThreadCount, r.Model,
		r.Reasoning, r.TaskProfile, r.CWD, r.Branch, runtimeNullableTime(r.LastEventAt), runtimeNullableTime(r.LastHeartbeatAt),
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func actualizarRuntimeInstanceTx(tx *Tx, r *RuntimeInstance) error {
	res, err := tx.Exec(`
		UPDATE runtime_instances
		SET agente = ?,
		    proyecto_id = ?,
		    sesion_id = ?,
		    parent_runtime_id = ?,
		    provider = ?,
		    connector = ?,
		    external_session_id = ?,
		    logical_state = ?,
		    process_state = ?,
		    pid = ?,
		    ppid = ?,
		    child_count = ?,
		    thread_count = ?,
		    model = ?,
		    reasoning = ?,
		    task_profile = ?,
		    cwd = ?,
		    branch = ?,
		    last_event_at = ?,
		    last_heartbeat_at = ?
		WHERE id = ?`,
		r.Agente, runtimeNullableInt64(r.ProyectoID), runtimeNullableInt64(r.SesionID), runtimeNullableInt64(r.ParentRuntimeID),
		r.Provider, r.Connector, r.ExternalSessionID, r.LogicalState, r.ProcessState,
		runtimeNullableInt64(r.PID), runtimeNullableInt64(r.PPID), r.ChildCount, r.ThreadCount, r.Model,
		r.Reasoning, r.TaskProfile, r.CWD, r.Branch, runtimeNullableTime(r.LastEventAt), runtimeNullableTime(r.LastHeartbeatAt),
		r.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("runtime instance #%d no encontrada", r.ID)
	}
	return nil
}

func runtimeInstancePorSesionTx(tx *Tx, sesionID *int64) (*RuntimeInstance, error) {
	if sesionID == nil {
		return nil, nil
	}
	row := tx.QueryRow(runtimeSelectBase()+` WHERE r.sesion_id = ?`, *sesionID)
	runtime, err := escanearRuntime(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func normalizarRuntimeInstance(r *RuntimeInstance) {
	if strings.TrimSpace(r.LogicalState) == "" {
		r.LogicalState = "arrancando"
	}
	if strings.TrimSpace(r.ProcessState) == "" {
		r.ProcessState = "desconocido"
	}
}

func runtimeNullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func runtimeNullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}
