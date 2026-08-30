package tooling

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestResourcePageRequestResolvesExactSpecAndNormalizesLimit(t *testing.T) {
	catalog := resourcePageCatalog(t, validResourceSpec("status.snapshot", "1"))
	request, err := catalog.NewResourcePageRequest("status.snapshot", "1", "", "", 0)
	if err != nil || request.ID != "status.snapshot" || request.Version != "1" ||
		request.URI != "orquesta://project/status/snapshot" || request.Limit != 2 ||
		!strings.HasPrefix(request.SpecDigest, "sha256:") {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	resumed, err := catalog.NewResourcePageRequest(
		"status.snapshot", "1", "cursor:opaque:2", "resource-snapshot:opaque:one", 4,
	)
	if err != nil || resumed.Cursor != "cursor:opaque:2" || resumed.Limit != 4 ||
		resumed.SpecDigest != request.SpecDigest {
		t.Fatalf("resumed=%+v error=%v", resumed, err)
	}
	invalid := []struct {
		id, version, cursor, snapshot string
		limit                         uint32
		code                          string
	}{
		{"missing.resource", "1", "", "", 1, ErrorSpecNotFound},
		{"status.snapshot", "2", "", "", 1, ErrorSpecNotFound},
		{"status.snapshot", "1", " cursor", "snapshot:one", 1, ErrorResourceRequestInvalid},
		{"status.snapshot", "1", "cursor\nsecret", "snapshot:one", 1, ErrorResourceRequestInvalid},
		{"status.snapshot", "1", "cursor:one", "", 1, ErrorResourceRequestInvalid},
		{"status.snapshot", "1", "", "snapshot:one", 1, ErrorResourceRequestInvalid},
		{"status.snapshot", "1", "", "", 5, ErrorResourceRequestInvalid},
	}
	for _, test := range invalid {
		if got, err := catalog.NewResourcePageRequest(test.id, test.version, test.cursor, test.snapshot, test.limit); !reflect.DeepEqual(got, ResourcePageRequest{}) ||
			ErrorCode(err) != test.code {
			t.Fatalf("input=%+v got=%+v error=%v code=%q", test, got, err, ErrorCode(err))
		}
	}
}

func TestResourcePagesBindSnapshotCursorSchemaAndBudgets(t *testing.T) {
	catalog := resourcePageCatalog(t, validResourceSpec("status.snapshot", "1"))
	firstRequest, err := catalog.NewResourcePageRequest("status.snapshot", "1", "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	first := pageForRequest(firstRequest,
		json.RawMessage(`{ "value": "one" }`), json.RawMessage(`{"value":"two"}`))
	first.NextCursor = "cursor:opaque:2"
	validated, err := catalog.ValidateResourcePage(firstRequest, first)
	if err != nil || validated.Complete || validated.NextCursor != "cursor:opaque:2" ||
		string(validated.Items[0]) != `{"value":"one"}` {
		t.Fatalf("validated=%+v error=%v", validated, err)
	}
	validated.Items[0][0] = '['
	if first.Items[0][0] != '{' {
		t.Fatal("validated page aliases connector-owned item bytes")
	}
	secondRequest, err := catalog.NewResourcePageRequest(
		"status.snapshot", "1", validated.NextCursor, validated.SnapshotRef, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	second := pageForRequest(secondRequest, json.RawMessage(`{"value":"three"}`))
	second.Complete = true
	final, err := catalog.ValidateResourcePage(secondRequest, second)
	if err != nil || !final.Complete || final.NextCursor != "" || final.SnapshotRef != first.SnapshotRef {
		t.Fatalf("final=%+v error=%v", final, err)
	}
}

func TestResourcePageRejectsEveryCausalAndPayloadDivergence(t *testing.T) {
	catalog := resourcePageCatalog(t, validResourceSpec("status.snapshot", "1"))
	request, err := catalog.NewResourcePageRequest(
		"status.snapshot", "1", "cursor:one", "resource-snapshot:opaque:one", 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	valid := pageForRequest(request, json.RawMessage(`{"value":"one"}`))
	valid.NextCursor = "cursor:two"
	tests := map[string]func(*ResourcePage){
		"id":             func(page *ResourcePage) { page.ID = "other.snapshot" },
		"version":        func(page *ResourcePage) { page.Version = "2" },
		"uri":            func(page *ResourcePage) { page.URI = "orquesta://project/other/snapshot" },
		"digest":         func(page *ResourcePage) { page.SpecDigest = "sha256:" + strings.Repeat("0", 64) },
		"request cursor": func(page *ResourcePage) { page.RequestCursor = "cursor:other" },
		"snapshot":       func(page *ResourcePage) { page.SnapshotRef = "" },
		"snapshot changed": func(page *ResourcePage) {
			page.SnapshotRef = "resource-snapshot:opaque:other"
		},
		"snapshot control": func(page *ResourcePage) {
			page.SnapshotRef = "snapshot\nsecret"
		},
		"complete cursor": func(page *ResourcePage) { page.Complete = true },
		"missing next":    func(page *ResourcePage) { page.NextCursor = "" },
		"same cursor":     func(page *ResourcePage) { page.NextCursor = page.RequestCursor },
		"empty progress": func(page *ResourcePage) {
			page.Items = nil
		},
		"too many": func(page *ResourcePage) {
			page.Items = append(page.Items,
				json.RawMessage(`{"value":"two"}`), json.RawMessage(`{"value":"three"}`))
		},
		"schema": func(page *ResourcePage) { page.Items[0] = json.RawMessage(`{"other":true}`) },
		"null":   func(page *ResourcePage) { page.Items[0] = json.RawMessage(`null`) },
		"bytes": func(page *ResourcePage) {
			page.Items[0] = json.RawMessage(`{"value":"` + strings.Repeat("x", 2100) + `"}`)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			page := cloneResourcePage(valid)
			mutate(&page)
			got, err := catalog.ValidateResourcePage(request, page)
			if !reflect.DeepEqual(got, ResourcePage{}) || ErrorCode(err) != ErrorResourcePageInvalid {
				t.Fatalf("page=%+v got=%+v error=%v code=%q", page, got, err, ErrorCode(err))
			}
		})
	}
	badRequests := []func(*ResourcePageRequest){
		func(value *ResourcePageRequest) { value.URI = "orquesta://project/other/snapshot" },
		func(value *ResourcePageRequest) { value.SpecDigest = "sha256:short" },
		func(value *ResourcePageRequest) { value.Limit = 0 },
		func(value *ResourcePageRequest) { value.Cursor = " invalid" },
		func(value *ResourcePageRequest) { value.SnapshotRef = "" },
	}
	for _, mutate := range badRequests {
		bad := request
		mutate(&bad)
		if _, err := catalog.ValidateResourcePage(bad, valid); ErrorCode(err) != ErrorResourceRequestInvalid {
			t.Fatalf("request=%+v error=%v code=%q", bad, err, ErrorCode(err))
		}
	}
}

func TestResourcePolicyAndSubscriptionAreBoundToCatalogDigest(t *testing.T) {
	base := validResourceSpec("status.snapshot", "1")
	first := resourcePageCatalog(t, base)
	changedPage := cloneResourceSpec(base)
	changedPage.Page.MaxItems++
	second := resourcePageCatalog(t, changedPage)
	changedSubscription := cloneResourceSpec(base)
	changedSubscription.Subscription = ResourceSubscriptionNone
	third := resourcePageCatalog(t, changedSubscription)
	if first.Digest() == second.Digest() || first.Digest() == third.Digest() || second.Digest() == third.Digest() {
		t.Fatalf("digests first=%q second=%q third=%q", first.Digest(), second.Digest(), third.Digest())
	}
}

func TestResourcePageRejectsCrossCandidateTransplant(t *testing.T) {
	base := validResourceSpec("status.snapshot", "1")
	oldCatalog := resourcePageCatalog(t, base)
	oldRequest, err := oldCatalog.NewResourcePageRequest("status.snapshot", "1", "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	oldPage := pageForRequest(oldRequest, json.RawMessage(`{"value":"one"}`))
	oldPage.Complete = true

	changed := cloneResourceSpec(base)
	changed.Page.MaxBytes--
	newCatalog := resourcePageCatalog(t, changed)
	newRequest, err := newCatalog.NewResourcePageRequest("status.snapshot", "1", "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if oldRequest.SpecDigest == newRequest.SpecDigest {
		t.Fatalf("candidate digests unexpectedly match: %q", oldRequest.SpecDigest)
	}

	if got, err := newCatalog.ValidateResourcePage(oldRequest, oldPage); !reflect.DeepEqual(got, ResourcePage{}) ||
		ErrorCode(err) != ErrorResourceRequestInvalid {
		t.Fatalf("stale request transplant got=%+v error=%v code=%q", got, err, ErrorCode(err))
	}
	if got, err := newCatalog.ValidateResourcePage(newRequest, oldPage); !reflect.DeepEqual(got, ResourcePage{}) ||
		ErrorCode(err) != ErrorResourcePageInvalid {
		t.Fatalf("stale page transplant got=%+v error=%v code=%q", got, err, ErrorCode(err))
	}
}

func resourcePageCatalog(t *testing.T, resource ResourceSpec) *SurfaceCatalog {
	t.Helper()
	tools, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewSurfaceCatalog(tools, []ResourceSpec{resource}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func pageForRequest(request ResourcePageRequest, items ...json.RawMessage) ResourcePage {
	return ResourcePage{
		ID: request.ID, Version: request.Version, URI: request.URI, SpecDigest: request.SpecDigest,
		RequestCursor: request.Cursor, SnapshotRef: "resource-snapshot:opaque:one", Items: items,
	}
}
