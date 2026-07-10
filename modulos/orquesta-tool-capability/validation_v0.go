package orquestatoolcapability

import (
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	ErrToolCapabilityRefRequiredV0             = "tool_capability_ref_required"
	ErrToolCapabilityRefInvalidV0              = "tool_capability_ref_invalid"
	ErrToolCapabilityVersionInvalidV0          = "tool_capability_version_invalid"
	ErrToolCapabilityHashInvalidV0             = "tool_capability_hash_invalid"
	ErrToolCapabilityHashMismatchV0            = "tool_capability_hash_mismatch"
	ErrToolCapabilityDuplicateV0               = "tool_capability_duplicate"
	ErrToolCapabilityModeInvalidV0             = "tool_capability_mode_invalid"
	ErrToolCapabilityModeIncompatibleV0        = "tool_capability_mode_incompatible"
	ErrToolCapabilityBundleMismatchV0          = "tool_capability_bundle_mismatch"
	ErrToolCapabilityAuthorityMismatchV0       = "tool_capability_authority_mismatch"
	ErrToolCapabilitySnapshotInvalidV0         = "tool_capability_snapshot_invalid"
	ErrToolCapabilityPlanFingerprintMismatchV0 = "tool_capability_plan_fingerprint_mismatch"
	ErrToolCapabilityPlanUnresolvedV0          = "tool_capability_plan_unresolved"
	ErrToolCapabilityIdempotencyConflictV0     = "tool_capability_idempotency_conflict"
	ErrToolCapabilityTargetBusyV0              = "tool_capability_target_busy"
	ErrToolCapabilityI18nMissingV0             = "tool_capability_i18n_missing"
	ErrToolCapabilityConfigMissingV0           = "tool_capability_config_missing"
	ErrToolCapabilityWriteSetInvalidV0         = "tool_capability_write_set_invalid"
	ErrToolCapabilityArchitectureInvalidV0     = "tool_capability_architecture_invalid"
	ErrToolCapabilityEffectUnauthorizedV0      = "tool_capability_effect_unauthorized"
	ErrToolCapabilityOperationInvalidV0        = "tool_capability_operation_invalid"
	ErrToolCapabilityOperationNotFoundV0       = "tool_capability_operation_not_found"
	ErrToolCapabilityOperationStateInvalidV0   = "tool_capability_operation_state_invalid"
	ErrToolCapabilityReceiptStoreMissingV0     = "tool_capability_receipt_store_missing"
	ErrToolCapabilityReceiptCASConflictV0      = "tool_capability_receipt_cas_conflict"
	ErrToolCapabilityStatusScopeRequiredV0     = "tool_capability_status_scope_required"
	ErrToolCapabilityMaterializationFailedV0   = "tool_capability_materialization_failed"
	ErrToolCapabilityUpgradeFailedV0           = "tool_capability_upgrade_failed"
	ErrToolCapabilityUpgradeIncompatibleV0     = "tool_capability_upgrade_incompatible"
	ErrToolCapabilityRollbackFailedV0          = "tool_capability_rollback_failed"
	ErrToolCapabilityPreviousReceiptInvalidV0  = "tool_capability_previous_receipt_invalid"
	ErrToolCapabilityReconciliationUnknownV0   = "tool_capability_reconciliation_unknown"
	ErrToolCapabilityPortUnavailableV0         = "tool_capability_port_unavailable"
)

var semverToolCapabilityV0 = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)
var sha256ToolCapabilityV0 = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var uriSchemeToolCapabilityV0 = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

