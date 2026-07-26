package config

import (
	"encoding/json"
	"testing"
)

func TestIntegerConfigLoadsStrictFileAndEnvironmentValues(t *testing.T) {
	snapshot := resolveTOML(t, `
[server]
max_request_bytes = 2048

[runtime]
max_output_bytes = 4096

[runtime.codex]
max_diagnostic_bytes = 8192
max_concurrent_executions = 23

[scheduler]
max_execution_attempts = 5

[intake]
max_question_rounds = 9

[api]
max_list_limit = 50
locale = "en"

[config]
effective_max_existing_bytes = 32768
`, map[string]string{
		"ORQUESTA_RUNTIME_MAX_OUTPUT_BYTES":   "16384",
		"ORQUESTA_INTAKE_MAX_QUESTION_ROUNDS": "12",
		"ORQUESTA_API_MAX_LIST_LIMIT":         "75",
	})
	if snapshot.ServerMaxRequestBytes() != 2048 || snapshot.RuntimeMaxOutputBytes() != 16384 ||
		snapshot.RuntimeCodexMaxDiagnosticBytes() != 8192 || snapshot.RuntimeCodexMaxConcurrentExecutions() != 23 ||
		snapshot.SchedulerMaxExecutionAttempts() != 5 || snapshot.IntakeMaxQuestionRounds() != 12 ||
		snapshot.APIMaxListLimit() != 75 || snapshot.APILocale() != "en" ||
		snapshot.ConfigEffectiveMaxExistingBytes() != 32768 {
		t.Fatal("unexpected typed integer snapshot")
	}
	assertSource(t, snapshot, KeyServerMaxRequestBytes, SourceFile)
	assertSource(t, snapshot, KeyRuntimeMaxOutputBytes, SourceEnv)
	assertSource(t, snapshot, KeyIntakeMaxQuestionRounds, SourceEnv)

	metadata, found := snapshot.Metadata(KeyServerMaxRequestBytes)
	if !found || metadata.Type != string(valueTypeInteger) || metadata.Minimum == nil || metadata.Maximum == nil {
		t.Fatalf("integer metadata incomplete: %+v/%v", metadata, found)
	}
	if *metadata.Minimum != 1 || *metadata.Maximum != 1073741824 {
		t.Fatalf("integer bounds = %d..%d", *metadata.Minimum, *metadata.Maximum)
	}
}

func TestIntakeRoundDefaultIsTypedValidatedAndEffectivelyProjected(t *testing.T) {
	snapshot := resolveTOML(t, "", nil)
	metadata, found := snapshot.Metadata(KeyIntakeMaxQuestionRounds)
	if !found || snapshot.IntakeMaxQuestionRounds() != 6 ||
		metadata.Source != SourceDefault || metadata.Type != string(valueTypeInteger) ||
		metadata.Minimum == nil || *metadata.Minimum != 1 ||
		metadata.Maximum == nil || *metadata.Maximum != 4294967295 {
		t.Fatalf("intake round default metadata=%+v found=%v value=%d",
			metadata, found, snapshot.IntakeMaxQuestionRounds())
	}
	for _, entry := range snapshot.effectiveDocument().Entries {
		if entry.Key == KeyIntakeMaxQuestionRounds {
			if entry.Value != int64(6) || entry.Source != SourceDefault ||
				entry.Minimum == nil || *entry.Minimum != 1 ||
				entry.Maximum == nil || *entry.Maximum != 4294967295 {
				t.Fatalf("effective intake rounds=%+v", entry)
			}
			return
		}
	}
	t.Fatal("effective intake rounds entry missing")
}

func TestSchedulerRegistryRetiresConflatedMaxActionAttempts(t *testing.T) {
	registry, err := loadRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	foundExecutionPolicy := false
	for _, definition := range registry.keys {
		if definition.Key == "scheduler.max_action_attempts" ||
			definition.EnvAlias == "ORQUESTA_SCHEDULER_MAX_ACTION_ATTEMPTS" {
			t.Fatalf("conflated delivery/execution policy survived: %+v", definition)
		}
		if definition.Key == KeySchedulerMaxExecutionAttempts {
			foundExecutionPolicy = definition.EnvAlias == "ORQUESTA_SCHEDULER_MAX_EXECUTION_ATTEMPTS"
		}
	}
	if !foundExecutionPolicy {
		t.Fatal("canonical execution-attempt policy missing or bound to wrong environment alias")
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
		{name: "intake rounds below minimum", toml: "[intake]\nmax_question_rounds = 0", key: KeyIntakeMaxQuestionRounds},
		{name: "intake rounds above uint32", toml: "[intake]\nmax_question_rounds = 4294967296", key: KeyIntakeMaxQuestionRounds},
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
			var content []byte
			if test.toml != "" {
				content = []byte(test.toml)
			}
			_, err := Resolve(ResolveOptions{TOML: content, Environment: test.environment})
			assertConfigError(t, err, ErrorValueInvalid, test.key)
		})
	}
}

func TestIntegerRegistryBoundsAreOptionalAndCoherent(t *testing.T) {
	definition := registryKeyDefinition{
		Key:             "test.count",
		GoName:          "TestCount",
		SemanticRef:     "orquesta.config.test.count",
		Type:            valueTypeInteger,
		Default:         json.RawMessage("12"),
		Scope:           "test",
		RestartRequired: true,
		EnvAlias:        "ORQUESTA_TEST_COUNT",
		ValidatorIDs:    []string{"integer_bounds"},
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
