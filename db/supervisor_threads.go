package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

const DefaultSupervisorThreadActiveWindow = 2 * time.Minute

type SupervisorThread struct {
	ID           int64     `json:"id"`
	Supervisor   string    `json:"supervisor"`
	ProyectoSlug string    `json:"proyecto_slug,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	ThreadID     string    `json:"thread_id"`
	Kind         string    `json:"kind"`
	Mode         string    `json:"mode,omitempty"`
	Status       string    `json:"status"`
	Source       string    `json:"source,omitempty"`
	LastTurnID   string    `json:"last_turn_id,omitempty"`
	TurnCount    int       `json:"turn_count"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
	FirstSeenAt  time.Time `json:"first_seen_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type RecordSupervisorThreadInput struct {
	Supervisor   string
	ProyectoSlug string
	SessionID    string
	ThreadID     string
	Kind         string
	Mode         string
	Status       string
	Source       string
	TurnID       string
	MetadataJSON string
	Timestamp    time.Time
}

type FiltroSupervisorThreads struct {
	Supervisor   string
	ProyectoSlug string
	SessionID    string
	Status       string
	Limit        int
}

type SupervisorThreadSessionSummary struct {
	Supervisor              string              `json:"supervisor"`
	ProyectoSlug            string              `json:"proyecto_slug,omitempty"`
	SessionID               string              `json:"session_id,omitempty"`
	LeaderThreadID          string              `json:"leader_thread_id,omitempty"`
	AllThreadIDs            []string            `json:"all_thread_ids"`
	AllSubagentThreadIDs    []string            `json:"all_subagent_thread_ids"`
	ActiveSubagentThreadIDs []string            `json:"active_subagent_thread_ids"`
	UpdatedAt               *time.Time          `json:"updated_at,omitempty"`
	Threads                 []*SupervisorThread `json:"threads,omitempty"`
}

func ensureSupervisorThreadsSchema() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS supervisor_threads (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			supervisor    TEXT    NOT NULL,
			proyecto_slug TEXT    NOT NULL DEFAULT '',
			session_id    TEXT    NOT NULL DEFAULT '',
			thread_id     TEXT    NOT NULL,
			kind          TEXT    NOT NULL DEFAULT 'subagent'
			                      CHECK (kind IN ('leader','subagent')),
			mode          TEXT    NOT NULL DEFAULT '',
			status        TEXT    NOT NULL DEFAULT 'active'
			                      CHECK (status IN ('active','idle','closed')),
			source        TEXT    NOT NULL DEFAULT '',
			last_turn_id  TEXT    NOT NULL DEFAULT '',
			turn_count    INTEGER NOT NULL DEFAULT 1,
			metadata_json TEXT    NOT NULL DEFAULT '{}',
			first_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(supervisor, session_id, thread_id)
		)`)
	if err != nil {
		return err
	}
	if _, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_supervisor_threads_supervisor_session ON supervisor_threads(supervisor, session_id, last_seen_at DESC)`); err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_supervisor_threads_project ON supervisor_threads(proyecto_slug, supervisor, last_seen_at DESC)`)
	return err
}

func RecordSupervisorThreadTurn(input RecordSupervisorThreadInput) (*SupervisorThread, error) {
	if err := ensureSupervisorThreadsSchema(); err != nil {
		return nil, err
	}
	supervisor := strings.TrimSpace(input.Supervisor)
	threadID := strings.TrimSpace(input.ThreadID)
	if supervisor == "" || threadID == "" {
		return nil, fmt.Errorf("supervisor y thread_id obligatorios")
	}
	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now().UTC()
	}
	kind := strings.TrimSpace(input.Kind)
	if kind != "leader" {
		kind = "subagent"
	}
	status := strings.TrimSpace(input.Status)
	switch status {
	case "idle", "closed", "active":
	default:
		status = "active"
	}
	if strings.TrimSpace(input.MetadataJSON) == "" {
		input.MetadataJSON = "{}"
	}
	driver := CurrentStorageDriver()
	insertColumns := []string{
		"supervisor", "proyecto_slug", "session_id", "thread_id", "kind", "mode", "status",
		"source", "last_turn_id", "turn_count", "metadata_json", "first_seen_at", "last_seen_at",
	}
	assignments := []upsertAssignment{
		{Column: "proyecto_slug"},
		{Column: "kind"},
		{Column: "mode"},
		{Column: "status"},
		{Column: "source"},
		{Column: "last_turn_id"},
		{Column: "turn_count", Expr: "supervisor_threads.turn_count + 1"},
		{Column: "metadata_json"},
		{Column: "last_seen_at"},
	}
	sqlText := buildUpsertValuesSQL(driver, "supervisor_threads", insertColumns, []string{"supervisor", "session_id", "thread_id"}, assignments)
	if _, err := DB.Exec(sqlText,
		supervisor,
		strings.TrimSpace(input.ProyectoSlug),
		strings.TrimSpace(input.SessionID),
		threadID,
		kind,
		strings.TrimSpace(input.Mode),
		status,
		strings.TrimSpace(input.Source),
		strings.TrimSpace(input.TurnID),
		1,
		strings.TrimSpace(input.MetadataJSON),
		input.Timestamp,
		input.Timestamp,
	); err != nil {
		return nil, err
	}
	items, err := ListarSupervisorThreads(FiltroSupervisorThreads{
		Supervisor: supervisor,
		SessionID:  strings.TrimSpace(input.SessionID),
		Limit:      100,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item != nil && strings.EqualFold(strings.TrimSpace(item.ThreadID), threadID) {
			return item, nil
		}
	}
	return nil, nil
}

