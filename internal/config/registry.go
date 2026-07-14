package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type valueType string

const (
	valueTypeString        valueType = "string"
	valueTypeInteger       valueType = "integer"
	valueTypeDuration      valueType = "duration"
	valueTypePath          valueType = "path"
	valueTypeStringList    valueType = "string_list"
	valueTypeCredentialRef valueType = "credential_ref"
)

type registryFile struct {
	SchemaVersion int                     `json:"schema_version"`
	Revision      string                  `json:"revision"`
	Keys          []registryKeyDefinition `json:"keys"`
}

type registryKeyDefinition struct {
	Key             Key             `json:"key"`
	GoName          string          `json:"go_name"`
	Type            valueType       `json:"type"`
	Default         json.RawMessage `json:"default"`
	Sensitive       bool            `json:"sensitive"`
	Scope           string          `json:"scope"`
	RestartRequired bool            `json:"restart_required"`
	EnvAlias        string          `json:"env_alias"`
	AllowedValues   []string        `json:"allowed_values,omitempty"`
	Minimum         *int64          `json:"minimum,omitempty"`
	Maximum         *int64          `json:"maximum,omitempty"`
}

type registry struct {
	schemaVersion int
	revision      string
	keys          []registryKeyDefinition
	byKey         map[Key]registryKeyDefinition
	prefixes      map[string]struct{}
}

func loadRegistry() (registry, error) {
	decoder := json.NewDecoder(strings.NewReader(generatedRegistryJSON))
	decoder.DisallowUnknownFields()
	var source registryFile
	if err := decoder.Decode(&source); err != nil {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: err}
	}
	return validateAndBuildRegistry(source)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("trailing JSON value")
	}
	return err
}

func validateAndBuildRegistry(source registryFile) (registry, error) {
	fail := func(cause error) (registry, error) {
		return registry{}, &Error{Code: ErrorRegistryInvalid, Cause: cause}
	}
	if source.SchemaVersion < 1 {
		return fail(fmt.Errorf("schema_version must be positive"))
	}
	if strings.TrimSpace(source.Revision) == "" || source.Revision != generatedRegistryRevision {
		return fail(fmt.Errorf("registry revision mismatch"))
	}
	if len(source.Keys) == 0 {
		return fail(fmt.Errorf("registry has no keys"))
	}

	result := registry{
		schemaVersion: source.SchemaVersion,
		revision:      source.Revision,
		keys:          append([]registryKeyDefinition(nil), source.Keys...),
		byKey:         make(map[Key]registryKeyDefinition, len(source.Keys)),
		prefixes:      make(map[string]struct{}),
	}
	goNames := make(map[string]struct{}, len(source.Keys))
	envAliases := make(map[string]struct{}, len(source.Keys))
	for _, definition := range source.Keys {
		if err := validateRegistryDefinition(definition); err != nil {
			return fail(err)
		}
		if _, exists := result.byKey[definition.Key]; exists {
			return fail(fmt.Errorf("duplicate key"))
		}
		if _, exists := goNames[definition.GoName]; exists {
			return fail(fmt.Errorf("duplicate go name"))
		}
		if _, exists := envAliases[definition.EnvAlias]; exists {
			return fail(fmt.Errorf("duplicate env alias"))
		}
		result.byKey[definition.Key] = definition
		goNames[definition.GoName] = struct{}{}
		envAliases[definition.EnvAlias] = struct{}{}

		parts := strings.Split(string(definition.Key), ".")
		for index := 1; index < len(parts); index++ {
			result.prefixes[strings.Join(parts[:index], ".")] = struct{}{}
		}
	}
	return result, nil
}

func validateRegistryDefinition(definition registryKeyDefinition) error {
	if !validCanonicalKey(string(definition.Key)) {
		return fmt.Errorf("invalid canonical key")
	}
	if strings.TrimSpace(definition.GoName) == "" || strings.TrimSpace(definition.Scope) == "" {
		return fmt.Errorf("go_name and scope are required")
	}
	if strings.TrimSpace(definition.EnvAlias) == "" {
		return fmt.Errorf("env_alias is required")
	}
	switch definition.Type {
	case valueTypeString, valueTypeInteger, valueTypeDuration, valueTypePath, valueTypeStringList, valueTypeCredentialRef:
	default:
		return fmt.Errorf("unsupported value type")
	}
	if definition.Type != valueTypeInteger && (definition.Minimum != nil || definition.Maximum != nil) {
		return fmt.Errorf("bounds require integer type")
	}
	if definition.Minimum != nil && definition.Maximum != nil && *definition.Minimum > *definition.Maximum {
		return fmt.Errorf("minimum exceeds maximum")
	}
	if definition.Sensitive != (definition.Type == valueTypeCredentialRef) {
		return fmt.Errorf("only credential references may be sensitive")
	}
	value, err := parseDefaultValue(definition)
	if err != nil {
		return err
	}
	return validateAllowedValue(definition, value)
}

func validCanonicalKey(key string) bool {
	if key == "" || strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") || strings.Contains(key, "..") {
		return false
	}
	for _, character := range key {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func parseDefaultValue(definition registryKeyDefinition) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(definition.Default))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return parseFileValue(definition, raw)
}

func (r registry) definition(key Key) (registryKeyDefinition, bool) {
	definition, found := r.byKey[key]
	return definition, found
}

func (r registry) hasPrefix(prefix string) bool {
	_, found := r.prefixes[prefix]
	return found
}
