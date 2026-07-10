package orquestatoolcapability

import "context"

const (
	CapabilityManifestSchemaV0      = "capability_manifest.v0"
	ToolBundleSchemaV0              = "tool_bundle.v0"
	ToolBundleSnapshotSchemaV0      = "tool_bundle_snapshot.v0"
	GeneratedAppToolBindingSchemaV0 = "generated_app_tool_binding.v0"
	ToolAttachPlanSchemaV0          = "generated_app_tool_attach_plan.v0"
	ToolOperationReceiptSchemaV0    = "tool_operation_receipt.v0"
	ToolOperationRecordSchemaV0     = "tool_operation_record.v0"

	IntegrationModeEmbeddedModuleV0  = "embedded_module"
	IntegrationModeLocalSidecarV0    = "local_sidecar"
	IntegrationModeRemoteConnectorV0 = "remote_connector"

	ToolOperationPrepareV0   = "prepare"
	ToolOperationAttachV0    = "attach"
	ToolOperationUpgradeV0   = "upgrade"
	ToolOperationUninstallV0 = "uninstall"

	ToolReceiptStatusClaimedV0          = "claimed"
	ToolReceiptStatusResumingV0         = "resuming"
	ToolReceiptStatusPreparedV0         = "prepared"
	ToolReceiptStatusAttachedV0         = "attached"
	ToolReceiptStatusUpgradedV0         = "upgraded"
	ToolReceiptStatusUninstalledV0      = "uninstalled"
	ToolReceiptStatusRolledBackV0       = "rolled_back"
	ToolReceiptStatusBlockedV0          = "blocked"
	ToolReceiptStatusRecoveryRequiredV0 = "recovery_required"

	ToolReconcileDispositionAppliedV0    = "applied"
	ToolReconcileDispositionNotAppliedV0 = "not_applied"
	ToolReconcileDispositionUnknownV0    = "unknown"
)

type CapabilityManifestV0 struct {
	SchemaVersion    string                    `json:"schema_version"`
	ManifestRef      string                    `json:"manifest_ref"`
	CapabilityRef    string                    `json:"capability_ref"`
	ToolRef          string                    `json:"tool_ref"`
	Version          string                    `json:"version"`
	InputSchemaRef   string                    `json:"input_schema_ref"`
	OutputSchemaRef  string                    `json:"output_schema_ref"`
	Locales          []string                  `json:"locales"`
	Permissions      []string                  `json:"permissions,omitempty"`
	EffectProfile    CapabilityEffectProfileV0 `json:"effect_profile"`
	Budgets          CapabilityBudgetsV0       `json:"budgets"`
	Idempotency      CapabilityIdempotencyV0   `json:"idempotency"`
	ConnectorSlots   []ConnectorSlotV0         `json:"connector_slots,omitempty"`
	ConfigSchemaRef  string                    `json:"config_schema_ref"`
	SecretRefs       []string                  `json:"secret_refs,omitempty"`
	DependencyRefs   []string                  `json:"dependency_refs,omitempty"`
	LicenseRefs      []string                  `json:"license_refs,omitempty"`
	TestRefs         []string                  `json:"test_refs"`
	HealthCheckRefs  []string                  `json:"health_check_refs,omitempty"`
	ReceiptSchemaRef string                    `json:"receipt_schema_ref"`
	IntegrationModes []string                  `json:"integration_modes"`
	Compatibility    ToolCompatibilityV0       `json:"compatibility"`
}

type CapabilityEffectProfileV0 struct {
	ProfileRef string   `json:"profile_ref"`
	EffectRefs []string `json:"effect_refs,omitempty"`
}

type CapabilityBudgetsV0 struct {
	BudgetRefs []string `json:"budget_refs,omitempty"`
}

type CapabilityIdempotencyV0 struct {
	KeySchemaRef string `json:"key_schema_ref"`
	ScopeRef     string `json:"scope_ref"`
}

