package tooling

import (
	"crypto/ed25519"
	"reflect"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestToolingBootstrapDoctorBindsExactDesiredDispositionAndDefendsAliases(t *testing.T) {
	doctor, spec := bootstrapDoctorFixture(t, false)
	snapshot := doctor.Snapshot()
	if snapshot.Capability != BootstrapCapability || snapshot.BootstrapVersion != "14" ||
		snapshot.ManifestDigest == "" || snapshot.Digest == "" || snapshot.Ref == "" ||
		snapshot.SelectionContext != spec.SelectionContext ||
		snapshot.BrokerDisposition != BootstrapSurfaceDisabled || snapshot.UIDisposition != BootstrapSurfaceDisabled ||
		snapshot.ExternalPorts != BootstrapSurfaceDisabled || len(snapshot.Subjects) != 4 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	observed := bootstrapProjection(spec, snapshot.ManifestDigest)
	if report := doctor.Diagnose(observed); report.Status != DoctorProjectionMatch || report.Scope != BootstrapProjectionScope ||
		report.DesiredSnapshotDigest != snapshot.ManifestDigest || report.ObservedSnapshotDigest != snapshot.ManifestDigest || len(report.Diagnostics) != 0 {
		t.Fatalf("report=%+v", report)
	}
	spec.Subjects[0].ID = "input-mutated"
	manifest := doctor.Manifest()
	manifest.Subjects[0].ID = "mutated"
	snapshot.Subjects[0].Digest = repeatedDigest("0")
	if doctor.Manifest().Subjects[0].ID == "input-mutated" || doctor.Manifest().Subjects[0].ID == "mutated" ||
		doctor.Snapshot().Subjects[0].Digest == repeatedDigest("0") {
		t.Fatal("caller mutation changed immutable bootstrap snapshot")
	}
	for _, field := range []string{"Path", "URL", "Bytes", "Installed", "Activated", "Authorized", "Persisted", "Attempt", "Wired", "Booted", "Restarted", "Published", "Operational"} {
		if _, exists := reflect.TypeOf(BootstrapDispositionSnapshot{}).FieldByName(field); exists {
			t.Fatalf("snapshot overclaims physical effect through %q", field)
		}
	}
}

func TestToolingBootstrapDoctorRequiresSelectedExactSubjectsAndSafeSurfaces(t *testing.T) {
	curated, context, rules, valid := bootstrapDoctorInputs(t)
	tests := map[string]func(*ToolingBootstrapSpec){
		"version":         func(value *ToolingBootstrapSpec) { value.Version = "latest" },
		"curation":        func(value *ToolingBootstrapSpec) { value.CuratedCatalogDigest = repeatedDigest("0") },
		"rule catalog":    func(value *ToolingBootstrapSpec) { value.RulePackCatalogDigest = repeatedDigest("0") },
		"product context": func(value *ToolingBootstrapSpec) { value.SelectionContext.ProductID = "other" },
		"project context": func(value *ToolingBootstrapSpec) { value.SelectionContext.ProjectRef = "project:other" },
		"role context":    func(value *ToolingBootstrapSpec) { value.SelectionContext.Role = "viewer" },
		"goal context":    func(value *ToolingBootstrapSpec) { value.SelectionContext.GoalRef = "goal:other" },
		"empty":           func(value *ToolingBootstrapSpec) { value.Subjects = nil },
		"duplicate":       func(value *ToolingBootstrapSpec) { value.Subjects = append(value.Subjects, value.Subjects[0]) },
		"kind":            func(value *ToolingBootstrapSpec) { value.Subjects[0].Kind = "resource" },
		"wildcard":        func(value *ToolingBootstrapSpec) { value.Subjects[0].ID = "status.*" },
		"implicit":        func(value *ToolingBootstrapSpec) { value.Subjects[0].Version = "latest" },
		"digest":          func(value *ToolingBootstrapSpec) { value.Subjects[0].Digest = repeatedDigest("0") },
		"ui":              func(value *ToolingBootstrapSpec) { value.UIEnabled = true },
		"external ports":  func(value *ToolingBootstrapSpec) { value.ExternalPortsEnabled = true },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			candidate.Subjects = append([]BootstrapSubject(nil), valid.Subjects...)
			mutate(&candidate)
			if doctor, err := NewToolingBootstrapDoctor(curated, context, rules, candidate); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
				t.Fatalf("doctor=%v error=%v", doctor, err)
			}
		})
	}
	if doctor, err := NewToolingBootstrapDoctor(nil, context, rules, valid); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
		t.Fatalf("nil curated doctor=%v error=%v", doctor, err)
	}
	foreignCapability, _ := goal.NewCapabilityRef("TLS-12")
	foreignContext, _ := NewCuratedContext(foreignCapability, context.scope)
	foreignSpec := valid
	foreignSpec.SelectionContext = bootstrapSelectionContext(foreignContext)
	if doctor, err := NewToolingBootstrapDoctor(curated, foreignContext, rules, foreignSpec); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
		t.Fatalf("foreign capability doctor=%v error=%v", doctor, err)
	}
	tools, releases, plugins, deniedSpec := curationFixture(t)
	denied, err := NewCuratedCatalog(tools, releases, plugins, deniedSpec)
	if err != nil {
		t.Fatal(err)
	}
	rulepackOnly := valid
	rulepackOnly.CuratedCatalogDigest = denied.Digest()
	rulepackOnly.Subjects = []BootstrapSubject{valid.Subjects[1]}
	if doctor, err := NewToolingBootstrapDoctor(denied, context, rules, rulepackOnly); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
		t.Fatalf("rulepack-only bypass doctor=%v error=%v", doctor, err)
	}

	privateKey, root := rulePackTrust(0x53)
	otherProjectPack := rulePackSpec("policy.bootstrap.other", "1", 10,
		SkillScope{Kind: SkillScopeProject, ProjectRef: "project:beta"},
		RulePackItem{Kind: RulePackItemRule, ID: "bootstrap.other", ContractRef: "artifact:" + repeatedDigest("c"), ContractDigest: repeatedDigest("c")},
	)
	origin := RulePackOrigin{Ref: "catalog:bootstrap:other", RevisionDigest: repeatedDigest("d")}
	otherRules, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{
		signedRulePack(t, otherProjectPack, origin, root.SignerRef, privateKey),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	otherSpec := valid
	otherSpec.RulePackCatalogDigest = otherRules.Digest()
	otherSpec.Subjects = []BootstrapSubject{{
		Kind: BootstrapKindRulePack, ID: otherProjectPack.ID, Version: otherProjectPack.Version,
		Digest: otherRules.List()[0].Digest,
	}}
	if doctor, err := NewToolingBootstrapDoctor(curated, context, otherRules, otherSpec); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
		t.Fatalf("cross-scope rulepack doctor=%v error=%v", doctor, err)
	}
}

