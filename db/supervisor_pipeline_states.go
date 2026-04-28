package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SupervisorPipelineState struct {
	ID             int64      `json:"id"`
	Supervisor     string     `json:"supervisor"`
	ProyectoSlug   string     `json:"proyecto_slug,omitempty"`
	PipelineName   string     `json:"pipeline_name,omitempty"`
	Role           string     `json:"role,omitempty"`
	CurrentPhase   string     `json:"current_phase,omitempty"`
	Status         string     `json:"status"`
	CurrentTaskID  *int64     `json:"current_task_id,omitempty"`
	CurrentGateID  *int64     `json:"current_gate_id,omitempty"`
	CurrentMergeID *int64     `json:"current_merge_id,omitempty"`
	ArtifactsJSON  string     `json:"artifacts_json,omitempty"`
	MetadataJSON   string     `json:"metadata_json,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type UpsertSupervisorPipelineStateInput struct {
	Supervisor     string
	ProyectoSlug   string
	PipelineName   string
	Role           string
	CurrentPhase   string
	Status         string
	CurrentTaskID  *int64
	CurrentGateID  *int64
	CurrentMergeID *int64
	ArtifactsJSON  string
	MetadataJSON   string
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

type FiltroSupervisorPipelineStates struct {
	Supervisor   string
	ProyectoSlug string
	Role         string
	Status       string
	CurrentPhase string
	Limit        int
}

func ensureSupervisorPipelineStatesSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS supervisor_pipeline_states (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			supervisor      TEXT    NOT NULL,
			proyecto_slug   TEXT    NOT NULL DEFAULT '',
			pipeline_name   TEXT    NOT NULL DEFAULT '',
			role            TEXT    NOT NULL DEFAULT 'executor',
			current_phase   TEXT    NOT NULL DEFAULT '',
			status          TEXT    NOT NULL DEFAULT 'active'
			                        CHECK (status IN ('active','paused','blocked','completed','failed')),
			current_task_id INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			current_gate_id INTEGER REFERENCES review_gates(id) ON DELETE SET NULL,
			current_merge_id INTEGER REFERENCES git_merges(id) ON DELETE SET NULL,
			artifacts_json  TEXT    NOT NULL DEFAULT '{}',
			metadata_json   TEXT    NOT NULL DEFAULT '{}',
			started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at    DATETIME,
			UNIQUE(supervisor, proyecto_slug, pipeline_name)
		)`)
	if err != nil {
		return err
	}
	exists, err := ColumnExists("supervisor_pipeline_states", "role")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := DB.Exec(`ALTER TABLE supervisor_pipeline_states ADD COLUMN role TEXT NOT NULL DEFAULT 'executor'`); err != nil {
			return err
		}
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_supervisor_pipeline_states_supervisor ON supervisor_pipeline_states(supervisor, updated_at DESC)`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_supervisor_pipeline_states_role_status ON supervisor_pipeline_states(role, status, updated_at DESC)`)
	return err
}

func UpsertSupervisorPipelineState(input UpsertSupervisorPipelineStateInput) (*SupervisorPipelineState, error) {
	if err := ensureSupervisorPipelineStatesSchema(); err != nil {
		return nil, err
	}
	supervisor := strings.TrimSpace(input.Supervisor)
	if supervisor == "" {
		return nil, fmt.Errorf("supervisor obligatorio")
	}
	pipelineName := canonicalSupervisorPipelineName(input.PipelineName)
	role := canonicalSupervisorPipelineRole(input.Role, pipelineName)
	status := canonicalSupervisorPipelineStatus(input.Status)
	currentPhase := canonicalSupervisorPipelinePhase(input.CurrentPhase)
	if strings.TrimSpace(input.ArtifactsJSON) == "" {
		input.ArtifactsJSON = "{}"
	}
	if strings.TrimSpace(input.MetadataJSON) == "" {
		input.MetadataJSON = "{}"
	}
	startedAt := time.Now().UTC()
	if input.StartedAt != nil && !input.StartedAt.IsZero() {
		startedAt = input.StartedAt.UTC()
	}
	assignments := []upsertAssignment{
		{Column: "role"},
		{Column: "current_phase"},
		{Column: "status"},
		{Column: "current_task_id"},
		{Column: "current_gate_id"},
		{Column: "current_merge_id"},
		{Column: "artifacts_json"},
		{Column: "metadata_json"},
		{Column: "updated_at", Expr: "CURRENT_TIMESTAMP"},
		{Column: "completed_at"},
	}
	sqlText := upsertValuesSQL(
		"supervisor_pipeline_states",
		[]string{"supervisor", "proyecto_slug", "pipeline_name", "role", "current_phase", "status", "current_task_id", "current_gate_id", "current_merge_id", "artifacts_json", "metadata_json", "started_at", "completed_at"},
		[]string{"supervisor", "proyecto_slug", "pipeline_name"},
		assignments,
	)
	if _, err := DB.Exec(sqlText,
		supervisor,
		strings.TrimSpace(input.ProyectoSlug),
		pipelineName,
		role,
		currentPhase,
		status,
		input.CurrentTaskID,
		input.CurrentGateID,
		input.CurrentMergeID,
		strings.TrimSpace(input.ArtifactsJSON),
		strings.TrimSpace(input.MetadataJSON),
		startedAt,
		input.CompletedAt,
	); err != nil {
		return nil, err
	}
	return GetSupervisorPipelineState(supervisor, strings.TrimSpace(input.ProyectoSlug), pipelineName)
}

