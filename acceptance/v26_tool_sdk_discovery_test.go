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

func TestV26TLS05PublicDiscoveryMatchesInternalTLS04IndexExactly(t *testing.T) {
	input := json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
	output := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`)
	internalSpec := func(id string) tooling.CapabilitySpec {
		return tooling.CapabilitySpec{
			ID: id, Version: "1", InputSchema: input, OutputSchema: output,
			Permissions: []identity.Permission{identity.PermissionArtifactsRead},
			Cost:        tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{DiskBytes: 64}},
			Output:      tooling.OutputDelivery{MaxBytes: 64, InlineBytes: 16},
			Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation,
		}
	}
	publicSpec := func(id string) tools.Spec {
		return tools.Spec{
			ID: id, Version: "1", InputSchema: input, OutputSchema: output,
			Permissions: []tools.Permission{"artifacts.read"},
			Cost:        tools.CostContract{Mode: tools.CostMaximum, Maximum: tools.ResourceCost{DiskBytes: 64}},
			Output:      tools.OutputDelivery{MaxBytes: 64, InlineBytes: 16},
			Idempotency: tools.IdempotencyReadReexecute, Receipt: tools.ReceiptObservation,
		}
	}
	internalRegistry, err := tooling.NewRegistry(internalSpec("status.read"), internalSpec("workspace.read"))
	if err != nil {
		t.Fatal(err)
	}
	internalDiscovery, err := tooling.NewToolDiscovery(internalRegistry)
	if err != nil {
		t.Fatal(err)
	}
	publicCatalog, err := tools.NewCatalog(publicSpec("workspace.read"), publicSpec("status.read"))
	if err != nil {
		t.Fatal(err)
	}
	publicDiscovery, err := tools.NewDiscovery(publicCatalog)
	if err != nil {
		t.Fatal(err)
	}
	internalNamespaces := internalDiscovery.ListNamespaces()
	publicNamespaces := publicDiscovery.ListNamespaces()
	if internalDiscovery.Digest() != publicDiscovery.Digest() || len(internalNamespaces) != len(publicNamespaces) {
		t.Fatalf("internal=%+v/%q public=%+v/%q", internalNamespaces, internalDiscovery.Digest(), publicNamespaces, publicDiscovery.Digest())
	}
	for index, namespace := range internalNamespaces {
		want := tools.Namespace{Name: namespace.Name, ToolCount: namespace.ToolCount, Digest: namespace.Digest}
		if !reflect.DeepEqual(publicNamespaces[index], want) {
			t.Fatalf("namespace[%d]=%+v want=%+v", index, publicNamespaces[index], want)
		}
		internalTools, _ := internalDiscovery.ListNamespace(namespace.Name)
		publicTools, _ := publicDiscovery.ListNamespace(namespace.Name)
		if len(internalTools) != len(publicTools) || internalTools[0].Digest != publicTools[0].Digest {
			t.Fatalf("namespace=%s internal=%+v public=%+v", namespace.Name, internalTools, publicTools)
		}
	}
}
