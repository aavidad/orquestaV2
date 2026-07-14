package config

import (
	"encoding/json"
	"testing"
)

func TestIntegerConfigLoadsStrictFileAndEnvironmentValues(t *testing.T) {
	path := writeTOML(t, `
[server]
max_request_bytes = 2048

[runtime]
max_output_bytes = 4096

[runtime.codex]
max_diagnostic_bytes = 8192
max_concurrent_executions = 23

[scheduler]
max_action_attempts = 7

[api]
max_list_limit = 50
locale = "en"

[config]
effective_max_existing_bytes = 32768
`)
	snapshot, err := loadWithEnvironment(LoadOptions{FilePath: path}, mapEnvironment(map[string]string{
		"ORQUESTA_RUNTIME_MAX_OUTPUT_BYTES": "16384",
		"ORQUESTA_API_MAX_LIST_LIMIT":       "75",
	}))
	if err != nil {
		t.Fatalf("load integer config: %v", err)
	}
	if snapshot.Server.MaxRequestBytes != 2048 ||
		snapshot.Runtime.MaxOutputBytes != 16384 ||
		snapshot.Runtime.Codex.MaxDiagnosticBytes != 8192 ||
		snapshot.Runtime.Codex.MaxConcurrentExecutions != 23 ||
		snapshot.Scheduler.MaxActionAttempts != 7 ||
		snapshot.API.MaxListLimit != 75 || snapshot.API.Locale != "en" ||
		snapshot.Effective.MaxExistingBytes != 32768 {
		t.Fatalf("unexpected typed snapshot: %+v %+v %+v %+v", snapshot.Server, snapshot.Runtime, snapshot.Scheduler, snapshot.API)
	}
	assertSource(t, snapshot, KeyServerMaxRequestBytes, SourceFile)
	assertSource(t, snapshot, KeyRuntimeMaxOutputBytes, SourceEnv)

	metadata, found := snapshot.Metadata(KeyServerMaxRequestBytes)
	if !found || metadata.Type != string(valueTypeInteger) || metadata.Minimum == nil || metadata.Maximum == nil {
		t.Fatalf("integer metadata incomplete: %+v/%v", metadata, found)
	}
	if *metadata.Minimum != 1 || *metadata.Maximum != 1073741824 {
		t.Fatalf("integer bounds = %d..%d", *metadata.Minimum, *metadata.Maximum)
	}
}

func TestIntegerConfigRejectsWrongTypeOverflowAndBounds(t *testing.T) {
	tests := []struct {
		name        string
		toml        string
		environment map[string]string
		key         Key
	}{
		{name: "float", toml: "[server]\nmax_request_bytes = 1.5", key: KeyServerMaxRequestBytes},
		{name: "string", toml: "[server]\nmax_request_bytes = \"12\"", key: KeyServerMaxRequestBytes},
		{name: "below minimum", toml: "[server]\nmax_request_bytes = 0", key: KeyServerMaxRequestBytes},
		{name: "above maximum", toml: "[server]\nmax_request_bytes = 1073741825", key: KeyServerMaxRequestBytes},
		{
			name:        "environment fractional",
			environment: map[string]string{"ORQUESTA_API_MAX_LIST_LIMIT": "1.5"},
			key:         KeyAPIMaxListLimit,
		},
		{
			name:        "environment whitespace",
			environment: map[string]string{"ORQUESTA_API_MAX_LIST_LIMIT": " 10"},
			key:         KeyAPIMaxListLimit,
		},
		{
			name:        "environment overflow",
			environment: map[string]string{"ORQUESTA_API_MAX_LIST_LIMIT": "9223372036854775808"},
			key:         KeyAPIMaxListLimit,
		},
		{name: "locale outside allowlist", toml: "[api]\nlocale = \"fr\"", key: KeyAPILocale},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := LoadOptions{}
			if test.toml != "" {
				options.FilePath = writeTOML(t, test.toml)
			}
			_, err := loadWithEnvironment(options, mapEnvironment(test.environment))
			assertConfigError(t, err, ErrorValueInvalid, test.key)
		})
	}
}

func TestIntegerRegistryBoundsAreOptionalAndCoherent(t *testing.T) {
	definition := registryKeyDefinition{
		Key:             "test.count",
		GoName:          "TestCount",
		Type:            valueTypeInteger,
		Default:         json.RawMessage("12"),
		Scope:           "test",
		RestartRequired: true,
		EnvAlias:        "ORQUESTA_TEST_COUNT",
	}
	if err := validateRegistryDefinition(definition); err != nil {
		t.Fatalf("optional bounds rejected: %v", err)
	}

	minimum, maximum := int64(10), int64(20)
	definition.Minimum = &minimum
	definition.Maximum = &maximum
	if err := validateRegistryDefinition(definition); err != nil {
		t.Fatalf("valid bounds rejected: %v", err)
	}

	definition.Default = json.RawMessage("21")
	if err := validateRegistryDefinition(definition); err == nil {
		t.Fatal("default above maximum was accepted")
	}
	definition.Default = json.RawMessage("12")
	minimum, maximum = 30, 20
	if err := validateRegistryDefinition(definition); err == nil {
		t.Fatal("inverted bounds were accepted")
	}
	definition.Type = valueTypeString
	definition.Default = json.RawMessage(`"12"`)
	if err := validateRegistryDefinition(definition); err == nil {
		t.Fatal("integer bounds on string were accepted")
	}
}
