package jsonimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
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
	switch delimiter {
	case '{':
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
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			itemPath := path + "[" + strconv.Itoa(len(array)) + "]"
			value, err := decodeJSONValue(ctx, decoder, itemPath, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: err}
		}
		return array, nil
	default:
		return nil, &Error{Code: ErrorSourceInvalid, Path: path, Cause: errors.New("legacy_json_delimiter_invalid")}
	}
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
			if _, expectedObject := prefixes[path]; expectedObject {
				return &Error{Code: ErrorSourceInvalid, Path: path, Cause: errors.New("legacy_json_object_required")}
			}
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
	valid := true
	switch mapping.SourceKind {
	case SourceKindString:
		_, valid = value.(string)
	case SourceKindBool:
		_, valid = value.(bool)
	case SourceKindInteger:
		number, ok := value.(json.Number)
		if !ok {
			valid = false
			break
		}
		_, err := strconv.ParseInt(number.String(), 10, 64)
		valid = err == nil
	case SourceKindStringArray:
		array, ok := value.([]any)
		if !ok {
			valid = false
			break
		}
		for _, item := range array {
			if _, ok := item.(string); !ok {
				valid = false
				break
			}
		}
	default:
		return &Error{Code: ErrorMappingInvalid, Path: mapping.LegacyPath, Cause: errors.New("legacy_source_kind_unknown")}
	}
	if !valid {
		return &Error{Code: ErrorValueInvalid, Path: mapping.LegacyPath}
	}
	return nil
}
