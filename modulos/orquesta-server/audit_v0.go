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

const (
	auditPayloadSummaryMaxRefsV0  = 16
	auditPayloadSummaryMaxBytesV0 = 8 << 10
)

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
	return appendServerDurableFileV0(sink.path, data, "orquesta_server_audit")
}

func normalizeAuditEventV0(event AuditEventV0) AuditEventV0 {
	event.SchemaVersion = AuditSchemaVersionV0
	event.Event = compactAuditEventNameV0(event.Event)
	event.OccurredAt = strings.TrimSpace(event.OccurredAt)
	event.Status = compactServerOperationalTokenV0(event.Status)
	event.Error = projectServerOperationalMessageV0("audit", event.Error)
	event.Payload = normalizeAuditPayloadV0(event.Payload)
	if len(event.Payload) == 0 {
		event.Payload = nil
	}
	return event
}

func normalizeAuditPayloadV0(payload map[string]interface{}) map[string]interface{} {
	if len(payload) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		key = compactAuditPayloadKeyV0(key)
		if key == "" {
			continue
		}
		if auditPayloadNeedsSummaryV0(key) {
			out[key+"_summary"] = auditPayloadValueSummaryV0(key, value)
			continue
		}
		out[key] = normalizeAuditPayloadValueV0(key, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func auditPayloadNeedsSummaryV0(key string) bool {
	switch key {
	case "request", "requests", "base_request", "result", "plan", "selected", "command":
		return true
	default:
		return false
	}
}

func auditPayloadValueSummaryV0(key string, value interface{}) map[string]interface{} {
	summary := map[string]interface{}{
		"kind":     key,
		"redacted": true,
	}
	data, err := json.Marshal(value)
	if err != nil {
		summary["status"] = "summary_unavailable"
		return summary
	}
	summary["json_bytes"] = len(data)
	if len(data) > auditPayloadSummaryMaxBytesV0 {
		summary["status"] = "payload_too_large"
		return summary
	}
	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		summary["status"] = "summary_unavailable"
		return summary
	}
	refs := auditPayloadCollectRefsV0(decoded, nil)
	if len(refs) > 0 {
		summary["refs"] = refs
		summary["refs_count"] = len(refs)
	}
	if count := auditPayloadItemCountV0(decoded); count > 0 {
		summary["items_count"] = count
	}
	summary["status"] = "summary_only"
	return summary
}

func normalizeAuditPayloadValueV0(key string, value interface{}) interface{} {
	switch typed := value.(type) {
	case string:
		return normalizeAuditPayloadStringV0(key, typed)
	case []string:
		return compactServerOperationalRefsV0(typed)
	case map[string]interface{}:
		return normalizeAuditPayloadV0(typed)
	default:
		return value
	}
}

func normalizeAuditPayloadStringV0(key string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if key == "path" || key == "endpoint" {
		return value
	}
	if auditPayloadKeyLooksSensitiveV0(key) {
		return serverOperationalMessageRedactedV0
	}
	return projectServerOperationalMessageV0("audit", value)
}

func compactAuditPayloadKeyV0(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			builder.WriteRune(r)
		} else if builder.Len() > 0 {
			break
		}
		if builder.Len() >= 96 {
			break
		}
	}
	return builder.String()
}

func auditPayloadKeyLooksSensitiveV0(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range []string{"token", "secret", "password", "credential", "prompt", "transcript", "raw"} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func auditPayloadCollectRefsV0(value interface{}, refs []string) []string {
	if len(refs) >= auditPayloadSummaryMaxRefsV0 {
		return compactServerOperationalRefsV0(refs)
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if strings.HasSuffix(key, "_ref") || key == "ref" {
				if ref, ok := item.(string); ok {
					refs = append(refs, ref)
				}
				continue
			}
			if strings.HasSuffix(key, "_refs") {
				refs = auditPayloadCollectRefsV0(item, refs)
				continue
			}
			refs = auditPayloadCollectRefsV0(item, refs)
		}
	case []interface{}:
		for _, item := range typed {
			refs = auditPayloadCollectRefsV0(item, refs)
		}
	case string:
		if ref := compactServerOperationalRefV0(typed); strings.Contains(ref, "-ref-") {
			refs = append(refs, ref)
		}
	}
	return compactServerOperationalRefsV0(refs)
}

func auditPayloadItemCountV0(value interface{}) int {
	switch typed := value.(type) {
	case []interface{}:
		return len(typed)
	case map[string]interface{}:
		return len(typed)
	default:
		return 0
	}
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
	if err := runtime.auditSink.AppendAuditEventV0(ctx, AuditEventV0{
		Event:      event,
		OccurredAt: formatTimeV0(runtime.clock.Now()),
		Status:     status,
		Error:      errText,
		Payload:    payload,
	}); err != nil {
		if runtime.tracker != nil {
			now := runtime.clock.Now()
			severity := auditWriteFailureSeverityV0(event)
			runtime.persistStateTransitionV0(
				context.Background(),
				runtime.tracker.MarkAuditWriteFailedV0(event, severity, now),
				"audit_write_failed",
			)
		}
		return
	}
	if runtime.tracker != nil {
		runtime.tracker.MarkAuditWriteConfirmedV0(runtime.clock.Now())
	}
}
