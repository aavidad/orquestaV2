package orquestaoperatormcp

import "strings"

func normalizeOperatorSectionsV0(values []string) []string {
	if len(values) == 0 {
		return []string{"status"}
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		canonical := normalizeOperatorSectionV0(value)
		if canonical == "" || seen[canonical] {
			continue
		}
		seen[canonical] = true
		out = append(out, canonical)
	}
	return out
}

func normalizeOperatorSectionV0(value string) string {
	value = normalizeOperatorTokenV0(value)
	switch value {
	case "state", "estado", "status", "salud", "health":
		return "status"
	case "blockers", "bloqueos", "closure", "cierre":
		return "closure_blockers"
	case "agents", "agentes", "workers":
		return "agents"
	case "queue", "cola":
		return "queue"
	default:
		return value
	}
}

func normalizeOperatorKindsV0(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, value := range values {
		canonical := normalizeOperatorKindV0(value)
		if canonical != "" {
			out[canonical] = true
		}
	}
	return out
}

func normalizeOperatorKindV0(value string) string {
	value = normalizeOperatorTokenV0(value)
	switch value {
	case "question", "consulta", "director-query", "director-question":
		return "director_question"
	case "launch-agent", "launch-subagent", "launch-runtime-agent":
		return "launch_runtime_agent"
	case "stop-agent", "stop-runtime-agent":
		return "stop_runtime_agent"
	default:
		return value
	}
}

func normalizeOperatorOutboxItemsV0(items []OperatorMCPOutboxItemV0) []OperatorMCPOutboxItemV0 {
	out := make([]OperatorMCPOutboxItemV0, 0, len(items))
	for _, item := range items {
		item.MessageRef = strings.TrimSpace(item.MessageRef)
		item.Kind = normalizeOperatorKindV0(item.Kind)
		item.TargetRef = strings.TrimSpace(item.TargetRef)
		if item.MessageRef != "" {
			out = append(out, item)
		}
	}
	return out
}

func filterOperatorOutboxItemsV0(
	items []OperatorMCPOutboxItemV0,
	kinds map[string]bool,
) []OperatorMCPOutboxItemV0 {
	if len(kinds) == 0 {
		return append([]OperatorMCPOutboxItemV0(nil), items...)
	}
	out := make([]OperatorMCPOutboxItemV0, 0, len(items))
	for _, item := range items {
		if kinds[item.Kind] {
			out = append(out, item)
		}
	}
	return out
}

func limitOperatorOutboxItemsV0(
	items []OperatorMCPOutboxItemV0,
	limit int,
) []OperatorMCPOutboxItemV0 {
	if limit <= 0 || limit >= len(items) {
		return append([]OperatorMCPOutboxItemV0(nil), items...)
	}
	return append([]OperatorMCPOutboxItemV0(nil), items[:limit]...)
}

func normalizeOperatorTokenV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.Join(strings.Fields(value), "-")
	return value
}
