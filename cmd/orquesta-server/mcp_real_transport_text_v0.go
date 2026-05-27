package main

import "strings"

func mcpTextBetweenFirstBracesV0(value string) (string, bool) {
	start := strings.Index(value, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	for idx := start; idx < len(value); idx++ {
		switch value[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return value[start+1 : idx], true
			}
		}
	}
	return "", false
}

func splitTopLevelMCPRealV0(value string, separator rune) []string {
	var out []string
	start := 0
	braceDepth := 0
	parenDepth := 0
	for idx, item := range value {
		switch item {
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		default:
			if item == separator && braceDepth == 0 && parenDepth == 0 {
				out = append(out, strings.TrimSpace(value[start:idx]))
				start = idx + len(string(item))
			}
		}
	}
	out = append(out, strings.TrimSpace(value[start:]))
	return out
}

func splitEnumMCPRealV0(value string) []string {
	parts := strings.Split(value, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func compactMCPRealStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
