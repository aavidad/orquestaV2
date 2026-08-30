package tooling

import (
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestSkillScopesAreCanonicalExplicitAndPartOfRegistrationDigest(t *testing.T) {
	tools := skillTestTools(t)
	scopes := []SkillScope{
		{Kind: SkillScopeRole, Role: identity.RoleReviewer},
		{Kind: SkillScopeProject, ProjectRef: "project:alpha"},
		{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"},
		{Kind: SkillScopeProduct, ProductID: "orquesta"},
	}
	candidate := skillCandidate("review.status", "1")
	candidate.Spec.Scopes = scopes
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	reorderedCandidate := skillCandidate("review.status", "1")
	reorderedCandidate.Spec.Scopes = []SkillScope{scopes[3], scopes[1], scopes[0], scopes[2]}
	reordered, err := NewSkillRegistry(tools, reorderedCandidate)
	if err != nil || registry.Digest() != reordered.Digest() {
		t.Fatalf("digest=%q reordered=%q error=%v", registry.Digest(), reordered.Digest(), err)
	}
	registration, found := registry.Lookup("review.status", "1")
	want := []SkillScope{
		{Kind: SkillScopeProduct, ProductID: "orquesta"},
		{Kind: SkillScopeProject, ProjectRef: "project:alpha"},
		{Kind: SkillScopeRole, Role: identity.RoleReviewer},
		{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"},
	}
	if !found || !reflect.DeepEqual(registration.Spec.Scopes, want) {
		t.Fatalf("registration=%+v found=%v", registration, found)
	}
	global := skillCandidate("review.status", "1")
	globalRegistry, err := NewSkillRegistry(tools, global)
	if err != nil || globalRegistry.Digest() == registry.Digest() {
		t.Fatalf("global digest=%q scoped=%q error=%v", globalRegistry.Digest(), registry.Digest(), err)
	}
}

func TestResolveScopedMatchesEveryScopeWithoutInferringOrGranting(t *testing.T) {
	tools := skillTestTools(t)
	candidates := []SkillCandidate{
		scopedSkillCandidate("scope.global", SkillScope{Kind: SkillScopeGlobal}),
		scopedSkillCandidate("scope.product", SkillScope{Kind: SkillScopeProduct, ProductID: "orquesta"}),
		scopedSkillCandidate("scope.project", SkillScope{Kind: SkillScopeProject, ProjectRef: "project:alpha"}),
		scopedSkillCandidate("scope.role", SkillScope{Kind: SkillScopeRole, Role: identity.RoleReviewer}),
		scopedSkillCandidate("scope.goal", SkillScope{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"}),
	}
	registry, err := NewSkillRegistry(tools, candidates...)
	if err != nil {
		t.Fatal(err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	goalAlpha, _ := goal.NewGoalRef("goal:alpha")
	full, err := NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goalAlpha)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"scope.global", "scope.product", "scope.project", "scope.role", "scope.goal"} {
		registration, found, err := registry.ResolveScoped(id, "1", full)
		if err != nil || !found || registration.Spec.ID != id || len(registration.Spec.Permissions) != 0 {
			t.Fatalf("id=%q registration=%+v found=%v error=%v", id, registration, found, err)
		}
	}

	empty, err := NewSkillScopeContext("", goal.ProjectRef{}, "", goal.GoalRef{})
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := registry.ResolveScoped("scope.global", "1", empty); err != nil || !found {
		t.Fatalf("global found=%v error=%v", found, err)
	}
	for _, id := range []string{"scope.product", "scope.project", "scope.role", "scope.goal"} {
		if registration, found, err := registry.ResolveScoped(id, "1", empty); err != nil || found || registration.Spec.ID != "" || registration.Digest != "" {
			t.Fatalf("empty context resolved %q: registration=%+v found=%v error=%v", id, registration, found, err)
		}
	}
	if registration, found, err := registry.ResolveScoped("scope.project", "2", full); err != nil || found || registration.Spec.ID != "" || registration.Digest != "" {
		t.Fatalf("version fallback registration=%+v found=%v error=%v", registration, found, err)
	}
}

func TestResolveScopedRejectsCrossProjectRoleGoalAndInvalidContext(t *testing.T) {
	tools := skillTestTools(t)
	registry, err := NewSkillRegistry(tools,
		scopedSkillCandidate("scope.project", SkillScope{Kind: SkillScopeProject, ProjectRef: "project:alpha"}),
		scopedSkillCandidate("scope.role", SkillScope{Kind: SkillScopeRole, Role: identity.RoleReviewer}),
		scopedSkillCandidate("scope.goal", SkillScope{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	projectBeta, _ := goal.NewProjectRef("project:beta")
	goalBeta, _ := goal.NewGoalRef("goal:beta")
	other, err := NewSkillScopeContext("other", projectBeta, identity.RoleViewer, goalBeta)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"scope.project", "scope.role", "scope.goal"} {
		if registration, found, err := registry.ResolveScoped(id, "1", other); err != nil || found || registration.Spec.ID != "" || registration.Digest != "" {
			t.Fatalf("cross-scope resolved %q: registration=%+v found=%v error=%v", id, registration, found, err)
		}
	}
	if _, found, err := registry.ResolveScoped("scope.project", "1", SkillScopeContext{}); found || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("zero context found=%v error=%v code=%q", found, err, SkillErrorCode(err))
	}
	if context, err := NewSkillScopeContext(" product ", goal.ProjectRef{}, "", goal.GoalRef{}); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("invalid product context=%+v error=%v", context, err)
	}
	if context, err := NewSkillScopeContext("", goal.ProjectRef{}, identity.Role("owner"), goal.GoalRef{}); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("invalid role context=%+v error=%v", context, err)
	}
	if context, err := NewSkillScopeContext("", goal.ProjectRef{}, identity.RoleReviewer, goal.GoalRef{}); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("unbound role context=%+v error=%v", context, err)
	}
	goalAlpha, _ := goal.NewGoalRef("goal:alpha")
	if context, err := NewSkillScopeContext("", goal.ProjectRef{}, "", goalAlpha); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("unbound goal context=%+v error=%v", context, err)
	}
	invalidProject, _ := goal.NewProjectRef("project:\xff")
	if context, err := NewSkillScopeContext("", invalidProject, "", goal.GoalRef{}); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("invalid UTF-8 project context=%+v error=%v", context, err)
	}
	oversizedProject, _ := goal.NewProjectRef(strings.Repeat("p", MaxSkillScopeValueBytes+1))
	if context, err := NewSkillScopeContext("", oversizedProject, "", goal.GoalRef{}); context != (SkillScopeContext{}) || SkillErrorCode(err) != ErrorSkillScopeInvalid {
		t.Fatalf("oversized project context=%+v error=%v", context, err)
	}
}

func TestResolveScopedReturnsDefensiveCopyAndExactReplay(t *testing.T) {
	tools := skillTestTools(t)
	status, _ := tools.LookupSkillTool("status.read", "1")
	candidate := skillCandidate("scope.replay", "1", status)
	candidate.Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}}
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil {
		t.Fatal(err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	context, _ := NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goal.GoalRef{})
	want, found, err := registry.ResolveScoped("scope.replay", "1", context)
	if err != nil || !found || len(want.Spec.Permissions) != 1 {
		t.Fatalf("registration=%+v found=%v error=%v", want, found, err)
	}
	mutated, _, _ := registry.ResolveScoped("scope.replay", "1", context)
	mutated.Spec.Scopes[0].ProjectRef = "project:forged"
	mutated.Spec.RequiredTools[0].ID = "forged.read"
	mutated.Spec.Permissions[0] = identity.Permission("forged")
	replayed, found, err := registry.ResolveScoped("scope.replay", "1", context)
	if err != nil || !found || !reflect.DeepEqual(replayed, want) || registry.Digest() == "" {
		t.Fatalf("replayed=%+v want=%+v found=%v error=%v", replayed, want, found, err)
	}
	caseVariant, _ := goal.NewProjectRef("project:Alpha")
	caseContext, _ := NewSkillScopeContext("orquesta", caseVariant, identity.RoleReviewer, goal.GoalRef{})
	if registration, found, err := registry.ResolveScoped("scope.replay", "1", caseContext); err != nil || found || registration.Spec.ID != "" || registration.Digest != "" {
		t.Fatalf("case variant registration=%+v found=%v error=%v", registration, found, err)
	}
}

