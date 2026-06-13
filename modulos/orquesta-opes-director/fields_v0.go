package orquestaopesdirector

import (
	"encoding/json"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func fieldStringV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) string {
	for _, name := range names {
		for _, field := range fields {
			if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
				continue
			}
			if strings.TrimSpace(field.Value) != "" {
				return strings.TrimSpace(field.Value)
			}
			if len(field.Values) > 0 {
				return strings.TrimSpace(field.Values[0])
			}
			var text string
			if len(field.ValueJSON) > 0 && json.Unmarshal(field.ValueJSON, &text) == nil {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func fieldIntV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) int {
	value := fieldStringV0(fields, names...)
	if value == "" {
		return 0
	}
	n, _ := strconv.Atoi(value)
	return n
}

func fieldStringsV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) []string {
	var out []string
	for _, name := range names {
		for _, field := range fields {
			if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
				continue
			}
			out = append(out, field.Value)
			out = append(out, field.Values...)
			if len(field.ValueJSON) > 0 {
				var texts []string
				if json.Unmarshal(field.ValueJSON, &texts) == nil {
					out = append(out, texts...)
					continue
				}
				var text string
				if json.Unmarshal(field.ValueJSON, &text) == nil {
					out = append(out, text)
				}
			}
		}
	}
	return compactStringsV0(out)
}

func fieldJSONV0[T any](
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	target *T,
) bool {
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
			continue
		}
		if len(field.ValueJSON) > 0 && json.Unmarshal(field.ValueJSON, target) == nil {
			return true
		}
		if strings.TrimSpace(field.Value) != "" && json.Unmarshal([]byte(field.Value), target) == nil {
			return true
		}
	}
	return false
}

func cloneFieldsV0(fields []orquestadomainwork.DomainWorkFieldV0) []orquestadomainwork.DomainWorkFieldV0 {
	normalized := orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:      "clone",
			CorrelationID:  "clone",
			IdempotencyKey: "clone",
			RequestedBy:    "clone",
			DomainRef:      "clone",
			WorkKind:       "clone",
			Objective:      "clone",
			InputFields:    fields,
		},
	)
	return normalized.InputFields
}
