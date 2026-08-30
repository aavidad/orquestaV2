package acceptance_test

import (
	"encoding/json"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

// TestAcceptanceV26ToolRegistryOutputMaxBytes ratchets the TLS-01 output
// boundary without claiming the rest of AC-V26-TOOLS-SKILLS-SDK complete.
func TestAcceptanceV26ToolRegistryOutputMaxBytes(t *testing.T) {
	const canonicalOutput = `{"accepted":true}`
	spec := v26BoundedToolSpec(int64(len(canonicalOutput)))
	registry, err := tooling.NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}

	got, err := registry.ValidateOutput(spec.ID, spec.Version, json.RawMessage(canonicalOutput))
	if err != nil || string(got) != canonicalOutput {
		t.Fatalf("exact output=%q err=%v", got, err)
	}

	// The extra whitespace disappears during JSON canonicalization, but it must
	// still count against the declared external-output budget.
	oversized := json.RawMessage(" \n" + canonicalOutput)
	if got, err = registry.ValidateOutput(spec.ID, spec.Version, oversized); got != nil ||
		tooling.ErrorCode(err) != tooling.ErrorPayloadInvalid {
		t.Fatalf("oversized output=%q err=%v code=%q", got, err, tooling.ErrorCode(err))
	}

	// Output.MaxBytes is not an input cap. Input keeps the registry's existing
	// schema validation and canonicalization semantics.
	input := json.RawMessage(" \n{\"request_ref\":\"request:one\"}")
	got, err = registry.ValidateInput(spec.ID, spec.Version, input)
	if err != nil || string(got) != `{"request_ref":"request:one"}` {
		t.Fatalf("canonical input=%q err=%v code=%q", got, err, tooling.ErrorCode(err))
	}
	if got, err = registry.ValidateOutput(spec.ID, spec.Version, json.RawMessage(`{"accepted":"yes"}`)); got != nil ||
		tooling.ErrorCode(err) != tooling.ErrorPayloadInvalid {
		t.Fatalf("schema-invalid output=%q err=%v code=%q", got, err, tooling.ErrorCode(err))
	}
}

func v26BoundedToolSpec(maxBytes int64) tooling.CapabilitySpec {
	return tooling.CapabilitySpec{
		ID:      "workspace.commit",
		Version: "1",
		InputSchema: json.RawMessage(
			`{"type":"object","properties":{"request_ref":{"type":"string","minLength":1}},"required":["request_ref"],"additionalProperties":false}`,
		),
		OutputSchema: json.RawMessage(
			`{"type":"object","properties":{"accepted":{"type":"boolean"}},"required":["accepted"],"additionalProperties":false}`,
		),
		Permissions: []identity.Permission{identity.PermissionChangesIntegrate},
		Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
			Tokens: 50, ActiveTimeNS: 1_000_000, DiskBytes: 4096,
		}},
		Output:      tooling.OutputDelivery{MaxBytes: maxBytes, InlineBytes: maxBytes},
		Idempotency: tooling.IdempotencyRequired,
		Receipt:     tooling.ReceiptApplication,
	}
}
