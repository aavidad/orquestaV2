package tooling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestSkillResourcesRemainCanonicalMetadataAndChangeExactRegistration(t *testing.T) {
	tools := skillTestTools(t)
	first := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.policy", "text/markdown", []byte("# Policy\n\nReviewed content.\n")),
		resourceFixture("fixtures.input", "application/json", []byte(`{"value":"reviewed"}`)),
	)
	registry, err := NewSkillRegistry(tools, first)
	if err != nil {
		t.Fatal(err)
	}
	reordered := resourceSkillCandidate("review.resources", "1",
		resourceFixture("fixtures.input", "application/json", []byte(`{"value":"reviewed"}`)),
		resourceFixture("references.policy", "text/markdown", []byte("# Policy\n\nReviewed content.\n")),
	)
	second, err := NewSkillRegistry(tools, reordered)
	if err != nil || registry.Digest() != second.Digest() {
		t.Fatalf("registry=%q reordered=%q error=%v", registry.Digest(), second.Digest(), err)
	}
	resources, found := registry.ListResources("review.resources", "1")
	if !found || len(resources) != 2 || resources[0].ID != "fixtures.input" ||
		resources[1].ID != "references.policy" {
		t.Fatalf("resources=%+v found=%v", resources, found)
	}
	registration, _ := registry.Lookup("review.resources", "1")
	encoded, err := json.Marshal(registration)
	if err != nil || bytes.Contains(encoded, []byte("Reviewed content")) ||
		bytes.Contains(encoded, []byte(`{"value":"reviewed"}`)) {
		t.Fatalf("metadata=%s error=%v", encoded, err)
	}
	changed := resourceSkillCandidate("review.resources", "1",
		resourceFixture("fixtures.input", "application/json", []byte(`{"value":"changed"}`)),
		resourceFixture("references.policy", "text/markdown", []byte("# Policy\n\nReviewed content.\n")),
	)
	changedRegistry, err := NewSkillRegistry(tools, changed)
	if err != nil || changedRegistry.Digest() == registry.Digest() {
		t.Fatalf("changed=%q original=%q error=%v", changedRegistry.Digest(), registry.Digest(), err)
	}
	first.Spec.Resources[0].ID = "mutated.resource"
	first.ResourceContents[0].Content[0] = 'x'
	resources[0].ID = "mutated.resource"
	if again, _ := registry.ListResources("review.resources", "1"); again[0].ID != "fixtures.input" {
		t.Fatal("caller mutation changed resource metadata")
	}
	if resources, found := registry.ListResources("review.resources", "2"); found || resources != nil {
		t.Fatalf("missing resources=%+v found=%v", resources, found)
	}
}