func TestToolingBootstrapDoctorRejectsInvalidUTF8BeforeDigestsAndSnapshots(t *testing.T) {
	curated, context, rules, valid := bootstrapDoctorInputs(t)
	invalid := string([]byte{0xff})
	tests := map[string]func(*ToolingBootstrapSpec){
		"version":        func(value *ToolingBootstrapSpec) { value.Version = "14" + invalid },
		"curated digest": func(value *ToolingBootstrapSpec) { value.CuratedCatalogDigest += invalid },
		"rule digest":    func(value *ToolingBootstrapSpec) { value.RulePackCatalogDigest += invalid },
		"capability":     func(value *ToolingBootstrapSpec) { value.SelectionContext.CapabilityRef += invalid },
		"product":        func(value *ToolingBootstrapSpec) { value.SelectionContext.ProductID += invalid },
		"project":        func(value *ToolingBootstrapSpec) { value.SelectionContext.ProjectRef += invalid },
		"role":           func(value *ToolingBootstrapSpec) { value.SelectionContext.Role += invalid },
		"goal":           func(value *ToolingBootstrapSpec) { value.SelectionContext.GoalRef += invalid },
		"subject kind":   func(value *ToolingBootstrapSpec) { value.Subjects[0].Kind = BootstrapSubjectKind(invalid) },
		"subject id":     func(value *ToolingBootstrapSpec) { value.Subjects[0].ID += invalid },
		"subject version": func(value *ToolingBootstrapSpec) {
			value.Subjects[0].Version += invalid
		},
		"subject digest": func(value *ToolingBootstrapSpec) { value.Subjects[0].Digest += invalid },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			candidate.Subjects = append([]BootstrapSubject(nil), valid.Subjects...)
			mutate(&candidate)
			if doctor, err := NewToolingBootstrapDoctor(curated, context, rules, candidate); doctor != nil || ErrorCode(err) != ErrorBootstrapInvalid {
				t.Fatalf("doctor=%v error=%v", doctor, err)
			}
		})
	}
	doctor, err := NewToolingBootstrapDoctor(curated, context, rules, valid)
	if err != nil {
		t.Fatal(err)
	}
	snapshotTests := map[string]func(*BootstrapDispositionSnapshot){
		"ref":              func(value *BootstrapDispositionSnapshot) { value.Ref = invalid },
		"digest":           func(value *BootstrapDispositionSnapshot) { value.Digest = invalid },
		"capability":       func(value *BootstrapDispositionSnapshot) { value.Capability = invalid },
		"version":          func(value *BootstrapDispositionSnapshot) { value.BootstrapVersion = invalid },
		"manifest":         func(value *BootstrapDispositionSnapshot) { value.ManifestDigest = invalid },
		"catalogs":         func(value *BootstrapDispositionSnapshot) { value.CuratedCatalogDigest = invalid },
		"rulepacks":        func(value *BootstrapDispositionSnapshot) { value.RulePackCatalogDigest = invalid },
		"selection":        func(value *BootstrapDispositionSnapshot) { value.SelectionContext.GoalRef = invalid },
		"broker":           func(value *BootstrapDispositionSnapshot) { value.BrokerDisposition = invalid },
		"ui":               func(value *BootstrapDispositionSnapshot) { value.UIDisposition = invalid },
		"external ports":   func(value *BootstrapDispositionSnapshot) { value.ExternalPorts = invalid },
		"snapshot subject": func(value *BootstrapDispositionSnapshot) { value.Subjects[0].ID = invalid },
	}
	for name, mutate := range snapshotTests {
		t.Run("snapshot "+name, func(t *testing.T) {
			snapshot := doctor.Snapshot()
			mutate(&snapshot)
			if digest := bootstrapSnapshotDigest(snapshot); digest != "" {
				t.Fatalf("invalid snapshot was digested: %q", digest)
			}
		})
	}
	projection := bootstrapProjection(valid, doctor.Snapshot().ManifestDigest)
	projection.Snapshot.SelectionContext.GoalRef = invalid
	if report := doctor.Diagnose(projection); report.Status != DoctorProjectionDrift || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "projection_invalid" || report.ObservedSnapshotDigest != "" {
		t.Fatalf("invalid projection report=%+v", report)
	}
}

