package orquestaserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const AuditSchemaVersionV0 = "orquesta_server_audit.v0"

type AuditEventV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	Event         string                 `json:"event"`
	OccurredAt    string                 `json:"occurred_at,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

type FileAuditSinkV0 struct {
	mu   sync.Mutex
	path string
}

func NewFileAuditSinkV0(path string) (*FileAuditSinkV0, error) {
	path = filepath.Clean(path)
	if path == "." || !filepath.IsAbs(path) || filepath.Ext(path) != ".jsonl" {
		return nil, fmt.Errorf("orquesta_server_audit: path_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("orquesta_server_audit: dir_unavailable")
	}
	return &FileAuditSinkV0{path: path}, nil
}

func (sink *FileAuditSinkV0) AppendAuditEventV0(
	ctx context.Context,
	event AuditEventV0,
) error {
	if sink == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	event = normalizeAuditEventV0(event)
	if event.Event == "" {
		return fmt.Errorf("orquesta_server_audit: event_required")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("orquesta_server_audit: json_failed")
	}
	data = append(data, '\n')
	sink.mu.Lock()
	defer sink.mu.Unlock()
	file, err := os.OpenFile(sink.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("orquesta_server_audit: open_failed")
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("orquesta_server_audit: write_failed")
	}
	return nil
}

func normalizeAuditEventV0(event AuditEventV0) AuditEventV0 {
	event.SchemaVersion = AuditSchemaVersionV0
	event.Event = strings.TrimSpace(event.Event)
	event.OccurredAt = strings.TrimSpace(event.OccurredAt)
	event.Status = strings.TrimSpace(event.Status)
	event.Error = strings.TrimSpace(event.Error)
	if len(event.Payload) == 0 {
		event.Payload = nil
	}
	return event
}

func (runtime *RuntimeV0) auditEventV0(
	ctx context.Context,
	event string,
	status string,
	errText string,
	payload map[string]interface{},
) {
	if runtime == nil || runtime.auditSink == nil {
		return
	}
	_ = runtime.auditSink.AppendAuditEventV0(ctx, AuditEventV0{
		Event:      event,
		OccurredAt: formatTimeV0(runtime.clock.Now()),
		Status:     status,
		Error:      errText,
		Payload:    payload,
	})
}
