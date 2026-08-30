package tooling

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func TestRegistryCanonicalizesOrdersAndDetachesCapabilitySpecs(t *testing.T) {
	read := validSpec("artifacts.read", "2", IdempotencyReadReexecute, ReceiptObservation)
	write := validSpec("workspace.commit", "10", IdempotencyRequired, ReceiptApplication)
	older := validSpec("workspace.commit", "2", IdempotencyRequired, ReceiptApplication)
	read.Permissions = []identity.Permission{identity.PermissionGoalsGet, identity.PermissionArtifactsRead}

	registry, err := NewRegistry(write, read, older)
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := NewRegistry(older, write, read)
	if err != nil {
		t.Fatal(err)
	}
	if registry.Digest() != reordered.Digest() || !strings.HasPrefix(registry.Digest(), "sha256:") {
		t.Fatalf("registry digests differ: %q / %q", registry.Digest(), reordered.Digest())
	}
	formatted := cloneTestSpec(read)
	formatted.InputSchema = json.RawMessage("\n  " + string(read.InputSchema) + "\n")
	formattedRegistry, err := NewRegistry(formatted)
	if err != nil {
		t.Fatal(err)
	}
	formattedRegistration, _ := formattedRegistry.Lookup(formatted.ID, formatted.Version)
	listed := registry.List()
	wantOrder := []string{"artifacts.read@2", "workspace.commit@2", "workspace.commit@10"}
	gotOrder := make([]string, len(listed))
	for index, registration := range listed {
		gotOrder[index] = registration.Spec.ID + "@" + registration.Spec.Version
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("registry order = %v", gotOrder)
	}
	got, ok := registry.Lookup("artifacts.read", "2")
	if !ok || !reflect.DeepEqual(got.Spec.Permissions, []identity.Permission{
		identity.PermissionArtifactsRead, identity.PermissionGoalsGet,
	}) || !strings.HasPrefix(got.Digest, "sha256:") || got.Digest != formattedRegistration.Digest {
		t.Fatalf("canonical registration = %+v, found=%v", got, ok)
	}

	read.InputSchema[0] = '['
	read.Permissions[0] = identity.PermissionEffectsApprove
	got.Spec.InputSchema[0] = '['
	got.Spec.Permissions[0] = identity.PermissionEffectsApprove
	listed[0].Spec.OutputSchema[0] = '['
	again, ok := registry.Lookup("artifacts.read", "2")
	if !ok || again.Spec.InputSchema[0] != '{' || again.Spec.OutputSchema[0] != '{' ||
		again.Spec.Permissions[0] != identity.PermissionArtifactsRead {
		t.Fatalf("registry state escaped immutability: %+v", again)
	}
	if _, found := registry.Lookup("artifacts.read", "3"); found {
		t.Fatal("lookup selected an undeclared version")
	}
}

func TestRegistryRejectsInvalidOrAmbiguousCapabilitySpecsAtomically(t *testing.T) {
	base := validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication)
	tests := map[string]func(*CapabilitySpec){
		"id without namespace": func(value *CapabilitySpec) { value.ID = "commit" },
		"noncanonical id":      func(value *CapabilitySpec) { value.ID = "Workspace.commit" },
		"zero version":         func(value *CapabilitySpec) { value.Version = "0" },
		"leading zero version": func(value *CapabilitySpec) {
			value.Version = "01"
		},
		"missing permission": func(value *CapabilitySpec) { value.Permissions = nil },
		"unknown permission": func(value *CapabilitySpec) {
			value.Permissions = []identity.Permission{"workspace.commit"}
		},
		"duplicate permission": func(value *CapabilitySpec) {
			value.Permissions = []identity.Permission{identity.PermissionChangesIntegrate, identity.PermissionChangesIntegrate}
		},
		"omitted cost":      func(value *CapabilitySpec) { value.Cost = CostContract{} },
		"negative cost":     func(value *CapabilitySpec) { value.Cost.Maximum.DiskBytes = -1 },
		"money no currency": func(value *CapabilitySpec) { value.Cost.Maximum.MoneyMicros = 1 },
		"output missing":    func(value *CapabilitySpec) { value.Output = OutputDelivery{} },
		"inline negative":   func(value *CapabilitySpec) { value.Output.InlineBytes = -1 },
		"inline over max": func(value *CapabilitySpec) {
			value.Output.InlineBytes = value.Output.MaxBytes + 1
		},
		"output over disk budget": func(value *CapabilitySpec) {
			value.Output.MaxBytes = value.Cost.Maximum.DiskBytes + 1
		},
		"omitted policies": func(value *CapabilitySpec) { value.Idempotency, value.Receipt = "", "" },
		"crossed policies": func(value *CapabilitySpec) {
			value.Idempotency, value.Receipt = IdempotencyRequired, ReceiptObservation
		},
		"malformed input schema": func(value *CapabilitySpec) { value.InputSchema = json.RawMessage(`{"type":`) },
		"open input schema": func(value *CapabilitySpec) {
			value.InputSchema = json.RawMessage(`{"type":"object","properties":{},"additionalProperties":true}`)
		},
		"nested open schema": func(value *CapabilitySpec) {
			value.InputSchema = json.RawMessage(`{"type":"object","properties":{"payload":{"type":"object","properties":{},"additionalProperties":true}},"additionalProperties":false}`)
		},
		"unknown required": func(value *CapabilitySpec) {
			value.OutputSchema = json.RawMessage(`{"type":"object","properties":{},"required":["receipt_ref"],"additionalProperties":false}`)
		},
		"unsupported keyword": func(value *CapabilitySpec) {
			value.OutputSchema = json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false,"oneOf":[]}`)
		},
		"null constraint": func(value *CapabilitySpec) {
			value.OutputSchema = json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","minLength":null}},"additionalProperties":false}`)
		},
		"null enum": func(value *CapabilitySpec) {
			value.OutputSchema = json.RawMessage(`{"type":"object","properties":{"value":{"type":"string","enum":null}},"additionalProperties":false}`)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := cloneTestSpec(base)
			mutate(&candidate)
			registry, err := NewRegistry(base, candidate)
			if registry != nil || ErrorCode(err) != ErrorSpecInvalid {
				t.Fatalf("registry=%v error=%v code=%q", registry, err, ErrorCode(err))
			}
		})
	}
}

