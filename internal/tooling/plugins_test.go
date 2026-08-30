package tooling

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/identity"
)

func TestPluginCatalogPackagesOnlyExactConnectorToolAndSkillDescriptors(t *testing.T) {
	tools, skills, plugin := pluginFixture(t)
	catalog, err := NewPluginCatalog(tools, skills, plugin)
	if err != nil {
		t.Fatal(err)
	}
	reordered := clonePluginSpec(plugin)
	reordered.Connectors = []PluginConnectorSpec{reordered.Connectors[1], reordered.Connectors[0]}
	reordered.Permissions = []identity.Permission{
		identity.PermissionChangesIntegrate, identity.PermissionArtifactsRead,
	}
	second, err := NewPluginCatalog(tools, skills, reordered)
	if err != nil || catalog.Digest() != second.Digest() || !strings.HasPrefix(catalog.Digest(), "sha256:") {
		t.Fatalf("catalog=%q reordered=%q error=%v", catalog.Digest(), second.Digest(), err)
	}
	registration, found := catalog.Lookup("forge.review", "1")
	if !found || registration.Spec.Format != PluginDescriptorFormatV1 ||
		!reflect.DeepEqual(registration.Spec.Capabilities, []string{"EXT-15", "TLS-10"}) ||
		len(registration.Spec.Scopes) != 1 || registration.Spec.Config == nil || registration.Spec.Config.SizeBytes != 4096 ||
		len(registration.Spec.Connectors) != 2 ||
		registration.Spec.Connectors[0].ID != "forge.events" ||
		registration.Spec.Tools[0].ID != "status.read" ||
		registration.Spec.Skills[0].ID != "review.status" ||
		!reflect.DeepEqual(registration.Spec.Permissions, []identity.Permission{
			identity.PermissionArtifactsRead, identity.PermissionChangesIntegrate,
		}) {
		t.Fatalf("registration=%+v found=%v", registration, found)
	}
	encoded, err := json.Marshal(registration)
	if err != nil || strings.Contains(string(encoded), "never-retain-this-body") {
		t.Fatalf("registration=%s error=%v", encoded, err)
	}
	assertPluginJSONShape(t, PluginSpec{}, []string{
		"Format:format", "ID:id", "Version:version", "DescriptionKey:description_key",
		"Capabilities:capabilities", "Scopes:scopes", "Config:config,omitempty", "Connectors:connectors",
		"Tools:tools", "Skills:skills", "Permissions:permissions",
	})
	assertPluginJSONShape(t, PluginConfigRequirement{}, []string{"ContractRef:contract_ref", "ContractDigest:contract_digest", "SizeBytes:size_bytes"})
	assertPluginJSONShape(t, PluginConnectorSpec{}, []string{"ID:id", "Version:version", "ContractRef:contract_ref", "ContractDigest:contract_digest", "Permissions:permissions"})
	assertPluginJSONShape(t, PluginToolRequirement{}, []string{"ID:id", "Version:version", "SpecDigest:spec_digest"})
	assertPluginJSONShape(t, PluginSkillRequirement{}, []string{"ID:id", "Version:version", "RegistrationDigest:registration_digest", "ReleaseDigest:release_digest,omitempty"})
	assertPluginJSONShape(t, PluginRegistration{}, []string{"Spec:spec", "Digest:digest"})
	plugin.Capabilities[0] = "mutated"
	plugin.Scopes[0].ProjectRef = "project:mutated"
	plugin.Config.SizeBytes = 1
	plugin.Connectors[0].Permissions[0] = identity.Permission("mutated")
	registration.Spec.Capabilities[0] = "mutated"
	registration.Spec.Scopes[0].ProjectRef = "project:mutated"
	registration.Spec.Config.SizeBytes = 1
	registration.Spec.Connectors[0].Permissions[0] = identity.Permission("mutated")
	registration.Spec.Tools[0].ID = "mutated.tool"
	registration.Spec.Skills[0].ID = "mutated.skill"
	registration.Spec.Permissions[0] = identity.Permission("mutated")
	again, _ := catalog.Lookup("forge.review", "1")
	if again.Spec.Capabilities[0] != "EXT-15" || again.Spec.Scopes[0].ProjectRef != "project:plugins" ||
		again.Spec.Config.SizeBytes != 4096 || again.Spec.Connectors[0].Permissions[0] != identity.PermissionArtifactsRead ||
		again.Spec.Tools[0].ID != "status.read" || again.Spec.Skills[0].ID != "review.status" ||
		again.Spec.Permissions[0] != identity.PermissionArtifactsRead {
		t.Fatal("caller mutation changed plugin catalog")
	}

	status, _ := tools.Lookup("status.read", "1")
	extra := validSpec("workspace.read", "1", IdempotencyReadReexecute, ReceiptObservation)
	extra.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
	extendedTools, err := NewRegistry(status.Spec, extra)
	if err != nil {
		t.Fatal(err)
	}
	extended, err := NewPluginCatalog(extendedTools, skills, reordered)
	if err != nil || extended.Digest() != catalog.Digest() {
		t.Fatalf("unrelated tool changed plugin catalog: %q/%q error=%v", extended.Digest(), catalog.Digest(), err)
	}
	if nilCatalog := (*PluginCatalog)(nil); nilCatalog.Digest() != "" || nilCatalog.List() != nil {
		t.Fatal("nil plugin catalog should be inert")
	}
	listed := catalog.List()
	listed[0].Spec.Capabilities[0] = "mutated"
	listed[0].Spec.Scopes[0].ProjectRef = "project:mutated"
	listed[0].Spec.Config.SizeBytes = 1
	if afterList, _ := catalog.Lookup("forge.review", "1"); afterList.Spec.Capabilities[0] != "EXT-15" ||
		afterList.Spec.Scopes[0].ProjectRef != "project:plugins" || afterList.Spec.Config.SizeBytes != 4096 {
		t.Fatal("List exposed plugin catalog memory")
	}
	changedConfig := clonePluginSpec(reordered)
	changedConfig.Config.ContractDigest = repeatedDigest("f")
	changedConfig.Config.ContractRef = "artifact:" + changedConfig.Config.ContractDigest
	changed, err := NewPluginCatalog(tools, skills, changedConfig)
	if err != nil || changed.Digest() == catalog.Digest() {
		t.Fatalf("config contract was not bound to descriptor digest: changed=%v error=%v", changed, err)
	}
}

