package tools_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	tools "orquesta/sdk/tools"
)

func TestPublicDiscoveryProjectsCanonicalNamespacesAndDetachedSpecs(t *testing.T) {
	catalog, err := tools.NewCatalog(
		validSpec("workspace.commit", "1", tools.IdempotencyRequired, tools.ReceiptApplication),
		validSpec("status.read", "2", tools.IdempotencyReadReexecute, tools.ReceiptObservation),
		validSpec("status.read", "1", tools.IdempotencyReadReexecute, tools.ReceiptObservation),
	)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := tools.NewDiscovery(catalog)
	if err != nil {
		t.Fatal(err)
	}
	namespaces := discovery.ListNamespaces()
	if len(namespaces) != 2 || namespaces[0].Name != "status" || namespaces[0].ToolCount != 2 ||
		namespaces[1].Name != "workspace" || discovery.Digest() == "" {
		t.Fatalf("namespaces=%+v digest=%q", namespaces, discovery.Digest())
	}
	status, found := discovery.ListNamespace("status")
	if !found || len(status) != 2 || status[0].Spec.Version != "1" || status[1].Spec.Version != "2" {
		t.Fatalf("status=%+v found=%v", status, found)
	}
	status[0].Spec.InputSchema[0] = '['
	again, _ := discovery.ListNamespace("status")
	if again[0].Spec.InputSchema[0] != '{' {
		t.Fatal("public discovery aliases canonical specs")
	}
	namespaces[0].Name = "mutated"
	if discovery.ListNamespaces()[0].Name == "mutated" {
		t.Fatal("public discovery aliases namespace descriptors")
	}
	if missing, found := discovery.ListNamespace("missing"); missing != nil || found {
		t.Fatalf("missing=%+v found=%v", missing, found)
	}
}

func TestPublicDiscoveryMapsInvalidAndOversizedCatalogs(t *testing.T) {
	if discovery, err := tools.NewDiscovery(nil); discovery != nil ||
		!errors.Is(err, tools.ErrDiscoveryInvalid) || tools.ErrorCode(err) != "toolsdk.discovery_invalid" {
		t.Fatalf("nil discovery=%v error=%v code=%q", discovery, err, tools.ErrorCode(err))
	}
	specs := make([]tools.Spec, tools.MaxToolsPerNamespace+1)
	for index := range specs {
		specs[index] = validSpec(
			fmt.Sprintf("dense.tool_%02d", index), "1", tools.IdempotencyReadReexecute, tools.ReceiptObservation,
		)
	}
	catalog, err := tools.NewCatalog(specs...)
	if err != nil {
		t.Fatal(err)
	}
	if discovery, err := tools.NewDiscovery(catalog); discovery != nil ||
		!errors.Is(err, tools.ErrNamespaceTooLarge) || tools.ErrorCode(err) != "toolsdk.namespace_too_large" {
		t.Fatalf("oversized discovery=%v error=%v code=%q", discovery, err, tools.ErrorCode(err))
	}
	emptyCatalog, err := tools.NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	empty, err := tools.NewDiscovery(emptyCatalog)
	if err != nil || !reflect.DeepEqual(empty.ListNamespaces(), []tools.Namespace{}) || empty.Digest() == "" {
		t.Fatalf("empty=%v namespaces=%+v digest=%q error=%v", empty, empty.ListNamespaces(), empty.Digest(), err)
	}
	if nilDiscovery := (*tools.Discovery)(nil); nilDiscovery.ListNamespaces() != nil ||
		nilDiscovery.Digest() != "" {
		t.Fatal("nil public discovery should be inert")
	}
}