func TestSkillRegistryRejectsAmbiguousOrMalformedScopesAtomically(t *testing.T) {
	tools := skillTestTools(t)
	tests := map[string][]SkillScope{
		"empty":         nil,
		"unknown":       {{Kind: "organization", ProductID: "orquesta"}},
		"global mixed":  {{Kind: SkillScopeGlobal}, {Kind: SkillScopeRole, Role: identity.RoleReviewer}},
		"global field":  {{Kind: SkillScopeGlobal, ProductID: "orquesta"}},
		"product empty": {{Kind: SkillScopeProduct}},
		"product extra": {{Kind: SkillScopeProduct, ProductID: "orquesta", ProjectRef: "project:alpha"}},
		"project bad":   {{Kind: SkillScopeProject, ProjectRef: " project:alpha"}},
		"project extra": {{Kind: SkillScopeProject, ProjectRef: "project:alpha", GoalRef: "goal:alpha"}},
		"role bad":      {{Kind: SkillScopeRole, Role: "owner"}},
		"role extra":    {{Kind: SkillScopeRole, Role: identity.RoleReviewer, ProductID: "orquesta"}},
		"goal unbound":  {{Kind: SkillScopeGoal, GoalRef: "goal:alpha"}},
		"goal bad":      {{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha\x00forged"}},
		"product utf8":  {{Kind: SkillScopeProduct, ProductID: "product:\xff"}},
		"product ctrl":  {{Kind: SkillScopeProduct, ProductID: "product:\nalpha"}},
		"project utf8":  {{Kind: SkillScopeProject, ProjectRef: "project:\xff"}},
		"goal utf8":     {{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:\xff"}},
		"duplicate":     {{Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"}, {Kind: SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha"}},
	}
	tooMany := make([]SkillScope, MaxSkillScopes+1)
	for index := range tooMany {
		tooMany[index] = SkillScope{Kind: SkillScopeProduct, ProductID: "product:" + strings.Repeat("x", index+1)}
	}
	tests["scope limit"] = tooMany
	for name, scopes := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := skillCandidate("scope.invalid", "1")
			candidate.Spec.Scopes = scopes
			registry, err := NewSkillRegistry(tools, candidate)
			if registry != nil || SkillErrorCode(err) != ErrorSkillSpecInvalid {
				t.Fatalf("registry=%v error=%v code=%q", registry, err, SkillErrorCode(err))
			}
		})
	}
}

func TestSkillScopeLimitsAcceptExactBoundaryAndRejectNextByte(t *testing.T) {
	tools := skillTestTools(t)
	boundary := strings.Repeat("p", MaxSkillScopeValueBytes)
	candidate := skillCandidate("scope.boundary", "1")
	candidate.Spec.Scopes = []SkillScope{{Kind: SkillScopeProduct, ProductID: boundary}}
	registry, err := NewSkillRegistry(tools, candidate)
	if err != nil || registry == nil {
		t.Fatalf("boundary registry=%v error=%v", registry, err)
	}
	tooLong := skillCandidate("scope.too-long", "1")
	tooLong.Spec.Scopes = []SkillScope{{Kind: SkillScopeProduct, ProductID: boundary + "x"}}
	if registry, err := NewSkillRegistry(tools, tooLong); registry != nil || SkillErrorCode(err) != ErrorSkillSpecInvalid {
		t.Fatalf("oversized registry=%v error=%v", registry, err)
	}
}

func scopedSkillCandidate(id string, scope SkillScope) SkillCandidate {
	candidate := skillCandidate(id, "1")
	candidate.Spec.Scopes = []SkillScope{scope}
	return candidate
}
