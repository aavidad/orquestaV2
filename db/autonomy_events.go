package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const autonomyEventActionPrefix = "autonomy_event_"

type AutonomyEvent struct {
	Kind         string         `json:"kind"`
	Actor        string         `json:"actor"`
	ProjectID    *int64         `json:"project_id,omitempty"`
	TaskID       *int64         `json:"task_id,omitempty"`
	RuntimeID    *int64         `json:"runtime_id,omitempty"`
	HandleID     *int64         `json:"handle_id,omitempty"`
	Source       string         `json:"source,omitempty"`
	Reason       string         `json:"reason,omitempty"`
	StateDelta   map[string]any `json:"state_delta,omitempty"`
	ArtifactsRef []string       `json:"artifacts_ref,omitempty"`
	CreatedAt    time.Time      `json:"created_at,omitempty"`
}

type FiltroAutonomyEvents struct {
	Actor     *string
	ProjectID *int64
	TaskID    *int64
	Kind      *string
	Desde     *time.Time
	Limite    int
}

func RegistrarAutonomyEvent(ev *AutonomyEvent) error {
	if ev == nil {
		return fmt.Errorf("autonomy event nil")
	}
	kind := strings.TrimSpace(ev.Kind)
	if kind == "" {
		return fmt.Errorf("kind obligatorio")
	}
	actor := strings.TrimSpace(ev.Actor)
	if actor == "" {
		actor = "orquesta"
	}
	payload := map[string]any{
		"kind": kind,
	}
	if strings.TrimSpace(ev.Source) != "" {
		payload["source"] = strings.TrimSpace(ev.Source)
	}
	if strings.TrimSpace(ev.Reason) != "" {
		payload["reason"] = strings.TrimSpace(ev.Reason)
	}
	if len(ev.StateDelta) > 0 {
		payload["state_delta"] = ev.StateDelta
	}
	if len(ev.ArtifactsRef) > 0 {
		payload["artifacts_ref"] = ev.ArtifactsRef
	}
	if ev.ProjectID != nil && *ev.ProjectID > 0 {
		payload["project_id"] = *ev.ProjectID
	}
	if ev.TaskID != nil && *ev.TaskID > 0 {
		payload["task_id"] = *ev.TaskID
	}
	if ev.RuntimeID != nil && *ev.RuntimeID > 0 {
		payload["runtime_id"] = *ev.RuntimeID
	}
	if ev.HandleID != nil && *ev.HandleID > 0 {
		payload["handle_id"] = *ev.HandleID
	}
	detailJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar autonomy event: %w", err)
	}
	Audit(actor, autonomyEventActionPrefix+kind, "autonomy_event", autonomyEventEntityID(ev), string(detailJSON))
	return nil
}

func ListarAutonomyEvents(f FiltroAutonomyEvents) ([]*AutonomyEvent, error) {
	limite := f.Limite
	if limite <= 0 {
		limite = 50
	}
	entidad := "autonomy_event"
	filtroAuditoria := FiltroAuditoria{
		Agente:  f.Actor,
		Entidad: &entidad,
		Desde:   f.Desde,
		Limite:  autonomyEventAuditScanLimit(limite, f),
	}
	if f.Kind != nil && strings.TrimSpace(*f.Kind) != "" {
		accion := autonomyEventActionPrefix + strings.TrimSpace(*f.Kind)
		filtroAuditoria.Accion = &accion
	}
	items, err := ListarAuditoria(filtroAuditoria)
	if err != nil {
		return nil, err
	}
	out := make([]*AutonomyEvent, 0, limite)
	for _, item := range items {
		if item == nil {
			continue
		}
		action := strings.TrimSpace(item.Accion)
		if !strings.HasPrefix(action, autonomyEventActionPrefix) {
			continue
		}
		ev, err := autonomyEventFromAudit(item)
		if err != nil || ev == nil {
			continue
		}
		if !autonomyEventMatchesFilter(ev, f) {
			continue
		}
		out = append(out, ev)
		if len(out) >= limite {
			break
		}
	}
	return out, nil
}

func autonomyEventAuditScanLimit(limite int, f FiltroAutonomyEvents) int {
	scanLimit := limite * 3
	if f.ProjectID != nil || f.TaskID != nil {
		scanLimit = limite * 20
	}
	if scanLimit < 50 {
		return 50
	}
	return scanLimit
}

func autonomyEventEntityID(ev *AutonomyEvent) int64 {
	for _, id := range []*int64{ev.TaskID, ev.RuntimeID, ev.HandleID, ev.ProjectID} {
		if id != nil && *id > 0 {
			return *id
		}
	}
	return 0
}

func autonomyEventFromAudit(item *LogAuditoria) (*AutonomyEvent, error) {
	if item == nil {
		return nil, nil
	}
	kind := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(item.Accion), autonomyEventActionPrefix))
	if kind == "" {
		return nil, nil
	}
	ev := &AutonomyEvent{
		Kind:      kind,
		Actor:     strings.TrimSpace(item.Agente),
		CreatedAt: item.CreatedAt.UTC(),
	}
	if strings.TrimSpace(item.Detalle) == "" {
		return ev, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(item.Detalle), &payload); err != nil {
		ev.Reason = strings.TrimSpace(item.Detalle)
		return ev, nil
	}
	if value, ok := payload["source"].(string); ok {
		ev.Source = strings.TrimSpace(value)
	}
	if value, ok := payload["reason"].(string); ok {
		ev.Reason = strings.TrimSpace(value)
	}
	if delta, ok := payload["state_delta"].(map[string]any); ok {
		ev.StateDelta = delta
	}
	if refs, ok := payload["artifacts_ref"].([]any); ok {
		ev.ArtifactsRef = make([]string, 0, len(refs))
		for _, ref := range refs {
			if s := strings.TrimSpace(fmt.Sprint(ref)); s != "" {
				ev.ArtifactsRef = append(ev.ArtifactsRef, s)
			}
		}
	}
	ev.ProjectID = autonomyEventInt64Ptr(payload["project_id"])
	ev.TaskID = autonomyEventInt64Ptr(payload["task_id"])
	ev.RuntimeID = autonomyEventInt64Ptr(payload["runtime_id"])
	ev.HandleID = autonomyEventInt64Ptr(payload["handle_id"])
	return ev, nil
}

func autonomyEventInt64Ptr(v any) *int64 {
	switch value := v.(type) {
	case float64:
		id := int64(value)
		if id > 0 {
			return &id
		}
	case int64:
		if value > 0 {
			id := value
			return &id
		}
	case int:
		if value > 0 {
			id := int64(value)
			return &id
		}
	}
	return nil
}

func autonomyEventMatchesFilter(ev *AutonomyEvent, f FiltroAutonomyEvents) bool {
	if ev == nil {
		return false
	}
	if f.Kind != nil && !strings.EqualFold(strings.TrimSpace(*f.Kind), strings.TrimSpace(ev.Kind)) {
		return false
	}
	if f.ProjectID != nil {
		if ev.ProjectID == nil || *ev.ProjectID != *f.ProjectID {
			return false
		}
	}
	if f.TaskID != nil {
		if ev.TaskID == nil || *ev.TaskID != *f.TaskID {
			return false
		}
	}
	if f.Desde != nil && ev.CreatedAt.Before(f.Desde.UTC()) {
		return false
	}
	return true
}