type ConnectorSlotV0 struct {
	SlotRef        string   `json:"slot_ref"`
	ContractRef    string   `json:"contract_ref"`
	Required       bool     `json:"required"`
	CapabilityRefs []string `json:"capability_refs,omitempty"`
}

type ToolCompatibilityV0 struct {
	AppContractRefs []string `json:"app_contract_refs,omitempty"`
	RuntimeRefs     []string `json:"runtime_refs,omitempty"`
	UpgradeFromRefs []string `json:"upgrade_from_refs,omitempty"`
}

type ToolBundleV0 struct {
	SchemaVersion string              `json:"schema_version"`
	BundleRef     string              `json:"bundle_ref"`
	ManifestRef   string              `json:"manifest_ref"`
	CapabilityRef string              `json:"capability_ref"`
	ToolRef       string              `json:"tool_ref"`
	Version       string              `json:"version"`
	ContentHash   string              `json:"content_hash"`
	ArtifactRefs  []HashedArtifactV0  `json:"artifact_refs"`
	I18nCatalogs  []I18nCatalogRefV0  `json:"i18n_catalogs"`
	MigrationRefs []string            `json:"migration_refs,omitempty"`
	TestRefs      []string            `json:"test_refs"`
	Compatibility ToolCompatibilityV0 `json:"compatibility"`
}

type HashedArtifactV0 struct {
	ArtifactRef string `json:"artifact_ref"`
	ContentHash string `json:"content_hash"`
}

type I18nCatalogRefV0 struct {
	Locale      string `json:"locale"`
	CatalogRef  string `json:"catalog_ref"`
	ContentHash string `json:"content_hash"`
}

// ToolBundleSnapshotV0 contains no path. HandleRef is an opaque capability
// owned by the resolver/materializer adapter pair.
type ToolBundleSnapshotV0 struct {
	SchemaVersion   string `json:"schema_version"`
	SnapshotRef     string `json:"snapshot_ref"`
	HandleRef       string `json:"handle_ref"`
	VerificationRef string `json:"verification_ref"`
	ManifestRef     string `json:"manifest_ref"`
	BundleRef       string `json:"bundle_ref"`
	ContentHash     string `json:"content_hash"`
}

type ResolvedToolBundleSnapshotV0 struct {
	Bundle   ToolBundleV0         `json:"bundle"`
	Snapshot ToolBundleSnapshotV0 `json:"snapshot"`
}

// GeneratedAppToolBindingV0 is authority-owned app configuration. Requests
// may reference it but cannot provide or alter it.
type GeneratedAppToolBindingV0 struct {
	SchemaVersion             string   `json:"schema_version"`
	BindingRef                string   `json:"binding_ref"`
	AppRef                    string   `json:"app_ref"`
	AppVersion                string   `json:"app_version"`
	ArchitectureContractRef   string   `json:"architecture_contract_ref"`
	PortRefs                  []string `json:"port_refs"`
	AdapterRefs               []string `json:"adapter_refs"`
	CompositionRef            string   `json:"composition_ref"`
	SupportedIntegrationModes []string `json:"supported_integration_modes"`
	ConfigSchemaRefs          []string `json:"config_schema_refs"`
	I18nCatalogRefs           []string `json:"i18n_catalog_refs"`
	AllowedEffectProfileRefs  []string `json:"allowed_effect_profile_refs"`
	AllowedWriteSet           []string `json:"allowed_write_set"`
	CompatibilityRefs         []string `json:"compatibility_refs,omitempty"`
}

// GeneratedAppToolAttachRequestV0 carries intent and opaque refs only.
type GeneratedAppToolAttachRequestV0 struct {
	RequestRef         string   `json:"request_ref"`
	Operation          string   `json:"operation"`
	IdempotencyKey     string   `json:"idempotency_key"`
	RequestedBy        string   `json:"requested_by"`
	AppRef             string   `json:"app_ref"`
	BindingRef         string   `json:"binding_ref"`
	IntegrationMode    string   `json:"integration_mode"`
	ManifestRef        string   `json:"manifest_ref"`
	BundleRef          string   `json:"bundle_ref"`
	ConfigRef          string   `json:"config_ref"`
	WriteSet           []string `json:"write_set"`
	PreviousReceiptRef string   `json:"previous_receipt_ref,omitempty"`
}

