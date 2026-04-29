package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type SupervisorSubagent struct {
	ID              int64      `json:"id"`
	Supervisor      string     `json:"supervisor"`
	ProyectoSlug    string     `json:"proyecto_slug,omitempty"`
	SessionID       string     `json:"session_id,omitempty"`
	ParentThreadID  string     `json:"parent_thread_id,omitempty"`
	ThreadID        string     `json:"thread_id"`
	SubagentName    string     `json:"subagent_name,omitempty"`
	SubagentType    string     `json:"subagent_type"`
	ToolProfileJSON string     `json:"tool_profile_json,omitempty"`
	Status          string     `json:"status"`
	ManifestPath    string     `json:"manifest_path,omitempty"`
	OutputPath      string     `json:"output_path,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	MetadataJSON    string     `json:"metadata_json,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       time.Time  `json:"started_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type SupervisorSubagentToolProfile struct {
	Name         string   `json:"name"`
	SubagentType string   `json:"subagent_type"`
	Description  string   `json:"description,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

type UpsertSupervisorSubagentInput struct {
	Supervisor      string
	ProyectoSlug    string
	SessionID       string
	ParentThreadID  string
	ThreadID        string
	SubagentName    string
	SubagentType    string
	ToolProfileJSON string
	Status          string
	ManifestPath    string
	OutputPath      string
	ErrorMessage    string
	MetadataJSON    string
	CreatedAt       *time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
}

type FiltroSupervisorSubagents struct {
	Supervisor   string
	ProyectoSlug string
	SessionID    string
	Status       string
	SubagentType string
	Limit        int
}

var supervisorSubagentProfiles = map[string]SupervisorSubagentToolProfile{
	"general-purpose": {
		Name:         "general-purpose",
		SubagentType: "general-purpose",
		Description:  "Subagente generalista para ejecuciones delegadas que combinan lectura, edición y verificación.",
		AllowedTools: []string{"read_file", "rg", "ls", "bash", "apply_patch", "status_read", "api_read"},
	},
	"explore": {
		Name:         "explore",
		SubagentType: "explore",
		Description:  "Subagente explorador centrado en leer código, estado y contexto sin modificar por defecto.",
		AllowedTools: []string{"read_file", "rg", "ls", "status_read", "api_read", "mcp_read"},
	},
	"plan": {
		Name:         "plan",
		SubagentType: "plan",
		Description:  "Subagente de planificación y análisis con foco en roadmap, tareas y contratos.",
		AllowedTools: []string{"read_file", "rg", "status_read", "api_read", "task_read", "proposal_read", "mcp_read"},
	},
	"verification": {
		Name:         "verification",
		SubagentType: "verification",
		Description:  "Subagente de verificación que prioriza tests, diagnósticos y comprobaciones server-first.",
		AllowedTools: []string{"read_file", "rg", "bash", "go_test", "status_read", "api_read", "diagnostico_read"},
	},
}

func ensureSupervisorSubagentsSchema() error {
	return ensureSupervisorSchemaObjects(
		"CREATE TABLE IF NOT EXISTS supervisor_subagents (",
		"CREATE INDEX IF NOT EXISTS idx_supervisor_subagents_supervisor_session",
		"CREATE INDEX IF NOT EXISTS idx_supervisor_subagents_project_status",
	)
}

func NormalizeSupervisorSubagentType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "general", "general-purpose", "general_purpose", "worker":
		return "general-purpose"
	case "explore", "explorer", "research":
		return "explore"
	case "plan", "planner", "planning":
		return "plan"
	case "verification", "verify", "verifier", "review":
		return "verification"
	default:
		return "general-purpose"
	}
}

func SupervisorSubagentToolProfileForType(subagentType string) SupervisorSubagentToolProfile {
	normalized := NormalizeSupervisorSubagentType(subagentType)
	profile, ok := supervisorSubagentProfiles[normalized]
	if !ok {
		profile = supervisorSubagentProfiles["general-purpose"]
	}
	clone := profile
	clone.AllowedTools = append([]string(nil), profile.AllowedTools...)
	return clone
}

func ListSupervisorSubagentToolProfiles() []SupervisorSubagentToolProfile {
	keys := make([]string, 0, len(supervisorSubagentProfiles))
	for key := range supervisorSubagentProfiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]SupervisorSubagentToolProfile, 0, len(keys))
	for _, key := range keys {
		out = append(out, SupervisorSubagentToolProfileForType(key))
	}
	return out
}

func marshalSupervisorSubagentToolProfile(subagentType string) string {
	profile := SupervisorSubagentToolProfileForType(subagentType)
	payload, err := json.Marshal(profile)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func UpsertSupervisorSubagent(input UpsertSupervisorSubagentInput) (*SupervisorSubagent, error) {
	if err := ensureSupervisorSubagentsSchema(); err != nil {
		return nil, err
	}
	supervisor := strings.TrimSpace(input.Supervisor)
	threadID := strings.TrimSpace(input.ThreadID)
	if supervisor == "" || threadID == "" {
		return nil, fmt.Errorf("supervisor y thread_id obligatorios")
	}
	subagentType := NormalizeSupervisorSubagentType(input.SubagentType)
	status := strings.TrimSpace(strings.ToLower(input.Status))
	switch status {
	case "", "running":
		status = "running"
	case "completed", "failed", "cancelled":
	default:
		status = "running"
	}
	toolProfileJSON := strings.TrimSpace(input.ToolProfileJSON)
	if toolProfileJSON == "" || toolProfileJSON == "{}" {
		toolProfileJSON = marshalSupervisorSubagentToolProfile(subagentType)
	}
	metadataJSON := strings.TrimSpace(input.MetadataJSON)
	if metadataJSON == "" {
		metadataJSON = "{}"
	}
	createdAt := time.Now().UTC()
	if input.CreatedAt != nil && !input.CreatedAt.IsZero() {
		createdAt = input.CreatedAt.UTC()
	}
	startedAt := createdAt
	if input.StartedAt != nil && !input.StartedAt.IsZero() {
		startedAt = input.StartedAt.UTC()
	}
	insertColumns := []string{
		"supervisor", "proyecto_slug", "session_id", "parent_thread_id", "thread_id",
		"subagent_name", "subagent_type", "tool_profile_json", "status", "manifest_path", "output_path",
		"error_message", "metadata_json", "created_at", "started_at", "updated_at", "completed_at",
	}
	assignments := []upsertAssignment{
		{Column: "proyecto_slug"},
		{Column: "parent_thread_id"},
		{Column: "subagent_name"},
		{Column: "subagent_type"},
		{Column: "tool_profile_json"},
		{Column: "status"},
		{Column: "manifest_path"},
		{Column: "output_path"},
		{Column: "error_message"},
		{Column: "metadata_json"},
		{Column: "updated_at", Expr: "CURRENT_TIMESTAMP"},
		{Column: "completed_at"},
	}
	sqlText := buildUpsertValuesSQL(CurrentStorageDriver(), "supervisor_subagents", insertColumns, []string{"supervisor", "session_id", "thread_id"}, assignments)
	if _, err := DB.Exec(sqlText,
		supervisor,
		strings.TrimSpace(input.ProyectoSlug),
		strings.TrimSpace(input.SessionID),
		strings.TrimSpace(input.ParentThreadID),
		threadID,
		strings.TrimSpace(input.SubagentName),
		subagentType,
		toolProfileJSON,
		status,
		strings.TrimSpace(input.ManifestPath),
		strings.TrimSpace(input.OutputPath),
		strings.TrimSpace(input.ErrorMessage),
		metadataJSON,
		createdAt,
		startedAt,
		time.Now().UTC(),
		input.CompletedAt,
	); err != nil {
		return nil, err
	}
	return GetSupervisorSubagent(supervisor, strings.TrimSpace(input.SessionID), threadID)
}

func GetSupervisorSubagent(supervisor, sessionID, threadID string) (*SupervisorSubagent, error) {
	exists, err := SchemaObjectExists("table", "supervisor_subagents")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	row := DB.QueryRow(`
		SELECT id, supervisor, proyecto_slug, session_id, parent_thread_id, thread_id,
		       subagent_name, subagent_type, tool_profile_json, status, manifest_path,
		       output_path, error_message, metadata_json, created_at, started_at, updated_at, completed_at
		FROM supervisor_subagents
		WHERE supervisor = ? AND session_id = ? AND thread_id = ?`,
		strings.TrimSpace(supervisor), strings.TrimSpace(sessionID), strings.TrimSpace(threadID),
	)
	item, err := scanSupervisorSubagent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func GetSupervisorSubagentByID(id int64) (*SupervisorSubagent, error) {
	exists, err := SchemaObjectExists("table", "supervisor_subagents")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	row := DB.QueryRow(`
		SELECT id, supervisor, proyecto_slug, session_id, parent_thread_id, thread_id,
		       subagent_name, subagent_type, tool_profile_json, status, manifest_path,
		       output_path, error_message, metadata_json, created_at, started_at, updated_at, completed_at
		FROM supervisor_subagents
		WHERE id = ?`, id,
	)
	item, err := scanSupervisorSubagent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

func ListarSupervisorSubagents(filter FiltroSupervisorSubagents) ([]*SupervisorSubagent, error) {
	exists, err := SchemaObjectExists("table", "supervisor_subagents")
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*SupervisorSubagent{}, nil
	}
	q := `
		SELECT id, supervisor, proyecto_slug, session_id, parent_thread_id, thread_id,
		       subagent_name, subagent_type, tool_profile_json, status, manifest_path,
		       output_path, error_message, metadata_json, created_at, started_at, updated_at, completed_at
		FROM supervisor_subagents
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
	if v := strings.TrimSpace(filter.SessionID); v != "" {
		q += ` AND session_id = ?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		q += ` AND status = ?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.SubagentType); v != "" {
		q += ` AND subagent_type = ?`
		args = append(args, NormalizeSupervisorSubagentType(v))
	}
	q += ` ORDER BY updated_at DESC, id DESC`
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	q += ` LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SupervisorSubagent
	for rows.Next() {
		item, err := scanSupervisorSubagent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanSupervisorSubagent(scanner interface{ Scan(dest ...any) error }) (*SupervisorSubagent, error) {
	var item SupervisorSubagent
	var completedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.Supervisor, &item.ProyectoSlug, &item.SessionID, &item.ParentThreadID, &item.ThreadID,
		&item.SubagentName, &item.SubagentType, &item.ToolProfileJSON, &item.Status, &item.ManifestPath,
		&item.OutputPath, &item.ErrorMessage, &item.MetadataJSON, &item.CreatedAt, &item.StartedAt, &item.UpdatedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		ts := completedAt.Time
		item.CompletedAt = &ts
	}
	return &item, nil
}
