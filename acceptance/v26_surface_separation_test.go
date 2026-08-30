package acceptance_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS02ToolsResourcesAndPromptsKeepDistinctControlContracts(t *testing.T) {
	tools, err := tooling.NewRegistry(tooling.CapabilitySpec{
		ID: "status.read", Version: "1",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"status":{"type":"string","minLength":1}},"required":["status"],"additionalProperties":false}`),
		Permissions:  []identity.Permission{identity.PermissionArtifactsRead},
		Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
			DiskBytes: 4096,
		}},
		Output:      tooling.OutputDelivery{MaxBytes: 4096, InlineBytes: 512},
		Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation,
	})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := tooling.NewSurfaceCatalog(
		tools,
		[]tooling.ResourceSpec{
			{
				ID: "status.read", Version: "1", URI: "orquesta://project/status/snapshot",
				MediaType: "application/json", Permissions: []identity.Permission{identity.PermissionArtifactsRead},
				MaxBytes:     4096,
				ItemSchema:   json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":1}},"required":["value"],"additionalProperties":false}`),
				Page:         tooling.ResourcePagePolicy{DefaultItems: 2, MaxItems: 8, MaxBytes: 2048},
				Subscription: tooling.ResourceSubscriptionSnapshotChanged,
			},
			{
				ID: "resource.only", Version: "1", URI: "orquesta://project/resource/only",
				MediaType: "application/json", Permissions: []identity.Permission{identity.PermissionArtifactsRead},
				MaxBytes:     1024,
				ItemSchema:   json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":1}},"required":["value"],"additionalProperties":false}`),
				Page:         tooling.ResourcePagePolicy{DefaultItems: 1, MaxItems: 2, MaxBytes: 512},
				Subscription: tooling.ResourceSubscriptionNone,
			},
		},
		[]tooling.PromptSpec{{
			ID: "status.read", Version: "1", TemplateKey: "prompt.status.explain",
			ArgumentsSchema: json.RawMessage(`{"type":"object","properties":{"locale":{"type":"string","minLength":2}},"required":["locale"],"additionalProperties":false}`),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	tool, toolFound := catalog.LookupTool("status.read", "1")
	resource, resourceFound := catalog.LookupResource("status.read", "1")
	prompt, promptFound := catalog.LookupPrompt("status.read", "1")
	if !toolFound || !resourceFound || !promptFound || tool.Spec.Receipt != tooling.ReceiptObservation ||
		resource.Spec.URI == "" || prompt.Spec.TemplateKey == "" || catalog.Digest() == "" {
		t.Fatalf("tool=%+v resource=%+v prompt=%+v digest=%q", tool, resource, prompt, catalog.Digest())
	}
	if _, found := reflect.TypeOf(catalog).MethodByName("Lookup"); found {
		t.Fatal("a generic surface lookup could disguise a resource or prompt as a tool")
	}
	if _, err := catalog.ValidatePromptInput("status.read", "1", json.RawMessage(`{"locale":"es"}`)); err != nil {
		t.Fatal(err)
	}
	if _, resourceFound := catalog.LookupResource("resource.only", "1"); !resourceFound {
		t.Fatal("resource-only identity missing from the resource catalog")
	}
	if _, toolFound := catalog.LookupTool("resource.only", "1"); toolFound {
		t.Fatal("resource-only identity resolved through the tool registry")
	}
	if _, promptFound := catalog.LookupPrompt("resource.only", "1"); promptFound {
		t.Fatal("resource-only identity resolved through the prompt catalog")
	}
	if _, resourceFound := catalog.LookupResource("missing", "1"); resourceFound {
		t.Fatal("unknown resource identity resolved")
	}
	if _, promptFound := catalog.LookupPrompt("status.read", "2"); promptFound {
		t.Fatal("prompt lookup selected another version implicitly")
	}

	resource.Spec.URI = "orquesta://mutated"
	prompt.Spec.TemplateKey = "prompt.mutated"
	tool.Spec.ID = "mutated"
	toolAgain, _ := catalog.LookupTool("status.read", "1")
	resourceAgain, _ := catalog.LookupResource("status.read", "1")
	promptAgain, _ := catalog.LookupPrompt("status.read", "1")
	if toolAgain.Spec.ID != "status.read" || resourceAgain.Spec.URI != "orquesta://project/status/snapshot" ||
		promptAgain.Spec.TemplateKey != "prompt.status.explain" {
		t.Fatalf("surface lookups alias caller memory: tool=%+v resource=%+v prompt=%+v", toolAgain, resourceAgain, promptAgain)
	}
}
