package orquestaopesbridge

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func fieldValueForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string, value string) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}

func fieldValuesForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string, values []string) bool {
	for _, field := range fields {
		if field.Name != name || len(field.Values) != len(values) {
			continue
		}
		ok := true
		for index := range values {
			if field.Values[index] != values[index] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func fieldValuesContainForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string, wants []string) bool {
	for _, field := range fields {
		if field.Name != name {
			continue
		}
		content := strings.Join(append([]string{field.Value}, field.Values...), "\n")
		for _, want := range wants {
			if !strings.Contains(content, want) {
				return false
			}
		}
		return true
	}
	return false
}

func fieldJSONForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string) bool {
	for _, field := range fields {
		if field.Name == name && json.Valid(field.ValueJSON) {
			return true
		}
	}
	return false
}

func containsStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