func ValidateCapabilityManifestV0(manifest CapabilityManifestV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilitySchemaV0(manifest.SchemaVersion, CapabilityManifestSchemaV0, "schema_version")...)
	for field, value := range map[string]string{
		"manifest_ref": manifest.ManifestRef, "capability_ref": manifest.CapabilityRef,
		"tool_ref": manifest.ToolRef, "input_schema_ref": manifest.InputSchemaRef,
		"output_schema_ref": manifest.OutputSchemaRef, "config_schema_ref": manifest.ConfigSchemaRef,
		"receipt_schema_ref": manifest.ReceiptSchemaRef, "effect_profile.profile_ref": manifest.EffectProfile.ProfileRef,
		"idempotency.key_schema_ref": manifest.Idempotency.KeySchemaRef, "idempotency.scope_ref": manifest.Idempotency.ScopeRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	if !semverToolCapabilityV0.MatchString(manifest.Version) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityVersionInvalidV0, "version"))
	}
	issues = append(issues, validateToolCapabilityRefsV0(manifest.Locales, "locales", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.Permissions, "permissions", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.EffectProfile.EffectRefs, "effect_profile.effect_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.Budgets.BudgetRefs, "budgets.budget_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.SecretRefs, "secret_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.DependencyRefs, "dependency_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.LicenseRefs, "license_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.TestRefs, "test_refs", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(manifest.HealthCheckRefs, "health_check_refs", false)...)
	issues = append(issues, validateIntegrationModesV0(manifest.IntegrationModes, "integration_modes", true)...)
	issues = append(issues, validateCompatibilityV0(manifest.Compatibility)...)
	seenSlots := make(map[string]struct{}, len(manifest.ConnectorSlots))
	for _, slot := range manifest.ConnectorSlots {
		issues = append(issues, requiredToolCapabilityRefV0(slot.SlotRef, "connector_slots.slot_ref")...)
		issues = append(issues, requiredToolCapabilityRefV0(slot.ContractRef, "connector_slots.contract_ref")...)
		issues = append(issues, validateToolCapabilityRefsV0(slot.CapabilityRefs, "connector_slots.capability_refs", false)...)
		if _, duplicate := seenSlots[slot.SlotRef]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "connector_slots.slot_ref"))
		}
		seenSlots[slot.SlotRef] = struct{}{}
	}
	return issues
}

func ValidateToolBundleV0(manifest CapabilityManifestV0, bundle ToolBundleV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilitySchemaV0(bundle.SchemaVersion, ToolBundleSchemaV0, "schema_version")...)
	for field, value := range map[string]string{
		"bundle_ref": bundle.BundleRef, "manifest_ref": bundle.ManifestRef,
		"capability_ref": bundle.CapabilityRef, "tool_ref": bundle.ToolRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	if !semverToolCapabilityV0.MatchString(bundle.Version) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityVersionInvalidV0, "version"))
	}
	if !sha256ToolCapabilityV0.MatchString(bundle.ContentHash) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityHashInvalidV0, "content_hash"))
	}
	if len(bundle.ArtifactRefs) == 0 {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityRefRequiredV0, "artifact_refs"))
	}
	seenArtifacts := make(map[string]struct{}, len(bundle.ArtifactRefs))
	seenHashes := make(map[string]struct{}, len(bundle.ArtifactRefs)+len(bundle.I18nCatalogs))
	for _, artifact := range bundle.ArtifactRefs {
		issues = append(issues, requiredToolCapabilityRefV0(artifact.ArtifactRef, "artifact_refs.artifact_ref")...)
		if !sha256ToolCapabilityV0.MatchString(artifact.ContentHash) {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityHashInvalidV0, "artifact_refs.content_hash"))
		}
		if _, duplicate := seenArtifacts[artifact.ArtifactRef]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "artifact_refs.artifact_ref"))
		}
		seenArtifacts[artifact.ArtifactRef] = struct{}{}
		if _, duplicate := seenHashes[artifact.ContentHash]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "artifact_refs.content_hash"))
		}
		seenHashes[artifact.ContentHash] = struct{}{}
	}
	if len(bundle.I18nCatalogs) == 0 {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityI18nMissingV0, "i18n_catalogs"))
	}
	seenCatalogRefs := make(map[string]struct{}, len(bundle.I18nCatalogs))
	seenCatalogLocales := make(map[string]struct{}, len(bundle.I18nCatalogs))
	for _, catalog := range bundle.I18nCatalogs {
		issues = append(issues, requiredToolCapabilityRefV0(catalog.Locale, "i18n_catalogs.locale")...)
		issues = append(issues, requiredToolCapabilityRefV0(catalog.CatalogRef, "i18n_catalogs.catalog_ref")...)
		if !sha256ToolCapabilityV0.MatchString(catalog.ContentHash) {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityHashInvalidV0, "i18n_catalogs.content_hash"))
		}
		if _, duplicate := seenCatalogRefs[catalog.CatalogRef]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "i18n_catalogs.catalog_ref"))
		}
		seenCatalogRefs[catalog.CatalogRef] = struct{}{}
		if _, duplicate := seenCatalogLocales[catalog.Locale]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "i18n_catalogs.locale"))
		}
		seenCatalogLocales[catalog.Locale] = struct{}{}
		if _, duplicate := seenHashes[catalog.ContentHash]; duplicate {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, "i18n_catalogs.content_hash"))
		}
		seenHashes[catalog.ContentHash] = struct{}{}
	}
	issues = append(issues, validateToolCapabilityRefsV0(bundle.MigrationRefs, "migration_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(bundle.TestRefs, "test_refs", true)...)
	issues = append(issues, validateCompatibilityV0(bundle.Compatibility)...)
	if sha256ToolCapabilityV0.MatchString(bundle.ContentHash) && bundle.ContentHash != CalculateToolBundleContentHashV0(manifest, bundle) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityHashMismatchV0, "content_hash"))
	}
	return issues
}

