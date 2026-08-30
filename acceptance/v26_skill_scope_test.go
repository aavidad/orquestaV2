package acceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS08SkillScopesAreExplicitAndProjectGoalIsolated(t *testing.T) {
	tools := v26SkillToolCatalog{}
	globalSkill := v26ScopedSkill("review.global", tooling.SkillScope{Kind: tooling.SkillScopeGlobal})
	productSkill := v26ScopedSkill("review.product", tooling.SkillScope{
		Kind: tooling.SkillScopeProduct, ProductID: "orquesta",
	})
	projectSkill := v26ScopedSkill("review.project", tooling.SkillScope{
		Kind: tooling.SkillScopeProject, ProjectRef: "project:alpha",
	})
	goalSkill := v26ScopedSkill("review.goal", tooling.SkillScope{
		Kind: tooling.SkillScopeGoal, ProjectRef: "project:alpha", GoalRef: "goal:alpha",
	})
	roleSkill := v26ScopedSkill("review.role", tooling.SkillScope{
		Kind: tooling.SkillScopeRole, Role: identity.RoleReviewer,
	})
	registry, err := tooling.NewSkillRegistry(tools, globalSkill, productSkill, projectSkill, goalSkill, roleSkill)
	if err != nil {
		t.Fatal(err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	goalAlpha, _ := goal.NewGoalRef("goal:alpha")
	allowed, err := tooling.NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goalAlpha)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"review.global", "review.product", "review.project", "review.goal", "review.role"} {
		if _, found, err := registry.ResolveScoped(id, "1", allowed); err != nil || !found {
			t.Fatalf("allowed scope %q found=%v error=%v", id, found, err)
		}
	}
	projectBeta, _ := goal.NewProjectRef("project:beta")
	goalBeta, _ := goal.NewGoalRef("goal:beta")
	denied, err := tooling.NewSkillScopeContext("orquesta", projectBeta, identity.RoleViewer, goalBeta)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"review.project", "review.goal", "review.role"} {
		if registration, found, err := registry.ResolveScoped(id, "1", denied); err != nil || found || registration.Spec.ID != "" || registration.Digest != "" {
			t.Fatalf("isolated scope %q registration=%+v found=%v error=%v", id, registration, found, err)
		}
	}
	for _, id := range []string{"review.global", "review.product"} {
		if _, found, err := registry.ResolveScoped(id, "1", denied); err != nil || !found {
			t.Fatalf("broad visibility %q found=%v error=%v", id, found, err)
		}
	}
	otherProduct, err := tooling.NewSkillScopeContext("other", projectBeta, identity.RoleViewer, goalBeta)
	if err != nil {
		t.Fatal(err)
	}
	if registration, found, err := registry.ResolveScoped("review.product", "1", otherProduct); err != nil || found || registration.Spec.ID != "" {
		t.Fatalf("cross-product registration=%+v found=%v error=%v", registration, found, err)
	}
	crossProject, err := tooling.NewSkillScopeContext("orquesta", projectBeta, identity.RoleViewer, goalAlpha)
	if err != nil {
		t.Fatal(err)
	}
	if registration, found, err := registry.ResolveScoped("review.goal", "1", crossProject); err != nil || found || registration.Spec.ID != "" {
		t.Fatalf("cross-project Goal registration=%+v found=%v error=%v", registration, found, err)
	}
	if _, found, err := registry.ResolveScoped("review.project", "1", tooling.SkillScopeContext{}); found || tooling.SkillErrorCode(err) != tooling.ErrorSkillScopeInvalid {
		t.Fatalf("unvalidated context found=%v error=%v", found, err)
	}
	first, found, err := registry.ResolveScoped("review.goal", "1", allowed)
	if err != nil || !found {
		t.Fatalf("first replay=%+v found=%v error=%v", first, found, err)
	}
	first.Spec.Scopes[0].GoalRef = "goal:forged"
	replayed, found, err := registry.ResolveScoped("review.goal", "1", allowed)
	if err != nil || !found || replayed.Spec.Scopes[0].GoalRef != "goal:alpha" {
		t.Fatalf("defensive replay=%+v found=%v error=%v", replayed, found, err)
	}
}

func v26ScopedSkill(id string, scope tooling.SkillScope) tooling.SkillCandidate {
	instructions := []byte("---\nname: " + id + "\ndescription: Scoped reviewed skill\n---\n\n# Scoped instructions\n")
	digestBytes := sha256.Sum256(instructions)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])
	return tooling.SkillCandidate{
		Spec: tooling.SkillSpec{
			ID: id, Version: "1", Format: tooling.SkillFormatMarkdownV1,
			DescriptionKey: "skill." + id + ".description", InstructionsRef: "artifact:" + digest,
			InstructionsDigest: digest, Scopes: []tooling.SkillScope{scope},
			RequiredTools: []tooling.SkillToolRequirement{}, Permissions: []identity.Permission{},
		},
		Instructions: instructions,
	}
}
