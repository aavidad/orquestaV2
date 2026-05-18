package orquestadomainwork

import (
	"encoding/json"
	"strconv"
	"strings"
)

func firstDocumentPlanStringV0(raw map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		value := documentPlanStringV0(raw[key])
		if value != "" {
			return value
		}
	}
	return ""
}

func firstDocumentPlanStringsV0(raw map[string]json.RawMessage, keys ...string) []string {
	for _, key := range keys {
		values := documentPlanStringsV0(raw[key])
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func documentPlanStringV0(raw json.RawMessage) string {
	var value string
	if len(raw) > 0 && json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func documentPlanStringsV0(raw json.RawMessage) []string {
	var values []string
	if len(raw) > 0 && json.Unmarshal(raw, &values) == nil {
		return compactDomainWorkStringsV0(values)
	}
	if values := documentPlanStringObjectsV0(raw); len(values) > 0 {
		return values
	}
	value := documentPlanStringV0(raw)
	if value == "" {
		return nil
	}
	return []string{value}
}

func documentPlanStringObjectsV0(raw json.RawMessage) []string {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, firstDocumentPlanStringV0(
			value,
			"title",
			"rule",
			"objective",
			"description",
			"validation_method",
		))
	}
	return compactDomainWorkStringsV0(out)
}

func documentPlanIntV0(raw map[string]json.RawMessage, key string) int {
	value, ok := raw[key]
	if !ok || len(value) == 0 {
		return 0
	}
	var number int
	if json.Unmarshal(value, &number) == nil {
		return number
	}
	var text string
	if json.Unmarshal(value, &text) == nil {
		parsed, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func firstPositiveDocumentPlanIntV0(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func documentPlanEstimatedPagesV0(
	raw map[string]json.RawMessage,
	defaults DomainDocumentPlanPayloadDefaultsV0,
) (int, int) {
	min := firstPositiveDocumentPlanIntV0(
		documentPlanIntV0(raw, "estimated_pages_min"),
		documentPlanIntV0(raw, "target_pages_min"),
		defaults.EstimatedPagesMin,
	)
	max := firstPositiveDocumentPlanIntV0(
		documentPlanIntV0(raw, "estimated_pages_max"),
		documentPlanIntV0(raw, "target_pages_max"),
		defaults.EstimatedPagesMax,
	)
	var nested map[string]json.RawMessage
	if json.Unmarshal(raw["target_pages"], &nested) == nil {
		min = firstPositiveDocumentPlanIntV0(documentPlanIntV0(nested, "min"), min)
		max = firstPositiveDocumentPlanIntV0(documentPlanIntV0(nested, "max"), max)
	}
	return min, max
}

func prefixedDocumentPlanRefV0(prefix string, value string) string {
	value = compactDocumentPlanRefTextV0(value)
	if value == "" {
		return ""
	}
	prefix = compactDocumentPlanRefTextV0(prefix)
	if prefix == "" || strings.HasPrefix(value, prefix+"-") {
		return value
	}
	return prefix + "-" + value
}

func compactDocumentPlanRefTextV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		" ", "_", "/", "_", "\\", "_", "\t", "_", "\n", "_", "\r", "_",
	)
	value = replacer.Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}
