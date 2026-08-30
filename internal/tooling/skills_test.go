package tooling

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/identity"
)

func TestSkillRegistryKeepsOnlyCanonicalMetadataAndExactToolRequirements(t *testing.T) {
	tools := skillTestTools(t)
	status, _ := tools.LookupSkillTool("status.read", "1")
	older := skillCandidate("review.status", "1", status)
	newer := skillCandidate("review.status", "2", status)
	document := skillCandidate("docs.explain", "1")
	registry, err := NewSkillRegistry(tools, newer, document, older)
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := NewSkillRegistry(tools, older, newer, document)
	if err != nil || registry.Digest() != reordered.Digest() || !strings.HasPrefix(registry.Digest(), "sha256:") {
		t.Fatalf("registry=%q reordered=%q error=%v", registry.Digest(), reordered.Digest(), err)
	}
	listed := registry.List()
	want := []string{"docs.explain@1", "review.status@1", "review.status@2"}
	got := make([]string, len(listed))
	for index, registration := range listed {
		got[index] = registration.Spec.ID + "@" + registration.Spec.Version
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order=%v", got)
	}
	registration, found := registry.Lookup("review.status", "1")
	if !found || len(registration.Spec.RequiredTools) != 1 ||
		registration.Spec.RequiredTools[0].SpecDigest != status.SpecDigest ||
		!reflect.DeepEqual(registration.Spec.Permissions, []identity.Permission{identity.PermissionArtifactsRead}) {
		t.Fatalf("registration=%+v found=%v", registration, found)
	}
	encoded, err := json.Marshal(listed)
	if err != nil || bytes.Contains(encoded, []byte("never-retain-this-body")) {
		t.Fatalf("metadata=%s error=%v", encoded, err)
	}
	encodedCandidate, err := json.Marshal(older)
	if err != nil || bytes.Contains(encodedCandidate, []byte("never-retain-this-body")) {
		t.Fatalf("candidate serialization leaked instructions: %s error=%v", encodedCandidate, err)
	}
	if _, retained := reflect.TypeOf(SkillSpec{}).FieldByName("Instructions"); retained {
		t.Fatal("skill metadata retains instruction bytes")
	}
	older.Instructions[0] = 'x'
	older.Spec.Scopes[0].Kind = SkillScopeRole
	older.Spec.RequiredTools[0].ID = "mutated.tool"
	registration.Spec.Scopes[0].Kind = SkillScopeRole
	registration.Spec.RequiredTools[0].ID = "mutated.tool"
	registration.Spec.Permissions[0] = identity.Permission("mutated")
	again, _ := registry.Lookup("review.status", "1")
	if again.Spec.Scopes[0].Kind != SkillScopeGlobal || again.Spec.RequiredTools[0].ID != "status.read" ||
		again.Spec.Permissions[0] != identity.PermissionArtifactsRead {
		t.Fatal("caller mutation changed registry state")
	}
	extendedTools := skillToolCatalog{entries: map[string]SkillToolRegistration{
		skillRegistryKey(status.ID, status.Version): status,
		skillRegistryKey("workspace.read", "1"): {
			ID: "workspace.read", Version: "1", SpecDigest: repeatedDigest("1"),
			Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		},
	}}
	extended, err := NewSkillRegistry(extendedTools, newer, document, skillCandidate("review.status", "1", status))
	if err != nil || extended.Digest() != registry.Digest() {
		t.Fatalf("unrelated tool changed skill catalog: %q/%q error=%v", extended.Digest(), registry.Digest(), err)
	}
}

