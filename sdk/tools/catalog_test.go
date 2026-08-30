package tools_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	tools "orquesta/sdk/tools"
)

func TestPublicCatalogIsCanonicalDeterministicAndDetached(t *testing.T) {
	read := validSpec("artifacts.read", "2", tools.IdempotencyReadReexecute, tools.ReceiptObservation)
	write := validSpec("workspace.commit", "10", tools.IdempotencyRequired, tools.ReceiptApplication)
	older := validSpec("workspace.commit", "2", tools.IdempotencyRequired, tools.ReceiptApplication)
	read.Permissions = []tools.Permission{"goals.get", "artifacts.read"}

	catalog, err := tools.NewCatalog(write, read, older)
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := tools.NewCatalog(older, write, read)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Digest() != reordered.Digest() || len(catalog.Digest()) != len("sha256:")+64 {
		t.Fatalf("digests=%q/%q", catalog.Digest(), reordered.Digest())
	}
	listed := catalog.List()
	wantOrder := []string{"artifacts.read@2", "workspace.commit@2", "workspace.commit@10"}
	gotOrder := make([]string, len(listed))
	for index, registration := range listed {
		gotOrder[index] = registration.Spec.ID + "@" + registration.Spec.Version
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("order=%v", gotOrder)
	}
	registration, found := catalog.Lookup("artifacts.read", "2")
	if !found || !reflect.DeepEqual(registration.Spec.Permissions,
		[]tools.Permission{"artifacts.read", "goals.get"}) {
		t.Fatalf("registration=%+v found=%v", registration, found)
	}

	read.InputSchema[0] = '['
	read.Permissions[0] = "effects.approve"
	registration.Spec.InputSchema[0] = '['
	registration.Spec.Permissions[0] = "effects.approve"
	listed[0].Spec.OutputSchema[0] = '['
	again, found := catalog.Lookup("artifacts.read", "2")
	if !found || again.Spec.InputSchema[0] != '{' || again.Spec.OutputSchema[0] != '{' ||
		again.Spec.Permissions[0] != "artifacts.read" {
		t.Fatalf("catalog state escaped: %+v", again)
	}
	if _, found := catalog.Lookup("artifacts.read", "3"); found {
		t.Fatal("lookup selected undeclared latest")
	}
}

func TestDecodeSpecIsStrictAndReturnsCanonicalPublicForm(t *testing.T) {
	encoded := []byte(`{
        "version":"1",
        "id":"workspace.commit",
        "output_schema":{"required":["receipt_ref"],"properties":{"receipt_ref":{"minLength":1,"type":"string"}},"additionalProperties":false,"type":"object"},
        "input_schema":{"required":["request_ref"],"properties":{"request_ref":{"type":"string","minLength":1}},"additionalProperties":false,"type":"object"},
        "permissions":["effects.approve","changes.integrate"],
        "cost":{"mode":"maximum","maximum":{"money_micros":25,"currency":"EUR","active_time_ns":1000,"disk_bytes":64,"tokens":0,"process_slots":0}},
        "output":{"max_bytes":64,"inline_bytes":16},
        "idempotency":"required",
        "receipt":"application"
    }`)
	spec, err := tools.DecodeSpec(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if spec.ID != "workspace.commit" || spec.Version != "1" ||
		!reflect.DeepEqual(spec.Permissions, []tools.Permission{"changes.integrate", "effects.approve"}) ||
		spec.Cost.Maximum.Currency != "EUR" || spec.InputSchema[0] != '{' {
		t.Fatalf("canonical spec=%+v", spec)
	}

	invalidEncoding := [][]byte{
		nil,
		[]byte(`{`),
		append(append([]byte(nil), encoded...), []byte(` {}`)...),
		[]byte(`{"id":"workspace.commit","unknown":true}`),
	}
	for _, candidate := range invalidEncoding {
		if _, err := tools.DecodeSpec(candidate); !errors.Is(err, tools.ErrSpecEncoding) ||
			tools.ErrorCode(err) != "toolsdk.spec_encoding_invalid" {
			t.Fatalf("encoding=%q err=%v code=%q", candidate, err, tools.ErrorCode(err))
		}
	}
	semantic := validSpec("workspace.commit", "1", tools.IdempotencyRequired, tools.ReceiptApplication)
	semantic.Permissions = nil
	encodedSemantic, err := json.Marshal(semantic)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tools.DecodeSpec(encodedSemantic); !errors.Is(err, tools.ErrSpecInvalid) ||
		tools.ErrorCode(err) != "toolsdk.spec_invalid" {
		t.Fatalf("semantic mutation error=%v code=%q", err, tools.ErrorCode(err))
	}
}

func TestPublicCatalogMapsCanonicalRegistryErrorsWithoutPartialState(t *testing.T) {
	valid := validSpec("workspace.commit", "1", tools.IdempotencyRequired, tools.ReceiptApplication)
	invalid := cloneSpec(valid)
	invalid.Permissions = nil
	if catalog, err := tools.NewCatalog(valid, invalid); catalog != nil ||
		!errors.Is(err, tools.ErrSpecInvalid) || tools.ErrorCode(err) != "toolsdk.spec_invalid" {
		t.Fatalf("catalog=%v err=%v code=%q", catalog, err, tools.ErrorCode(err))
	}
	duplicate := cloneSpec(valid)
	duplicate.OutputSchema = schema(`"receipt_ref":{"type":"string"}`, `"receipt_ref"`)
	if catalog, err := tools.NewCatalog(valid, duplicate); catalog != nil ||
		!errors.Is(err, tools.ErrSpecDuplicate) || tools.ErrorCode(err) != "toolsdk.spec_duplicate" {
		t.Fatalf("catalog=%v err=%v code=%q", catalog, err, tools.ErrorCode(err))
	}
	var nilCatalog *tools.Catalog
	if nilCatalog.Digest() != "" || nilCatalog.List() != nil {
		t.Fatal("nil catalog is not inert")
	}
}

func validSpec(id, version string, idempotency tools.IdempotencyPolicy, receipt tools.ReceiptPolicy) tools.Spec {
	return tools.Spec{
		ID: id, Version: version,
		InputSchema:  schema(`"request_ref":{"type":"string","minLength":1}`, `"request_ref"`),
		OutputSchema: schema(`"accepted":{"type":"boolean"}`, `"accepted"`),
		Permissions:  []tools.Permission{"changes.integrate"},
		Cost: tools.CostContract{Mode: tools.CostMaximum, Maximum: tools.ResourceCost{
			Tokens: 10, ActiveTimeNS: 1000, DiskBytes: 64,
		}},
		Output:      tools.OutputDelivery{MaxBytes: 64, InlineBytes: 16},
		Idempotency: idempotency,
		Receipt:     receipt,
	}
}

func schema(properties, required string) json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` + properties + `},"required":[` + required + `],"additionalProperties":false}`)
}

func cloneSpec(source tools.Spec) tools.Spec {
	source.InputSchema = append(json.RawMessage(nil), source.InputSchema...)
	source.OutputSchema = append(json.RawMessage(nil), source.OutputSchema...)
	source.Permissions = append([]tools.Permission(nil), source.Permissions...)
	return source
}
