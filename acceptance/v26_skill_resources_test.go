package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS07SkillResourcesLoadOnlyByExactCASSubject(t *testing.T) {
	tools := v26SkillToolCatalog{}
	instructions := []byte("---\nname: review.resources\ndescription: Progressive skill resources\n---\n\n# Review resources\n")
	instructionsDigest := v26BytesDigest(instructions)
	resource := []byte("# Policy\n\nLoad this resource only when requested.\n")
	resourceDigest := v26BytesDigest(resource)
	fixture := []byte(`{"input":"reviewed"}`)
	fixtureDigest := v26BytesDigest(fixture)
	candidate := tooling.SkillCandidate{
		Spec: tooling.SkillSpec{
			ID: "review.resources", Version: "1", Format: tooling.SkillFormatMarkdownV1,
			DescriptionKey:  "skill.review.resources.description",
			InstructionsRef: "artifact:" + instructionsDigest, InstructionsDigest: instructionsDigest,
			Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeGlobal}},
			Resources: []tooling.SkillResourceSpec{{
				ID: "references.policy", MediaType: "text/markdown",
				ContentRef: "artifact:" + resourceDigest, ContentDigest: resourceDigest,
				SizeBytes: int64(len(resource)),
			}, {
				ID: "fixtures.input", MediaType: "application/json",
				ContentRef: "artifact:" + fixtureDigest, ContentDigest: fixtureDigest,
				SizeBytes: int64(len(fixture)),
			}},
			RequiredTools: []tooling.SkillToolRequirement{}, Permissions: []identity.Permission{},
		},
		Instructions: instructions,
		ResourceContents: []tooling.SkillResourceContent{{
			ID: "references.policy", Content: resource,
		}, {
			ID: "fixtures.input", Content: fixture,
		}},
	}
	registry, err := tooling.NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	listed, found := registry.ListResources("review.resources", "1")
	encoded, encodeErr := json.Marshal(registry.List())
	if !found || len(listed) != 2 || listed[1].ContentRef != "artifact:"+resourceDigest ||
		encodeErr != nil || bytes.Contains(encoded, []byte("Load this resource")) {
		t.Fatalf("listed=%+v metadata=%s found=%v error=%v", listed, encoded, found, encodeErr)
	}
	instructionsRequest, err := registry.NewLoadRequest("review.resources", "1")
	if err != nil {
		t.Fatal(err)
	}
	loadedInstructions, err := registry.ValidateLoadedInstructions(instructionsRequest, instructions)
	if err != nil || !bytes.Equal(loadedInstructions, instructions) || bytes.Contains(loadedInstructions, resource) {
		t.Fatalf("instructions=%q error=%v", loadedInstructions, err)
	}
	pageRequest, err := registry.NewResourcePageRequest("review.resources", "1", "", 1)
	if err != nil || pageRequest.ScopesDigest == "" {
		t.Fatalf("page request=%+v error=%v", pageRequest, err)
	}
	firstPage, err := registry.ListResourcePage(pageRequest)
	if err != nil || firstPage.Complete || len(firstPage.Items) != 1 ||
		firstPage.Items[0].ID != "fixtures.input" || firstPage.NextCursor == "" {
		t.Fatalf("first page=%+v error=%v", firstPage, err)
	}
	nextRequest, err := registry.NewResourcePageRequest("review.resources", "1", firstPage.NextCursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	finalPage, err := registry.ListResourcePage(nextRequest)
	if err != nil || !finalPage.Complete || len(finalPage.Items) != 1 ||
		finalPage.Items[0].ID != "references.policy" {
		t.Fatalf("final page=%+v error=%v", finalPage, err)
	}
	request, err := registry.NewResourceLoadRequest("review.resources", "1", "references.policy")
	if err != nil || request.RegistrationDigest == "" || request.ContentDigest != resourceDigest ||
		request.ScopesDigest == "" || request.SizeBytes != int64(len(resource)) ||
		request.MaxBytes != tooling.MaxSkillResourceBytes {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	loaded, err := registry.ValidateLoadedResource(request, resource)
	if err != nil || !bytes.Equal(loaded, resource) {
		t.Fatalf("loaded=%q error=%v", loaded, err)
	}
	forged := request
	forged.ScopesDigest = "sha256:" + string(bytes.Repeat([]byte{'0'}, 64))
	if content, err := registry.ValidateLoadedResource(forged, resource); content != nil || tooling.SkillErrorCode(err) != tooling.ErrorSkillResourceLoadInvalid {
		t.Fatalf("forged content=%q error=%v", content, err)
	}
	mutated := append([]byte(nil), resource...)
	mutated[len(mutated)-2] = 'x'
	if content, err := registry.ValidateLoadedResource(request, mutated); content != nil || tooling.SkillErrorCode(err) != tooling.ErrorSkillResourceContentInvalid {
		t.Fatalf("mutated content=%q error=%v", content, err)
	}
}

func v26BytesDigest(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}
