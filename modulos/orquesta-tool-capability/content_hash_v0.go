package orquestatoolcapability

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CalculateToolBundleContentHashV0 hashes a provider-neutral canonical value.
// Every repeated field is a set: validation rejects duplicates and hashing
// sorts equivalent declarations before encoding them.
func CalculateToolBundleContentHashV0(manifest CapabilityManifestV0, bundle ToolBundleV0) string {
	canonical := map[string]any{
		"contract": "orquesta.tool_bundle.content.v0",
		"manifest": canonicalToolCapabilityManifestV0(manifest),
		"bundle":   canonicalToolBundleV0(bundle),
	}
	var encoded strings.Builder
	writeCanonicalToolCapabilityValueV0(&encoded, canonical)
	digest := sha256.Sum256([]byte(encoded.String()))
	return fmt.Sprintf("sha256:%x", digest)
}

// CalculateGeneratedAppToolPlanFingerprintV0 binds immutable authority and
// resolved content. Snapshot.HandleRef is deliberately excluded: a resolver
// may reissue a safe handle for the same content-addressed snapshot on replay.
func CalculateGeneratedAppToolPlanFingerprintV0(plan GeneratedAppToolAttachPlanV0) string {
	canonical := map[string]any{
		"contract":             "orquesta.generated_app_tool_plan_fingerprint.v0",
		"operation":            plan.Operation,
		"idempotency_key":      plan.IdempotencyKey,
		"manifest_ref":         plan.Manifest.ManifestRef,
		"capability_ref":       plan.Manifest.CapabilityRef,
		"tool_ref":             plan.Manifest.ToolRef,
		"bundle_ref":           plan.Bundle.BundleRef,
		"bundle_content_hash":  plan.Bundle.ContentHash,
		"snapshot_ref":         plan.Snapshot.SnapshotRef,
		"binding":              canonicalGeneratedAppToolBindingV0(plan.Binding),
		"integration_mode":     plan.IntegrationMode,
		"config_ref":           plan.ConfigRef,
		"write_set":            canonicalToolCapabilityStringsV0(plan.WriteSet),
		"previous_receipt_ref": plan.PreviousReceiptRef,
	}
	var encoded strings.Builder
	writeCanonicalToolCapabilityValueV0(&encoded, canonical)
	digest := sha256.Sum256([]byte(encoded.String()))
	return fmt.Sprintf("sha256:%x", digest)
}

func canonicalToolCapabilityManifestV0(manifest CapabilityManifestV0) map[string]any {
	return map[string]any{
		"schema_version":    manifest.SchemaVersion,
		"manifest_ref":      manifest.ManifestRef,
		"capability_ref":    manifest.CapabilityRef,
		"tool_ref":          manifest.ToolRef,
		"version":           manifest.Version,
		"input_schema_ref":  manifest.InputSchemaRef,
		"output_schema_ref": manifest.OutputSchemaRef,
		"locales":           canonicalToolCapabilityStringsV0(manifest.Locales),
		"permissions":       canonicalToolCapabilityStringsV0(manifest.Permissions),
		"effect_profile": map[string]any{
			"profile_ref": manifest.EffectProfile.ProfileRef,
			"effect_refs": canonicalToolCapabilityStringsV0(manifest.EffectProfile.EffectRefs),
		},
		"budgets": map[string]any{
			"budget_refs": canonicalToolCapabilityStringsV0(manifest.Budgets.BudgetRefs),
		},
		"idempotency": map[string]any{
			"key_schema_ref": manifest.Idempotency.KeySchemaRef,
			"scope_ref":      manifest.Idempotency.ScopeRef,
		},
		"connector_slots":    canonicalToolCapabilityConnectorSlotsV0(manifest.ConnectorSlots),
		"config_schema_ref":  manifest.ConfigSchemaRef,
		"secret_refs":        canonicalToolCapabilityStringsV0(manifest.SecretRefs),
		"dependency_refs":    canonicalToolCapabilityStringsV0(manifest.DependencyRefs),
		"license_refs":       canonicalToolCapabilityStringsV0(manifest.LicenseRefs),
		"test_refs":          canonicalToolCapabilityStringsV0(manifest.TestRefs),
		"health_check_refs":  canonicalToolCapabilityStringsV0(manifest.HealthCheckRefs),
		"receipt_schema_ref": manifest.ReceiptSchemaRef,
		"integration_modes":  canonicalToolCapabilityStringsV0(manifest.IntegrationModes),
		"compatibility":      canonicalToolCapabilityCompatibilityV0(manifest.Compatibility),
	}
}

func canonicalToolBundleV0(bundle ToolBundleV0) map[string]any {
	return map[string]any{
		"schema_version": bundle.SchemaVersion,
		"bundle_ref":     bundle.BundleRef,
		"manifest_ref":   bundle.ManifestRef,
		"capability_ref": bundle.CapabilityRef,
		"tool_ref":       bundle.ToolRef,
		"version":        bundle.Version,
		"artifact_refs":  canonicalToolCapabilityArtifactsV0(bundle.ArtifactRefs),
		"i18n_catalogs":  canonicalToolCapabilityCatalogsV0(bundle.I18nCatalogs),
		"migration_refs": canonicalToolCapabilityStringsV0(bundle.MigrationRefs),
		"test_refs":      canonicalToolCapabilityStringsV0(bundle.TestRefs),
		"compatibility":  canonicalToolCapabilityCompatibilityV0(bundle.Compatibility),
	}
}