func TestPluginCatalogRejectsInvalidOrImplicitComponentsAndPermissionsAtomically(t *testing.T) {
	tools, skills, valid := pluginFixture(t)
	tests := map[string]func(*PluginSpec){
		"format missing": func(value *PluginSpec) { value.Format = "" },
		"format changed": func(value *PluginSpec) { value.Format = "plugin_descriptor_v2" },
		"id":             func(value *PluginSpec) { value.ID = "forge" },
		"version":        func(value *PluginSpec) { value.Version = "01" },
		"description":    func(value *PluginSpec) { value.DescriptionKey = "plugin.other.description" },
		"empty": func(value *PluginSpec) {
			value.Connectors, value.Tools, value.Skills, value.Permissions = nil, nil, nil, nil
		},
		"connector id":      func(value *PluginSpec) { value.Connectors[0].ID = "forge" },
		"connector version": func(value *PluginSpec) { value.Connectors[0].Version = "0" },
		"connector ref":     func(value *PluginSpec) { value.Connectors[0].ContractRef = "./contract.json" },
		"connector digest":  func(value *PluginSpec) { value.Connectors[0].ContractDigest = "sha256:short" },
		"connector uppercase": func(value *PluginSpec) {
			value.Connectors[0].ContractDigest = strings.ToUpper(value.Connectors[0].ContractDigest)
			value.Connectors[0].ContractRef = "artifact:" + value.Connectors[0].ContractDigest
		},
		"connector permission": func(value *PluginSpec) {
			value.Connectors[0].Permissions = nil
		},
		"duplicate connector": func(value *PluginSpec) {
			value.Connectors = append(value.Connectors, value.Connectors[0])
		},
		"tool missing":        func(value *PluginSpec) { value.Tools[0].ID = "missing.read" },
		"tool digest":         func(value *PluginSpec) { value.Tools[0].SpecDigest = repeatedDigest("0") },
		"duplicate tool":      func(value *PluginSpec) { value.Tools = append(value.Tools, value.Tools[0]) },
		"skill missing":       func(value *PluginSpec) { value.Skills[0].ID = "missing.skill" },
		"skill registry":      func(value *PluginSpec) { value.Skills[0].RegistrationDigest = repeatedDigest("0") },
		"skill release":       func(value *PluginSpec) { value.Skills[0].ReleaseDigest = repeatedDigest("0") },
		"duplicate skill":     func(value *PluginSpec) { value.Skills = append(value.Skills, value.Skills[0]) },
		"skill tool omitted":  func(value *PluginSpec) { value.Tools = nil },
		"capability wildcard": func(value *PluginSpec) { value.Capabilities[0] = "TLS-*" },
		"duplicate capability": func(value *PluginSpec) {
			value.Capabilities = append(value.Capabilities, value.Capabilities[0])
		},
		"duplicate scope":  func(value *PluginSpec) { value.Scopes = append(value.Scopes, value.Scopes[0]) },
		"config ref":       func(value *PluginSpec) { value.Config.ContractRef = "./config.json" },
		"config size":      func(value *PluginSpec) { value.Config.SizeBytes = MaxPluginConfigContractBytes + 1 },
		"capability limit": func(value *PluginSpec) { value.Capabilities = make([]string, MaxPluginCapabilities+1) },
		"scope limit":      func(value *PluginSpec) { value.Scopes = make([]SkillScope, MaxPluginScopes+1) },
		"connector limit":  func(value *PluginSpec) { value.Connectors = make([]PluginConnectorSpec, MaxPluginComponentsPerKind+1) },
		"tool limit":       func(value *PluginSpec) { value.Tools = make([]PluginToolRequirement, MaxPluginComponentsPerKind+1) },
		"skill limit":      func(value *PluginSpec) { value.Skills = make([]PluginSkillRequirement, MaxPluginComponentsPerKind+1) },
		"permission limit": func(value *PluginSpec) { value.Permissions = make([]identity.Permission, MaxPluginPermissions+1) },
		"descriptor size": func(value *PluginSpec) {
			value.Connectors = make([]PluginConnectorSpec, MaxPluginComponentsPerKind)
			for index := range value.Connectors {
				value.Connectors[index] = PluginConnectorSpec{
					ID: fmt.Sprintf("connector.%s.v%d", strings.Repeat("a", 60), index+1), Version: "1",
					ContractRef: "artifact:" + repeatedDigest("c"), ContractDigest: repeatedDigest("c"),
					Permissions: []identity.Permission{identity.PermissionArtifactsRead},
				}
			}
			value.Tools, value.Skills = nil, nil
			value.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
		},
		"missing permission": func(value *PluginSpec) {
			value.Permissions = value.Permissions[:1]
		},
		"extra permission": func(value *PluginSpec) {
			value.Permissions = append(value.Permissions, identity.PermissionGoalsDirect)
		},
		"duplicate permission": func(value *PluginSpec) {
			value.Permissions = append(value.Permissions, value.Permissions[0])
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			plugin := clonePluginSpec(valid)
			mutate(&plugin)
			catalog, err := NewPluginCatalog(tools, skills, plugin)
			if catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
				t.Fatalf("catalog=%v error=%v code=%q", catalog, err, ErrorCode(err))
			}
		})
	}
	if catalog, err := NewPluginCatalog(nil, skills, valid); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("nil tools catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewPluginCatalog(tools, nil, valid); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("nil skills catalog=%v error=%v", catalog, err)
	}
	var nilSkills *SkillRegistry
	if catalog, err := NewPluginCatalog(tools, nilSkills, valid); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("typed nil skills catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewPluginCatalog(tools, skills, valid, valid); catalog != nil || ErrorCode(err) != ErrorPluginSpecDuplicate {
		t.Fatalf("duplicate catalog=%v error=%v", catalog, err)
	}
	if catalog, err := NewPluginCatalog(tools, skills, make([]PluginSpec, MaxPluginsPerCatalog+1)...); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("oversized catalog=%v error=%v", catalog, err)
	}
	invalidSecond := clonePluginSpec(valid)
	invalidSecond.ID = "invalid"
	if catalog, err := NewPluginCatalog(tools, skills, valid, invalidSecond); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("partially published catalog=%v error=%v", catalog, err)
	}
	status, _ := tools.Lookup("status.read", "1")
	changedSkill := skillCandidate("review.status", "1", skillToolRegistration(status))
	changedSkill.Spec.Scopes = []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:changed"}}
	changedSkills, err := NewSkillRegistry(tools, changedSkill)
	if err != nil {
		t.Fatal(err)
	}
	if catalog, err := NewPluginCatalog(tools, changedSkills, valid); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("changed skill scope catalog=%v error=%v", catalog, err)
	}

	registry := skillGovernanceRegistry(t, "1", "2")
	privateKey, root := skillGovernanceTrust(0x61, "signer:plugin:1")
	releaseOne := signedSkillRelease(t, registry, "review.status", "1", root.SignerRef, privateKey)
	releaseTwo := signedSkillRelease(t, registry, "review.status", "2", root.SignerRef, privateKey)
	revocation := signedSkillRevocation(t, registry, "2", "1", root.SignerRef, privateKey)
	revokedCatalog, err := NewSkillReleaseCatalog(registry, []SkillTrustRoot{root},
		[]SkillReleaseCandidate{releaseOne, releaseTwo}, []SkillRevocationCandidate{revocation})
	if err != nil {
		t.Fatal(err)
	}
	revoked, _ := exactPluginSkillRelease(revokedCatalog, "review.status", "2")
	revokedPlugin := clonePluginSpec(valid)
	revokedPlugin.Skills[0] = PluginSkillRequirement{
		ID: "review.status", Version: "2", RegistrationDigest: revoked.Registration.Digest,
		ReleaseDigest: revoked.ReleaseDigest,
	}
	if catalog, err := NewPluginCatalog(tools, revokedCatalog, revokedPlugin); catalog != nil || ErrorCode(err) != ErrorPluginSpecInvalid {
		t.Fatalf("revoked skill catalog=%v error=%v", catalog, err)
	}
}

