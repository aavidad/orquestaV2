package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

type runtimeRowScanner interface {
	Scan(dest ...any) error
}

// RegistrarRuntimeInstance crea o actualiza una instancia runtime.
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

// RegistrarRuntimeEvent persiste un evento de observabilidad y refresca la última actividad.
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
	Audit("orquesta", "registrar_runtime_event", "runtime_event", id, e.Kind)
	return id, nil
}

// RegistrarRuntimeTelemetrySample guarda una muestra de telemetría y refresca la instancia.
func RegistrarRuntimeTelemetrySample(s *RuntimeTelemetrySample) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("runtime telemetry sample nula")
	}
	if s.RuntimeID <= 0 {
		return 0, fmt.Errorf("runtime_id es obligatorio")
	}
	if strings.TrimSpace(s.Source) == "" {
		return 0, fmt.Errorf("source es obligatorio")
	}
	if strings.TrimSpace(s.SampleJSON) == "" {
		s.SampleJSON = "{}"
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO runtime_telemetry_samples (
		    runtime_id, cpu_pct, mem_bytes, rss_bytes, open_fds,
		    child_count, thread_count, logical_state, source, sample_json
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		s.RuntimeID, s.CPUPct, s.MemBytes, s.RSSBytes, s.OpenFDs,
		s.ChildCount, s.ThreadCount, s.LogicalState, s.Source, s.SampleJSON,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`
		UPDATE runtime_instances
		SET last_heartbeat_at = CURRENT_TIMESTAMP,
		    child_count = ?,
		    thread_count = ?,
		    logical_state = COALESCE(NULLIF(?, ''), logical_state)
		WHERE id = ?`,
		s.ChildCount, s.ThreadCount, s.LogicalState, s.RuntimeID,
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	Audit("orquesta", "registrar_runtime_telemetry_sample", "runtime_telemetry_sample", id, s.Source)
	return id, nil
}

// UltimaMuestraRuntime devuelve la última muestra registrada para una instancia.
func UltimaMuestraRuntime(runtimeID int64) (*RuntimeTelemetrySample, error) {
	row := DB.QueryRow(`
		SELECT id, runtime_id, cpu_pct, mem_bytes, rss_bytes, open_fds,
		       child_count, thread_count, logical_state, source, sample_json, created_at
		FROM runtime_telemetry_samples
		WHERE runtime_id = ?
		ORDER BY id DESC
		LIMIT 1`, runtimeID)
	return escanearRuntimeTelemetrySample(row)
}

func RuntimePrincipalAgente(agente string) (*RuntimeInstance, error) {
	row := DB.QueryRow(`
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug, ''), COALESCE(p.nombre, ''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch,
		       r.last_event_at, r.last_heartbeat_at, r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id
		WHERE r.agente = ?
		ORDER BY COALESCE(r.last_heartbeat_at, r.last_event_at, r.updated_at, r.created_at) DESC, r.id DESC
		LIMIT 1`, strings.TrimSpace(agente))
	return escanearRuntimeInstance(row)
}

// ListarRuntimesConUltimaActividad devuelve las instancias con su actividad más reciente.
func ListarRuntimesConUltimaActividad() ([]*RuntimeInstance, error) {
	rows, err := DB.Query(`
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug, ''), COALESCE(p.nombre, ''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch,
		       r.last_event_at, r.last_heartbeat_at,
		       COALESCE(MAX(e.created_at), MAX(t.created_at), r.last_event_at, r.last_heartbeat_at, r.updated_at, r.created_at) AS ultima_actividad_at,
		       r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id
		LEFT JOIN runtime_events e ON e.runtime_id = r.id
		LEFT JOIN runtime_telemetry_samples t ON t.runtime_id = r.id
		GROUP BY r.id
		ORDER BY ultima_actividad_at DESC, r.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RuntimeInstance
	for rows.Next() {
		r, err := escanearRuntimeInstanceConActividad(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func upsertRuntimeInstanceTx(tx *sql.Tx, r *RuntimeInstance) (int64, error) {
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

	res, err := tx.Exec(`
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
	id, _ := res.LastInsertId()
	return id, nil
}

func actualizarRuntimeInstanceTx(tx *sql.Tx, r *RuntimeInstance) error {
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

func runtimeInstancePorSesionTx(tx *sql.Tx, sesionID *int64) (*RuntimeInstance, error) {
	if sesionID == nil {
		return nil, nil
	}
	row := tx.QueryRow(`
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug, ''), COALESCE(p.nombre, ''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch, r.last_event_at, r.last_heartbeat_at,
		       r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id
		WHERE r.sesion_id = ?`,
		*sesionID,
	)
	runtime, err := escanearRuntimeInstance(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return runtime, err
}

func RuntimeInstancePorSesionID(sesionID int64) (*RuntimeInstance, error) {
	row := DB.QueryRow(`
		SELECT r.id, r.agente, r.proyecto_id, COALESCE(p.slug, ''), COALESCE(p.nombre, ''),
		       r.sesion_id, r.parent_runtime_id, r.provider, r.connector, r.external_session_id,
		       r.logical_state, r.process_state, r.pid, r.ppid, r.child_count, r.thread_count,
		       r.model, r.reasoning, r.task_profile, r.cwd, r.branch, r.last_event_at, r.last_heartbeat_at,
		       r.created_at, r.updated_at
		FROM runtime_instances r
		LEFT JOIN proyectos p ON p.id = r.proyecto_id
		WHERE r.sesion_id = ?`, sesionID)
	return escanearRuntimeInstance(row)
}

func escanearRuntimeInstance(s runtimeRowScanner) (*RuntimeInstance, error) {
	var r RuntimeInstance
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var parentID sql.NullInt64
	var pid sql.NullInt64
	var ppid sql.NullInt64
	var lastEvent sql.NullTime
	var lastHeartbeat sql.NullTime
	if err := s.Scan(
		&r.ID, &r.Agente, &proyectoID, &r.ProyectoSlug, &r.ProyectoNombre,
		&sesionID, &parentID, &r.Provider, &r.Connector, &r.ExternalSessionID,
		&r.LogicalState, &r.ProcessState, &pid, &ppid, &r.ChildCount, &r.ThreadCount,
		&r.Model, &r.Reasoning, &r.TaskProfile, &r.CWD, &r.Branch,
		&lastEvent, &lastHeartbeat, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		r.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		r.SesionID = &sesionID.Int64
	}
	if parentID.Valid {
		r.ParentRuntimeID = &parentID.Int64
	}
	if pid.Valid {
		r.PID = &pid.Int64
	}
	if ppid.Valid {
		r.PPID = &ppid.Int64
	}
	if lastEvent.Valid {
		r.LastEventAt = &lastEvent.Time
	}
	if lastHeartbeat.Valid {
		r.LastHeartbeatAt = &lastHeartbeat.Time
	}
	return &r, nil
}

func escanearRuntimeInstanceConActividad(s runtimeRowScanner) (*RuntimeInstance, error) {
	r, err := escanearRuntimeInstanceConBase(s)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func escanearRuntimeInstanceConBase(s runtimeRowScanner) (*RuntimeInstance, error) {
	var r RuntimeInstance
	var proyectoID sql.NullInt64
	var sesionID sql.NullInt64
	var parentID sql.NullInt64
	var pid sql.NullInt64
	var ppid sql.NullInt64
	var lastEvent sql.NullTime
	var lastHeartbeat sql.NullTime
	var ultimaActividad sql.NullString
	if err := s.Scan(
		&r.ID, &r.Agente, &proyectoID, &r.ProyectoSlug, &r.ProyectoNombre,
		&sesionID, &parentID, &r.Provider, &r.Connector, &r.ExternalSessionID,
		&r.LogicalState, &r.ProcessState, &pid, &ppid, &r.ChildCount, &r.ThreadCount,
		&r.Model, &r.Reasoning, &r.TaskProfile, &r.CWD, &r.Branch,
		&lastEvent, &lastHeartbeat, &ultimaActividad, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if proyectoID.Valid {
		r.ProyectoID = &proyectoID.Int64
	}
	if sesionID.Valid {
		r.SesionID = &sesionID.Int64
	}
	if parentID.Valid {
		r.ParentRuntimeID = &parentID.Int64
	}
	if pid.Valid {
		r.PID = &pid.Int64
	}
	if ppid.Valid {
		r.PPID = &ppid.Int64
	}
	if lastEvent.Valid {
		r.LastEventAt = &lastEvent.Time
	}
	if lastHeartbeat.Valid {
		r.LastHeartbeatAt = &lastHeartbeat.Time
	}
	if ultimaActividad.Valid {
		if parsed, ok := parseSQLiteTimestamp(ultimaActividad.String); ok {
			r.UltimaActividadAt = &parsed
		}
	}
	return &r, nil
}

func escanearRuntimeTelemetrySample(s runtimeRowScanner) (*RuntimeTelemetrySample, error) {
	var sample RuntimeTelemetrySample
	if err := s.Scan(
		&sample.ID, &sample.RuntimeID, &sample.CPUPct, &sample.MemBytes, &sample.RSSBytes, &sample.OpenFDs,
		&sample.ChildCount, &sample.ThreadCount, &sample.LogicalState, &sample.Source, &sample.SampleJSON, &sample.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &sample, nil
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

func sampleJSONConFuente(source string) string {
	b, err := json.Marshal(map[string]any{"source": source})
	if err != nil {
		return "{}"
	}
	return string(b)
}

func parseSQLiteTimestamp(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
