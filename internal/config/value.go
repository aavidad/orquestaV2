package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseFileValue(definition registryKeyDefinition, raw any) (any, error) {
	switch definition.Type {
	case valueTypeString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		if err := validateAllowedValue(definition, value); err != nil {
			return nil, err
		}
		return value, nil
	case valueTypeInteger:
		value, err := parseInteger(raw)
		if err != nil {
			return nil, err
		}
		if err := validateIntegerBounds(definition, value); err != nil {
			return nil, err
		}
		return value, nil
	case valueTypePath:
		value, ok := raw.(string)
		if !ok || value == "" {
			return nil, fmt.Errorf("expected non-empty path")
		}
		return value, nil
	case valueTypeCredentialRef:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected credential reference")
		}
		return CredentialRef(value), nil
	case valueTypeDuration:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected duration string")
		}
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("expected positive duration")
		}
		return duration, nil
	case valueTypeStringList:
		return parseStringList(raw)
	default:
		return nil, fmt.Errorf("unsupported value type")
	}
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
	return parseStringList(values)
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
