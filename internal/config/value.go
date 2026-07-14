package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseFileValue(definition registryKeyDefinition, raw any) (any, error) {
	var value any
	switch definition.Type {
	case valueTypeString:
		parsed, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		if err := validateAllowedValue(definition, parsed); err != nil {
			return nil, err
		}
		value = parsed
	case valueTypeInteger:
		parsed, err := parseInteger(raw)
		if err != nil {
			return nil, err
		}
		if err := validateIntegerBounds(definition, parsed); err != nil {
			return nil, err
		}
		value = parsed
	case valueTypePath:
		parsed, ok := raw.(string)
		if !ok || parsed == "" || strings.TrimSpace(parsed) != parsed || strings.ContainsRune(parsed, '\x00') {
			return nil, fmt.Errorf("expected non-empty path")
		}
		value = parsed
	case valueTypeCredentialRef:
		parsed, ok := raw.(string)
		if !ok || !validCredentialRef(parsed) {
			return nil, fmt.Errorf("expected canonical credential reference")
		}
		value = CredentialRef(parsed)
	case valueTypeDuration:
		parsed, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected duration string")
		}
		duration, err := time.ParseDuration(parsed)
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("expected positive duration")
		}
		value = duration
	case valueTypeStringList:
		parsed, err := parseStringList(raw)
		if err != nil {
			return nil, err
		}
		value = parsed
	default:
		return nil, fmt.Errorf("unsupported value type")
	}
	if err := validateDeclaredValue(definition, value); err != nil {
		return nil, err
	}
	return value, nil
}

func validateDeclaredValue(definition registryKeyDefinition, value any) error {
	for _, validator := range definition.ValidatorIDs {
		switch validator {
		case "trimmed_non_empty_string":
			text, ok := value.(string)
			if !ok || text == "" || strings.TrimSpace(text) != text || strings.ContainsRune(text, '\x00') {
				return fmt.Errorf("expected trimmed non-empty string")
			}
		case "trimmed_optional_string":
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) != text || strings.ContainsRune(text, '\x00') {
				return fmt.Errorf("expected trimmed optional string")
			}
		case "environment_name_list":
			names, ok := value.([]string)
			if !ok {
				return fmt.Errorf("expected environment name list")
			}
			for _, name := range names {
				if !validChildEnvironmentName(name) {
					return fmt.Errorf("invalid child environment name")
				}
			}
		case "opaque_ref":
			text, ok := value.(string)
			if !ok || text == "" || strings.TrimSpace(text) != text || strings.ContainsRune(text, '\x00') {
				return fmt.Errorf("expected opaque reference")
			}
		case "integer_bounds", "positive_duration", "non_empty_path", "unique_non_empty_string_list", "credential_ref", "allowed_values":
			// Enforced by the typed parser before semantic validators run.
		default:
			return fmt.Errorf("unsupported declared validator")
		}
	}
	return nil
}

func validCredentialRef(value string) bool {
	if value == "" {
		return true
	}
	const prefix = "credential:"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	identifier := strings.TrimPrefix(value, prefix)
	if len(identifier) == 0 || len(identifier) > 128 {
		return false
	}
	for index, character := range identifier {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') ||
			(index > 0 && (character == '.' || character == '_' || character == '-')) {
			continue
		}
		return false
	}
	return true
}

func parseEnvironmentValue(definition registryKeyDefinition, raw string) (any, error) {
	if definition.Type == valueTypeInteger {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected int64")
		}
		if err := validateIntegerBounds(definition, value); err != nil {
			return nil, err
		}
		return value, nil
	}
	if definition.Type != valueTypeStringList {
		return parseFileValue(definition, raw)
	}
	if raw == "" {
		return []string{}, nil
	}
	parts := strings.Split(raw, ",")
	values := make([]any, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.TrimSpace(part))
	}
	parsed, err := parseStringList(values)
	if err != nil {
		return nil, err
	}
	if err := validateDeclaredValue(definition, parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func parseInteger(raw any) (int64, error) {
	switch value := raw.(type) {
	case int64:
		return value, nil
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0, fmt.Errorf("expected int64")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("expected int64")
	}
}

func validateIntegerBounds(definition registryKeyDefinition, value int64) error {
	if definition.Minimum != nil && value < *definition.Minimum {
		return fmt.Errorf("integer is below minimum")
	}
	if definition.Maximum != nil && value > *definition.Maximum {
		return fmt.Errorf("integer is above maximum")
	}
	return nil
}

func parseStringList(raw any) ([]string, error) {
	var values []string
	switch list := raw.(type) {
	case []string:
		values = append(values, list...)
	case []any:
		values = make([]string, 0, len(list))
		for _, item := range list {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string list")
			}
			values = append(values, value)
		}
	default:
		return nil, fmt.Errorf("expected string list")
	}

	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return nil, fmt.Errorf("string list contains empty value")
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("string list contains duplicate value")
		}
		seen[value] = struct{}{}
	}
	return values, nil
}

func validateAllowedValue(definition registryKeyDefinition, raw any) error {
	if len(definition.AllowedValues) == 0 {
		return nil
	}
	value, ok := raw.(string)
	if !ok {
		return fmt.Errorf("allowed values require string")
	}
	for _, allowed := range definition.AllowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("value is not allowed")
}

func canonicalValue(value any) any {
	switch typed := value.(type) {
	case time.Duration:
		return typed.String()
	case CredentialRef:
		return string(typed)
	case []string:
		return append([]string(nil), typed...)
	default:
		return typed
	}
}
