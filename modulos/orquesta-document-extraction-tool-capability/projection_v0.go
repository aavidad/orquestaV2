package orquestadocumentextractiontoolcapability

import (
	"fmt"

	document "orquesta/modulos/orquesta-document-extraction"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// ProjectDocumentExtractionToolBundleV0 is deterministic conversion from the
// provisional descriptor plus authority metadata to the generic SDK bundle.
func ProjectDocumentExtractionToolBundleV0(registration DocumentExtractionToolRegistrationV0) (toolcapability.ToolBundleV0, error) {
	descriptor := registration.Descriptor
	if err := document.ValidateDocumentExtractionBundleDescriptorV0(descriptor); err != nil {
		return toolcapability.ToolBundleV0{}, fmt.Errorf("document_tool_registration_descriptor_invalid: %w", err)
	}
	if registration.ManifestRef == "" || registration.ToolRef == "" || registration.ToolVersion == "" || len(registration.I18nCatalogs) == 0 {
		return toolcapability.ToolBundleV0{}, fmt.Errorf("document_tool_registration_invalid")
	}
	artifacts := make([]toolcapability.HashedArtifactV0, 0, len(descriptor.ArtifactHashes))
	for _, artifact := range descriptor.ArtifactHashes {
		artifacts = append(artifacts, toolcapability.HashedArtifactV0{ArtifactRef: artifact.ArtifactRef, ContentHash: artifact.Hash})
	}
	bundle := toolcapability.ToolBundleV0{
		SchemaVersion: toolcapability.ToolBundleSchemaV0,
		BundleRef:     descriptor.BundleRef, ManifestRef: registration.ManifestRef,
		CapabilityRef: descriptor.Capability.CapabilityRef, ToolRef: registration.ToolRef,
		Version: registration.ToolVersion, ArtifactRefs: artifacts,
		I18nCatalogs: append([]toolcapability.I18nCatalogRefV0(nil), registration.I18nCatalogs...),
		TestRefs:     append([]string(nil), descriptor.TestArtifactRefs...), Compatibility: registration.Compatibility,
	}
	return bundle, nil
}

func documentExtractionIntegrationModeV0(mode document.DocumentToolAttachModeV0) string {
	return string(mode)
}

func documentExtractionRequestV0(intent DocumentExtractionToolAttachIntentV0, registration DocumentExtractionToolRegistrationV0) toolcapability.GeneratedAppToolAttachRequestV0 {
	return toolcapability.GeneratedAppToolAttachRequestV0{
		RequestRef: intent.RequestRef, Operation: intent.Operation, IdempotencyKey: intent.IdempotencyKey,
		RequestedBy: intent.RequestedBy, AppRef: intent.AppRef, BindingRef: intent.BindingRef,
		IntegrationMode: documentExtractionIntegrationModeV0(registration.Descriptor.AttachMode),
		ManifestRef:     registration.ManifestRef, BundleRef: registration.Descriptor.BundleRef,
		ConfigRef: registration.Descriptor.ConfigurationRef, WriteSet: append([]string(nil), intent.WriteSet...),
		PreviousReceiptRef: intent.PreviousReceiptRef,
	}
}
