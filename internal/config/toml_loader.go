package config

import (
	"bytes"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func loadTOMLFile(path string, registry registry) (map[Key]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Code: ErrorFileRead, Cause: err}
	}
	if strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
		return nil, &Error{Code: ErrorEffectiveInputForbidden}
	}

	var document map[string]any
	if err := toml.NewDecoder(bytes.NewReader(content)).Decode(&document); err != nil {
		return nil, &Error{Code: ErrorFileInvalid, Cause: err}
	}
	flattened := make(map[Key]any)
	if err := flattenTOMLMap(registry, "", document, flattened); err != nil {
		return nil, err
	}
	return flattened, nil
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

		key := Key(path)
		if _, found := registry.definition(key); !found {
			return &Error{Code: ErrorUnknownKey, Key: key}
		}
		if _, duplicate := flattened[key]; duplicate {
			return &Error{Code: ErrorFileInvalid, Key: key}
		}
		flattened[key] = raw
	}
	return nil
}
