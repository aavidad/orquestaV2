package acceptance_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/tooling"
)

func TestV26TLS10PluginDescriptorBindsExactTLS01AndTLS06Contracts(t *testing.T) {
	toolSpec := tooling.CapabilitySpec{
		ID: "status.read", Version: "1",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":false}`),
		Permissions:  []identity.Permission{identity.PermissionArtifactsRead},
		Cost: tooling.CostContract{Mode: tooling.CostMaximum, Maximum: governance.ResourceVector{
			DiskBytes: 4096,
		}},
		Output:      tooling.OutputDelivery{MaxBytes: 4096, InlineBytes: 256},
		Idempotency: tooling.IdempotencyReadReexecute, Receipt: tooling.ReceiptObservation,
	}
	tools, err := tooling.NewRegistry(toolSpec)
	if err != nil {
		t.Fatal(err)
	}
	tool, _ := tools.Lookup(toolSpec.ID, toolSpec.Version)
	skill := v26GovernedSkill("review.governed", "1")
	skill.Spec.RequiredTools = []tooling.SkillToolRequirement{{
		ID: tool.Spec.ID, Version: tool.Spec.Version, SpecDigest: tool.Digest,
	}}
	skill.Spec.Permissions = []identity.Permission{identity.PermissionArtifactsRead}
	skillRegistry, err := tooling.NewSkillRegistry(tools, skill)
	if err != nil {
		t.Fatal(err)
	}
	registeredSkill, _ := skillRegistry.Lookup("review.governed", "1")
	contractDigest := "sha256:" + strings.Repeat("c", 64)
	configDigest := "sha256:" + strings.Repeat("d", 64)
	plugin := tooling.PluginSpec{
		Format: tooling.PluginDescriptorFormatV1,
		ID:     "forge.review", Version: "1", DescriptionKey: "plugin.forge.review.description",
		Capabilities: []string{"TLS-10"},
		Scopes:       []tooling.SkillScope{{Kind: tooling.SkillScopeProject, ProjectRef: "project:alpha"}},
		Config: &tooling.PluginConfigRequirement{
			ContractRef: "artifact:" + configDigest, ContractDigest: configDigest, SizeBytes: 2048,
		},
		Connectors: []tooling.PluginConnectorSpec{{
			ID: "forge.mutations", Version: "1", ContractRef: "artifact:" + contractDigest,
			ContractDigest: contractDigest,
			Permissions:    []identity.Permission{identity.PermissionChangesIntegrate},
		}},
		Tools: []tooling.PluginToolRequirement{{ID: tool.Spec.ID, Version: tool.Spec.Version, SpecDigest: tool.Digest}},
		Skills: []tooling.PluginSkillRequirement{{
			ID: registeredSkill.Spec.ID, Version: registeredSkill.Spec.Version,
			RegistrationDigest: registeredSkill.Digest,
		}},
		Permissions: []identity.Permission{identity.PermissionArtifactsRead, identity.PermissionChangesIntegrate},
	}
	catalog, err := tooling.NewPluginCatalog(tools, skillRegistry, plugin)
	if err != nil {
		t.Fatal(err)
	}
	registration, found := catalog.Lookup("forge.review", "1")
	encoded, encodeErr := json.Marshal(registration)
	if !found || registration.Digest == "" || registration.Spec.Format != tooling.PluginDescriptorFormatV1 ||
		len(registration.Spec.Capabilities) != 1 || len(registration.Spec.Scopes) != 1 ||
		registration.Spec.Config == nil || registration.Spec.Config.SizeBytes != 2048 ||
		len(registration.Spec.Connectors) != 1 || len(registration.Spec.Tools) != 1 ||
		len(registration.Spec.Skills) != 1 || registration.Spec.Skills[0].ReleaseDigest != "" ||
		encodeErr != nil || strings.Contains(string(encoded), "Governed instructions") {
		t.Fatalf("registration=%+v encoded=%s found=%v error=%v", registration, encoded, found, encodeErr)
	}
	if reflect.TypeOf(tooling.PluginSpec{}).NumField() != 11 {
		t.Fatalf("plugin descriptor shape changed: %v", reflect.TypeOf(tooling.PluginSpec{}))
	}
	forged := plugin
	forged.Skills = append([]tooling.PluginSkillRequirement(nil), plugin.Skills...)
	forged.Skills[0].RegistrationDigest = "sha256:" + strings.Repeat("0", 64)
	if result, err := tooling.NewPluginCatalog(tools, skillRegistry, forged); result != nil || tooling.ErrorCode(err) != tooling.ErrorPluginSpecInvalid {
		t.Fatalf("forged catalog=%v error=%v", result, err)
	}
}
