package cmd

import (
	"fmt"
	"strings"
	"time"

	"orquesta/db"
)

type autonomyEventSummary struct {
	Kind          string    `json:"kind"`
	Source        string    `json:"source,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	TaskID        *int64    `json:"task_id,omitempty"`
	RuntimeID     *int64    `json:"runtime_id,omitempty"`
	HandleID      *int64    `json:"handle_id,omitempty"`
	Agent         string    `json:"agent,omitempty"`
	OriginAgent   string    `json:"origin_agent,omitempty"`
	TargetAgent   string    `json:"target_agent,omitempty"`
	Supervisor    string    `json:"supervisor,omitempty"`
	ControlAction string    `json:"control_action,omitempty"`
	Artifacts     []string  `json:"artifacts,omitempty"`
	ArtifactsMore int       `json:"artifacts_more,omitempty"`
}

func buildProjectAutonomyEventSummaries(projectID int64, since time.Time, limit int) ([]autonomyEventSummary, error) {
	if projectID <= 0 {
		return nil, nil
	}
	projectRef := projectID
	items, err := db.ListarAutonomyEvents(db.FiltroAutonomyEvents{
		ProjectID: &projectRef,
		Desde:     &since,
		Limite:    limit,
	})
	if err != nil {
		return nil, err
	}
	return projectAutonomyEventSummariesFromEvents(items), nil
}

func projectAutonomyEventSummariesFromEvents(items []*db.AutonomyEvent) []autonomyEventSummary {
	out := make([]autonomyEventSummary, 0, len(items))
	for _, item := range items {
		if summary, ok := autonomyEventSummaryFromEvent(item); ok {
			out = append(out, summary)
		}
	}
	return out
}

func buildAgentAutonomyEventSummaries(agent string, projectID *int64, since time.Time, limit int) ([]autonomyEventSummary, error) {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return nil, nil
	}
	f := db.FiltroAutonomyEvents{Desde: &since, Limite: max(limit*4, limit)}
	if projectID != nil && *projectID > 0 {
		f.ProjectID = projectID
	}
	items, err := db.ListarAutonomyEvents(f)
	if err != nil {
		return nil, err
	}
	out := make([]autonomyEventSummary, 0, min(limit, len(items)))
	for _, item := range items {
		if !autonomyEventMentionsAgent(item, agent) {
			continue
		}
		if summary, ok := autonomyEventSummaryFromEvent(item); ok {
			out = append(out, summary)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func autonomyEventSummaryFromEvent(item *db.AutonomyEvent) (autonomyEventSummary, bool) {
	if item == nil {
		return autonomyEventSummary{}, false
	}
	out := autonomyEventSummary{
		Kind:      strings.TrimSpace(item.Kind),
		Source:    strings.TrimSpace(item.Source),
		Reason:    strings.TrimSpace(item.Reason),
		CreatedAt: item.CreatedAt.UTC(),
		TaskID:    item.TaskID,
		RuntimeID: item.RuntimeID,
		HandleID:  item.HandleID,
	}
	out.Agent = autonomyEventDeltaString(item.StateDelta, "agente")
	out.OriginAgent = autonomyEventDeltaString(item.StateDelta, "agente_origen")
	out.TargetAgent = firstNonEmpty(
		autonomyEventDeltaString(item.StateDelta, "agente_destino"),
		autonomyEventDeltaString(item.StateDelta, "target_agente"),
	)
	out.Supervisor = autonomyEventDeltaString(item.StateDelta, "supervisor")
	out.ControlAction = autonomyEventDeltaString(item.StateDelta, "control_action")
	out.Artifacts, out.ArtifactsMore = compactAutonomyArtifactRefs(item.ArtifactsRef, 3)
	if out.Kind == "" {
		return autonomyEventSummary{}, false
	}
	return out, true
}

func compactAutonomyArtifactRefs(items []string, maxItems int) ([]string, int) {
	if len(items) == 0 {
		return nil, 0
	}
	if maxItems <= 0 {
		maxItems = 3
	}
	seen := make(map[string]struct{}, len(items))
	compact := make([]string, 0, min(len(items), maxItems))
	total := 0
	for _, raw := range items {
		ref := compactAutonomyArtifactRef(raw)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		total++
		if len(compact) < maxItems {
			compact = append(compact, ref)
		}
	}
	if len(compact) == 0 {
		return nil, 0
	}
	if total <= len(compact) {
		return compact, 0
	}
	return compact, total - len(compact)
}

func compactAutonomyArtifactRef(raw string) string {
	ref := strings.TrimSpace(raw)
	if ref == "" {
		return ""
	}
	if len(ref) <= 96 {
		return ref
	}
	if idx := strings.LastIndex(ref, "/"); idx >= 0 && idx+1 < len(ref) {
		tail := strings.TrimSpace(ref[idx+1:])
		if tail != "" && len(tail) <= 64 {
			return ".../" + tail
		}
	}
	if idx := strings.LastIndex(ref, ":"); idx >= 0 && idx+1 < len(ref) {
		head := strings.TrimSpace(ref[:idx])
		tail := strings.TrimSpace(ref[idx+1:])
		if head != "" && tail != "" && len(tail) <= 40 {
			return head + ":..." + tail
		}
	}
	if len(ref) <= 64 {
		return ref
	}
	return ref[:61] + "..."
}

func autonomyEventMentionsAgent(item *db.AutonomyEvent, agent string) bool {
	if item == nil || strings.TrimSpace(agent) == "" {
		return false
	}
	agent = strings.ToLower(strings.TrimSpace(agent))
	for _, value := range []string{
		autonomyEventDeltaString(item.StateDelta, "agente"),
		autonomyEventDeltaString(item.StateDelta, "agente_origen"),
		autonomyEventDeltaString(item.StateDelta, "agente_destino"),
		autonomyEventDeltaString(item.StateDelta, "target_agente"),
		autonomyEventDeltaString(item.StateDelta, "supervisor"),
	} {
		if strings.EqualFold(strings.TrimSpace(value), agent) {
			return true
		}
	}
	return false
}

func autonomyEventDeltaString(delta map[string]any, key string) string {
	if len(delta) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	value, ok := delta[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func min(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