func TestRegistryValidatesAndCanonicalizesInputAndOutputPayloads(t *testing.T) {
	registry, err := NewRegistry(validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication))
	if err != nil {
		t.Fatal(err)
	}
	input, err := registry.ValidateInput("workspace.commit", "1", json.RawMessage(" \n{\"request_ref\":\"request:one\"}"))
	if err != nil || string(input) != `{"request_ref":"request:one"}` {
		t.Fatalf("input=%s error=%v", input, err)
	}
	output, err := registry.ValidateOutput("workspace.commit", "1", json.RawMessage(`{"accepted":true}`))
	if err != nil || string(output) != `{"accepted":true}` {
		t.Fatalf("output=%s error=%v", output, err)
	}
	limited := validSpec("workspace.limited", "1", IdempotencyRequired, ReceiptApplication)
	limited.Output.MaxBytes = int64(len(`{"accepted":true}`))
	limited.Output.InlineBytes = limited.Output.MaxBytes
	limitedRegistry, err := NewRegistry(limited)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := limitedRegistry.ValidateOutput("workspace.limited", "1", json.RawMessage(" \n{\"accepted\":true}")); got != nil ||
		ErrorCode(err) != ErrorPayloadInvalid {
		t.Fatalf("oversized output=%q err=%v code=%q", got, err, ErrorCode(err))
	}
	invalid := []json.RawMessage{
		nil, json.RawMessage(`null`), json.RawMessage(`{}`),
		json.RawMessage(`{"request_ref":""}`), json.RawMessage(`{"request_ref":"ok","extra":true}`),
		json.RawMessage([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}),
	}
	for _, payload := range invalid {
		if got, err := registry.ValidateInput("workspace.commit", "1", payload); got != nil ||
			ErrorCode(err) != ErrorPayloadInvalid {
			t.Fatalf("payload=%q got=%q err=%v code=%q", payload, got, err, ErrorCode(err))
		}
	}
	if _, err := registry.ValidateInput("workspace.commit", "2", json.RawMessage(`{}`)); ErrorCode(err) != ErrorSpecNotFound {
		t.Fatalf("not found error=%v code=%q", err, ErrorCode(err))
	}
}

func TestRegistryOutputMaxBytesMeasuresRawPayloadBeforeCanonicalization(t *testing.T) {
	const compactOutput = `{"accepted":true}`
	spec := validSpec("workspace.bounded", "1", IdempotencyRequired, ReceiptApplication)
	spec.Output.MaxBytes = int64(len(compactOutput))
	spec.Output.InlineBytes = spec.Output.MaxBytes

	registry, err := NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}
	wantRegistration, ok := registry.Lookup(spec.ID, spec.Version)
	if !ok {
		t.Fatal("bounded tool was not registered")
	}

	got, err := registry.ValidateOutput(spec.ID, spec.Version, json.RawMessage(compactOutput))
	if err != nil || string(got) != compactOutput {
		t.Fatalf("exact-limit output=%q err=%v code=%q", got, err, ErrorCode(err))
	}

	for name, payload := range map[string]json.RawMessage{
		"one byte over":                        json.RawMessage(compactOutput + " "),
		"whitespace canonicalizes under limit": json.RawMessage(" \n" + compactOutput),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := registry.ValidateOutput(spec.ID, spec.Version, payload)
			if got != nil || ErrorCode(err) != ErrorPayloadInvalid {
				t.Fatalf("output=%q err=%v code=%q", got, err, ErrorCode(err))
			}
		})
	}

	// The output delivery budget must not leak into input validation or mutate
	// the canonical, detached capability schema kept by the registry.
	input, err := registry.ValidateInput(spec.ID, spec.Version, json.RawMessage(" \n{\"request_ref\":\"request:one\"}"))
	if err != nil || string(input) != `{"request_ref":"request:one"}` {
		t.Fatalf("input=%q err=%v code=%q", input, err, ErrorCode(err))
	}
	after, ok := registry.Lookup(spec.ID, spec.Version)
	if !ok || after.Digest != wantRegistration.Digest ||
		!reflect.DeepEqual(after.Spec.InputSchema, wantRegistration.Spec.InputSchema) ||
		!reflect.DeepEqual(after.Spec.OutputSchema, wantRegistration.Spec.OutputSchema) {
		t.Fatalf("registration mutated after validation: before=%+v after=%+v", wantRegistration, after)
	}
}