func TestPluginComponentKindsStayDistinctEvenWithSameIdentity(t *testing.T) {
	tools, skills, plugin := pluginFixture(t)
	plugin.Connectors = []PluginConnectorSpec{{
		ID: "status.read", Version: "1", ContractRef: "artifact:" + repeatedDigest("c"),
		ContractDigest: repeatedDigest("c"), Permissions: []identity.Permission{identity.PermissionArtifactsRead},
	}}
	plugin.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
	catalog, err := NewPluginCatalog(tools, skills, plugin)
	if err != nil {
		t.Fatal(err)
	}
	registration, _ := catalog.Lookup(plugin.ID, plugin.Version)
	if registration.Spec.Connectors[0].ID != "status.read" || registration.Spec.Tools[0].ID != "status.read" ||
		registration.Spec.Skills[0].ID != "review.status" {
		t.Fatalf("plugin collapsed component kinds: %+v", registration)
	}
}

func pluginFixture(t *testing.T) (*Registry, *SkillRegistry, PluginSpec) {
	t.Helper()
	tools := pluginTestTools(t)
	status, _ := tools.Lookup("status.read", "1")
	skillCandidate := skillCandidate("review.status", "1", skillToolRegistration(status))
	skillCandidate.Spec.Scopes = []SkillScope{{Kind: SkillScopeGlobal}}
	skillRegistry, err := NewSkillRegistry(tools, skillCandidate)
	if err != nil {
		t.Fatal(err)
	}
	skill, _ := skillRegistry.Lookup("review.status", "1")
	return tools, skillRegistry, PluginSpec{
		Format: PluginDescriptorFormatV1, ID: "forge.review", Version: "1", DescriptionKey: "plugin.forge.review.description",
		Capabilities: []string{"TLS-10", "EXT-15"},
		Scopes:       []SkillScope{{Kind: SkillScopeProject, ProjectRef: "project:plugins"}},
		Config: &PluginConfigRequirement{
			ContractRef: "artifact:" + repeatedDigest("e"), ContractDigest: repeatedDigest("e"), SizeBytes: 4096,
		},
		Connectors: []PluginConnectorSpec{
			{ID: "forge.mutations", Version: "1", ContractRef: "artifact:" + repeatedDigest("d"), ContractDigest: repeatedDigest("d"), Permissions: []identity.Permission{identity.PermissionChangesIntegrate}},
			{ID: "forge.events", Version: "1", ContractRef: "artifact:" + repeatedDigest("c"), ContractDigest: repeatedDigest("c"), Permissions: []identity.Permission{identity.PermissionArtifactsRead}},
		},
		Tools: []PluginToolRequirement{{ID: status.Spec.ID, Version: status.Spec.Version, SpecDigest: status.Digest}},
		Skills: []PluginSkillRequirement{{
			ID: skill.Spec.ID, Version: skill.Spec.Version, RegistrationDigest: skill.Digest,
		}},
		Permissions: []identity.Permission{identity.PermissionChangesIntegrate, identity.PermissionArtifactsRead},
	}
}