func ValidateToolBundleSnapshotV0(manifest CapabilityManifestV0, bundle ToolBundleV0, snapshot ToolBundleSnapshotV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilitySchemaV0(snapshot.SchemaVersion, ToolBundleSnapshotSchemaV0, "snapshot.schema_version")...)
	for field, value := range map[string]string{
		"snapshot.snapshot_ref": snapshot.SnapshotRef, "snapshot.handle_ref": snapshot.HandleRef,
		"snapshot.verification_ref": snapshot.VerificationRef, "snapshot.manifest_ref": snapshot.ManifestRef,
		"snapshot.bundle_ref": snapshot.BundleRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	if !sha256ToolCapabilityV0.MatchString(snapshot.ContentHash) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityHashInvalidV0, "snapshot.content_hash"))
	}
	if snapshot.ManifestRef != manifest.ManifestRef || snapshot.BundleRef != bundle.BundleRef ||
		snapshot.ContentHash != bundle.ContentHash || snapshot.SnapshotRef != ToolBundleSnapshotRefV0(bundle.ContentHash) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilitySnapshotInvalidV0, "snapshot"))
	}
	return issues
}

func ValidateGeneratedAppToolBindingV0(binding GeneratedAppToolBindingV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilitySchemaV0(binding.SchemaVersion, GeneratedAppToolBindingSchemaV0, "schema_version")...)
	for field, value := range map[string]string{
		"binding_ref": binding.BindingRef, "app_ref": binding.AppRef,
		"app_version": binding.AppVersion, "architecture_contract_ref": binding.ArchitectureContractRef,
		"composition_ref": binding.CompositionRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	issues = append(issues, validateToolCapabilityRefsV0(binding.PortRefs, "port_refs", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(binding.AdapterRefs, "adapter_refs", true)...)
	issues = append(issues, validateIntegrationModesV0(binding.SupportedIntegrationModes, "supported_integration_modes", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(binding.ConfigSchemaRefs, "config_schema_refs", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(binding.I18nCatalogRefs, "i18n_catalog_refs", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(binding.AllowedEffectProfileRefs, "allowed_effect_profile_refs", true)...)
	issues = append(issues, validateToolCapabilityRefsV0(binding.CompatibilityRefs, "compatibility_refs", false)...)
	issues = append(issues, validateToolCapabilityWriteSetV0(binding.AllowedWriteSet, "allowed_write_set", true)...)
	return issues
}

func ValidateGeneratedAppToolAttachRequestV0(request GeneratedAppToolAttachRequestV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	for field, value := range map[string]string{
		"request_ref": request.RequestRef, "idempotency_key": request.IdempotencyKey,
		"requested_by": request.RequestedBy, "app_ref": request.AppRef,
		"binding_ref": request.BindingRef, "manifest_ref": request.ManifestRef,
		"bundle_ref": request.BundleRef, "config_ref": request.ConfigRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	if !validToolOperationV0(request.Operation) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityOperationInvalidV0, "operation"))
	}
	if request.Operation == ToolOperationUpgradeV0 && request.PreviousReceiptRef == "" {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityPreviousReceiptInvalidV0, "previous_receipt_ref"))
	}
	issues = append(issues, validateIntegrationModesV0([]string{request.IntegrationMode}, "integration_mode", true)...)
	issues = append(issues, validateToolCapabilityWriteSetV0(request.WriteSet, "write_set", true)...)
	if request.PreviousReceiptRef != "" {
		issues = append(issues, requiredToolCapabilityRefV0(request.PreviousReceiptRef, "previous_receipt_ref")...)
	}
	return issues
}