func TestSkillResourceLoadRequiresExactRegistrationRefDigestSizeAndBytes(t *testing.T) {
	tools := skillTestTools(t)
	content := []byte("# Policy\n\nLoad only when requested.\n")
	candidate := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.policy", "text/markdown", content),
	)
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	request, err := registry.NewResourceLoadRequest("review.resources", "1", "references.policy")
	if err != nil || request.MediaType != "text/markdown" || request.SizeBytes != int64(len(content)) ||
		request.MaxBytes != MaxSkillResourceBytes || request.ContentRef != "artifact:"+request.ContentDigest {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	loaded, err := registry.ValidateLoadedResource(request, content)
	if err != nil || !bytes.Equal(loaded, content) {
		t.Fatalf("loaded=%q error=%v", loaded, err)
	}
	loaded[0] = 'x'
	if content[0] != '#' {
		t.Fatal("loaded resource aliases caller bytes")
	}
	for _, mutate := range []func(*SkillResourceLoadRequest){
		func(value *SkillResourceLoadRequest) { value.SkillID = "other.resources" },
		func(value *SkillResourceLoadRequest) { value.SkillVersion = "2" },
		func(value *SkillResourceLoadRequest) { value.RegistrationDigest = repeatedDigest("0") },
		func(value *SkillResourceLoadRequest) { value.ScopesDigest = repeatedDigest("0") },
		func(value *SkillResourceLoadRequest) { value.ResourceID = "fixtures.input" },
		func(value *SkillResourceLoadRequest) { value.MediaType = "text/plain" },
		func(value *SkillResourceLoadRequest) { value.ContentRef = "artifact:" + repeatedDigest("0") },
		func(value *SkillResourceLoadRequest) { value.ContentDigest = repeatedDigest("0") },
		func(value *SkillResourceLoadRequest) { value.SizeBytes++ },
		func(value *SkillResourceLoadRequest) { value.MaxBytes-- },
	} {
		forged := request
		mutate(&forged)
		if loaded, err := registry.ValidateLoadedResource(forged, content); loaded != nil || SkillErrorCode(err) != ErrorSkillResourceLoadInvalid {
			t.Fatalf("forged=%+v loaded=%q error=%v", forged, loaded, err)
		}
	}
	replayed, err := registry.ValidateLoadedResource(request, content)
	if err != nil || !bytes.Equal(replayed, content) || &replayed[0] == &loaded[0] {
		t.Fatalf("replayed=%q loaded=%q error=%v", replayed, loaded, err)
	}
	changed := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.policy", "text/markdown", []byte("fresh content")),
	)
	freshRegistry, err := NewSkillRegistry(tools, changed)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := freshRegistry.ValidateLoadedResource(request, content); got != nil || SkillErrorCode(err) != ErrorSkillResourceLoadInvalid {
		t.Fatalf("stale request loaded=%q error=%v", got, err)
	}
	mutated := append([]byte(nil), content...)
	mutated[len(mutated)-2] = 'x'
	if loaded, err := registry.ValidateLoadedResource(request, mutated); loaded != nil || SkillErrorCode(err) != ErrorSkillResourceContentInvalid {
		t.Fatalf("mutated=%q loaded=%q error=%v", mutated, loaded, err)
	}
	if request, err := registry.NewResourceLoadRequest("review.resources", "1", "fixtures.input"); !reflect.DeepEqual(request, SkillResourceLoadRequest{}) || SkillErrorCode(err) != ErrorSkillResourceNotFound {
		t.Fatalf("missing request=%+v error=%v", request, err)
	}
	if request, err := registry.NewResourceLoadRequest("review.resources", "2", "references.policy"); !reflect.DeepEqual(request, SkillResourceLoadRequest{}) || SkillErrorCode(err) != ErrorSkillNotFound {
		t.Fatalf("missing skill request=%+v error=%v", request, err)
	}
}