func GetSupervisorPipelineState(supervisor, proyectoSlug, pipelineName string) (*SupervisorPipelineState, error) {
	exists, err := SchemaObjectExists("table", "supervisor_pipeline_states")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	pipelineName = canonicalSupervisorPipelineName(pipelineName)
	row := DB.QueryRow(`
		SELECT id, supervisor, proyecto_slug, pipeline_name, role, current_phase, status,
		       current_task_id, current_gate_id, current_merge_id, artifacts_json,
		       metadata_json, started_at, updated_at, completed_at
		FROM supervisor_pipeline_states
		WHERE supervisor = ? AND proyecto_slug = ? AND pipeline_name = ?`,
		strings.TrimSpace(supervisor), strings.TrimSpace(proyectoSlug), strings.TrimSpace(pipelineName),
	)
	item, err := scanSupervisorPipelineState(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func ListarSupervisorPipelineStates(filter FiltroSupervisorPipelineStates) ([]*SupervisorPipelineState, error) {
	exists, err := SchemaObjectExists("table", "supervisor_pipeline_states")
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*SupervisorPipelineState{}, nil
	}
	q := `
		SELECT id, supervisor, proyecto_slug, pipeline_name, role, current_phase, status,
		       current_task_id, current_gate_id, current_merge_id, artifacts_json,
		       metadata_json, started_at, updated_at, completed_at
		FROM supervisor_pipeline_states
		WHERE 1=1`
	args := []any{}
	if v := strings.TrimSpace(filter.Supervisor); v != "" {
		q += ` AND supervisor = ?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.ProyectoSlug); v != "" {
		q += ` AND proyecto_slug = ?`
		args = append(args, v)
	}
	if v := canonicalSupervisorPipelineRoleFilter(filter.Role); v != "" {
		q += ` AND role = ?`
		args = append(args, v)
	}
	if v := canonicalSupervisorPipelineStatusFilter(filter.Status); v != "" {
		q += ` AND status = ?`
		args = append(args, v)
	}
	if v := canonicalSupervisorPipelinePhase(filter.CurrentPhase); v != "" {
		q += ` AND current_phase = ?`
		args = append(args, v)
	}
	q += ` ORDER BY updated_at DESC, id DESC`
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	q += ` LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SupervisorPipelineState
	for rows.Next() {
		item, err := scanSupervisorPipelineState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanSupervisorPipelineState(scanner interface{ Scan(dest ...any) error }) (*SupervisorPipelineState, error) {
	var item SupervisorPipelineState
	var taskID, gateID, mergeID sql.NullInt64
	var completedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.Supervisor, &item.ProyectoSlug, &item.PipelineName, &item.Role, &item.CurrentPhase, &item.Status,
		&taskID, &gateID, &mergeID, &item.ArtifactsJSON, &item.MetadataJSON, &item.StartedAt, &item.UpdatedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	if taskID.Valid {
		item.CurrentTaskID = &taskID.Int64
	}
	if gateID.Valid {
		item.CurrentGateID = &gateID.Int64
	}
	if mergeID.Valid {
		item.CurrentMergeID = &mergeID.Int64
	}
	if completedAt.Valid {
		ts := completedAt.Time
		item.CompletedAt = &ts
	}
	return &item, nil
}

func canonicalSupervisorPipelineName(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "default"
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-")
	raw = replacer.Replace(raw)
	return compactSupervisorPipelineSlug(raw, "-")
}

func canonicalSupervisorPipelineRole(raw, pipelineName string) string {
	if role := canonicalSupervisorPipelineRoleFilter(raw); role != "" {
		return role
	}
	switch {
	case strings.Contains(pipelineName, "supervisor"):
		return "supervisor"
	case strings.Contains(pipelineName, "review"):
		return "reviewer"
	case strings.Contains(pipelineName, "fix"), strings.Contains(pipelineName, "repair"):
		return "fixer"
	default:
		return "executor"
	}
}

func canonicalSupervisorPipelineRoleFilter(raw string) string {
	switch compactSupervisorPipelineSlug(strings.TrimSpace(strings.ToLower(raw)), "_") {
	case "supervisor", "lead", "manager":
		return "supervisor"
	case "executor", "worker", "implementer", "developer":
		return "executor"
	case "reviewer", "review", "qa", "validator":
		return "reviewer"
	case "fixer", "repair", "bugfix", "hotfix":
		return "fixer"
	default:
		return ""
	}
}

func canonicalSupervisorPipelineStatus(raw string) string {
	if status := canonicalSupervisorPipelineStatusFilter(raw); status != "" {
		return status
	}
	return "active"
}

func canonicalSupervisorPipelineStatusFilter(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "active", "paused", "blocked", "completed", "failed":
		return raw
	default:
		return ""
	}
}

func canonicalSupervisorPipelinePhase(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	raw = replacer.Replace(raw)
	return compactSupervisorPipelineSlug(raw, "_")
}

func compactSupervisorPipelineSlug(raw, sep string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '-', '_', ' ', '\t', '\n', '\r', '/':
			return true
		default:
			return false
		}
	})
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, sep)
}