type GeneratedAppToolAttachPlanV0 struct {
	SchemaVersion      string                    `json:"schema_version"`
	PlanRef            string                    `json:"plan_ref"`
	PlanFingerprint    string                    `json:"plan_fingerprint"`
	OperationRef       string                    `json:"operation_ref"`
	TargetRef          string                    `json:"target_ref"`
	RequestRef         string                    `json:"request_ref"`
	Operation          string                    `json:"operation"`
	IdempotencyKey     string                    `json:"idempotency_key"`
	Manifest           CapabilityManifestV0      `json:"manifest"`
	Bundle             ToolBundleV0              `json:"bundle"`
	Snapshot           ToolBundleSnapshotV0      `json:"snapshot"`
	Binding            GeneratedAppToolBindingV0 `json:"binding"`
	IntegrationMode    string                    `json:"integration_mode"`
	ConfigRef          string                    `json:"config_ref"`
	WriteSet           []string                  `json:"write_set"`
	PreviousReceiptRef string                    `json:"previous_receipt_ref,omitempty"`
	DryRun             bool                      `json:"dry_run"`
	EffectAuthorized   bool                      `json:"effect_authorized"`
	Issues             []ToolCapabilityIssueV0   `json:"issues,omitempty"`
}

type ToolOperationReceiptV0 struct {
	SchemaVersion      string                  `json:"schema_version"`
	ReceiptRef         string                  `json:"receipt_ref"`
	OperationRef       string                  `json:"operation_ref"`
	TargetRef          string                  `json:"target_ref"`
	PlanRef            string                  `json:"plan_ref"`
	PlanFingerprint    string                  `json:"plan_fingerprint"`
	Operation          string                  `json:"operation"`
	Status             string                  `json:"status"`
	StateVersion       uint64                  `json:"state_version"`
	IdempotencyKey     string                  `json:"idempotency_key"`
	AppRef             string                  `json:"app_ref"`
	BindingRef         string                  `json:"binding_ref"`
	ToolRef            string                  `json:"tool_ref"`
	CapabilityRef      string                  `json:"capability_ref"`
	ManifestRef        string                  `json:"manifest_ref"`
	BundleRef          string                  `json:"bundle_ref"`
	ContentHash        string                  `json:"content_hash"`
	ConfigRef          string                  `json:"config_ref"`
	SnapshotRef        string                  `json:"snapshot_ref"`
	SnapshotHandleRef  string                  `json:"snapshot_handle_ref"`
	PreviousReceiptRef string                  `json:"previous_receipt_ref,omitempty"`
	RecoveryRef        string                  `json:"recovery_ref,omitempty"`
	EvidenceRefs       []string                `json:"evidence_refs,omitempty"`
	Issues             []ToolCapabilityIssueV0 `json:"issues,omitempty"`
}

type ToolOperationRecordV0 struct {
	SchemaVersion string                       `json:"schema_version"`
	Plan          GeneratedAppToolAttachPlanV0 `json:"plan"`
	Receipt       ToolOperationReceiptV0       `json:"receipt"`
}

type ToolCapabilityIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type ToolOperationReceiptFilterV0 struct {
	ReceiptRef     string `json:"receipt_ref,omitempty"`
	AppRef         string `json:"app_ref,omitempty"`
	BindingRef     string `json:"binding_ref,omitempty"`
	ToolRef        string `json:"tool_ref,omitempty"`
	Operation      string `json:"operation,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type ToolCapabilityPortsV0 struct {
	Catalog          CapabilityCatalogPortV0
	BindingAuthority GeneratedAppToolBindingAuthorityPortV0
	Registry         CapabilityRegistryPortV0
	SnapshotResolver ToolBundleSnapshotResolverPortV0
	Validator        GeneratedAppToolAttachValidatorPortV0
	Installer        ToolBundleInstallerPortV0
	ReceiptStore     ToolOperationReceiptStorePortV0
}

type CapabilityCatalogPortV0 interface {
	ListCapabilityManifestsV0(context.Context, CapabilityManifestFilterV0) ([]CapabilityManifestV0, error)
}

type CapabilityManifestFilterV0 struct {
	CapabilityRef string `json:"capability_ref,omitempty"`
	ToolRef       string `json:"tool_ref,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

type GeneratedAppToolBindingAuthorityPortV0 interface {
	ResolveGeneratedAppToolBindingV0(context.Context, string, string) (GeneratedAppToolBindingV0, error)
}

type CapabilityRegistryPortV0 interface {
	GetCapabilityManifestV0(context.Context, string) (CapabilityManifestV0, error)
}

// ToolBundleSnapshotResolverPortV0 must verify every declared hash and return
// a content-addressed snapshot. Filesystem implementations must use a trusted
// root and no-follow/safe handles rather than re-opening request paths.
type ToolBundleSnapshotResolverPortV0 interface {
	ResolveVerifiedToolBundleSnapshotV0(context.Context, CapabilityManifestV0, string) (ResolvedToolBundleSnapshotV0, error)
}

type GeneratedAppToolAttachValidatorPortV0 interface {
	// The plan contains the authority-resolved binding, never request data.
	ValidateGeneratedAppToolAttachPlanV0(context.Context, GeneratedAppToolAttachPlanV0) []ToolCapabilityIssueV0
}

type ToolBundleMaterializerPortV0 interface {
	// Materializers consume only plan.Snapshot.HandleRef. OperationRef is the
	// durable idempotency key and PlanRef identifies immutable input.
	MaterializeToolBundleV0(context.Context, GeneratedAppToolAttachPlanV0) (ToolMaterializationResultV0, error)
}

type ToolBundleInstallerPortV0 interface {
	ToolBundleMaterializerPortV0
	UninstallToolBundleV0(context.Context, GeneratedAppToolAttachPlanV0) (ToolMaterializationResultV0, error)
	RollbackToolBundleV0(context.Context, ToolRollbackRequestV0) (ToolMaterializationResultV0, error)
	ReconcileToolBundleV0(context.Context, GeneratedAppToolAttachPlanV0) (ToolOperationReconciliationV0, error)
}

type ToolRollbackRequestV0 struct {
	FailedPlan      GeneratedAppToolAttachPlanV0 `json:"failed_plan"`
	PreviousPlan    GeneratedAppToolAttachPlanV0 `json:"previous_plan"`
	PreviousReceipt ToolOperationReceiptV0       `json:"previous_receipt"`
}

type ToolMaterializationResultV0 struct {
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type ToolOperationReconciliationV0 struct {
	Disposition  string   `json:"disposition"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type ToolOperationReceiptStorePortV0 interface {
	// ClaimToolOperationV0 atomically persists Plan+Receipt and acquires the
	// TargetRef lease. Only one non-terminal operation may lease a target. A
	// repeated OperationRef returns its original immutable record.
	ClaimToolOperationV0(context.Context, ToolOperationRecordV0) (record ToolOperationRecordV0, acquired bool, err error)
	GetToolOperationRecordV0(context.Context, string) (ToolOperationRecordV0, bool, error)
	GetToolOperationRecordByReceiptRefV0(context.Context, string) (ToolOperationRecordV0, bool, error)
	// CompareAndSwapToolOperationV0 updates only Receipt. Terminal transitions
	// release the target lease; claimed/recovery_required retain it.
	CompareAndSwapToolOperationV0(context.Context, string, uint64, ToolOperationReceiptV0) (record ToolOperationRecordV0, swapped bool, err error)
	ListToolOperationReceiptsV0(context.Context, ToolOperationReceiptFilterV0) ([]ToolOperationReceiptV0, error)
}
