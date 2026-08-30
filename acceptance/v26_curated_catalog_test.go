package acceptance_test

import (
	"bytes"
	"crypto/ed25519"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS12CuratedCatalogNeverResolvesUnselectedOrCrossScopeSkill(t *testing.T) {
	tools, err := tooling.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	one := v26GovernedSkill("review.governed", "1")
	two := v26GovernedSkill("review.governed", "2")
	skillRegistry, err := tooling.NewSkillRegistry(tools, two, one)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x79}, ed25519.SeedSize))
	root := tooling.SkillTrustRoot{
		SignerRef: "signer:v26:curation:1",
		PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
	releaseOne := v26SignedRelease(t, skillRegistry, "1", root.SignerRef, privateKey)
	releaseTwo := v26SignedRelease(t, skillRegistry, "2", root.SignerRef, privateKey)
	releases, err := tooling.NewSkillReleaseCatalog(skillRegistry, []tooling.SkillTrustRoot{root},
		[]tooling.SkillReleaseCandidate{releaseTwo, releaseOne}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reviewedOne := releases.List()[0]
	pluginSpec := tooling.PluginSpec{
		ID: "review.bundle", Version: "1", DescriptionKey: "plugin.review.bundle.description",
		Connectors: []tooling.PluginConnectorSpec{}, Tools: []tooling.PluginToolRequirement{},
		Skills: []tooling.PluginSkillRequirement{{
			ID: reviewedOne.Registration.Spec.ID, Version: reviewedOne.Registration.Spec.Version,
			RegistrationDigest: reviewedOne.Registration.Digest, ReleaseDigest: reviewedOne.ReleaseDigest,
		}},
		Permissions: []identity.Permission{},
	}
	plugins, err := tooling.NewPluginCatalog(tools, releases, pluginSpec)
	if err != nil {
		t.Fatal(err)
	}
	plugin, _ := plugins.Lookup("review.bundle", "1")
	curated, err := tooling.NewCuratedCatalog(tools, releases, plugins, tooling.CuratedCatalogSpec{
		ID: "default.tooling", Version: "1", DescriptionKey: "curation.default.tooling.description",
		ReviewRef: "review:v26:curation:1", ReviewDigest: "sha256:" + strings.Repeat("e", 64),
		Capabilities: []string{"TLS-12"}, Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeGlobal}},
		Tools: []tooling.CuratedToolSelection{},
		Skills: []tooling.CuratedSkillSelection{{
			ID: reviewedOne.Registration.Spec.ID, Version: reviewedOne.Registration.Spec.Version,
			RegistrationDigest: reviewedOne.Registration.Digest, ReleaseDigest: reviewedOne.ReleaseDigest,
		}},
		Plugins: []tooling.CuratedPluginSelection{{
			ID: plugin.Spec.ID, Version: plugin.Spec.Version, PluginDigest: plugin.Digest,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	registration := curated.Registration()
	if registration.Spec.ID != "default.tooling" || registration.Spec.Version != "1" ||
		registration.Spec.ReviewRef != "review:v26:curation:1" ||
		registration.Spec.ReviewDigest != "sha256:"+strings.Repeat("e", 64) ||
		registration.Digest != curated.Digest() || !strings.HasPrefix(registration.Digest, "sha256:") ||
		len(registration.Spec.Capabilities) != 1 || len(registration.Spec.Scopes) != 1 ||
		len(registration.Spec.Skills) != 1 || len(registration.Spec.Plugins) != 1 {
		t.Fatalf("unexpected curated snapshot: %+v", registration)
	}
	capability, _ := goal.NewCapabilityRef("TLS-12")
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	alphaScope, _ := tooling.NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goal.GoalRef{})
	alpha, _ := tooling.NewCuratedContext(capability, alphaScope)
	if _, found, err := curated.ResolveSkill(alpha, "review.governed", "1"); err != nil || !found {
		t.Fatalf("selected skill found=%v error=%v", found, err)
	}
	if _, found, err := curated.ResolvePlugin(alpha, "review.bundle", "1"); err != nil || !found {
		t.Fatalf("selected plugin found=%v error=%v", found, err)
	}
	if value, found, err := curated.ResolveSkill(alpha, "review.governed", "2"); err != nil || found || value.Registration.Spec.ID != "" {
		t.Fatalf("unselected version=%+v found=%v error=%v", value, found, err)
	}
	projectBeta, _ := goal.NewProjectRef("project:beta")
	betaScope, _ := tooling.NewSkillScopeContext("orquesta", projectBeta, identity.RoleReviewer, goal.GoalRef{})
	beta, _ := tooling.NewCuratedContext(capability, betaScope)
	for _, resolve := range []func() (bool, error){
		func() (bool, error) {
			_, found, err := curated.ResolveSkill(beta, "review.governed", "1")
			return found, err
		},
		func() (bool, error) {
			_, found, err := curated.ResolvePlugin(beta, "review.bundle", "1")
			return found, err
		},
	} {
		if found, err := resolve(); err != nil || found {
			t.Fatalf("cross-project selection found=%v error=%v", found, err)
		}
	}
}
