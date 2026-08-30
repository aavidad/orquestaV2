package tooling

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestToolDiscoveryDefersSpecsBehindSmallNamespaces(t *testing.T) {
	registry, err := NewRegistry(
		validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication),
		validSpec("status.read", "2", IdempotencyReadReexecute, ReceiptObservation),
		validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation),
	)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := NewToolDiscovery(registry)
	if err != nil {
		t.Fatal(err)
	}
	namespaces := discovery.ListNamespaces()
	if len(namespaces) != 2 || namespaces[0].Name != "status" || namespaces[0].ToolCount != 2 ||
		namespaces[1].Name != "workspace" || namespaces[1].ToolCount != 1 ||
		!strings.HasPrefix(namespaces[0].Digest, "sha256:") || !strings.HasPrefix(discovery.Digest(), "sha256:") {
		t.Fatalf("namespaces=%+v digest=%q", namespaces, discovery.Digest())
	}
	if _, found := reflect.TypeOf(ToolNamespace{}).FieldByName("Specs"); found {
		t.Fatal("namespace index eagerly exposes tool specs")
	}
	status, found := discovery.ListNamespace("status")
	if !found || len(status) != 2 || status[0].Spec.ID != "status.read" ||
		status[0].Spec.Version != "1" || status[1].Spec.Version != "2" {
		t.Fatalf("status=%+v found=%v", status, found)
	}
	status[0].Spec.InputSchema[0] = '['
	again, _ := discovery.ListNamespace("status")
	if again[0].Spec.InputSchema[0] != '{' {
		t.Fatal("namespace listing aliases registry-owned specs")
	}
	if missing, found := discovery.ListNamespace("missing"); missing != nil || found {
		t.Fatalf("missing=%+v found=%v", missing, found)
	}
}

func TestToolDiscoveryDigestIsStableAndSensitiveToExactRegistry(t *testing.T) {
	firstRegistry, err := NewRegistry(
		validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication),
		validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondRegistry, err := NewRegistry(
		validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation),
		validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := NewToolDiscovery(firstRegistry)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewToolDiscovery(secondRegistry)
	if err != nil || first.Digest() != second.Digest() ||
		!reflect.DeepEqual(first.ListNamespaces(), second.ListNamespaces()) {
		t.Fatalf("first=%q second=%q error=%v", first.Digest(), second.Digest(), err)
	}
	changedRegistry, err := NewRegistry(
		validSpec("status.read", "2", IdempotencyReadReexecute, ReceiptObservation),
		validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication),
	)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := NewToolDiscovery(changedRegistry)
	if err != nil || changed.Digest() == first.Digest() {
		t.Fatalf("changed=%q first=%q error=%v", changed.Digest(), first.Digest(), err)
	}
	namespaces := first.ListNamespaces()
	namespaces[0].Name = "mutated"
	if first.ListNamespaces()[0].Name == "mutated" {
		t.Fatal("namespace descriptors alias discovery-owned state")
	}
}

func TestToolDiscoveryRejectsNilOrOversizedNamespaceAtomically(t *testing.T) {
	if discovery, err := NewToolDiscovery(nil); discovery != nil || ErrorCode(err) != ErrorToolDiscoveryInvalid {
		t.Fatalf("nil discovery=%v error=%v code=%q", discovery, err, ErrorCode(err))
	}
	specs := make([]CapabilitySpec, MaxToolsPerNamespace+1)
	for index := range specs {
		specs[index] = validSpec(
			fmt.Sprintf("dense.tool_%02d", index), "1", IdempotencyReadReexecute, ReceiptObservation,
		)
	}
	registry, err := NewRegistry(specs...)
	if err != nil {
		t.Fatal(err)
	}
	if discovery, err := NewToolDiscovery(registry); discovery != nil ||
		ErrorCode(err) != ErrorToolNamespaceTooLarge {
		t.Fatalf("oversized discovery=%v error=%v code=%q", discovery, err, ErrorCode(err))
	}
	emptyRegistry, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	empty, err := NewToolDiscovery(emptyRegistry)
	if err != nil || empty.ListNamespaces() == nil || len(empty.ListNamespaces()) != 0 || empty.Digest() == "" {
		t.Fatalf("empty=%v namespaces=%+v digest=%q error=%v", empty, empty.ListNamespaces(), empty.Digest(), err)
	}
	if nilDiscovery := (*ToolDiscovery)(nil); nilDiscovery.ListNamespaces() != nil ||
		nilDiscovery.Digest() != "" {
		t.Fatal("nil discovery should be inert")
	}
}
