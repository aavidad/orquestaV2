package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS04ToolDiscoveryLoadsOnlyOneSmallNamespaceAtATime(t *testing.T) {
	toolSpec := func(id, version string) tooling.CapabilitySpec {
		return tooling.CapabilitySpec{
			ID: id, Version: version,
			InputSchema:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
			Permissions:  []identity.Permission{identity.PermissionArtifactsRead},
			Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
				ActiveTimeNS: int64(time.Second), DiskBytes: 1024,
			}},
			Output:      tooling.OutputDelivery{MaxBytes: 1024, InlineBytes: 256},
			Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation,
		}
	}
	registry, err := tooling.NewRegistry(
		toolSpec("status.read", "1"), toolSpec("status.health", "1"), toolSpec("workspace.read", "1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := tooling.NewToolDiscovery(registry)
	if err != nil {
		t.Fatal(err)
	}
	namespaces := discovery.ListNamespaces()
	encoded, err := json.Marshal(namespaces)
	if err != nil || len(namespaces) != 2 || namespaces[0].Name != "status" ||
		namespaces[0].ToolCount != 2 || bytes.Contains(encoded, []byte("input_schema")) ||
		bytes.Contains(encoded, []byte("permissions")) {
		t.Fatalf("namespaces=%+v encoded=%s error=%v", namespaces, encoded, err)
	}
	status, found := discovery.ListNamespace("status")
	if !found || len(status) != 2 || status[0].Spec.ID != "status.health" ||
		status[1].Spec.ID != "status.read" || len(status) > tooling.MaxToolsPerNamespace {
		t.Fatalf("status=%+v found=%v", status, found)
	}
	if workspace, found := discovery.ListNamespace("workspace"); !found || len(workspace) != 1 ||
		workspace[0].Spec.ID != "workspace.read" {
		t.Fatalf("workspace=%+v found=%v", workspace, found)
	}
	if _, found := discovery.ListNamespace("missing"); found {
		t.Fatal("an unknown namespace resolved")
	}

	status[0].Spec.ID = "mutated"
	again, found := discovery.ListNamespace("status")
	if !found || again[0].Spec.ID != "status.health" {
		t.Fatalf("namespace projection aliases caller memory: %+v", again)
	}

	revisedRegistry, err := tooling.NewRegistry(
		toolSpec("status.read", "1"), toolSpec("status.health", "2"), toolSpec("workspace.read", "1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	revised, err := tooling.NewToolDiscovery(revisedRegistry)
	if err != nil {
		t.Fatal(err)
	}
	revisedNamespaces := revised.ListNamespaces()
	if revisedNamespaces[0].Name != "status" || revisedNamespaces[0].Digest == namespaces[0].Digest {
		t.Fatalf("namespace digest does not bind exact tool revisions: before=%+v after=%+v", namespaces, revisedNamespaces)
	}
}

func TestV26TLS04ToolDiscoveryRejectsOversizedNamespaceAtomically(t *testing.T) {
	specs := make([]tooling.CapabilitySpec, 0, tooling.MaxToolsPerNamespace+1)
	for index := 0; index <= tooling.MaxToolsPerNamespace; index++ {
		specs = append(specs, tooling.CapabilitySpec{
			ID: fmt.Sprintf("bulk.tool%02d", index), Version: "1",
			InputSchema:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
			OutputSchema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
			Permissions:  []identity.Permission{identity.PermissionArtifactsRead},
			Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
				ActiveTimeNS: int64(time.Second), DiskBytes: 1024,
			}},
			Output:      tooling.OutputDelivery{MaxBytes: 1024, InlineBytes: 256},
			Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation,
		})
	}
	registry, err := tooling.NewRegistry(specs...)
	if err != nil {
		t.Fatal(err)
	}
	if discovery, err := tooling.NewToolDiscovery(registry); tooling.ErrorCode(err) != tooling.ErrorToolNamespaceTooLarge || discovery != nil {
		t.Fatalf("oversized namespace admitted: discovery=%+v error=%v", discovery, err)
	}
}
