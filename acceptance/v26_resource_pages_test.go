package acceptance_test

import (
	"encoding/json"
	"testing"

	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS03ResourcePagesAreAddressedBoundedAndSnapshotCausal(t *testing.T) {
	tools, err := tooling.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := tooling.NewSurfaceCatalog(tools, []tooling.ResourceSpec{{
		ID: "timeline.read", Version: "1", URI: "orquesta://project/timeline/items",
		MediaType: "application/json", Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		MaxBytes:     4096,
		ItemSchema:   json.RawMessage(`{"type":"object","properties":{"ref":{"type":"string","minLength":1}},"required":["ref"],"additionalProperties":false}`),
		Page:         tooling.ResourcePagePolicy{DefaultItems: 1, MaxItems: 2, MaxBytes: 1024},
		Subscription: tooling.ResourceSubscriptionSnapshotChanged,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registration, found := catalog.LookupResource("timeline.read", "1")
	if !found || registration.Spec.Subscription != tooling.ResourceSubscriptionSnapshotChanged {
		t.Fatalf("registration=%+v found=%v", registration, found)
	}
	firstRequest, err := catalog.NewResourcePageRequest("timeline.read", "1", "", "", 0)
	if err != nil || firstRequest.Limit != 1 || firstRequest.URI != registration.Spec.URI ||
		firstRequest.SpecDigest != registration.Digest {
		t.Fatalf("request=%+v error=%v", firstRequest, err)
	}
	first, err := catalog.ValidateResourcePage(firstRequest, tooling.ResourcePage{
		ID: firstRequest.ID, Version: firstRequest.Version, URI: firstRequest.URI,
		SpecDigest: firstRequest.SpecDigest, SnapshotRef: "snapshot:timeline:opaque",
		Items: []json.RawMessage{json.RawMessage(`{ "ref": "event:1" }`)}, NextCursor: "cursor:timeline:opaque:2",
	})
	if err != nil || first.Complete || string(first.Items[0]) != `{"ref":"event:1"}` {
		t.Fatalf("first=%+v error=%v", first, err)
	}
	secondRequest, err := catalog.NewResourcePageRequest(
		"timeline.read", "1", first.NextCursor, first.SnapshotRef, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.ValidateResourcePage(secondRequest, tooling.ResourcePage{
		ID: secondRequest.ID, Version: secondRequest.Version, URI: secondRequest.URI,
		SpecDigest: secondRequest.SpecDigest, RequestCursor: secondRequest.Cursor,
		SnapshotRef: first.SnapshotRef, Complete: true,
	})
	if err != nil || !second.Complete || second.SnapshotRef != first.SnapshotRef {
		t.Fatalf("second=%+v error=%v", second, err)
	}
	forged := second
	forged.SpecDigest = "sha256:forged"
	if _, err := catalog.ValidateResourcePage(secondRequest, forged); tooling.ErrorCode(err) != tooling.ErrorResourcePageInvalid {
		t.Fatalf("forged error=%v code=%q", err, tooling.ErrorCode(err))
	}

	mutations := []struct {
		name   string
		mutate func(*tooling.ResourcePage)
	}{
		{"resource identity", func(page *tooling.ResourcePage) { page.ID = "timeline.other" }},
		{"resource version", func(page *tooling.ResourcePage) { page.Version = "2" }},
		{"resource uri", func(page *tooling.ResourcePage) { page.URI = "orquesta://project/other/items" }},
		{"request cursor", func(page *tooling.ResourcePage) { page.RequestCursor = "cursor:transplanted" }},
		{"snapshot transplant", func(page *tooling.ResourcePage) { page.SnapshotRef = "snapshot:other:opaque" }},
		{"complete page with cursor", func(page *tooling.ResourcePage) { page.NextCursor = "cursor:unexpected" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			page := tooling.ResourcePage{
				ID: secondRequest.ID, Version: secondRequest.Version, URI: secondRequest.URI,
				SpecDigest: secondRequest.SpecDigest, RequestCursor: secondRequest.Cursor,
				SnapshotRef: first.SnapshotRef, Complete: true,
			}
			mutation.mutate(&page)
			if _, err := catalog.ValidateResourcePage(secondRequest, page); tooling.ErrorCode(err) != tooling.ErrorResourcePageInvalid {
				t.Fatalf("mutation accepted: page=%+v error=%v code=%q", page, err, tooling.ErrorCode(err))
			}
		})
	}

	if _, err := catalog.NewResourcePageRequest("timeline.read", "1", "", "", 3); tooling.ErrorCode(err) != tooling.ErrorResourceRequestInvalid {
		t.Fatalf("oversized page admitted: error=%v code=%q", err, tooling.ErrorCode(err))
	}
}
