package main

import "strings"

type mcpInputShapeFieldV0 struct {
	name     string
	required bool
	value    string
}

func mcpInputShapeFieldsV0(inputShape string) []mcpInputShapeFieldV0 {
	inside, ok := mcpTextBetweenFirstBracesV0(inputShape)
	if !ok {
		return nil
	}
	tokens := splitTopLevelMCPRealV0(inside, ',')
	fields := make([]mcpInputShapeFieldV0, 0, len(tokens))
	for _, token := range tokens {
		field, ok := mcpInputShapeFieldFromTokenV0(token)
		if ok {
			fields = append(fields, field)
		}
	}
	return fields
}

func mcpInputShapeFieldFromTokenV0(token string) (mcpInputShapeFieldV0, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return mcpInputShapeFieldV0{}, false
	}
	namePart := token
	value := ""
	if idx := strings.Index(token, ":"); idx >= 0 {
		namePart = token[:idx]
		value = strings.TrimSpace(token[idx+1:])
	}
	namePart = strings.TrimSpace(namePart)
	required := !strings.HasSuffix(namePart, "?")
	namePart = strings.TrimSuffix(namePart, "?")
	if namePart == "" || strings.ContainsAny(namePart, "{}()|") {
		return mcpInputShapeFieldV0{}, false
	}
	return mcpInputShapeFieldV0{name: namePart, required: required, value: value}, true
}

func mcpInputShapeFieldSchemaV0(field mcpInputShapeFieldV0) map[string]any {
	value := strings.TrimSpace(field.value)
	name := strings.TrimSpace(field.name)
	if strings.Contains(value, "|") && !strings.Contains(value, "{") && !strings.Contains(value, "(") {
		return map[string]any{"type": "string", "enum": splitEnumMCPRealV0(value)}
	}
	switch {
	case strings.HasSuffix(name, "_refs") || strings.HasPrefix(name, "include_") && strings.HasSuffix(name, "s"):
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	case strings.HasPrefix(name, "include_") ||
		strings.HasPrefix(name, "auto_") ||
		strings.HasPrefix(name, "allow_") ||
		strings.HasPrefix(name, "use_") ||
		name == "forced" ||
		name == "resident_mode" ||
		name == "worktree_isolated" ||
		name == "raise_operator_question":
		return map[string]any{"type": "boolean"}
	case strings.Contains(name, "limit") ||
		strings.Contains(name, "max_") ||
		strings.Contains(name, "score") ||
		strings.Contains(name, "ticks") ||
		strings.Contains(name, "executions") ||
		strings.Contains(name, "waits"):
		return map[string]any{"type": "integer"}
	case strings.Contains(value, "{") || strings.Contains(value, "V0") || strings.Contains(value, "Request"):
		return map[string]any{"type": "object", "description": value}
	default:
		return map[string]any{"type": "string", "description": value}
	}
}