func TestRegistryPayloadValidationCoversTheDeclaredSchemaDialect(t *testing.T) {
	spec := validSpec("metrics.read", "1", IdempotencyReadReexecute, ReceiptObservation)
	spec.InputSchema = json.RawMessage(`{
        "type":"object",
        "properties":{
            "count":{"type":"integer","minimum":1,"maximum":3},
            "ratio":{"type":"number","minimum":0.5,"maximum":2.5},
            "flags":{"type":"array","minItems":1,"maxItems":2,"items":{"type":"boolean"}},
            "mode":{"type":"string","enum":["brief","full"]},
            "nested":{"type":"object","properties":{"name":{"type":"string","minLength":1}},"required":["name"],"additionalProperties":false}
        },
        "required":["count","ratio","flags","mode","nested"],
        "additionalProperties":false
    }`)
	registry, err := NewRegistry(spec)
	if err != nil {
		t.Fatal(err)
	}
	valid := json.RawMessage(`{"count":2,"ratio":1.25,"flags":[true,false],"mode":"brief","nested":{"name":"one"}}`)
	if canonical, err := registry.ValidateInput("metrics.read", "1", valid); err != nil || !json.Valid(canonical) {
		t.Fatalf("canonical=%s error=%v", canonical, err)
	}
	invalid := []json.RawMessage{
		json.RawMessage(`{"count":0,"ratio":1,"flags":[true],"mode":"brief","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":1.5,"ratio":1,"flags":[true],"mode":"brief","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":2,"ratio":3,"flags":[true],"mode":"brief","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":2,"ratio":1,"flags":[],"mode":"brief","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":2,"ratio":1,"flags":[true,null],"mode":"brief","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":2,"ratio":1,"flags":[true],"mode":"other","nested":{"name":"one"}}`),
		json.RawMessage(`{"count":2,"ratio":1,"flags":[true],"mode":"brief","nested":{"name":"one","extra":true}}`),
	}
	for _, payload := range invalid {
		if _, err := registry.ValidateInput("metrics.read", "1", payload); ErrorCode(err) != ErrorPayloadInvalid {
			t.Fatalf("payload=%s error=%v code=%q", payload, err, ErrorCode(err))
		}
	}
}

func TestRegistryRejectsDuplicateExactToolVersion(t *testing.T) {
	first := validSpec("workspace.commit", "1", IdempotencyRequired, ReceiptApplication)
	duplicate := cloneTestSpec(first)
	duplicate.OutputSchema = schema(`"receipt_ref":{"type":"string","minLength":1}`, `"receipt_ref"`)
	registry, err := NewRegistry(first, duplicate)
	if registry != nil || ErrorCode(err) != ErrorSpecDuplicate {
		t.Fatalf("registry=%v error=%v code=%q", registry, err, ErrorCode(err))
	}
	if nilRegistry := (*Registry)(nil); nilRegistry.Digest() != "" || nilRegistry.List() != nil {
		t.Fatal("nil registry should be inert")
	}
}

func validSpec(id, version string, idempotency IdempotencyPolicy, receipt ReceiptPolicy) CapabilitySpec {
	return CapabilitySpec{
		ID: id, Version: version,
		InputSchema:  schema(`"request_ref":{"type":"string","minLength":1}`, `"request_ref"`),
		OutputSchema: schema(`"accepted":{"type":"boolean"}`, `"accepted"`),
		Permissions:  []identity.Permission{identity.PermissionChangesIntegrate},
		Cost: CostContract{Mode: CostMaximum, Maximum: governance.ResourceVector{
			Tokens: 50, ActiveTimeNS: 1_000_000, DiskBytes: 4096,
		}},
		Output:      OutputDelivery{MaxBytes: 4096, InlineBytes: 512},
		Idempotency: idempotency,
		Receipt:     receipt,
	}
}

func schema(properties, required string) json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` + properties + `},"required":[` + required + `],"additionalProperties":false}`)
}

func cloneTestSpec(source CapabilitySpec) CapabilitySpec {
	source.InputSchema = append(json.RawMessage(nil), source.InputSchema...)
	source.OutputSchema = append(json.RawMessage(nil), source.OutputSchema...)
	source.Permissions = append([]identity.Permission(nil), source.Permissions...)
	return source
}
