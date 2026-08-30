package tooling

import (
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestCuratedCatalogResolvesOnlyExactCapabilityScopeAndReviewedSelections(t *testing.T) {
	tools, releases, plugins, spec := curationFixture(t)
	catalog, err := NewCuratedCatalog(tools, releases, plugins, spec)
	if err != nil {
		t.Fatal(err)
	}
	reordered := cloneCuratedSpec(spec)
	reordered.Capabilities = []string{"TLS-12", "TLS-10"}
	second, err := NewCuratedCatalog(tools, releases, plugins, reordered)
	if err != nil || catalog.Digest() != second.Digest() {
		t.Fatalf("catalog=%q reordered=%q error=%v", catalog.Digest(), second.Digest(), err)
	}
	projectAlpha, _ := goal.NewProjectRef("project:alpha")
	scopeAlpha, _ := NewSkillScopeContext("orquesta", projectAlpha, identity.RoleReviewer, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef("TLS-12")
	context, err := NewCuratedContext(capability, scopeAlpha)
	if err != nil {
		t.Fatal(err)
	}
	tool, toolFound, toolErr := catalog.ResolveTool(context, "status.read", "1")
	skill, skillFound, skillErr := catalog.ResolveSkill(context, "review.status", "1")
	plugin, pluginFound, pluginErr := catalog.ResolvePlugin(context, "forge.review", "1")
	if toolErr != nil || skillErr != nil || pluginErr != nil || !toolFound || !skillFound || !pluginFound ||
		tool.Spec.ID != "status.read" || skill.Registration.Spec.ID != "review.status" ||
		plugin.Spec.ID != "forge.review" {
		t.Fatalf("tool=%+v/%v/%v skill=%+v/%v/%v plugin=%+v/%v/%v",
			tool, toolFound, toolErr, skill, skillFound, skillErr, plugin, pluginFound, pluginErr)
	}
	if value, found, err := catalog.ResolveTool(context, "workspace.read", "1"); err != nil || found || value.Spec.ID != "" {
		t.Fatalf("uncurated tool=%+v found=%v error=%v", value, found, err)
	}
	if value, found, err := catalog.ResolveSkill(context, "review.status", "2"); err != nil || found || value.Registration.Spec.ID != "" {
		t.Fatalf("version fallback skill=%+v found=%v error=%v", value, found, err)
	}

	projectBeta, _ := goal.NewProjectRef("project:beta")
	scopeBeta, _ := NewSkillScopeContext("orquesta", projectBeta, identity.RoleReviewer, goal.GoalRef{})
	betaContext, _ := NewCuratedContext(capability, scopeBeta)
	if _, found, err := catalog.ResolveTool(betaContext, "status.read", "1"); err != nil || !found {
		t.Fatalf("global curated tool found=%v error=%v", found, err)
	}
	if _, found, err := catalog.ResolveSkill(betaContext, "review.status", "1"); err != nil || found {
		t.Fatalf("cross-project skill found=%v error=%v", found, err)
	}
	if _, found, err := catalog.ResolvePlugin(betaContext, "forge.review", "1"); err != nil || found {
		t.Fatalf("cross-project plugin found=%v error=%v", found, err)
	}
	otherCapability, _ := goal.NewCapabilityRef("TLS-11")
	denied, _ := NewCuratedContext(otherCapability, scopeAlpha)
	if _, found, err := catalog.ResolveTool(denied, "status.read", "1"); found || ErrorCode(err) != ErrorCuratedContextDenied {
		t.Fatalf("wrong capability found=%v error=%v", found, err)
	}
	if _, found, err := catalog.ResolveTool(CuratedContext{}, "status.read", "1"); found || ErrorCode(err) != ErrorCuratedContextInvalid {
		t.Fatalf("zero context found=%v error=%v", found, err)
	}

	registration := catalog.Registration()
	registration.Spec.Capabilities[0] = "mutated"
	registration.Spec.Scopes[0].Kind = SkillScopeRole
	registration.Spec.Tools[0].ID = "mutated.tool"
	registration.Spec.Skills[0].ID = "mutated.skill"
	registration.Spec.Plugins[0].ID = "mutated.plugin"
	spec.Capabilities[0], spec.Scopes[0].Kind = "source-mutated", SkillScopeRole
	spec.Tools[0].ID, spec.Skills[0].ID, spec.Plugins[0].ID = "source.tool", "source.skill", "source.plugin"
	if again := catalog.Registration(); again.Spec.Capabilities[0] != "TLS-10" ||
		again.Spec.Scopes[0].Kind != SkillScopeGlobal || again.Spec.Tools[0].ID != "status.read" ||
		again.Spec.Skills[0].ID != "review.status" || again.Spec.Plugins[0].ID != "forge.review" {
		t.Fatal("caller mutation changed curated catalog")
	}
	for _, method := range []string{"ListAll", "Authorize", "Execute", "Install", "Persist", "Refresh", "Revoke"} {
		if _, exists := reflect.TypeOf(CuratedCatalog{}).MethodByName(method); exists {
			t.Fatalf("curated catalog exposes out-of-scope method %s", method)
		}
	}
	if nilCatalog := (*CuratedCatalog)(nil); nilCatalog.Digest() != "" ||
		nilCatalog.Registration().Spec.ID != "" || nilCatalog.Registration().Digest != "" {
		t.Fatal("nil curated catalog should be inert")
	}
}

func TestCuratedCatalogIsStableWhenUnderlyingCatalogsGainUnselectedEntries(t *testing.T) {
	tools, releases, plugins, spec := curationFixture(t)
	first, err := NewCuratedCatalog(tools, releases, plugins, spec)
	if err != nil {
		t.Fatal(err)
	}
	status, _ := tools.Lookup("status.read", "1")
	extra := validSpec("workspace.read", "1", IdempotencyReadReexecute, ReceiptObservation)
	extra.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
	extendedTools, err := NewRegistry(status.Spec, extra)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewCuratedCatalog(extendedTools, releases, plugins, spec)
	if err != nil || first.Digest() != second.Digest() {
		t.Fatalf("unselected addition changed curation: %q/%q error=%v", first.Digest(), second.Digest(), err)
	}
	changedReview := cloneCuratedSpec(spec)
	changedReview.ReviewDigest = repeatedDigest("d")
	third, err := NewCuratedCatalog(tools, releases, plugins, changedReview)
	if err != nil || first.Digest() == third.Digest() {
		t.Fatalf("changed review did not change curation: %q/%q error=%v", first.Digest(), third.Digest(), err)
	}
}

func TestCuratedCatalogRejectsWildcardsImplicitOrDivergentSelectionsAtomically(t *testing.T) {
	tools, releases, plugins, valid := curationFixture(t)
	tests := map[string]func(*CuratedCatalogSpec){
		"id":          func(value *CuratedCatalogSpec) { value.ID = "default" },
		"version":     func(value *CuratedCatalogSpec) { value.Version = "01" },
		"description": func(value *CuratedCatalogSpec) { value.DescriptionKey = "curation.other.description" },
		"review ref":  func(value *CuratedCatalogSpec) { value.ReviewRef = "" },
		"review wildcard": func(value *CuratedCatalogSpec) {
			value.ReviewRef = "review:curation:*"
		},
		"review control": func(value *CuratedCatalogSpec) {
			value.ReviewRef = "review:curation:\n1"
		},
		"review invalid utf8": func(value *CuratedCatalogSpec) {
			value.ReviewRef = string([]byte{'r', 0xff})
		},
		"review digest": func(value *CuratedCatalogSpec) {
			value.ReviewDigest = "sha256:short"
		},
		"review digest uppercase": func(value *CuratedCatalogSpec) {
			value.ReviewDigest = "sha256:" + strings.Repeat("A", 64)
		},
		"capabilities empty": func(value *CuratedCatalogSpec) { value.Capabilities = nil },
		"capability wildcard": func(value *CuratedCatalogSpec) {
			value.Capabilities = []string{"TLS-*"}
		},
		"capability control": func(value *CuratedCatalogSpec) {
			value.Capabilities = []string{"TLS-\n12"}
		},
		"capability invalid utf8": func(value *CuratedCatalogSpec) {
			value.Capabilities = []string{string([]byte{'T', 0xff})}
		},
		"capability duplicate": func(value *CuratedCatalogSpec) {
			value.Capabilities = []string{"TLS-12", "TLS-12"}
		},
		"capability limit": func(value *CuratedCatalogSpec) {
			value.Capabilities = make([]string, MaxCuratedCapabilities+1)
		},
		"scope empty": func(value *CuratedCatalogSpec) { value.Scopes = nil },
		"scope duplicate": func(value *CuratedCatalogSpec) {
			value.Scopes = append(value.Scopes, value.Scopes[0])
		},
		"scope wildcard": func(value *CuratedCatalogSpec) {
			value.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:*"}}
		},
		"scope limit": func(value *CuratedCatalogSpec) {
			value.Scopes = make([]SkillScope, MaxCuratedScopes+1)
		},
		"catalog size": func(value *CuratedCatalogSpec) {
			value.Scopes = make([]SkillScope, MaxCuratedScopes)
			for index := range value.Scopes {
				value.Scopes[index] = SkillScope{Kind: SkillScopeProject,
					ProjectRef: "project:" + strings.Repeat("p", 1000) + string(rune('a'+index/26)) + string(rune('a'+index%26))}
			}
		},
		"selections empty": func(value *CuratedCatalogSpec) {
			value.Tools, value.Skills, value.Plugins = nil, nil, nil
		},
		"tool missing": func(value *CuratedCatalogSpec) { value.Tools[0].ID = "missing.read" },
		"tool digest":  func(value *CuratedCatalogSpec) { value.Tools[0].SpecDigest = repeatedDigest("0") },
		"tool duplicate": func(value *CuratedCatalogSpec) {
			value.Tools = append(value.Tools, value.Tools[0])
		},
		"tool limit": func(value *CuratedCatalogSpec) {
			value.Tools = make([]CuratedToolSelection, MaxCuratedSelectionsPerKind+1)
		},
		"skill missing": func(value *CuratedCatalogSpec) { value.Skills[0].ID = "missing.skill" },
		"skill registration": func(value *CuratedCatalogSpec) {
			value.Skills[0].RegistrationDigest = repeatedDigest("0")
		},
		"skill release": func(value *CuratedCatalogSpec) { value.Skills[0].ReleaseDigest = repeatedDigest("0") },
		"skill duplicate": func(value *CuratedCatalogSpec) {
			value.Skills = append(value.Skills, value.Skills[0])
		},
		"skill limit": func(value *CuratedCatalogSpec) {
			value.Skills = make([]CuratedSkillSelection, MaxCuratedSelectionsPerKind+1)
		},
		"plugin missing": func(value *CuratedCatalogSpec) { value.Plugins[0].ID = "missing.plugin" },
		"plugin digest":  func(value *CuratedCatalogSpec) { value.Plugins[0].PluginDigest = repeatedDigest("0") },
		"plugin duplicate": func(value *CuratedCatalogSpec) {
			value.Plugins = append(value.Plugins, value.Plugins[0])
		},
		"plugin limit": func(value *CuratedCatalogSpec) {
			value.Plugins = make([]CuratedPluginSelection, MaxCuratedSelectionsPerKind+1)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := cloneCuratedSpec(valid)
			mutate(&spec)
			catalog, err := NewCuratedCatalog(tools, releases, plugins, spec)
			if catalog != nil || (ErrorCode(err) != ErrorCuratedCatalogInvalid && ErrorCode(err) != ErrorCuratedSelectionInvalid) {
				t.Fatalf("catalog=%v error=%v code=%q", catalog, err, ErrorCode(err))
			}
		})
	}
	if catalog, err := NewCuratedCatalog(nil, releases, plugins, valid); catalog != nil || ErrorCode(err) != ErrorCuratedCatalogInvalid {
		t.Fatalf("nil tools catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewCuratedCatalog(tools, nil, plugins, valid); catalog != nil || ErrorCode(err) != ErrorCuratedCatalogInvalid {
		t.Fatalf("nil skills catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewCuratedCatalog(tools, releases, nil, valid); catalog != nil || ErrorCode(err) != ErrorCuratedCatalogInvalid {
		t.Fatalf("nil plugins catalog=%v error=%v", catalog, err)
	}
	wildcard, _ := goal.NewCapabilityRef("TLS-*")
	scope, _ := NewSkillScopeContext("", goal.ProjectRef{}, "", goal.GoalRef{})
	if context, err := NewCuratedContext(wildcard, scope); context != (CuratedContext{}) || ErrorCode(err) != ErrorCuratedContextInvalid {
		t.Fatalf("wildcard context=%+v error=%v", context, err)
	}
	control, _ := goal.NewCapabilityRef("TLS-\n12")
	if context, err := NewCuratedContext(control, scope); context != (CuratedContext{}) || ErrorCode(err) != ErrorCuratedContextInvalid {
		t.Fatalf("control context=%+v error=%v", context, err)
	}
	invalidProject, _ := goal.NewProjectRef(string([]byte{'p', 0xff}))
	invalidScope, _ := NewSkillScopeContext("orquesta", invalidProject, identity.RoleReviewer, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef("TLS-12")
	if context, err := NewCuratedContext(capability, invalidScope); context != (CuratedContext{}) || ErrorCode(err) != ErrorCuratedContextInvalid {
		t.Fatalf("invalid scope context=%+v error=%v", context, err)
	}
}

func TestCuratedCatalogBindsExactComponentSnapshotsAndFailsClosedOnDrift(t *testing.T) {
	tools, releases, plugins, spec := curationFixture(t)
	catalog, err := NewCuratedCatalog(tools, releases, plugins, spec)
	if err != nil {
		t.Fatal(err)
	}
	project, _ := goal.NewProjectRef("project:alpha")
	scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleReviewer, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef("TLS-12")
	context, _ := NewCuratedContext(capability, scope)
	tools.entries[tools.byKey[registryKey(spec.Tools[0].ID, spec.Tools[0].Version)]].Digest = repeatedDigest("0")
	releases.entries[releases.byKey[registryKey(spec.Skills[0].ID, spec.Skills[0].Version)]].ReleaseDigest = repeatedDigest("0")
	plugins.entries[plugins.byKey[registryKey(spec.Plugins[0].ID, spec.Plugins[0].Version)]].Digest = repeatedDigest("0")
	if _, found, err := catalog.ResolveTool(context, spec.Tools[0].ID, spec.Tools[0].Version); err != nil || found {
		t.Fatalf("drifted tool found=%v error=%v", found, err)
	}
	if _, found, err := catalog.ResolveSkill(context, spec.Skills[0].ID, spec.Skills[0].Version); err != nil || found {
		t.Fatalf("drifted skill found=%v error=%v", found, err)
	}
	if _, found, err := catalog.ResolvePlugin(context, spec.Plugins[0].ID, spec.Plugins[0].Version); err != nil || found {
		t.Fatalf("drifted plugin found=%v error=%v", found, err)
	}
}

func TestCuratedCatalogRejectsPluginFromDivergentToolSkillOrRevocationSnapshot(t *testing.T) {
	tools, releases, plugins, spec := curationFixture(t)
	pluginOnly := cloneCuratedSpec(spec)
	pluginOnly.Tools, pluginOnly.Skills = nil, nil
	emptyTools, _ := NewRegistry()
	if catalog, err := NewCuratedCatalog(emptyTools, releases, plugins, pluginOnly); catalog != nil || ErrorCode(err) != ErrorCuratedSelectionInvalid {
		t.Fatalf("divergent tools catalog=%v error=%v", catalog, err)
	}
	emptyReleases, err := NewSkillReleaseCatalog(releases.registry, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if catalog, err := NewCuratedCatalog(tools, emptyReleases, plugins, pluginOnly); catalog != nil || ErrorCode(err) != ErrorCuratedSelectionInvalid {
		t.Fatalf("divergent skills catalog=%v error=%v", catalog, err)
	}
	revoked := *releases
	revoked.entries = append([]SkillRelease(nil), releases.entries...)
	revoked.entries[revoked.byKey[registryKey(spec.Skills[0].ID, spec.Skills[0].Version)]].Revoked = true
	if catalog, err := NewCuratedCatalog(tools, &revoked, plugins, pluginOnly); catalog != nil || ErrorCode(err) != ErrorCuratedSelectionInvalid {
		t.Fatalf("revoked plugin skill catalog=%v error=%v", catalog, err)
	}
}

func TestCuratedCatalogOwnScopeDeniesOtherwiseSelectedTool(t *testing.T) {
	tools, releases, plugins, spec := curationFixture(t)
	spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}}
	catalog, err := NewCuratedCatalog(tools, releases, plugins, spec)
	if err != nil {
		t.Fatal(err)
	}
	project, _ := goal.NewProjectRef("project:beta")
	scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleReviewer, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef("TLS-12")
	context, _ := NewCuratedContext(capability, scope)
	if _, found, err := catalog.ResolveTool(context, spec.Tools[0].ID, spec.Tools[0].Version); found || ErrorCode(err) != ErrorCuratedContextDenied {
		t.Fatalf("cross-scope tool found=%v error=%v", found, err)
	}
}

func TestCuratedCatalogAppliesPluginOwnCapabilityAndScope(t *testing.T) {
	tools, releases, _, spec := curationFixture(t)
	status, _ := tools.Lookup("status.read", "1")
	plugins, err := NewPluginCatalog(tools, releases, PluginSpec{
		ID: "scoped.plugin", Version: "1", DescriptionKey: "plugin.scoped.plugin.description",
		Capabilities: []string{"TLS-12"},
		Scopes:       []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}},
		Tools:        []PluginToolRequirement{{ID: status.Spec.ID, Version: status.Spec.Version, SpecDigest: status.Digest}},
		Permissions:  append([]identity.Permission(nil), status.Spec.Permissions...),
	})
	if err != nil {
		t.Fatal(err)
	}
	plugin, _ := plugins.Lookup("scoped.plugin", "1")
	spec.Plugins = []CuratedPluginSelection{{ID: plugin.Spec.ID, Version: plugin.Spec.Version, PluginDigest: plugin.Digest}}
	catalog, err := NewCuratedCatalog(tools, releases, plugins, spec)
	if err != nil {
		t.Fatal(err)
	}
	capability, _ := goal.NewCapabilityRef("TLS-12")
	for _, projectValue := range []string{"project:alpha", "project:beta"} {
		project, _ := goal.NewProjectRef(projectValue)
		scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleReviewer, goal.GoalRef{})
		context, _ := NewCuratedContext(capability, scope)
		_, found, resolveErr := catalog.ResolvePlugin(context, plugin.Spec.ID, plugin.Spec.Version)
		if resolveErr != nil || found != (projectValue == "project:alpha") {
			t.Fatalf("project=%s found=%v error=%v", projectValue, found, resolveErr)
		}
	}
	project, _ := goal.NewProjectRef("project:alpha")
	scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleReviewer, goal.GoalRef{})
	otherCapability, _ := goal.NewCapabilityRef("TLS-10")
	context, _ := NewCuratedContext(otherCapability, scope)
	if _, found, err := catalog.ResolvePlugin(context, plugin.Spec.ID, plugin.Spec.Version); err != nil || found {
		t.Fatalf("other capability found=%v error=%v", found, err)
	}
}