func canonicalToolCapabilityCompatibilityV0(value ToolCompatibilityV0) map[string]any {
	return map[string]any{
		"app_contract_refs": canonicalToolCapabilityStringsV0(value.AppContractRefs),
		"runtime_refs":      canonicalToolCapabilityStringsV0(value.RuntimeRefs),
		"upgrade_from_refs": canonicalToolCapabilityStringsV0(value.UpgradeFromRefs),
	}
}

func canonicalGeneratedAppToolBindingV0(binding GeneratedAppToolBindingV0) map[string]any {
	return map[string]any{
		"schema_version":              binding.SchemaVersion,
		"binding_ref":                 binding.BindingRef,
		"app_ref":                     binding.AppRef,
		"app_version":                 binding.AppVersion,
		"architecture_contract_ref":   binding.ArchitectureContractRef,
		"port_refs":                   canonicalToolCapabilityStringsV0(binding.PortRefs),
		"adapter_refs":                canonicalToolCapabilityStringsV0(binding.AdapterRefs),
		"composition_ref":             binding.CompositionRef,
		"supported_integration_modes": canonicalToolCapabilityStringsV0(binding.SupportedIntegrationModes),
		"config_schema_refs":          canonicalToolCapabilityStringsV0(binding.ConfigSchemaRefs),
		"i18n_catalog_refs":           canonicalToolCapabilityStringsV0(binding.I18nCatalogRefs),
		"allowed_effect_profile_refs": canonicalToolCapabilityStringsV0(binding.AllowedEffectProfileRefs),
		"allowed_write_set":           canonicalToolCapabilityStringsV0(binding.AllowedWriteSet),
		"compatibility_refs":          canonicalToolCapabilityStringsV0(binding.CompatibilityRefs),
	}
}

func canonicalToolCapabilityStringsV0(values []string) []any {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	result := make([]any, len(sorted))
	for index := range sorted {
		result[index] = sorted[index]
	}
	return result
}

func canonicalToolCapabilityConnectorSlotsV0(values []ConnectorSlotV0) []any {
	sorted := append([]ConnectorSlotV0(nil), values...)
	sort.Slice(sorted, func(left, right int) bool {
		if sorted[left].SlotRef != sorted[right].SlotRef {
			return sorted[left].SlotRef < sorted[right].SlotRef
		}
		if sorted[left].ContractRef != sorted[right].ContractRef {
			return sorted[left].ContractRef < sorted[right].ContractRef
		}
		return !sorted[left].Required && sorted[right].Required
	})
	result := make([]any, len(sorted))
	for index, slot := range sorted {
		result[index] = map[string]any{
			"slot_ref":        slot.SlotRef,
			"contract_ref":    slot.ContractRef,
			"required":        slot.Required,
			"capability_refs": canonicalToolCapabilityStringsV0(slot.CapabilityRefs),
		}
	}
	return result
}

func canonicalToolCapabilityArtifactsV0(values []HashedArtifactV0) []any {
	sorted := append([]HashedArtifactV0(nil), values...)
	sort.Slice(sorted, func(left, right int) bool {
		if sorted[left].ArtifactRef != sorted[right].ArtifactRef {
			return sorted[left].ArtifactRef < sorted[right].ArtifactRef
		}
		return sorted[left].ContentHash < sorted[right].ContentHash
	})
	result := make([]any, len(sorted))
	for index, artifact := range sorted {
		result[index] = map[string]any{"artifact_ref": artifact.ArtifactRef, "content_hash": artifact.ContentHash}
	}
	return result
}

func canonicalToolCapabilityCatalogsV0(values []I18nCatalogRefV0) []any {
	sorted := append([]I18nCatalogRefV0(nil), values...)
	sort.Slice(sorted, func(left, right int) bool {
		if sorted[left].Locale != sorted[right].Locale {
			return sorted[left].Locale < sorted[right].Locale
		}
		if sorted[left].CatalogRef != sorted[right].CatalogRef {
			return sorted[left].CatalogRef < sorted[right].CatalogRef
		}
		return sorted[left].ContentHash < sorted[right].ContentHash
	})
	result := make([]any, len(sorted))
	for index, catalog := range sorted {
		result[index] = map[string]any{
			"locale": catalog.Locale, "catalog_ref": catalog.CatalogRef, "content_hash": catalog.ContentHash,
		}
	}
	return result
}

func writeCanonicalToolCapabilityValueV0(output *strings.Builder, value any) {
	switch typed := value.(type) {
	case string:
		output.WriteByte('s')
		output.WriteString(strconv.Itoa(len([]byte(typed))))
		output.WriteByte(':')
		output.WriteString(typed)
		output.WriteByte(';')
	case bool:
		if typed {
			output.WriteString("b1;")
		} else {
			output.WriteString("b0;")
		}
	case []any:
		output.WriteByte('a')
		output.WriteString(strconv.Itoa(len(typed)))
		output.WriteByte('[')
		for _, item := range typed {
			writeCanonicalToolCapabilityValueV0(output, item)
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		output.WriteByte('o')
		output.WriteString(strconv.Itoa(len(keys)))
		output.WriteByte('{')
		for _, key := range keys {
			writeCanonicalToolCapabilityValueV0(output, key)
			writeCanonicalToolCapabilityValueV0(output, typed[key])
		}
		output.WriteByte('}')
	default:
		panic(fmt.Sprintf("unsupported canonical tool capability value %T", value))
	}
}