func TestSkillResourceMetadataPagesBindScopeFreshnessAndReplay(t *testing.T) {
	tools := skillTestTools(t)
	candidate := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.third", "text/plain", []byte("third")),
		resourceFixture("references.first", "text/plain", []byte("first")),
		resourceFixture("references.second", "text/plain", []byte("second")),
	)
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	request, err := registry.NewResourcePageRequest("review.resources", "1", "", 2)
	if err != nil || request.Limit != 2 || request.ScopesDigest == "" || request.RegistrationDigest == "" {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	first, err := registry.ListResourcePage(request)
	if err != nil || first.Complete || first.NextCursor == "" || len(first.Items) != 2 ||
		first.Items[0].ID != "references.first" || first.Items[1].ID != "references.second" {
		t.Fatalf("first=%+v error=%v", first, err)
	}
	replayed, err := registry.ListResourcePage(request)
	if err != nil || !reflect.DeepEqual(replayed, first) {
		t.Fatalf("replayed=%+v first=%+v error=%v", replayed, first, err)
	}
	replayed.Items[0].ID = "mutated.resource"
	again, _ := registry.ListResourcePage(request)
	if again.Items[0].ID != "references.first" {
		t.Fatal("page aliases registry metadata")
	}
	nextRequest, err := registry.NewResourcePageRequest("review.resources", "1", first.NextCursor, 0)
	if err != nil || nextRequest.Limit != DefaultSkillResourcePageItems {
		t.Fatalf("next request=%+v error=%v", nextRequest, err)
	}
	final, err := registry.ListResourcePage(nextRequest)
	if err != nil || !final.Complete || final.NextCursor != "" || len(final.Items) != 1 ||
		final.Items[0].ID != "references.third" {
		t.Fatalf("final=%+v error=%v", final, err)
	}
	for _, mutate := range []func(*SkillResourcePageRequest){
		func(value *SkillResourcePageRequest) { value.SkillID = "other.resources" },
		func(value *SkillResourcePageRequest) { value.SkillVersion = "2" },
		func(value *SkillResourcePageRequest) { value.RegistrationDigest = repeatedDigest("0") },
		func(value *SkillResourcePageRequest) { value.ScopesDigest = repeatedDigest("0") },
		func(value *SkillResourcePageRequest) { value.Cursor = "../references.policy" },
		func(value *SkillResourcePageRequest) { value.Limit = 0 },
	} {
		forged := request
		mutate(&forged)
		if page, err := registry.ListResourcePage(forged); !reflect.DeepEqual(page, SkillResourcePage{}) || SkillErrorCode(err) != ErrorSkillResourcePageInvalid {
			t.Fatalf("forged=%+v page=%+v error=%v", forged, page, err)
		}
	}
	if got, err := registry.NewResourcePageRequest("review.resources", "1", "", MaxSkillResourcePageItems+1); !reflect.DeepEqual(got, SkillResourcePageRequest{}) || SkillErrorCode(err) != ErrorSkillResourcePageInvalid {
		t.Fatalf("oversized request=%+v error=%v", got, err)
	}
	registration, _ := registry.Lookup("review.resources", "1")
	finalCursor := skillResourceCursor(registration, len(registration.Spec.Resources))
	if got, err := registry.NewResourcePageRequest("review.resources", "1", finalCursor, 1); !reflect.DeepEqual(got, SkillResourcePageRequest{}) || SkillErrorCode(err) != ErrorSkillResourcePageInvalid {
		t.Fatalf("terminal cursor request=%+v error=%v", got, err)
	}
	changed := cloneSkillCandidate(candidate)
	changed.Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}}
	freshRegistry, err := NewSkillRegistry(tools, changed)
	if err != nil {
		t.Fatal(err)
	}
	if page, err := freshRegistry.ListResourcePage(request); !reflect.DeepEqual(page, SkillResourcePage{}) || SkillErrorCode(err) != ErrorSkillResourcePageInvalid {
		t.Fatalf("stale request page=%+v error=%v", page, err)
	}
}