func curationFixture(t *testing.T) (*Registry, *SkillReleaseCatalog, *PluginCatalog, CuratedCatalogSpec) {
	t.Helper()
	tools := pluginTestTools(t)
	status, _ := tools.Lookup("status.read", "1")
	skill := skillCandidate("review.status", "1", skillToolRegistration(status))
	skill.Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:alpha"}}
	skillRegistry, err := NewSkillRegistry(tools, skill)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, root := skillGovernanceTrust(0x71, "signer:curation:1")
	releaseCandidate := signedSkillRelease(t, skillRegistry, "review.status", "1", root.SignerRef, privateKey)
	releases, err := NewSkillReleaseCatalog(skillRegistry, []SkillTrustRoot{root}, []SkillReleaseCandidate{releaseCandidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	release, _ := exactPluginSkillRelease(releases, "review.status", "1")
	pluginSpec := PluginSpec{
		ID: "forge.review", Version: "1", DescriptionKey: "plugin.forge.review.description",
		Tools: []PluginToolRequirement{{ID: status.Spec.ID, Version: status.Spec.Version, SpecDigest: status.Digest}},
		Skills: []PluginSkillRequirement{{
			ID: release.Registration.Spec.ID, Version: release.Registration.Spec.Version,
			RegistrationDigest: release.Registration.Digest, ReleaseDigest: release.ReleaseDigest,
		}},
		Connectors:  []PluginConnectorSpec{},
		Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}
	plugins, err := NewPluginCatalog(tools, releases, pluginSpec)
	if err != nil {
		t.Fatal(err)
	}
	plugin, _ := plugins.Lookup("forge.review", "1")
	return tools, releases, plugins, CuratedCatalogSpec{
		ID: "default.tooling", Version: "1", DescriptionKey: "curation.default.tooling.description",
		ReviewRef: "review:curation:1", ReviewDigest: repeatedDigest("e"),
		Capabilities: []string{"TLS-12", "TLS-10"},
		Scopes:       []SkillScope{{Kind: SkillScopeGlobal}},
		Tools:        []CuratedToolSelection{{ID: status.Spec.ID, Version: status.Spec.Version, SpecDigest: status.Digest}},
		Skills: []CuratedSkillSelection{{
			ID: release.Registration.Spec.ID, Version: release.Registration.Spec.Version,
			RegistrationDigest: release.Registration.Digest, ReleaseDigest: release.ReleaseDigest,
		}},
		Plugins: []CuratedPluginSelection{{ID: plugin.Spec.ID, Version: plugin.Spec.Version, PluginDigest: plugin.Digest}},
	}
}

func cloneCuratedSpec(source CuratedCatalogSpec) CuratedCatalogSpec {
	source.Capabilities = append([]string(nil), source.Capabilities...)
	source.Scopes = append([]SkillScope(nil), source.Scopes...)
	source.Tools = append([]CuratedToolSelection(nil), source.Tools...)
	source.Skills = append([]CuratedSkillSelection(nil), source.Skills...)
	source.Plugins = append([]CuratedPluginSelection(nil), source.Plugins...)
	return source
}
