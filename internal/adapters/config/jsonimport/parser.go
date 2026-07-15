package jsonimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/config"
)

const (
	legacySchemaVersion = "orquesta_config.v0"
	maximumJSONDepth    = 32
)

type legacyLeaf struct {
	path  string
	value any
}

func parseLegacyDocument(ctx context.Context, source []byte) ([]legacyLeaf, error) {
	if err := readyContext(ctx); err != nil {
		return nil, err
	}
	maximum := config.SourceMaxBytes()
	if maximum <= 0 {
		return nil, &Error{Code: ErrorMappingInvalid, Cause: errors.New("canonical_source_limit_unavailable")}
	}
	if int64(len(source)) > maximum {
		return nil, &Error{Code: ErrorSourceTooLarge}
	}
	if len(bytes.TrimSpace(source)) == 0 || !utf8.Valid(source) {
		return nil, &Error{Code: ErrorSourceInvalid}
	}

	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.UseNumber()
	decoded, err := decodeJSONValue(ctx, decoder, "", 0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, &Error{Code: ErrorSourceInvalid, Cause: errors.New("legacy_json_trailing_value")}
		}
		return nil, &Error{Code: ErrorSourceInvalid, Cause: err}
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, &Error{Code: ErrorSourceInvalid, Cause: errors.New("legacy_json_root_not_object")}
	}
	index, prefixes := canonicalMappingIndex()
	leaves := make([]legacyLeaf, 0, len(index))
	if err := flattenLegacyObject(root, "", index, prefixes, &leaves); err != nil {
		return nil, err
	}
	schemaFound := false
	for _, leaf := range leaves {
		if leaf.path != "schema_version" {
			continue
		}
		schemaFound = true
		version, ok := leaf.value.(string)
		if !ok {
			return nil, &Error{Code: ErrorValueInvalid, Path: leaf.path}
		}
		if version != legacySchemaVersion {
			return nil, &Error{Code: ErrorSchemaUnsupported, Path: leaf.path}
		}
	}
	if !schemaFound {
		return nil, &Error{Code: ErrorSchemaRequired, Path: "schema_version"}
	}
	return leaves, nil
}

func decodeJSONValue(ctx context.Context, decoder *json.Decoder, path string, depth int) (any, error) {
	if err := readyContext(ctx); err != nil {
		return nil, err
	}
	if depth > maximumJSONDepth {
		return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: errors.New("legacy_json_depth_exceeded")}
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: err}
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return token, nil
	}
	if delimiter != '{' {
		return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: errors.New("legacy_json_arrays_not_supported")}
	}
	object := make(map[string]any)
	for decoder.More() {
		if err := readyContext(ctx); err != nil {
			return nil, err
		}
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: err}
		}
		key, ok := keyToken.(string)
		if !ok || !validLegacySegment(key) {
			return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: errors.New("legacy_json_key_invalid")}
		}
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		if _, duplicate := object[key]; duplicate {
			return nil, &Error{Code: ErrorSourceInvalid, Path: childPath, Cause: errors.New("legacy_json_duplicate_key")}
		}
		value, err := decodeJSONValue(ctx, decoder, childPath, depth+1)
		if err != nil {
			return nil, err
		}
		object[key] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: err}
	}
	return object, nil
}

func validLegacySegment(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, ".") || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func flattenLegacyObject(
	object map[string]any,
	prefix string,
	index map[string]Mapping,
	prefixes map[string]struct{},
	leaves *[]legacyLeaf,
) error {
	if len(object) == 0 {
		return &Error{Code: ErrorSourceInvalid, Path: prefix, Cause: errors.New("legacy_json_empty_object")}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		value := object[key]
		if nested, ok := value.(map[string]any); ok {
			if _, known := prefixes[path]; !known {
				return &Error{Code: ErrorUnknownPath, Path: path}
			}
			if err := flattenLegacyObject(nested, path, index, prefixes, leaves); err != nil {
				return err
			}
			continue
		}
		mapping, known := index[path]
		if !known {
			return &Error{Code: ErrorUnknownPath, Path: path}
		}
		if err := validateLegacyValue(mapping, value); err != nil {
			return err
		}
		*leaves = append(*leaves, legacyLeaf{path: path, value: value})
	}
	return nil
}

func validateLegacyValue(mapping Mapping, value any) error {
	valid := false
	switch mapping.LegacyPath {
	case "schema_version", "server.addr", "server.state_dir", "control_plane.token":
		_, valid = value.(string)
	case "autoprogramming.checkpoint_only_high_consumption_tokens",
		"operator_director_mailbox.enabled", "runtime_models.enabled":
		_, valid = value.(bool)
	}
	if !valid {
		return &Error{Code: ErrorValueInvalid, Path: mapping.LegacyPath}
	}
	return nil
}