func ListarSupervisorThreads(filter FiltroSupervisorThreads) ([]*SupervisorThread, error) {
	exists, err := SchemaObjectExists("table", "supervisor_threads")
	if err != nil {
		return nil, err
	}
	if !exists {
		return []*SupervisorThread{}, nil
	}
	q := `
		SELECT id, supervisor, proyecto_slug, session_id, thread_id, kind, mode, status,
		       source, last_turn_id, turn_count, metadata_json, first_seen_at, last_seen_at
		FROM supervisor_threads
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
	q += ` ORDER BY last_seen_at DESC, id DESC`
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
	var out []*SupervisorThread
	for rows.Next() {
		item, err := scanSupervisorThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func SummarizeSupervisorThreadSession(supervisor, sessionID string, activeWindow time.Duration) (*SupervisorThreadSessionSummary, error) {
	if activeWindow <= 0 {
		activeWindow = DefaultSupervisorThreadActiveWindow
	}
	items, err := ListarSupervisorThreads(FiltroSupervisorThreads{
		Supervisor: strings.TrimSpace(supervisor),
		SessionID:  strings.TrimSpace(sessionID),
		Limit:      200,
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	now := time.Now().UTC()
	var leaderThreadID string
	threadIDs := make([]string, 0, len(items))
	subagentIDs := []string{}
	activeSubagentIDs := []string{}
	seenIDs := map[string]struct{}{}
	projectSlug := ""
	var updatedAt *time.Time
	for _, item := range items {
		if item == nil {
			continue
		}
		if projectSlug == "" && strings.TrimSpace(item.ProyectoSlug) != "" {
			projectSlug = strings.TrimSpace(item.ProyectoSlug)
		}
		if updatedAt == nil || item.LastSeenAt.After(*updatedAt) {
			ts := item.LastSeenAt
			updatedAt = &ts
		}
		threadID := strings.TrimSpace(item.ThreadID)
		if threadID == "" {
			continue
		}
		if _, ok := seenIDs[threadID]; !ok {
			seenIDs[threadID] = struct{}{}
			threadIDs = append(threadIDs, threadID)
		}
		if strings.TrimSpace(item.Kind) == "leader" && leaderThreadID == "" {
			leaderThreadID = threadID
		}
		if strings.TrimSpace(item.Kind) != "subagent" {
			continue
		}
		subagentIDs = appendIfMissing(subagentIDs, threadID)
		if now.Sub(item.LastSeenAt) <= activeWindow && strings.TrimSpace(item.Status) != "closed" {
			activeSubagentIDs = appendIfMissing(activeSubagentIDs, threadID)
		}
	}
	sort.Strings(threadIDs)
	sort.Strings(subagentIDs)
	sort.Strings(activeSubagentIDs)
	return &SupervisorThreadSessionSummary{
		Supervisor:              strings.TrimSpace(supervisor),
		ProyectoSlug:            projectSlug,
		SessionID:               strings.TrimSpace(sessionID),
		LeaderThreadID:          leaderThreadID,
		AllThreadIDs:            threadIDs,
		AllSubagentThreadIDs:    subagentIDs,
		ActiveSubagentThreadIDs: activeSubagentIDs,
		UpdatedAt:               updatedAt,
		Threads:                 items,
	}, nil
}

func appendIfMissing(items []string, value string) []string {
	for _, existing := range items {
		if existing == value {
			return items
		}
	}
	return append(items, value)
}

func scanSupervisorThread(scanner interface{ Scan(dest ...any) error }) (*SupervisorThread, error) {
	var item SupervisorThread
	if err := scanner.Scan(
		&item.ID, &item.Supervisor, &item.ProyectoSlug, &item.SessionID, &item.ThreadID, &item.Kind, &item.Mode,
		&item.Status, &item.Source, &item.LastTurnID, &item.TurnCount, &item.MetadataJSON, &item.FirstSeenAt, &item.LastSeenAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}
