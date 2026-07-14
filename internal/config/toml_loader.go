package config

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ParseExplicit validates and canonicalizes one human-editable TOML document.
// The returned map is detached and contains no secret material.
func ParseExplicit(content []byte) (map[Key]any, error) {
	registry, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	return parseExplicitWithRegistry(content, registry)
}

func parseExplicitWithRegistry(content []byte, registry registry) (map[Key]any, error) {
	if int64(len(content)) > registry.documentLimits.SourceMaxBytes {
		return nil, &Error{Code: ErrorFileInvalid, Cause: errors.New("config_source_too_large")}
	}
	if strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
		return nil, &Error{Code: ErrorEffectiveInputForbidden}
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return map[Key]any{}, nil
	}

	var document map[string]any
	decoder := toml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&document); err != nil {
		return nil, &Error{Code: ErrorFileInvalid, Cause: err}
	}
	flattened := make(map[Key]any)
	if err := flattenTOMLMap(registry, "", document, flattened); err != nil {
		return nil, err
	}
	result := make(map[Key]any, len(flattened))
	for key, raw := range flattened {
		definition, _ := registry.definition(key)
		value, err := parseFileValue(definition, raw)
		if err != nil {
			return nil, &Error{Code: ErrorValueInvalid, Key: key, Cause: err}
		}
		result[key] = cloneValue(value)
	}
	return result, nil
}

func flattenTOMLMap(registry registry, prefix string, document map[string]any, flattened map[Key]any) error {
	for name, raw := range document {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if nested, ok := raw.(map[string]any); ok {
			if !registry.hasPrefix(path) {
				return &Error{Code: ErrorUnknownKey, Key: Key(path)}
			}
			if err := flattenTOMLMap(registry, path, nested, flattened); err != nil {
				return err
			}
			continue
		}

		inputKey := Key(path)
		key, found := registry.canonicalKey(inputKey)
		if !found {
			return &Error{Code: ErrorUnknownKey, Key: inputKey}
		}
		if _, duplicate := flattened[key]; duplicate {
			return &Error{Code: ErrorFileInvalid, Key: key, Cause: errors.New("config_alias_ambiguous")}
		}
		flattened[key] = raw
	}
	return nil
}

// RenderExplicit serializes canonical explicit values in registry order. It
// emits only values provided by the caller; defaults stay in the registry.
func RenderExplicit(values map[Key]any) ([]byte, error) {
	registry, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	normalized := make(map[Key]any, len(values))
	for key, raw := range values {
		definition, found := registry.definition(key)
		if !found {
			return nil, &Error{Code: ErrorUnknownKey, Key: key}
		}
		value, err := parseFileValue(definition, canonicalValue(raw))
		if err != nil {
			return nil, &Error{Code: ErrorValueInvalid, Key: key, Cause: err}
		}
		normalized[key] = value
	}

	var output strings.Builder
	lastSection := ""
	for _, definition := range registry.keys {
		value, present := normalized[definition.Key]
		if !present {
			continue
		}
		key := string(definition.Key)
		dot := strings.LastIndexByte(key, '.')
		if dot <= 0 || dot == len(key)-1 {
			return nil, &Error{Code: ErrorRegistryInvalid, Key: definition.Key}
		}
		section, name := key[:dot], key[dot+1:]
		if section != lastSection {
			if output.Len() > 0 {
				output.WriteByte('\n')
			}
			fmt.Fprintf(&output, "[%s]\n", section)
			lastSection = section
		}
		fmt.Fprintf(&output, "%s = %s\n", name, renderTOMLValue(value))
	}
	content := []byte(output.String())
	if int64(len(content)) > registry.documentLimits.SourceMaxBytes {
		return nil, &Error{Code: ErrorFileInvalid, Cause: errors.New("config_source_too_large")}
	}
	return content, nil
}

func renderTOMLValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strconv.Quote(typed)
	case CredentialRef:
		return strconv.Quote(string(typed))
	case int64:
		return strconv.FormatInt(typed, 10)
	case []string:
		parts := make([]string, len(typed))
		for index, item := range typed {
			parts[index] = strconv.Quote(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return strconv.Quote(fmt.Sprint(canonicalValue(typed)))
	}
}
