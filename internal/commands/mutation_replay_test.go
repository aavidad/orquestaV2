package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestEveryMutatingHandlerReturnsStableImmutableReceiptOnReplay(t *testing.T) {
	dispatcher, api, _ := testDispatcher(t)
	payloads := canonicalPayloads()
	mutations := 0
	for index, definition := range dispatcher.Definitions() {
		if definition.Kind != KindCommand {
			continue
		}
		mutations++
		requestRef := fmt.Sprintf("request:immutable:%02d", index)
		first := invoke(t, dispatcher, definition.ID, requestRef, payloads[definition.ID], definition.ExecutionBound)
		second := invoke(t, dispatcher, definition.ID, requestRef, payloads[definition.ID], definition.ExecutionBound)
		if first.Failure != nil || second.Failure != nil || first.AuditRef != second.AuditRef || !bytes.Equal(first.Data, second.Data) {
			t.Errorf("%s first=%+v second=%+v", definition.ID, first, second)
			continue
		}
		if api.calls[definition.Handler] != 2 {
			t.Errorf("%s calls=%d", definition.ID, api.calls[definition.Handler])
		}
		assertNoTransientMutationFlags(t, definition.ID, first.Data)
	}
	if mutations != 22 {
		t.Fatalf("mutating command count=%d", mutations)
	}
}

func assertNoTransientMutationFlags(t *testing.T, commandID string, encoded json.RawMessage) {
	t.Helper()
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatalf("%s output=%s: %v", commandID, encoded, err)
	}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				switch key {
				case "created", "changed", "claimed", "revoked", "membership", "mailbox", "state", "status", "work_item_count", "attempt_count":
					t.Errorf("%s exposes transient or mutable snapshot field %q in %s", commandID, key, encoded)
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(value)
}

func TestMutationSchemasExcludeTransientFlagsAndMailboxClaimCarriesFence(t *testing.T) {
	for _, definition := range CanonicalDefinitions() {
		if definition.Kind != KindCommand {
			continue
		}
		assertNoTransientMutationFlags(t, definition.ID, definition.OutputSchema)
	}
	var schema map[string]any
	definition := compiledDefinitionsByID()["orquesta.mailbox.claim"]
	if err := json.Unmarshal(definition.OutputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	receipt := schema["properties"].(map[string]any)["receipt"].(map[string]any)
	required := receipt["required"].([]any)
	for _, want := range []string{"claim_token", "fence", "recipient_execution_ref"} {
		found := false
		for _, value := range required {
			found = found || value == want
		}
		if !found {
			t.Errorf("mailbox claim missing required %s", want)
		}
	}
}

func compiledDefinitionsByID() map[string]Definition {
	result := make(map[string]Definition, len(compiledDefinitions))
	for _, definition := range compiledDefinitions {
		result[definition.ID] = definition
	}
	return result
}