func pluginTestTools(t *testing.T) *Registry {
	t.Helper()
	status := validSpec("status.read", "1", IdempotencyReadReexecute, ReceiptObservation)
	status.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
	registry, err := NewRegistry(status)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func skillToolRegistration(registration Registration) SkillToolRegistration {
	return SkillToolRegistration{
		ID: registration.Spec.ID, Version: registration.Spec.Version,
		SpecDigest:  registration.Digest,
		Permissions: append([]identity.Permission(nil), registration.Spec.Permissions...),
	}
}

func clonePluginSpec(source PluginSpec) PluginSpec {
	source.Capabilities = append([]string(nil), source.Capabilities...)
	source.Scopes = append([]SkillScope(nil), source.Scopes...)
	if source.Config != nil {
		config := *source.Config
		source.Config = &config
	}
	source.Connectors = append([]PluginConnectorSpec(nil), source.Connectors...)
	for index := range source.Connectors {
		source.Connectors[index].Permissions = append([]identity.Permission(nil), source.Connectors[index].Permissions...)
	}
	source.Tools = append([]PluginToolRequirement(nil), source.Tools...)
	source.Skills = append([]PluginSkillRequirement(nil), source.Skills...)
	source.Permissions = append([]identity.Permission(nil), source.Permissions...)
	return source
}

func assertPluginJSONShape(t *testing.T, value any, expected []string) {
	t.Helper()
	typeOfValue := reflect.TypeOf(value)
	actual := make([]string, typeOfValue.NumField())
	for index := range actual {
		field := typeOfValue.Field(index)
		actual[index] = field.Name + ":" + field.Tag.Get("json")
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected %s JSON shape: %v", typeOfValue.Name(), actual)
	}
}
