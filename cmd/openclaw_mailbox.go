package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"orquesta/db"
)

type openClawMailboxAgg struct {
	count             int
	kinds             map[string]bool
	supervisorActions map[string]bool
	contexts          map[string]bool
	oldest            *time.Time
}

func buildOpenClawMailboxLiteFromItems(items []*db.RuntimeMailboxMessage, relevantAgents map[string]struct{}) []apiOpenClawMailboxLite {
	byAgent := map[string]*openClawMailboxAgg{}
	for _, item := range items {
		if item == nil {
			continue
		}
		agente := strings.TrimSpace(item.ToAgente)
		if agente == "" {
			continue
		}
		if len(relevantAgents) > 0 {
			if _, ok := relevantAgents[strings.ToLower(agente)]; !ok {
				continue
			}
		}
		entry := byAgent[agente]
		if entry == nil {
			entry = &openClawMailboxAgg{
				kinds:             map[string]bool{},
				supervisorActions: map[string]bool{},
				contexts:          map[string]bool{},
			}
			byAgent[agente] = entry
		}
		entry.count++
		if entry.oldest == nil || item.CreatedAt.Before(*entry.oldest) {
			ts := item.CreatedAt
			entry.oldest = &ts
		}
		if kind := strings.TrimSpace(item.Kind); kind != "" {
			entry.kinds[kind] = true
		}
		action, context := summarizeOpenClawMailboxPayload(item.PayloadJSON)
		if action != "" {
			entry.supervisorActions[action] = true
		}
		if context != "" {
			entry.contexts[context] = true
		}
	}

	out := make([]apiOpenClawMailboxLite, 0, len(byAgent))
	for agente, entry := range byAgent {
		kinds := sortedOpenClawMailboxValues(entry.kinds)
		supervisorActions := sortedOpenClawMailboxValues(entry.supervisorActions)
		contexts := sortedOpenClawMailboxValues(entry.contexts)
		oldestAge := 0
		if entry.oldest != nil && !entry.oldest.IsZero() {
			oldestAge = int(time.Since(*entry.oldest).Minutes())
			if oldestAge < 0 {
				oldestAge = 0
			}
		}
		out = append(out, apiOpenClawMailboxLite{
			Agente:               agente,
			Count:                entry.count,
			Kinds:                kinds,
			KindsCSV:             strings.Join(kinds, ", "),
			SupervisorActions:    supervisorActions,
			SupervisorActionsCSV: strings.Join(supervisorActions, ", "),
			Contexts:             contexts,
			ContextsCSV:          strings.Join(contexts, " | "),
			OldestCreatedAt:      entry.oldest,
			OldestAgeMin:         oldestAge,
		})
	}
	return out
}

func sortOpenClawMailboxLiteByCount(items []apiOpenClawMailboxLite) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Agente < items[j].Agente
	})
}

func sortOpenClawMailboxLiteByAgent(items []apiOpenClawMailboxLite) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Agente) < strings.ToLower(items[j].Agente)
	})
}

func sortedOpenClawMailboxValues(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func summarizeOpenClawMailboxPayload(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return "", ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload) == 0 {
		return "", ""
	}
	action := strings.TrimSpace(stringDesdeAny(payload["supervisor_action"]))
	return action, buildOpenClawMailboxContext(payload)
}

func buildOpenClawMailboxContext(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	parts := make([]string, 0, 4)
	if lane := strings.TrimSpace(stringDesdeAny(payload["carril"])); lane != "" {
		parts = append(parts, lane)
	}
	taskID := int64DesdeAny(payload["tarea_objetivo_id"])
	if taskID <= 0 {
		taskID = int64DesdeAny(payload["task_id"])
	}
	if taskID > 0 {
		parts = append(parts, fmt.Sprintf("tarea#%d", taskID))
	}
	if worktreeID := int64DesdeAny(payload["worktree_id"]); worktreeID > 0 {
		parts = append(parts, fmt.Sprintf("wt#%d", worktreeID))
	}
	if writeSet := openClawMailboxWriteSetSummary(payload); writeSet != "" {
		parts = append(parts, writeSet)
	}
	return strings.Join(parts, " · ")
}

func openClawMailboxWriteSetSummary(payload map[string]any) string {
	writeSet := stringSliceFromAny(payload["write_set"])
	if len(writeSet) == 0 {
		writeSet = stringSliceFromAny(payload["write_set_slice"])
	}
	if len(writeSet) == 0 {
		return ""
	}
	first := strings.TrimSpace(writeSet[0])
	if first == "" {
		return fmt.Sprintf("write_set=%d", len(writeSet))
	}
	if len(writeSet) == 1 {
		return first
	}
	return fmt.Sprintf("%s (+%d)", first, len(writeSet)-1)
}