func ValidateToolOperationRecordV0(record ToolOperationRecordV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilitySchemaV0(record.SchemaVersion, ToolOperationRecordSchemaV0, "record.schema_version")...)
	plan := record.Plan
	receipt := record.Receipt
	issues = append(issues, validateToolCapabilitySchemaV0(plan.SchemaVersion, ToolAttachPlanSchemaV0, "plan.schema_version")...)
	issues = append(issues, validateToolCapabilitySchemaV0(receipt.SchemaVersion, ToolOperationReceiptSchemaV0, "receipt.schema_version")...)
	if !sha256ToolCapabilityV0.MatchString(plan.PlanFingerprint) || plan.PlanFingerprint != CalculateGeneratedAppToolPlanFingerprintV0(plan) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityPlanFingerprintMismatchV0, "plan.plan_fingerprint"))
	}
	for field, value := range map[string]string{
		"plan.plan_ref": plan.PlanRef, "plan.operation_ref": plan.OperationRef,
		"plan.target_ref": plan.TargetRef, "receipt.receipt_ref": receipt.ReceiptRef,
		"receipt.operation_ref": receipt.OperationRef, "receipt.target_ref": receipt.TargetRef,
	} {
		issues = append(issues, requiredToolCapabilityRefV0(value, field)...)
	}
	if receipt.StateVersion == 0 || !validToolReceiptStatusV0(receipt.Status) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityOperationStateInvalidV0, "receipt.status"))
	}
	if ToolReceiptStatusIsTargetTerminalV0(receipt.Status) && receipt.RecoveryRef != "" {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityOperationStateInvalidV0, "receipt.recovery_ref"))
	}
	if !ToolReceiptStatusIsTargetTerminalV0(receipt.Status) && receipt.RecoveryRef == "" {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityOperationStateInvalidV0, "receipt.recovery_ref"))
	}
	if plan.PlanRef != ToolAttachPlanRefV0(plan.PlanFingerprint) || receipt.PlanRef != plan.PlanRef ||
		receipt.PlanFingerprint != plan.PlanFingerprint || receipt.OperationRef != plan.OperationRef ||
		receipt.TargetRef != plan.TargetRef || receipt.Operation != plan.Operation ||
		receipt.IdempotencyKey != plan.IdempotencyKey ||
		receipt.AppRef != plan.Binding.AppRef || receipt.BindingRef != plan.Binding.BindingRef ||
		receipt.ToolRef != plan.Manifest.ToolRef || receipt.CapabilityRef != plan.Manifest.CapabilityRef ||
		receipt.ManifestRef != plan.Manifest.ManifestRef || receipt.BundleRef != plan.Bundle.BundleRef ||
		receipt.ContentHash != plan.Bundle.ContentHash || receipt.ConfigRef != plan.ConfigRef ||
		receipt.SnapshotRef != plan.Snapshot.SnapshotRef || receipt.SnapshotHandleRef != plan.Snapshot.HandleRef ||
		receipt.PreviousReceiptRef != plan.PreviousReceiptRef {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityPlanFingerprintMismatchV0, "record"))
	}
	issues = append(issues, validateToolCapabilityRefsV0(receipt.EvidenceRefs, "receipt.evidence_refs", false)...)
	if receipt.Status != ToolReceiptStatusBlockedV0 {
		issues = append(issues, ValidateGeneratedAppToolBindingV0(plan.Binding)...)
		issues = append(issues, ValidateCapabilityManifestV0(plan.Manifest)...)
		issues = append(issues, ValidateToolBundleV0(plan.Manifest, plan.Bundle)...)
		issues = append(issues, ValidateToolBundleSnapshotV0(plan.Manifest, plan.Bundle, plan.Snapshot)...)
		issues = append(issues, validateToolAttachCompatibilityV0(plan)...)
		if len(plan.Issues) > 0 || !plan.EffectAuthorized {
			issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityOperationStateInvalidV0, "plan.issues"))
		}
	}
	return issues
}

