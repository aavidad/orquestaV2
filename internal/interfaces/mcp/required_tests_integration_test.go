package mcpinterface

import (
	"bytes"
	"encoding/json"
	"testing"

	commandcore "orquesta/internal/commands"
)

func TestMCPPlanCarriesStructuredRequiredTestsIntoGoalView(t *testing.T) {
	definition := commandDefinitionByID(t, "orquesta.goals.create")
	schema, err := commandInputSchema(definition)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(schema, &decoded); err != nil {
		t.Fatal(err)
	}
	if !containsJSONKeys(decoded, "properties", "payload", "properties", "plan", "properties", "work_items", "items", "properties", "required_tests") {
		t.Fatalf("MCP schema dropped structured required_tests: %s", schema)
	}
}

func TestMCPPlanSchemasExposeOptionalEgressPolicyRefForCreateAndReplan(t *testing.T) {
	for _, commandID := range []string{"orquesta.goals.create", "orquesta.director.plan.propose"} {
		definition := commandDefinitionByID(t, commandID)
		schema, err := commandInputSchema(definition)
		if err != nil {
			t.Fatalf("%s: %v", commandID, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(schema, &decoded); err != nil {
			t.Fatalf("%s: %v", commandID, err)
		}
		if !containsJSONKeys(decoded,
			"properties", "payload", "properties", "plan", "properties", "work_items",
			"items", "properties", "egress_policy_ref",
		) {
			t.Fatalf("MCP schema dropped egress_policy_ref for %s: %s", commandID, schema)
		}
	}
}

func TestApplicationPlanDeepCopiesRequiredTestArguments(t *testing.T) {
	one := commandDefinitionByID(t, "orquesta.goals.create")
	want := append([]byte(nil), one.InputSchema...)
	one.InputSchema[0] ^= 0xff
	two := commandDefinitionByID(t, "orquesta.goals.create")
	if !bytes.Equal(two.InputSchema, want) {
		t.Fatal("canonical definitions retained caller-mutated schema storage")
	}
}

func commandDefinitionByID(t *testing.T, id string) commandcore.Definition {
	t.Helper()
	for _, definition := range commandcore.CanonicalDefinitions() {
		if definition.ID == id {
			return definition
		}
	}
	t.Fatalf("command %q missing", id)
	return commandcore.Definition{}
}

func containsJSONKeys(value any, keys ...string) bool {
	current := value
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = object[key]
		if !ok {
			return false
		}
	}
	return true
}