func TestSkillRegistryRejectsMalformedOrDivergentResourcesAtomically(t *testing.T) {
	tools := skillTestTools(t)
	valid := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.policy", "text/markdown", []byte("reviewed")),
	)
	tests := map[string]func(*SkillCandidate){
		"missing content": func(value *SkillCandidate) { value.ResourceContents = nil },
		"extra content": func(value *SkillCandidate) {
			value.ResourceContents = append(value.ResourceContents, SkillResourceContent{ID: "extra.resource", Content: []byte("extra")})
		},
		"duplicate content": func(value *SkillCandidate) {
			value.ResourceContents = append(value.ResourceContents, value.ResourceContents[0])
		},
		"id": func(value *SkillCandidate) { value.Spec.Resources[0].ID = "policy" },
		"id traversal": func(value *SkillCandidate) {
			value.Spec.Resources[0].ID = "references.../policy"
		},
		"content traversal": func(value *SkillCandidate) {
			value.ResourceContents[0].ID = "../references.policy"
		},
		"media type": func(value *SkillCandidate) { value.Spec.Resources[0].MediaType = "text/markdown; charset=utf-8" },
		"ref":        func(value *SkillCandidate) { value.Spec.Resources[0].ContentRef = "../policy.md" },
		"digest":     func(value *SkillCandidate) { value.Spec.Resources[0].ContentDigest = "sha256:short" },
		"empty": func(value *SkillCandidate) {
			value.ResourceContents[0].Content = nil
			value.Spec.Resources[0].SizeBytes = 0
			value.Spec.Resources[0].ContentDigest = skillContentDigest(nil)
			value.Spec.Resources[0].ContentRef = "artifact:" + value.Spec.Resources[0].ContentDigest
		},
		"size":    func(value *SkillCandidate) { value.Spec.Resources[0].SizeBytes++ },
		"content": func(value *SkillCandidate) { value.ResourceContents[0].Content[0] = 'x' },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := cloneSkillCandidate(valid)
			mutate(&candidate)
			registry, err := NewSkillRegistry(tools, candidate)
			if registry != nil || SkillErrorCode(err) != ErrorSkillResourceInvalid {
				t.Fatalf("registry=%v error=%v code=%q", registry, err, SkillErrorCode(err))
			}
		})
	}
	oversized := cloneSkillCandidate(valid)
	oversized.ResourceContents[0].Content = bytes.Repeat([]byte{'x'}, MaxSkillResourceBytes+1)
	oversized.Spec.Resources[0].SizeBytes = int64(len(oversized.ResourceContents[0].Content))
	oversized.Spec.Resources[0].ContentDigest = skillContentDigest(oversized.ResourceContents[0].Content)
	oversized.Spec.Resources[0].ContentRef = "artifact:" + oversized.Spec.Resources[0].ContentDigest
	if registry, err := NewSkillRegistry(tools, oversized); registry != nil || SkillErrorCode(err) != ErrorSkillResourceInvalid {
		t.Fatalf("oversized registry=%v error=%v", registry, err)
	}
	duplicate := resourceSkillCandidate("review.resources", "1",
		resourceFixture("references.policy", "text/markdown", []byte("first")),
		resourceFixture("references.policy", "text/markdown", []byte("second")),
	)
	if registry, err := NewSkillRegistry(tools, duplicate); registry != nil || SkillErrorCode(err) != ErrorSkillResourceInvalid {
		t.Fatalf("duplicate registry=%v error=%v", registry, err)
	}
	tooMany := skillCandidate("review.too-many", "1")
	for index := 0; index <= MaxSkillResourcesPerSkill; index++ {
		fixture := resourceFixture(fmt.Sprintf("references.r%04d", index), "text/plain", []byte("x"))
		tooMany.Spec.Resources = append(tooMany.Spec.Resources, fixture.spec)
		tooMany.ResourceContents = append(tooMany.ResourceContents, fixture.content)
	}
	if registry, err := NewSkillRegistry(tools, tooMany); registry != nil || SkillErrorCode(err) != ErrorSkillResourceInvalid {
		t.Fatalf("resource count registry=%v error=%v", registry, err)
	}
	encoded, err := json.Marshal(valid)
	if err != nil || bytes.Contains(encoded, []byte("reviewed")) {
		t.Fatalf("candidate leaked resource content: %s error=%v", encoded, err)
	}
}

func resourceFixture(id, mediaType string, content []byte) struct {
	spec    SkillResourceSpec
	content SkillResourceContent
} {
	digest := skillContentDigest(content)
	return struct {
		spec    SkillResourceSpec
		content SkillResourceContent
	}{
		spec: SkillResourceSpec{
			ID: id, MediaType: mediaType, ContentRef: "artifact:" + digest,
			ContentDigest: digest, SizeBytes: int64(len(content)),
		},
		content: SkillResourceContent{ID: id, Content: append([]byte(nil), content...)},
	}
}

func resourceSkillCandidate(id, version string, resources ...struct {
	spec    SkillResourceSpec
	content SkillResourceContent
}) SkillCandidate {
	candidate := skillCandidate(id, version)
	for _, resource := range resources {
		candidate.Spec.Resources = append(candidate.Spec.Resources, resource.spec)
		candidate.ResourceContents = append(candidate.ResourceContents, resource.content)
	}
	return candidate
}
