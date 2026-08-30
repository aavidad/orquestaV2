package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

type v26SkillToolCatalog map[string]tooling.SkillToolRegistration

func (catalog v26SkillToolCatalog) LookupSkillTool(id, version string) (tooling.SkillToolRegistration, bool) {
	registration, found := catalog[id+"@"+version]
	return registration, found
}

func TestV26TLS06SkillRegistryKeepsSKILLMarkdownInCASAndMetadataInCatalog(t *testing.T) {
	tool := tooling.SkillToolRegistration{
		ID: "status.read", Version: "1", SpecDigest: "sha256:" + string(bytes.Repeat([]byte{'a'}, 64)),
		Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}
	tools := v26SkillToolCatalog{"status.read@1": tool}
	instructions := []byte("---\nname: review.status\ndescription: Review one status result\n---\n\n# Review status\n\nbody-must-load-only-on-demand\n")
	digestBytes := sha256.Sum256(instructions)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	registry, err := tooling.NewSkillRegistry(tools, tooling.SkillCandidate{
		Spec: tooling.SkillSpec{
			ID: "review.status", Version: "1", Format: tooling.SkillFormatMarkdownV1,
			DescriptionKey:  "skill.review.status.description",
			InstructionsRef: "artifact:" + digest, InstructionsDigest: digest,
			Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeGlobal}},
			RequiredTools: []tooling.SkillToolRequirement{{
				ID: tool.ID, Version: tool.Version, SpecDigest: tool.SpecDigest,
			}},
			Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		},
		Instructions: instructions,
	})
	if err != nil {
		t.Fatal(err)
	}
	listed := registry.List()
	encoded, err := json.Marshal(listed)
	if err != nil || len(listed) != 1 || listed[0].Spec.RequiredTools[0].SpecDigest != tool.SpecDigest ||
		bytes.Contains(encoded, []byte("body-must-load-only-on-demand")) {
		t.Fatalf("listed=%+v encoded=%s error=%v", listed, encoded, err)
	}
	if _, found := reflect.TypeOf(tooling.SkillSpec{}).FieldByName("Instructions"); found {
		t.Fatal("catalog metadata exposes instruction body")
	}
	if _, found := reflect.TypeOf(tooling.SkillSpec{}).FieldByName("Active"); found {
		t.Fatal("descriptive registry became an activation authority")
	}
	request, err := registry.NewLoadRequest("review.status", "1")
	if err != nil || request.InstructionsRef != "artifact:"+digest || request.RegistrationDigest != listed[0].Digest {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	loaded, err := registry.ValidateLoadedInstructions(request, instructions)
	if err != nil || !bytes.Equal(loaded, instructions) {
		t.Fatalf("loaded=%q error=%v", loaded, err)
	}
	mutated := append([]byte(nil), instructions...)
	mutated[len(mutated)-2] = 'x'
	if _, err := registry.ValidateLoadedInstructions(request, mutated); tooling.SkillErrorCode(err) != tooling.ErrorSkillContentInvalid {
		t.Fatalf("mutated error=%v code=%q", err, tooling.SkillErrorCode(err))
	}
}
