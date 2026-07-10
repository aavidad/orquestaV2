package orquestapresentationextractiontoolcapability

import (
	"context"

	presentation "orquesta/modulos/orquesta-presentation-extraction"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// PresentationExtractionToolRegistrationV0 is authority-owned metadata that
// completes the neutral presentation descriptor for the generic SDK.
type PresentationExtractionToolRegistrationV0 struct {
	RegistrationRef string                                                `json:"registration_ref"`
	Descriptor      presentation.PresentationExtractionBundleDescriptorV0 `json:"descriptor"`
	ManifestRef     string                                                `json:"manifest_ref"`
	ToolRef         string                                                `json:"tool_ref"`
	ToolVersion     string                                                `json:"tool_version"`
	I18nCatalogs    []toolcapability.I18nCatalogRefV0                     `json:"i18n_catalogs"`
	Compatibility   toolcapability.ToolCompatibilityV0                    `json:"compatibility"`
}

// PresentationExtractionToolAttachIntentV0 contains caller-selected refs only.
// Descriptor, manifest and tool binding remain authority-owned.
type PresentationExtractionToolAttachIntentV0 struct {
	RequestRef            string   `json:"request_ref"`
	Operation             string   `json:"operation"`
	IdempotencyKey        string   `json:"idempotency_key"`
	RequestedBy           string   `json:"requested_by"`
	AppRef                string   `json:"app_ref"`
	BindingRef            string   `json:"binding_ref"`
	PresentationBundleRef string   `json:"presentation_bundle_ref"`
	WriteSet              []string `json:"write_set"`
	PreviousReceiptRef    string   `json:"previous_receipt_ref,omitempty"`
}

type PresentationExtractionToolRegistrationAuthorityPortV0 interface {
	ResolvePresentationExtractionToolRegistrationV0(context.Context, string) (PresentationExtractionToolRegistrationV0, error)
}

type PresentationExtractionToolSnapshotResolverPortV0 interface {
	ResolveVerifiedPresentationExtractionToolSnapshotV0(context.Context, toolcapability.CapabilityManifestV0, toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error)
}

// PresentationExtractionToolEffectPortV0 belongs to an opt-in composition.
// Effects must durably deduplicate by plan.OperationRef.
type PresentationExtractionToolEffectPortV0 interface {
	MaterializePresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	UninstallPresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error)
	RollbackPresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error)
	ReconcilePresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error)
}

type PresentationExtractionToolCapabilityPortsV0 struct {
	RegistrationAuthority PresentationExtractionToolRegistrationAuthorityPortV0
	SnapshotResolver      PresentationExtractionToolSnapshotResolverPortV0
	BindingAuthority      toolcapability.GeneratedAppToolBindingAuthorityPortV0
	Registry              toolcapability.CapabilityRegistryPortV0
	Validator             toolcapability.GeneratedAppToolAttachValidatorPortV0
	Effect                PresentationExtractionToolEffectPortV0
	ReceiptStore          toolcapability.ToolOperationReceiptStorePortV0
}