func TestSkillRegistryRejectsInvalidCandidatesAtomically(t *testing.T) {
	tools := skillTestTools(t)
	status, _ := tools.LookupSkillTool("status.read", "1")
	valid := skillCandidate("review.status", "1", status)
	tests := map[string]func(*SkillCandidate){
		"id":              func(value *SkillCandidate) { value.Spec.ID = "review" },
		"version":         func(value *SkillCandidate) { value.Spec.Version = "01" },
		"format":          func(value *SkillCandidate) { value.Spec.Format = "markdown" },
		"description key": func(value *SkillCandidate) { value.Spec.DescriptionKey = "skill.other.description" },
		"artifact ref":    func(value *SkillCandidate) { value.Spec.InstructionsRef = "../SKILL.md" },
		"digest":          func(value *SkillCandidate) { value.Spec.InstructionsDigest = "sha256:short" },
		"content":         func(value *SkillCandidate) { value.Instructions = append(value.Instructions, 'x') },
		"missing scope":   func(value *SkillCandidate) { value.Spec.Scopes = nil },
		"invalid scope": func(value *SkillCandidate) {
			value.Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: ""}}
		},
		"frontmatter": func(value *SkillCandidate) { value.Instructions[0] = '#' },
		"unknown metadata": func(value *SkillCandidate) {
			value.Instructions = bytes.Replace(value.Instructions,
				[]byte("description: "), []byte("unknown: value\ndescription: "), 1)
			value.Spec.InstructionsDigest = skillContentDigest(value.Instructions)
			value.Spec.InstructionsRef = "artifact:" + value.Spec.InstructionsDigest
		},
		"missing tool": func(value *SkillCandidate) {
			value.Spec.RequiredTools[0].ID = "missing.read"
		},
		"tool digest": func(value *SkillCandidate) {
			value.Spec.RequiredTools[0].SpecDigest = "sha256:" + strings.Repeat("0", 64)
		},
		"duplicate tool": func(value *SkillCandidate) {
			value.Spec.RequiredTools = append(value.Spec.RequiredTools, value.Spec.RequiredTools[0])
		},
		"missing permission": func(value *SkillCandidate) { value.Spec.Permissions = nil },
		"extra permission": func(value *SkillCandidate) {
			value.Spec.Permissions = append(value.Spec.Permissions, identity.PermissionChangesIntegrate)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := cloneSkillCandidate(valid)
			mutate(&candidate)
			registry, err := NewSkillRegistry(tools, candidate)
			if registry != nil || (SkillErrorCode(err) != ErrorSkillSpecInvalid && SkillErrorCode(err) != ErrorSkillContentInvalid) {
				t.Fatalf("registry=%v error=%v code=%q", registry, err, SkillErrorCode(err))
			}
		})
	}
	oversized := cloneSkillCandidate(valid)
	oversized.Instructions = bytes.Repeat([]byte{'x'}, MaxSkillInstructionsBytes+1)
	oversized.Spec.InstructionsDigest = skillContentDigest(oversized.Instructions)
	oversized.Spec.InstructionsRef = "artifact:" + oversized.Spec.InstructionsDigest
	if registry, err := NewSkillRegistry(tools, oversized); registry != nil || SkillErrorCode(err) != ErrorSkillContentInvalid {
		t.Fatalf("oversized registry=%v error=%v", registry, err)
	}
	if registry, err := NewSkillRegistry(nil, valid); registry != nil || SkillErrorCode(err) != ErrorSkillSpecInvalid {
		t.Fatalf("nil tools registry=%v error=%v", registry, err)
	}
	if registry, err := NewSkillRegistry(tools, valid, valid); registry != nil ||
		SkillErrorCode(err) != ErrorSkillSpecDuplicate {
		t.Fatalf("duplicate registry=%v error=%v", registry, err)
	}
	empty, err := NewSkillRegistry(tools)
	if err != nil || empty.Digest() == "" || empty.List() == nil || len(empty.List()) != 0 {
		t.Fatalf("empty=%v digest=%q list=%+v error=%v", empty, empty.Digest(), empty.List(), err)
	}
}

