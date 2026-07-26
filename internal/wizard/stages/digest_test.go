package stages

import "testing"

func TestBuiltInV1CatalogDigestIsFrozen(t *testing.T) {
	t.Parallel()

	catalog := builtInV1()
	got := catalog.Digest().String()
	if got != frozenV1CatalogDigestValue {
		t.Fatalf("v1 catalog digest drifted: got=%s want=%s", got, frozenV1CatalogDigestValue)
	}
	if catalog.Version() != VersionV1() ||
		VersionV1().String() != v1VersionValue {
		t.Fatal("v1 builder depends on mutable current version")
	}
	resolved, err := BuiltInVersion(VersionV1())
	if err != nil || resolved.Digest() != catalog.Digest() {
		t.Fatalf("resolve v1: catalog=%+v error=%v", resolved, err)
	}
}

func TestTemplateDigestCoversPreviewOnlySemantics(t *testing.T) {
	t.Parallel()

	base, found := builtInV1().Template(mustTemplateRef("template:research"))
	if !found {
		t.Fatal("research template missing")
	}
	baseDigest := base.Digest()
	tests := []struct {
		name   string
		mutate func(*Template)
	}{
		{
			name: "roadmap",
			mutate: func(value *Template) {
				value.roadmapCapabilityRefs = append(
					value.roadmapCapabilityRefs,
					mustRoadmapCapabilityRef("WIZ-25"),
				)
				sortRoadmapCapabilityRefs(value.roadmapCapabilityRefs)
			},
		},
		{
			name: "effect",
			mutate: func(value *Template) {
				value.units[0].effects[0].Kind = EffectWriteArtifact
			},
		},
		{
			name: "policy",
			mutate: func(value *Template) {
				value.units[0].policy.Effort = EffortDeep
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			changed := cloneTemplates([]Template{base})[0]
			test.mutate(&changed)
			if changed.Digest() == baseDigest {
				t.Fatal("semantic change did not change template digest")
			}
		})
	}
}

func TestBuiltInVersionAndDigestFailClosed(t *testing.T) {
	t.Parallel()

	unknown := mustCatalogVersion("orquesta.wizard.stages.v2")
	if _, err := BuiltInVersion(unknown); ErrorCodeOf(err) !=
		ErrorCatalogVersionUnknown {
		t.Fatalf("unknown version error=%v", err)
	}
	for _, value := range []string{
		"", "ABC", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	} {
		if _, err := NewDigest(value); ErrorCodeOf(err) != ErrorInvalidDigest {
			t.Fatalf("digest %q error=%v", value, err)
		}
	}
}
