package main

import (
	"strings"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func serverWorkspaceTimelineFilterItemsV0(
	items []orquestaobservability.WorkspaceTimelineItemV0,
	query orquestaobservability.WorkspaceTimelineQueryV0,
	generatedAt string,
) []orquestaobservability.WorkspaceTimelineItemV0 {
	from, to, ok := serverWorkspaceTimelineWindowBoundsV0(query.TimeWindow, generatedAt)
	if !ok {
		return items
	}
	out := make([]orquestaobservability.WorkspaceTimelineItemV0, 0, len(items))
	for _, item := range items {
		occurred, err := time.Parse(time.RFC3339, strings.TrimSpace(item.OccurredAt))
		if err != nil {
			continue
		}
		if (from.IsZero() || !occurred.Before(from)) && (to.IsZero() || !occurred.After(to)) {
			out = append(out, item)
		}
	}
	return out
}

func serverWorkspaceTimelineWindowBoundsV0(
	window orquestaobservability.OperationalStatusTimeWindowV0,
	generatedAt string,
) (time.Time, time.Time, bool) {
	var from time.Time
	var to time.Time
	if strings.TrimSpace(window.From) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(window.From))
		if err == nil {
			from = parsed
		}
	}
	if strings.TrimSpace(window.To) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(window.To))
		if err == nil {
			to = parsed
		}
	}
	preset := strings.ToLower(strings.TrimSpace(window.Preset))
	if preset != "" && from.IsZero() && to.IsZero() {
		now, err := time.Parse(time.RFC3339, strings.TrimSpace(generatedAt))
		if err != nil {
			return time.Time{}, time.Time{}, false
		}
		to = now
		switch preset {
		case "last_30m", "last30m", "30m":
			from = now.Add(-30 * time.Minute)
		case "last_hour", "last_1h", "last60m", "1h":
			from = now.Add(-1 * time.Hour)
		default:
			return time.Time{}, time.Time{}, false
		}
	}
	return from, to, !from.IsZero() || !to.IsZero()
}

func serverWorkspaceTimelineCountersV0(
	items []orquestaobservability.WorkspaceTimelineItemV0,
) map[string]float64 {
	counters := map[string]float64{
		"events":        float64(len(items)),
		"tasks_visible": float64(len(items)),
	}
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "closed", "completed", "cerrada", "cerrado":
			counters["tasks_closed"]++
		}
	}
	return counters
}
