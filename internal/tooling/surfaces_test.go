package tooling

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/identity"
)

func TestSurfaceCatalogKeepsToolsResourcesAndPromptsTypeSeparated(t *testing.T) {
	tools, err := NewRegistry(validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation))
	if err != nil {
		t.Fatal(err)
	}
	resource := validResourceSpec("status.read", "1")
	prompt := validPromptSpec("status.read", "1")
	catalog, err := NewSurfaceCatalog(tools, []ResourceSpec{resource}, []PromptSpec{prompt})
	if err != nil {
		t.Fatal(err)
	}
	tool, toolFound := catalog.LookupTool("status.read", "1")
	resourceRegistration, resourceFound := catalog.LookupResource("status.read", "1")
	promptRegistration, promptFound := catalog.LookupPrompt("status.read", "1")
	if !toolFound || !resourceFound || !promptFound || tool.Digest == "" ||
		resourceRegistration.Digest == "" || promptRegistration.Digest == "" {
		t.Fatalf("tool=%+v/%v resource=%+v/%v prompt=%+v/%v",
			tool, toolFound, resourceRegistration, resourceFound, promptRegistration, promptFound)
	}
	if tool.Digest == resourceRegistration.Digest || tool.Digest == promptRegistration.Digest ||
		resourceRegistration.Digest == promptRegistration.Digest {
		t.Fatal("surface kind is absent from a registration digest")
	}
	if _, genericLookup := reflect.TypeOf(catalog).MethodByName("Lookup"); genericLookup {
		t.Fatal("generic lookup could erase the surface control boundary")
	}
	assertStructOmitsFields(t, reflect.TypeOf(ResourceSpec{}),
		"InputSchema", "OutputSchema", "Idempotency", "Receipt", "Handler", "TemplateKey")
	assertStructOmitsFields(t, reflect.TypeOf(PromptSpec{}),
		"Permissions", "Cost", "Receipt", "Handler", "URI", "MediaType")
	assertStructOmitsFields(t, reflect.TypeOf(CapabilitySpec{}),
		"URI", "MediaType", "TemplateKey", "ArgumentsSchema", "Handler")
}

func TestSurfaceCatalogCanonicalizesOrdersDetachesSpecsAndHasStableDigest(t *testing.T) {
	tools, err := NewRegistry(
		validSpec("workspace.read", "2", IdempotencyReadReexecute, ReceiptObservation),
		validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation),
	)
	if err != nil {
		t.Fatal(err)
	}
	resources := []ResourceSpec{
		validResourceSpec("workspace.tree", "2"), validResourceSpec("status.snapshot", "1"),
	}
	prompts := []PromptSpec{
		validPromptSpec("workspace.explain", "2"), validPromptSpec("status.explain", "1"),
	}
	first, err := NewSurfaceCatalog(tools, resources, prompts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSurfaceCatalog(
		tools,
		[]ResourceSpec{resources[1], resources[0]},
		[]PromptSpec{prompts[1], prompts[0]},
	)
	if err != nil || first.Digest() != second.Digest() || !strings.HasPrefix(first.Digest(), "sha256:") {
		t.Fatalf("first=%q second=%q error=%v", first.Digest(), second.Digest(), err)
	}
	resourceList, promptList := first.ListResources(), first.ListPrompts()
	if len(first.ListTools()) != 2 || resourceList[0].Spec.ID != "status.snapshot" ||
		promptList[0].Spec.ID != "status.explain" {
		t.Fatalf("resources=%+v prompts=%+v", resourceList, promptList)
	}
	resources[0].Permissions[0] = identity.Permission("mutated")
	prompts[0].ArgumentsSchema[0] = '['
	resourceList[0].Spec.Permissions[0] = identity.Permission("mutated")
	promptList[0].Spec.ArgumentsSchema[0] = '['
	storedResource, _ := first.LookupResource("workspace.tree", "2")
	storedPrompt, _ := first.LookupPrompt("workspace.explain", "2")
	if identity.ValidatePermission(storedResource.Spec.Permissions[0]) != nil ||
		!json.Valid(storedPrompt.Spec.ArgumentsSchema) || first.Digest() != second.Digest() {
		t.Fatal("caller mutation changed the immutable surface catalog")
	}
	if nilCatalog := (*SurfaceCatalog)(nil); nilCatalog.Digest() != "" ||
		nilCatalog.ListTools() != nil || nilCatalog.ListResources() != nil || nilCatalog.ListPrompts() != nil {
		t.Fatal("nil surface catalog should be inert")
	}
}