func validateToolAttachCompatibilityV0(plan GeneratedAppToolAttachPlanV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	if plan.Manifest.ManifestRef != plan.Bundle.ManifestRef ||
		plan.Manifest.CapabilityRef != plan.Bundle.CapabilityRef ||
		plan.Manifest.ToolRef != plan.Bundle.ToolRef ||
		plan.Manifest.Version != plan.Bundle.Version {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityBundleMismatchV0, "bundle"))
	}
	if !containsToolCapabilityStringV0(plan.Manifest.IntegrationModes, plan.IntegrationMode) ||
		!containsToolCapabilityStringV0(plan.Binding.SupportedIntegrationModes, plan.IntegrationMode) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityModeIncompatibleV0, "integration_mode"))
	}
	if !containsToolCapabilityStringV0(plan.Binding.ConfigSchemaRefs, plan.Manifest.ConfigSchemaRef) || plan.ConfigRef == "" {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityConfigMissingV0, "config_ref"))
	}
	if !sameToolCapabilityLocaleSetV0(plan.Manifest.Locales, toolCapabilityCatalogLocalesV0(plan.Bundle.I18nCatalogs)) || len(plan.Binding.I18nCatalogRefs) == 0 {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityI18nMissingV0, "i18n"))
	}
	if !toolCapabilityWriteSetWithinV0(plan.WriteSet, plan.Binding.AllowedWriteSet) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityWriteSetInvalidV0, "write_set"))
	}
	if !containsToolCapabilityStringV0(plan.Binding.AllowedEffectProfileRefs, plan.Manifest.EffectProfile.ProfileRef) {
		issues = append(issues, toolCapabilityIssueV0(ErrToolCapabilityEffectUnauthorizedV0, "effect_profile"))
	}
	return issues
}

func ToolBundleSnapshotRefV0(contentHash string) string {
	return "tool-bundle-snapshot-" + strings.TrimPrefix(contentHash, "sha256:")
}

func ToolAttachPlanRefV0(fingerprint string) string {
	return "tool-attach-plan-" + strings.TrimPrefix(fingerprint, "sha256:")
}

func ToolReceiptStatusIsTargetTerminalV0(status string) bool {
	switch status {
	case ToolReceiptStatusPreparedV0, ToolReceiptStatusAttachedV0, ToolReceiptStatusUpgradedV0,
		ToolReceiptStatusUninstalledV0, ToolReceiptStatusRolledBackV0, ToolReceiptStatusBlockedV0:
		return true
	default:
		return false
	}
}

func validToolReceiptStatusV0(status string) bool {
	return status == ToolReceiptStatusClaimedV0 || status == ToolReceiptStatusResumingV0 ||
		status == ToolReceiptStatusRecoveryRequiredV0 || ToolReceiptStatusIsTargetTerminalV0(status)
}

func validateToolCapabilitySchemaV0(value, expected, field string) []ToolCapabilityIssueV0 {
	if value != expected {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityRefInvalidV0, field)}
	}
	return nil
}

func requiredToolCapabilityRefV0(value, field string) []ToolCapabilityIssueV0 {
	if strings.TrimSpace(value) == "" {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityRefRequiredV0, field)}
	}
	if !compactToolCapabilityRefV0(value) {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityRefInvalidV0, field)}
	}
	return nil
}