func TestToolingBootstrapDoctorReportsExactDriftWithoutMutatingObservation(t *testing.T) {
	doctor, spec := bootstrapDoctorFixture(t, true)
	snapshot := doctor.Snapshot()
	observed := bootstrapProjection(spec, repeatedDigest("0"))
	observed.Snapshot.Version, observed.Snapshot.UIEnabled = "13", true
	observed.Snapshot.ExternalPortsEnabled, observed.Snapshot.BrokerOptIn = true, false
	observed.Snapshot.Subjects = append([]BootstrapSubject(nil), snapshot.Subjects[1:]...)
	observed.Snapshot.Subjects[0].Digest = repeatedDigest("f")
	observed.Snapshot.Subjects = append(observed.Snapshot.Subjects, BootstrapSubject{Kind: BootstrapKindTool, ID: "extra.read", Version: "1", Digest: repeatedDigest("e")})
	before := append([]BootstrapSubject(nil), observed.Snapshot.Subjects...)
	report := doctor.Diagnose(observed)
	wantedCodes := map[string]bool{
		"projection_digest_mismatch": true, "desired_snapshot_mismatch": true, "bootstrap_version_mismatch": true,
		"broker_disposition_mismatch": true, "ui_not_disabled": true, "external_ports_not_disabled": true,
		"subject_digest_mismatch": true, "subject_unexpected": true, "subject_missing": true,
	}
	for _, diagnostic := range report.Diagnostics {
		delete(wantedCodes, diagnostic.Code)
	}
	if report.Status != DoctorProjectionDrift || len(report.Diagnostics) != 9 || len(wantedCodes) != 0 || !reflect.DeepEqual(before, observed.Snapshot.Subjects) {
		t.Fatalf("report=%+v observation mutated=%v", report, !reflect.DeepEqual(before, observed.Snapshot.Subjects))
	}
	if spec.BrokerOptIn != true || doctor.Snapshot().BrokerDisposition != BootstrapBrokerRequested {
		t.Fatal("broker opt-in was not recorded as requested disposition")
	}
	left := bootstrapProjection(spec, snapshot.ManifestDigest)
	wrong := left.Snapshot.Subjects[0]
	wrong.Digest = repeatedDigest("1")
	left.Snapshot.Subjects = append(left.Snapshot.Subjects, wrong)
	right := bootstrapProjection(left.Snapshot, left.SnapshotDigest)
	right.Snapshot.Subjects[0], right.Snapshot.Subjects[len(right.Snapshot.Subjects)-1] = right.Snapshot.Subjects[len(right.Snapshot.Subjects)-1], right.Snapshot.Subjects[0]
	if !reflect.DeepEqual(doctor.Diagnose(left), doctor.Diagnose(right)) {
		t.Fatal("duplicate diagnostics depend on observation order")
	}
	if nilReport := (*ToolingBootstrapDoctor)(nil).Diagnose(ObservedBootstrapProjection{}); nilReport.Status != DoctorProjectionDrift || nilReport.Scope != BootstrapProjectionScope {
		t.Fatalf("nil report=%+v", nilReport)
	}
}