func TestSurfaceCatalogRejectsInvalidDefinitionsAtomically(t *testing.T) {
	tools, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	validResource := validResourceSpec("status.snapshot", "1")
	validPrompt := validPromptSpec("status.explain", "1")
	resourceTests := map[string]func(*ResourceSpec){
		"id":           func(spec *ResourceSpec) { spec.ID = "status" },
		"version":      func(spec *ResourceSpec) { spec.Version = "01" },
		"relative uri": func(spec *ResourceSpec) { spec.URI = "/status/snapshot" },
		"uri query":    func(spec *ResourceSpec) { spec.URI += "?secret=value" },
		"media type":   func(spec *ResourceSpec) { spec.MediaType = "application/json; charset=utf-8" },
		"permission":   func(spec *ResourceSpec) { spec.Permissions = nil },
		"duplicate permission": func(spec *ResourceSpec) {
			spec.Permissions = append(spec.Permissions, spec.Permissions[0])
		},
		"max bytes":   func(spec *ResourceSpec) { spec.MaxBytes = 0 },
		"item schema": func(spec *ResourceSpec) { spec.ItemSchema = nil },
		"page default": func(spec *ResourceSpec) {
			spec.Page.DefaultItems = 0
		},
		"page maximum": func(spec *ResourceSpec) {
			spec.Page.MaxItems = spec.Page.DefaultItems - 1
		},
		"page bytes": func(spec *ResourceSpec) { spec.Page.MaxBytes = spec.MaxBytes + 1 },
		"subscription": func(spec *ResourceSpec) {
			spec.Subscription = ResourceSubscriptionPolicy("push_mutation")
		},
	}
	for name, mutate := range resourceTests {
		t.Run("resource "+name, func(t *testing.T) {
			spec := cloneResourceSpec(validResource)
			mutate(&spec)
			catalog, err := NewSurfaceCatalog(tools, []ResourceSpec{spec}, []PromptSpec{validPrompt})
			if catalog != nil || ErrorCode(err) != ErrorSurfaceSpecInvalid {
				t.Fatalf("catalog=%v error=%v code=%q", catalog, err, ErrorCode(err))
			}
		})
	}
	promptTests := map[string]func(*PromptSpec){
		"id":       func(spec *PromptSpec) { spec.ID = "status" },
		"version":  func(spec *PromptSpec) { spec.Version = "0" },
		"template": func(spec *PromptSpec) { spec.TemplateKey = "status.prompt" },
		"schema":   func(spec *PromptSpec) { spec.ArgumentsSchema = json.RawMessage(`{"type":"object"}`) },
	}
	for name, mutate := range promptTests {
		t.Run("prompt "+name, func(t *testing.T) {
			spec := clonePromptSpec(validPrompt)
			mutate(&spec)
			catalog, err := NewSurfaceCatalog(tools, []ResourceSpec{validResource}, []PromptSpec{spec})
			if catalog != nil || ErrorCode(err) != ErrorSurfaceSpecInvalid {
				t.Fatalf("catalog=%v error=%v code=%q", catalog, err, ErrorCode(err))
			}
		})
	}
	if catalog, err := NewSurfaceCatalog(nil, nil, nil); catalog != nil || ErrorCode(err) != ErrorSurfaceSpecInvalid {
		t.Fatalf("nil tools catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewSurfaceCatalog(tools, []ResourceSpec{validResource, validResource}, nil); catalog != nil ||
		ErrorCode(err) != ErrorSurfaceSpecDuplicate {
		t.Fatalf("duplicate resource catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewSurfaceCatalog(tools, nil, []PromptSpec{validPrompt, validPrompt}); catalog != nil ||
		ErrorCode(err) != ErrorSurfaceSpecDuplicate {
		t.Fatalf("duplicate prompt catalog=%v error=%v", catalog, err)
	}
}

func TestSurfaceCatalogValidatesOnlyExactPromptInputs(t *testing.T) {
	tools, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewSurfaceCatalog(tools, nil, []PromptSpec{validPromptSpec("status.explain", "1")})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := catalog.ValidatePromptInput("status.explain", "1", json.RawMessage(`{ "locale": "es" }`))
	if err != nil || string(canonical) != `{"locale":"es"}` {
		t.Fatalf("canonical=%s error=%v", canonical, err)
	}
	invalid := []json.RawMessage{
		nil, json.RawMessage(`null`), json.RawMessage(`{}`),
		json.RawMessage(`{"locale":"es","extra":true}`),
	}
	for _, payload := range invalid {
		if got, err := catalog.ValidatePromptInput("status.explain", "1", payload); got != nil ||
			ErrorCode(err) != ErrorPromptInputInvalid {
			t.Fatalf("payload=%s got=%s error=%v code=%q", payload, got, err, ErrorCode(err))
		}
	}
	if _, err := catalog.ValidatePromptInput("status.explain", "2", json.RawMessage(`{"locale":"es"}`)); ErrorCode(err) != ErrorSpecNotFound {
		t.Fatalf("wrong version error=%v code=%q", err, ErrorCode(err))
	}
}

func validResourceSpec(id, version string) ResourceSpec {
	return ResourceSpec{
		ID: id, Version: version, URI: "orquesta://project/status/snapshot",
		MediaType: "application/json", Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		MaxBytes:     4096,
		ItemSchema:   json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":1}},"required":["value"],"additionalProperties":false}`),
		Page:         ResourcePagePolicy{DefaultItems: 2, MaxItems: 4, MaxBytes: 2048},
		Subscription: ResourceSubscriptionSnapshotChanged,
	}
}

func validPromptSpec(id, version string) PromptSpec {
	return PromptSpec{
		ID: id, Version: version, TemplateKey: "prompt.status.explain",
		ArgumentsSchema: json.RawMessage(`{"type":"object","properties":{"locale":{"type":"string","enum":["en","es"]}},"required":["locale"],"additionalProperties":false}`),
	}
}

func cloneResourceSpec(source ResourceSpec) ResourceSpec {
	source.Permissions = append([]identity.Permission(nil), source.Permissions...)
	source.ItemSchema = append(json.RawMessage(nil), source.ItemSchema...)
	return source
}

func clonePromptSpec(source PromptSpec) PromptSpec {
	source.ArgumentsSchema = append(json.RawMessage(nil), source.ArgumentsSchema...)
	return source
}

func assertStructOmitsFields(t *testing.T, subject reflect.Type, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, found := subject.FieldByName(field); found {
			t.Fatalf("%s exposes forbidden %s", subject.Name(), field)
		}
	}
}