func TestSkillRegistryProgressiveLoadRejectsEveryDivergence(t *testing.T) {
	tools := skillTestTools(t)
	status, _ := tools.LookupSkillTool("status.read", "1")
	candidate := skillCandidate("review.status", "1", status)
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	request, err := registry.NewLoadRequest("review.status", "1")
	if err != nil || request.InstructionsRef != candidate.Spec.InstructionsRef ||
		request.InstructionsDigest != candidate.Spec.InstructionsDigest || request.MaxBytes != MaxSkillInstructionsBytes {
		t.Fatalf("request=%+v error=%v", request, err)
	}
	loaded, err := registry.ValidateLoadedInstructions(request, candidate.Instructions)
	if err != nil || !bytes.Equal(loaded, candidate.Instructions) {
		t.Fatalf("loaded=%q error=%v", loaded, err)
	}
	loaded[0] = 'x'
	if candidate.Instructions[0] != '-' {
		t.Fatal("loaded instructions alias caller bytes")
	}
	mutatedContent := append([]byte(nil), candidate.Instructions...)
	mutatedContent[len(mutatedContent)-1] = 'x'
	if got, err := registry.ValidateLoadedInstructions(request, mutatedContent); got != nil ||
		SkillErrorCode(err) != ErrorSkillContentInvalid {
		t.Fatalf("mutated content=%q error=%v", got, err)
	}
	requestMutations := []func(*SkillLoadRequest){
		func(value *SkillLoadRequest) { value.ID = "other.skill" },
		func(value *SkillLoadRequest) { value.Version = "2" },
		func(value *SkillLoadRequest) { value.RegistrationDigest = "sha256:" + strings.Repeat("0", 64) },
		func(value *SkillLoadRequest) { value.InstructionsRef = "artifact:sha256:" + strings.Repeat("0", 64) },
		func(value *SkillLoadRequest) { value.InstructionsDigest = "sha256:" + strings.Repeat("0", 64) },
		func(value *SkillLoadRequest) { value.MaxBytes-- },
	}
	for _, mutate := range requestMutations {
		forged := request
		mutate(&forged)
		if got, err := registry.ValidateLoadedInstructions(forged, candidate.Instructions); got != nil ||
			SkillErrorCode(err) != ErrorSkillLoadRequestInvalid {
			t.Fatalf("forged=%+v got=%q error=%v code=%q", forged, got, err, SkillErrorCode(err))
		}
	}
	if request, err := registry.NewLoadRequest("review.status", "2"); !reflect.DeepEqual(request, SkillLoadRequest{}) ||
		SkillErrorCode(err) != ErrorSkillNotFound {
		t.Fatalf("missing request=%+v error=%v", request, err)
	}
	if nilRegistry := (*SkillRegistry)(nil); nilRegistry.List() != nil || nilRegistry.Digest() != "" {
		t.Fatal("nil skill registry should be inert")
	}
}

func skillTestTools(t *testing.T) skillToolCatalog {
	t.Helper()
	status := SkillToolRegistration{
		ID: "status.read", Version: "1", SpecDigest: repeatedDigest("a"),
		Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}
	return skillToolCatalog{entries: map[string]SkillToolRegistration{skillRegistryKey(status.ID, status.Version): status}}
}

type skillToolCatalog struct {
	entries map[string]SkillToolRegistration
}

func (catalog skillToolCatalog) LookupSkillTool(id, version string) (SkillToolRegistration, bool) {
	registration, found := catalog.entries[skillRegistryKey(id, version)]
	registration.Permissions = append([]identity.Permission(nil), registration.Permissions...)
	return registration, found
}

func skillCandidate(id, version string, tools ...SkillToolRegistration) SkillCandidate {
	instructions := []byte("---\nname: " + id + "\ndescription: Reviewed skill metadata\n---\n\n# Skill instructions\n\nnever-retain-this-body\n")
	digest := skillContentDigest(instructions)
	requirements := make([]SkillToolRequirement, len(tools))
	permissionSet := make(map[identity.Permission]struct{})
	for index, tool := range tools {
		requirements[index] = SkillToolRequirement{
			ID: tool.ID, Version: tool.Version, SpecDigest: tool.SpecDigest,
		}
		for _, permission := range tool.Permissions {
			permissionSet[permission] = struct{}{}
		}
	}
	permissions := make([]identity.Permission, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	return SkillCandidate{
		Spec: SkillSpec{
			ID: id, Version: version, Format: SkillFormatMarkdownV1,
			DescriptionKey:  "skill." + id + ".description",
			InstructionsRef: "artifact:" + digest, InstructionsDigest: digest,
			Scopes:        []SkillScope{{Kind: SkillScopeGlobal}},
			RequiredTools: requirements, Permissions: permissions,
		},
		Instructions: instructions,
	}
}

func cloneSkillCandidate(source SkillCandidate) SkillCandidate {
	source.Spec.Scopes = append([]SkillScope(nil), source.Spec.Scopes...)
	source.Spec.Resources = append([]SkillResourceSpec(nil), source.Spec.Resources...)
	source.Spec.RequiredTools = append([]SkillToolRequirement(nil), source.Spec.RequiredTools...)
	source.Spec.Permissions = append([]identity.Permission(nil), source.Spec.Permissions...)
	source.Instructions = append([]byte(nil), source.Instructions...)
	source.ResourceContents = append([]SkillResourceContent(nil), source.ResourceContents...)
	for index := range source.ResourceContents {
		source.ResourceContents[index].Content = append([]byte(nil), source.ResourceContents[index].Content...)
	}
	return source
}
