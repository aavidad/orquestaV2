package orquestaautoprogramming

import (
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestFrozenRequiredTestContextRefV0NormalizaYParseaV0(t *testing.T) {
	ref, ok := FrozenRequiredTestContextRefV0(FrozenRequiredTestV0{
		Path:   "modulos/example/../example/frozen_actor_test.go",
		SHA256: "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	})
	if !ok {
		t.Fatalf("context ref no construido")
	}
	parsed, ok := ParseFrozenRequiredTestContextRefV0(ref)
	if !ok {
		t.Fatalf("context ref no parseado: %+v", ref)
	}
	if parsed.Path != "modulos/example/frozen_actor_test.go" ||
		parsed.SHA256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("parsed=%+v", parsed)
	}
}

func TestFrozenRequiredTestContextRefV0RechazaRutasNoCongelablesV0(t *testing.T) {
	invalid := []string{
		"../modulos/example/frozen_actor_test.go",
		"/tmp/frozen_actor_test.go",
		"modulos/example/frozen_actor.go",
	}
	for _, path := range invalid {
		if _, ok := FrozenRequiredTestContextRefV0(FrozenRequiredTestV0{
			Path:   path,
			SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}); ok {
			t.Fatalf("path invalida aceptada: %s", path)
		}
	}
}

func TestGoalHasFrozenRequiredTestsPhaseV0(t *testing.T) {
	spec := orquestagoal.GoalWorkSpecV0{ContextRefs: []orquestagoal.GoalContextRefV0{
		FrozenRequiredTestsPhaseContextRefV0(FrozenRequiredTestsImplementerPhaseV0),
		FrozenRequiredTestsBaseRequestContextRefV0("request-ref-base"),
	}}
	if !GoalHasFrozenRequiredTestsPhaseV0(spec, FrozenRequiredTestsImplementerPhaseV0) {
		t.Fatalf("phase implementer no detectada")
	}
	if got := GoalFrozenRequiredTestsBaseRequestRefV0(spec); got != "request-ref-base" {
		t.Fatalf("base=%q", got)
	}
}