func bootstrapProjection(spec ToolingBootstrapSpec, digest string) ObservedBootstrapProjection {
	spec.Subjects = append([]BootstrapSubject(nil), spec.Subjects...)
	return ObservedBootstrapProjection{Snapshot: spec, SnapshotDigest: digest}
}

func bootstrapDoctorFixture(t *testing.T, broker bool) (*ToolingBootstrapDoctor, ToolingBootstrapSpec) {
	t.Helper()
	curated, context, rules, spec := bootstrapDoctorInputs(t)
	spec.BrokerOptIn = broker
	for left, right := 0, len(spec.Subjects)-1; left < right; left, right = left+1, right-1 {
		spec.Subjects[left], spec.Subjects[right] = spec.Subjects[right], spec.Subjects[left]
	}
	doctor, err := NewToolingBootstrapDoctor(curated, context, rules, spec)
	if err != nil {
		t.Fatal(err)
	}
	return doctor, spec
}

func bootstrapDoctorInputs(t *testing.T) (*CuratedCatalog, CuratedContext, *RulePackCatalog, ToolingBootstrapSpec) {
	t.Helper()
	tools, releases, plugins, curatedSpec := curationFixture(t)
	curatedSpec.Capabilities = append(curatedSpec.Capabilities, BootstrapCapability)
	curated, err := NewCuratedCatalog(tools, releases, plugins, curatedSpec)
	if err != nil {
		t.Fatal(err)
	}
	project, _ := goal.NewProjectRef("project:alpha")
	scope, _ := NewSkillScopeContext("orquesta", project, identity.RoleReviewer, goal.GoalRef{})
	capability, _ := goal.NewCapabilityRef(BootstrapCapability)
	context, err := NewCuratedContext(capability, scope)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, root := rulePackTrust(0x52)
	packSpec := rulePackSpec("policy.bootstrap", "1", 10, SkillScope{Kind: SkillScopeGlobal},
		RulePackItem{Kind: RulePackItemRule, ID: "bootstrap.exact", ContractRef: "artifact:" + repeatedDigest("a"), ContractDigest: repeatedDigest("a")})
	origin := RulePackOrigin{Ref: "catalog:bootstrap:rules", RevisionDigest: repeatedDigest("b")}
	claim, err := NewRulePackClaim(packSpec, origin, root.SignerRef)
	if err != nil {
		t.Fatal(err)
	}
	candidate := RulePackCandidate{Spec: packSpec, Claim: claim, Signature: ed25519.Sign(privateKey, claim.SignaturePayload())}
	rules, err := NewRulePackCatalog(origin, []SkillTrustRoot{root}, nil, []RulePackCandidate{candidate}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tool, _, _ := curated.ResolveTool(context, curatedSpec.Tools[0].ID, curatedSpec.Tools[0].Version)
	skill, _, _ := curated.ResolveSkill(context, curatedSpec.Skills[0].ID, curatedSpec.Skills[0].Version)
	plugin, _, _ := curated.ResolvePlugin(context, curatedSpec.Plugins[0].ID, curatedSpec.Plugins[0].Version)
	pack := rules.List()[0]
	return curated, context, rules, ToolingBootstrapSpec{
		Version: "14", CuratedCatalogDigest: curated.Digest(), RulePackCatalogDigest: rules.Digest(),
		SelectionContext: BootstrapSelectionContext{
			CapabilityRef: BootstrapCapability, ProductID: "orquesta", ProjectRef: "project:alpha",
			Role: string(identity.RoleReviewer),
		},
		Subjects: []BootstrapSubject{
			{Kind: BootstrapKindPlugin, ID: plugin.Spec.ID, Version: plugin.Spec.Version, Digest: plugin.Digest},
			{Kind: BootstrapKindRulePack, ID: pack.Spec.ID, Version: pack.Spec.Version, Digest: pack.Digest},
			{Kind: BootstrapKindSkill, ID: skill.Registration.Spec.ID, Version: skill.Registration.Spec.Version, Digest: skill.ReleaseDigest},
			{Kind: BootstrapKindTool, ID: tool.Spec.ID, Version: tool.Spec.Version, Digest: tool.Digest},
		},
	}
}
