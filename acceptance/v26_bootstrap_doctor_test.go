package acceptance_test

import (
	"bytes"
	"crypto/ed25519"
	"reflect"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS14BootstrapDoctorKeepsExactDesiredDispositionAndSafeDefaults(t *testing.T) {
	curated, project, plugins := v26BootstrapCatalogFixture(t)
	context := v26BootstrapContext(t, project)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x74}, ed25519.SeedSize))
	root := tooling.SkillTrustRoot{
		SignerRef: "signer:v26:bootstrap:1",
		PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...),
	}
	packSpec := v26RulePack("policy.bootstrap", 10, tooling.SkillScope{Kind: tooling.SkillScopeGlobal}, "d")
	origin := tooling.RulePackOrigin{Ref: "catalog:v26:bootstrap", RevisionDigest: v26RulePackDigest("e")}
	claim, err := tooling.NewRulePackClaim(packSpec, origin, root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	packCandidate := tooling.RulePackCandidate{Spec: packSpec, Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
	rules, err := tooling.NewRulePackCatalog(
		origin,
		[]tooling.SkillTrustRoot{root}, nil, []tooling.RulePackCandidate{packCandidate}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	pack := rules.List()[0]
	spec := tooling.ToolingBootstrapSpec{
		Version: "14", CuratedCatalogDigest: curated.Digest(), RulePackCatalogDigest: rules.Digest(),
		SelectionContext: tooling.BootstrapSelectionContext{
			CapabilityRef: tooling.BootstrapCapability, ProductID: "orquesta",
			ProjectRef: project.String(), Role: "operator",
		},
		Subjects: []tooling.BootstrapSubject{
			{Kind: tooling.BootstrapKindPlugin, ID: plugins[0].Spec.ID, Version: plugins[0].Spec.Version, Digest: plugins[0].Digest},
			{Kind: tooling.BootstrapKindRulePack, ID: pack.Spec.ID, Version: pack.Spec.Version, Digest: pack.Digest},
		},
	}
	doctor, err := tooling.NewToolingBootstrapDoctor(curated, context, rules, spec)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := doctor.Snapshot()
	report := doctor.Diagnose(v26BootstrapProjection(spec, snapshot.ManifestDigest))
	if report.Status != tooling.DoctorProjectionMatch || report.Scope != tooling.BootstrapProjectionScope ||
		report.DesiredSnapshotDigest != snapshot.ManifestDigest || report.ObservedSnapshotDigest != snapshot.ManifestDigest || len(report.Diagnostics) != 0 ||
		snapshot.Capability != tooling.BootstrapCapability || snapshot.SelectionContext != spec.SelectionContext ||
		snapshot.BrokerDisposition != tooling.BootstrapSurfaceDisabled ||
		snapshot.UIDisposition != tooling.BootstrapSurfaceDisabled || snapshot.ExternalPorts != tooling.BootstrapSurfaceDisabled {
		t.Fatalf("snapshot=%+v report=%+v", snapshot, report)
	}

	drifted := snapshot.Subjects
	drifted[0].Digest = v26RulePackDigest("0")
	driftedProjection := v26BootstrapProjection(spec, snapshot.ManifestDigest)
	driftedProjection.Snapshot.BrokerOptIn, driftedProjection.Snapshot.UIEnabled = true, true
	driftedProjection.Snapshot.ExternalPortsEnabled, driftedProjection.Snapshot.Subjects = true, drifted
	drift := doctor.Diagnose(driftedProjection)
	if drift.Status != tooling.DoctorProjectionDrift || len(drift.Diagnostics) < 5 {
		t.Fatalf("drift=%+v", drift)
	}
	for name, mutate := range map[string]func(*tooling.ToolingBootstrapSpec){
		"curated catalog":  func(value *tooling.ToolingBootstrapSpec) { value.CuratedCatalogDigest = v26RulePackDigest("1") },
		"rulepack catalog": func(value *tooling.ToolingBootstrapSpec) { value.RulePackCatalogDigest = v26RulePackDigest("2") },
		"selection context": func(value *tooling.ToolingBootstrapSpec) {
			value.SelectionContext.ProjectRef = "project:other"
		},
	} {
		t.Run(name, func(t *testing.T) {
			projection := v26BootstrapProjection(spec, snapshot.ManifestDigest)
			mutate(&projection.Snapshot)
			if report := doctor.Diagnose(projection); report.Status != tooling.DoctorProjectionDrift || report.ObservedSnapshotDigest == snapshot.ManifestDigest {
				t.Fatalf("projection drift not bound: %+v", report)
			}
		})
	}
	for _, field := range []string{"Path", "URL", "Installed", "Downloaded", "Authorized", "Persisted", "Activated", "Wired", "Booted", "Restarted", "Published", "Operational"} {
		if _, exists := reflect.TypeOf(tooling.BootstrapDispositionSnapshot{}).FieldByName(field); exists {
			t.Fatalf("TLS-14 snapshot overclaims %q", field)
		}
	}
}

func TestV26TLS14RejectsUncuratedSubjectAndExternalSurfaceActivation(t *testing.T) {
	curated, project, plugins := v26BootstrapCatalogFixture(t)
	context := v26BootstrapContext(t, project)
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x75}, ed25519.SeedSize))
	root := tooling.SkillTrustRoot{SignerRef: "signer:v26:bootstrap:2", PublicKey: append([]byte(nil), privateKey[ed25519.SeedSize:]...)}
	packSpec := v26RulePack("policy.bootstrap", 10, tooling.SkillScope{Kind: tooling.SkillScopeGlobal}, "c")
	origin := tooling.RulePackOrigin{Ref: "catalog:v26:bootstrap", RevisionDigest: v26RulePackDigest("b")}
	claim, err := tooling.NewRulePackClaim(packSpec, origin, root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	candidate := tooling.RulePackCandidate{Spec: packSpec, Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
	rules, err := tooling.NewRulePackCatalog(origin, []tooling.SkillTrustRoot{root}, nil, []tooling.RulePackCandidate{candidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := tooling.ToolingBootstrapSpec{
		Version: "14", CuratedCatalogDigest: curated.Digest(), RulePackCatalogDigest: rules.Digest(),
		SelectionContext: tooling.BootstrapSelectionContext{
			CapabilityRef: tooling.BootstrapCapability, ProductID: "orquesta",
			ProjectRef: project.String(), Role: string(identity.RoleOperator),
		},
		Subjects: []tooling.BootstrapSubject{{Kind: tooling.BootstrapKindPlugin, ID: plugins[0].Spec.ID + ".uncurated", Version: plugins[0].Spec.Version, Digest: plugins[0].Digest}},
	}
	if doctor, err := tooling.NewToolingBootstrapDoctor(curated, context, rules, base); doctor != nil || tooling.ErrorCode(err) != tooling.ErrorBootstrapInvalid {
		t.Fatalf("uncurated doctor=%v error=%v", doctor, err)
	}
	base.Subjects[0] = tooling.BootstrapSubject{Kind: tooling.BootstrapKindPlugin, ID: plugins[0].Spec.ID, Version: plugins[0].Spec.Version, Digest: plugins[0].Digest}
	base.ExternalPortsEnabled = true
	if doctor, err := tooling.NewToolingBootstrapDoctor(curated, context, rules, base); doctor != nil || tooling.ErrorCode(err) != tooling.ErrorBootstrapInvalid {
		t.Fatalf("external surface doctor=%v error=%v", doctor, err)
	}
	base.ExternalPortsEnabled = false
	base.Subjects[0].ID += string([]byte{0xff})
	if doctor, err := tooling.NewToolingBootstrapDoctor(curated, context, rules, base); doctor != nil || tooling.ErrorCode(err) != tooling.ErrorBootstrapInvalid {
		t.Fatalf("invalid UTF-8 doctor=%v error=%v", doctor, err)
	}

	otherPack := v26RulePack("policy.bootstrap.other", 10,
		tooling.SkillScope{Kind: tooling.SkillScopeProject, ProjectRef: "project:other"}, "f")
	otherOrigin := tooling.RulePackOrigin{Ref: "catalog:v26:bootstrap:other", RevisionDigest: v26RulePackDigest("a")}
	otherClaim, err := tooling.NewRulePackClaim(otherPack, otherOrigin, root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	otherRules, err := tooling.NewRulePackCatalog(otherOrigin, []tooling.SkillTrustRoot{root}, nil,
		[]tooling.RulePackCandidate{{
			Spec: otherPack, Claim: otherClaim, Signature: ed25519.Sign(privateKey, otherClaim.SignaturePayload()),
		}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	base.RulePackCatalogDigest = otherRules.Digest()
	base.Subjects[0] = tooling.BootstrapSubject{
		Kind: tooling.BootstrapKindRulePack, ID: otherPack.ID, Version: otherPack.Version,
		Digest: otherRules.List()[0].Digest,
	}
	if doctor, err := tooling.NewToolingBootstrapDoctor(curated, context, otherRules, base); doctor != nil || tooling.ErrorCode(err) != tooling.ErrorBootstrapInvalid {
		t.Fatalf("cross-scope rulepack doctor=%v error=%v", doctor, err)
	}
}

func v26BootstrapContext(t *testing.T, project goal.ProjectRef) tooling.CuratedContext {
	t.Helper()
	scope, err := tooling.NewSkillScopeContext("orquesta", project, identity.RoleOperator, goal.GoalRef{})
	if err != nil {
		t.Fatal(err)
	}
	capability, err := goal.NewCapabilityRef(tooling.BootstrapCapability)
	if err != nil {
		t.Fatal(err)
	}
	context, err := tooling.NewCuratedContext(capability, scope)
	if err != nil {
		t.Fatal(err)
	}
	return context
}

func v26BootstrapCatalogFixture(t *testing.T) (*tooling.CuratedCatalog, goal.ProjectRef, []tooling.PluginRegistration) {
	t.Helper()
	tools, err := tooling.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	skills, err := tooling.NewSkillRegistry(tools)
	if err != nil {
		t.Fatal(err)
	}
	releases, err := tooling.NewSkillReleaseCatalog(skills, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	contractDigest := v26RulePackDigest("9")
	plugins, err := tooling.NewPluginCatalog(tools, releases, tooling.PluginSpec{
		ID: "bootstrap.fixture", Version: "1", DescriptionKey: "plugin.bootstrap.fixture.description",
		Connectors: []tooling.PluginConnectorSpec{{
			ID: "bootstrap.fixture", Version: "1", ContractRef: "artifact:" + contractDigest,
			ContractDigest: contractDigest, Permissions: []identity.Permission{identity.PermissionArtifactsRead},
		}}, Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	plugin := plugins.List()[0]
	curated, err := tooling.NewCuratedCatalog(tools, releases, plugins, tooling.CuratedCatalogSpec{
		ID: "bootstrap.fixture", Version: "1", DescriptionKey: "curation.bootstrap.fixture.description",
		ReviewRef: "review:v26:bootstrap", ReviewDigest: v26RulePackDigest("8"),
		Capabilities: []string{tooling.BootstrapCapability}, Scopes: []tooling.SkillScope{{Kind: tooling.SkillScopeGlobal}},
		Plugins: []tooling.CuratedPluginSelection{{ID: plugin.Spec.ID, Version: plugin.Spec.Version, PluginDigest: plugin.Digest}},
	})
	if err != nil {
		t.Fatal(err)
	}
	project, _ := goal.NewProjectRef("project:v26-tls14")
	return curated, project, plugins.List()
}

func v26BootstrapProjection(spec tooling.ToolingBootstrapSpec, digest string) tooling.ObservedBootstrapProjection {
	spec.Subjects = append([]tooling.BootstrapSubject(nil), spec.Subjects...)
	return tooling.ObservedBootstrapProjection{Snapshot: spec, SnapshotDigest: digest}
}
