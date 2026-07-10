package orquestadocumentextractiontoolcapability

import (
	"context"

	document "orquesta/modulos/orquesta-document-extraction"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// DocumentExtractionToolRegistrationV0 is authority-owned metadata missing
// from the provisional document descriptor (manifest/tool/version/i18n).
type DocumentExtractionToolRegistrationV0 struct {
	RegistrationRef string                                        `json:"registration_ref"`
	Descriptor      document.DocumentExtractionBundleDescriptorV0 `json:"descriptor"`
	ManifestRef     string                                        `json:"manifest_ref"`
	ToolRef         string                                        `json:"tool_ref"`
	ToolVersion     string                                        `json:"tool_version"`
	I18nCatalogs    []toolcapability.I18nCatalogRefV0             `json:"i18n_catalogs"`
	Compatibility   toolcapability.ToolCompatibilityV0            `json:"compatibility"`
}

// DocumentExtractionToolAttachIntentV0 carries caller intent only. Descriptor
// metadata, manifest and binding remain authority-owned.
type DocumentExtractionToolAttachIntentV0 struct {
	RequestRef         string   `json:"request_ref"`
	Operation          string   `json:"operation"`
	IdempotencyKey     string   `json:"idempotency_key"`
	RequestedBy        string   `json:"requested_by"`
	AppRef             string   `json:"app_ref"`
	BindingRef         string   `json:"binding_ref"`
	DocumentBundleRef  string   `json:"document_bundle_ref"`
	WriteSet           []string `json:"write_set"`
	PreviousReceiptRef string   `json:"previous_receipt_ref,omitempty"`
}

type DocumentExtractionToolRegistrationAuthorityPortV0 interface {
	ResolveDocumentExtractionToolRegistrationV0(context.Context, string) (DocumentExtractionToolRegistrationV0, error)
}

// DocumentExtractionToolSnapshotResolverPortV0 verifies projected content and
// returns an opaque, content-addressed snapshot. No filesystem path crosses it.
type DocumentExtractionToolSnapshotResolverPortV0 interface {
	ResolveVerifiedDocumentExtractionToolSnapshotV0(context.Context, toolcapability.CapabilityManifestV0, toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error)
}

// DocumentExtractionToolEffectPortV0 is implemented by an opt-in composition.
// Every effect must deduplicate durably by plan.OperationRef.
type DocumentExtractionToolEffectPortV0 interface {
	MaterializeDocumentExtractionToolV0(context.Context, document.DocumentExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	UninstallDocumentExtractionToolV0(context.Context, document.DocumentExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	RollbackDocumentExtractionToolV0(context.Context, document.DocumentExtractionBundleDescriptorV0, toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error)
	ReconcileDocumentExtractionToolV0(context.Context, document.DocumentExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error)
}

type DocumentExtractionToolCapabilityPortsV0 struct {
	RegistrationAuthority DocumentExtractionToolRegistrationAuthorityPortV0
	SnapshotResolver      DocumentExtractionToolSnapshotResolverPortV0
	BindingAuthority      toolcapability.GeneratedAppToolBindingAuthorityPortV0
	Registry              toolcapability.CapabilityRegistryPortV0
	Validator             toolcapability.GeneratedAppToolAttachValidatorPortV0
	Effect                DocumentExtractionToolEffectPortV0
	ReceiptStore          toolcapability.ToolOperationReceiptStorePortV0
}
