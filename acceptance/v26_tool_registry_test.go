package acceptance_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS01ToolRegistryPreservesOneCanonicalCapabilitySpec(t *testing.T) {
	spec := tooling.CapabilitySpec{
		ID:      "workspace.commit",
		Version: "1",
		InputSchema: json.RawMessage(
			`{"additionalProperties":false,"required":["request_ref"],"properties":{"request_ref":{"minLength":1,"type":"string"}},"type":"object"}`,
		),
		OutputSchema: json.RawMessage(
			`{"type":"object","properties":{"receipt_ref":{"type":"string","minLength":1}},"required":["receipt_ref"],"additionalProperties":false}`,
		),
		Permissions: []identity.Permission{identity.PermissionEffectsApprove, identity.PermissionChangesIntegrate},
		Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
			MoneyMicros: 25_000, Currency: "EUR", ActiveTimeNS: 2_000_000_000, DiskBytes: 1 << 20,
		}},
		Output:      tooling.OutputDelivery{MaxBytes: 1 << 20, InlineBytes: 16 << 10},
		Idempotency: tooling.IdempotencyRequired,
		Receipt:     tooling.ReceiptApplication,
	}
	registry, err := tooling.NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}
	registration, found := registry.Lookup("workspace.commit", "1")
	if !found || registration.Spec.ID != spec.ID || registration.Spec.Version != spec.Version ||
		registration.Spec.Cost != spec.Cost || registration.Spec.Idempotency != spec.Idempotency ||
		registration.Spec.Receipt != spec.Receipt || !reflect.DeepEqual(registration.Spec.Permissions,
		[]identity.Permission{identity.PermissionChangesIntegrate, identity.PermissionEffectsApprove}) ||
		len(registry.List()) != 1 || len(registration.Digest) != len("sha256:")+64 ||
		len(registry.Digest()) != len("sha256:")+64 {
		t.Fatalf("canonical registration mismatch: %+v", registration)
	}

	unauthorized := spec
	unauthorized.Permissions = nil
	if denied, deniedErr := tooling.NewRegistry(unauthorized); denied != nil ||
		tooling.ErrorCode(deniedErr) != tooling.ErrorSpecInvalid {
		t.Fatalf("permission-free tool entered registry: registry=%v error=%v", denied, deniedErr)
	}
	openSchema := spec
	openSchema.InputSchema = json.RawMessage(`{"type":"object","properties":{},"additionalProperties":true}`)
	if admitted, admittedErr := tooling.NewRegistry(openSchema); admitted != nil ||
		tooling.ErrorCode(admittedErr) != tooling.ErrorSpecInvalid {
		t.Fatalf("open schema entered registry: registry=%v error=%v", admitted, admittedErr)
	}
}