func validateToolCapabilityRefsV0(values []string, field string, required bool) []ToolCapabilityIssueV0 {
	if required && len(values) == 0 {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityRefRequiredV0, field)}
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !compactToolCapabilityRefV0(value) {
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityRefInvalidV0, field)}
		}
		if _, duplicate := seen[value]; duplicate {
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, field)}
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateIntegrationModesV0(values []string, field string, required bool) []ToolCapabilityIssueV0 {
	if required && len(values) == 0 {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityModeInvalidV0, field)}
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		switch value {
		case IntegrationModeEmbeddedModuleV0, IntegrationModeLocalSidecarV0, IntegrationModeRemoteConnectorV0:
		default:
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityModeInvalidV0, field)}
		}
		if _, duplicate := seen[value]; duplicate {
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, field)}
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateCompatibilityV0(compatibility ToolCompatibilityV0) []ToolCapabilityIssueV0 {
	var issues []ToolCapabilityIssueV0
	issues = append(issues, validateToolCapabilityRefsV0(compatibility.AppContractRefs, "compatibility.app_contract_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(compatibility.RuntimeRefs, "compatibility.runtime_refs", false)...)
	issues = append(issues, validateToolCapabilityRefsV0(compatibility.UpgradeFromRefs, "compatibility.upgrade_from_refs", false)...)
	return issues
}

func validateToolCapabilityWriteSetV0(values []string, field string, required bool) []ToolCapabilityIssueV0 {
	if required && len(values) == 0 {
		return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityWriteSetInvalidV0, field)}
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !safeToolCapabilityWriteSetV0(value) {
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityWriteSetInvalidV0, field)}
		}
		if _, duplicate := seen[value]; duplicate {
			return []ToolCapabilityIssueV0{toolCapabilityIssueV0(ErrToolCapabilityDuplicateV0, field)}
		}
		seen[value] = struct{}{}
	}
	return nil
}

func compactToolCapabilityRefV0(value string) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != "" &&
		!strings.ContainsRune(value, '\x00') && !strings.ContainsAny(value, " /\\\t\n\r")
}

func safeToolCapabilityWriteSetV0(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed != value || strings.ContainsRune(trimmed, '\x00') ||
		strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") ||
		strings.HasPrefix(trimmed, "~") || strings.Contains(trimmed, "\\") ||
		uriSchemeToolCapabilityV0.MatchString(trimmed) {
		return false
	}
	clean := path.Clean(trimmed)
	if clean == "." || clean != trimmed {
		return false
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func validToolOperationV0(value string) bool {
	switch value {
	case ToolOperationPrepareV0, ToolOperationAttachV0, ToolOperationUpgradeV0, ToolOperationUninstallV0:
		return true
	default:
		return false
	}
}

func containsToolCapabilityStringV0(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func sameToolCapabilityLocaleSetV0(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]bool, len(left))
	for _, value := range left {
		seen[value] = true
	}
	for _, value := range right {
		if !seen[value] {
			return false
		}
		delete(seen, value)
	}
	return len(seen) == 0
}

func toolCapabilityCatalogLocalesV0(catalogs []I18nCatalogRefV0) []string {
	locales := make([]string, 0, len(catalogs))
	for _, catalog := range catalogs {
		locales = append(locales, catalog.Locale)
	}
	return locales
}

func toolCapabilityWriteSetWithinV0(requested, allowed []string) bool {
	if len(requested) == 0 || len(allowed) == 0 {
		return false
	}
	for _, requestedEntry := range requested {
		if !safeToolCapabilityWriteSetV0(requestedEntry) {
			return false
		}
		requestedSegments := strings.Split(requestedEntry, "/")
		allowedEntry := false
		for _, allowedRoot := range allowed {
			if !safeToolCapabilityWriteSetV0(allowedRoot) {
				return false
			}
			if toolCapabilityPathSegmentsPrefixV0(requestedSegments, strings.Split(allowedRoot, "/")) {
				allowedEntry = true
				break
			}
		}
		if !allowedEntry {
			return false
		}
	}
	return true
}

func toolCapabilityPathSegmentsPrefixV0(pathSegments, rootSegments []string) bool {
	if len(rootSegments) > len(pathSegments) {
		return false
	}
	for index := range rootSegments {
		if pathSegments[index] != rootSegments[index] {
			return false
		}
	}
	return true
}

func toolCapabilityIssueV0(code, field string) ToolCapabilityIssueV0 {
	return ToolCapabilityIssueV0{Code: code, Field: field}
}
