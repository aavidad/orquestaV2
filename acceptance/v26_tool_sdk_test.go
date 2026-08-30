package acceptance_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
	tools "orquesta/sdk/tools"
)

func TestV26TLS05PublicSDKProjectsTheCanonicalTLS01Registry(t *testing.T) {
	publicSpec := tools.Spec{
		ID: "workspace.commit", Version: "1",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"request_ref":{"type":"string","minLength":1}},"required":["request_ref"],"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"receipt_ref":{"type":"string","minLength":1}},"required":["receipt_ref"],"additionalProperties":false}`),
		Permissions:  []tools.Permission{"effects.approve", "changes.integrate"},
		Cost: tools.CostContract{Mode: tools.CostMaximum, Maximum: tools.ResourceCost{
			MoneyMicros: 25_000, Currency: "EUR", ActiveTimeNS: 2_000_000_000, DiskBytes: 1 << 20,
		}},
		Output:      tools.OutputDelivery{MaxBytes: 1 << 20, InlineBytes: 16 << 10},
		Idempotency: tools.IdempotencyRequired,
		Receipt:     tools.ReceiptApplication,
	}
	publicCatalog, err := tools.NewCatalog(publicSpec)
	if err != nil {
		t.Fatal(err)
	}
	internalRegistry, err := tooling.NewRegistry(tooling.CapabilitySpec{
		ID: publicSpec.ID, Version: publicSpec.Version,
		InputSchema: publicSpec.InputSchema, OutputSchema: publicSpec.OutputSchema,
		Permissions: []identity.Permission{identity.PermissionEffectsApprove, identity.PermissionChangesIntegrate},
		Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
			MoneyMicros: 25_000, Currency: "EUR", ActiveTimeNS: 2_000_000_000, DiskBytes: 1 << 20,
		}},
		Output:      tooling.OutputDelivery{MaxBytes: 1 << 20, InlineBytes: 16 << 10},
		Idempotency: tooling.IdempotencyRequired,
		Receipt:     tooling.ReceiptApplication,
	})
	if err != nil {
		t.Fatal(err)
	}
	publicRegistration, publicFound := publicCatalog.Lookup(publicSpec.ID, publicSpec.Version)
	internalRegistration, internalFound := internalRegistry.Lookup(publicSpec.ID, publicSpec.Version)
	if !publicFound || !internalFound || publicCatalog.Digest() != internalRegistry.Digest() ||
		publicRegistration.Digest != internalRegistration.Digest ||
		!reflect.DeepEqual(publicRegistration.Spec.Permissions,
			[]tools.Permission{"changes.integrate", "effects.approve"}) {
		t.Fatalf("public=%+v internal=%+v", publicRegistration, internalRegistration)
	}

	unauthorized := publicSpec
	unauthorized.Permissions = nil
	if catalog, denied := tools.NewCatalog(unauthorized); catalog != nil || denied == nil {
		t.Fatalf("permission-free public spec admitted: catalog=%v error=%v", catalog, denied)
	}
}
