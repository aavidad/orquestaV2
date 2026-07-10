package orquestadataingestiontoolcapability

import (
	"context"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// DataIngestionToolRegistrationV0 is authority-owned metadata missing from
// the provisional ingestion descriptor (manifest/tool/version/i18n).
type DataIngestionToolRegistrationV0 struct {
	RegistrationRef string                                    `json:"registration_ref"`
	Descriptor      ingestion.DataIngestionBundleDescriptorV0 `json:"descriptor"`
	ManifestRef     string                                    `json:"manifest_ref"`
	ToolRef         string                                    `json:"tool_ref"`
	ToolVersion     string                                    `json:"tool_version"`
	I18nCatalogs    []toolcapability.I18nCatalogRefV0         `json:"i18n_catalogs"`
	Compatibility   toolcapability.ToolCompatibilityV0        `json:"compatibility"`
}

// DataIngestionToolAttachIntentV0 carries caller intent only. Descriptor
// metadata, manifest and binding remain authority-owned.
type DataIngestionToolAttachIntentV0 struct {
	RequestRef         string   `json:"request_ref"`
	Operation          string   `json:"operation"`
	IdempotencyKey     string   `json:"idempotency_key"`
	RequestedBy        string   `json:"requested_by"`
	AppRef             string   `json:"app_ref"`
	BindingRef         string   `json:"binding_ref"`
	DataBundleRef      string   `json:"data_bundle_ref"`
	WriteSet           []string `json:"write_set"`
	PreviousReceiptRef string   `json:"previous_receipt_ref,omitempty"`
}

type DataIngestionToolRegistrationAuthorityPortV0 interface {
	ResolveDataIngestionToolRegistrationV0(context.Context, string) (DataIngestionToolRegistrationV0, error)
}

// DataIngestionToolSnapshotResolverPortV0 verifies projected content and
// returns an opaque, content-addressed snapshot. No path crosses this port.
type DataIngestionToolSnapshotResolverPortV0 interface {
	ResolveVerifiedDataIngestionToolSnapshotV0(context.Context, toolcapability.CapabilityManifestV0, toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error)
}

// DataIngestionToolEffectPortV0 is implemented by an opt-in composition.
// Every effect must deduplicate durably by plan.OperationRef.
type DataIngestionToolEffectPortV0 interface {
	MaterializeDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	UninstallDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	RollbackDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error)
	ReconcileDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error)
}

type DataIngestionToolCapabilityPortsV0 struct {
	RegistrationAuthority DataIngestionToolRegistrationAuthorityPortV0
	SnapshotResolver      DataIngestionToolSnapshotResolverPortV0
	BindingAuthority      toolcapability.GeneratedAppToolBindingAuthorityPortV0
	Registry              toolcapability.CapabilityRegistryPortV0
	Validator             toolcapability.GeneratedAppToolAttachValidatorPortV0
	Effect                DataIngestionToolEffectPortV0
	ReceiptStore          toolcapability.ToolOperationReceiptStorePortV0
}
