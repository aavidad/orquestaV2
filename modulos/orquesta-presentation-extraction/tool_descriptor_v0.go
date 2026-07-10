package orquestapresentationextraction

import "fmt"

type PresentationToolSurfaceV0 string

const (
	PresentationToolSurfaceInspectV0 PresentationToolSurfaceV0 = "presentations.inspect.v0"
	PresentationToolSurfaceExtractV0 PresentationToolSurfaceV0 = "presentations.extract.v0"
	PresentationToolSurfaceStatusV0  PresentationToolSurfaceV0 = "presentations.status.v0"
)

type PresentationToolAttachModeV0 string

const (
	PresentationToolAttachModeEmbeddedModuleV0  PresentationToolAttachModeV0 = "embedded_module"
	PresentationToolAttachModeLocalSidecarV0    PresentationToolAttachModeV0 = "local_sidecar"
	PresentationToolAttachModeRemoteConnectorV0 PresentationToolAttachModeV0 = "remote_connector"
)

// PresentationToolCapabilityDescriptorV0 declares the neutral presentation
// extraction capability. Implementations remain behind ports and opaque refs.
type PresentationToolCapabilityDescriptorV0 struct {
	CapabilityRef          string                         `json:"capability_ref"`
	Version                string                         `json:"version"`
	SurfaceRefs            []PresentationToolSurfaceV0    `json:"surface_refs"`
	RequiredCapabilityRefs []string                       `json:"required_capability_refs"`
	SupportedFormats       []PresentationFormatV0         `json:"supported_formats"`
	AttachModes            []PresentationToolAttachModeV0 `json:"attach_modes"`
	ReceiptSchemaVersion   string                         `json:"receipt_schema_version"`
}

// PresentationExtractionBundleDescriptorV0 is the extraction-specific input
// to the generic ToolBundle/AttachPlan SDK. It contains no implementation path.
type PresentationExtractionBundleDescriptorV0 struct {
	BundleRef        string                                 `json:"bundle_ref"`
	Capability       PresentationToolCapabilityDescriptorV0 `json:"capability"`
	AttachMode       PresentationToolAttachModeV0           `json:"attach_mode"`
	ModuleRef        string                                 `json:"module_ref,omitempty"`
	ConnectorRef     string                                 `json:"connector_ref,omitempty"`
	ConfigurationRef string                                 `json:"configuration_ref"`
	I18NRef          string                                 `json:"i18n_ref"`
	TestArtifactRefs []string                               `json:"test_artifact_refs"`
	ArtifactHashes   []PresentationArtifactHashV0           `json:"artifact_hashes"`
	ReceiptRef       string                                 `json:"receipt_ref,omitempty"`
}

type PresentationArtifactHashV0 struct {
	ArtifactRef string `json:"artifact_ref"`
	Hash        string `json:"hash"`
}

func ValidatePresentationToolCapabilityDescriptorV0(descriptor PresentationToolCapabilityDescriptorV0) error {
	if blankPresentationValueV0(descriptor.CapabilityRef) || blankPresentationValueV0(descriptor.Version) || blankPresentationValueV0(descriptor.ReceiptSchemaVersion) || len(descriptor.SurfaceRefs) == 0 || len(descriptor.SupportedFormats) == 0 || len(descriptor.AttachModes) == 0 {
		return fmt.Errorf("presentation_tool_capability_descriptor_invalid")
	}
	for _, format := range descriptor.SupportedFormats {
		if !validPresentationFormatV0(format) {
			return fmt.Errorf("presentation_tool_capability_format_invalid")
		}
	}
	return nil
}

func ValidatePresentationExtractionBundleDescriptorV0(descriptor PresentationExtractionBundleDescriptorV0) error {
	if blankPresentationValueV0(descriptor.BundleRef) || blankPresentationValueV0(descriptor.ConfigurationRef) || blankPresentationValueV0(descriptor.I18NRef) {
		return fmt.Errorf("presentation_tool_bundle_descriptor_invalid")
	}
	if err := ValidatePresentationToolCapabilityDescriptorV0(descriptor.Capability); err != nil {
		return err
	}
	switch descriptor.AttachMode {
	case PresentationToolAttachModeEmbeddedModuleV0:
		if blankPresentationValueV0(descriptor.ModuleRef) {
			return fmt.Errorf("presentation_tool_bundle_module_ref_required")
		}
	case PresentationToolAttachModeLocalSidecarV0, PresentationToolAttachModeRemoteConnectorV0:
		if blankPresentationValueV0(descriptor.ConnectorRef) {
			return fmt.Errorf("presentation_tool_bundle_connector_ref_required")
		}
	default:
		return fmt.Errorf("presentation_tool_bundle_attach_mode_invalid")
	}
	if len(descriptor.TestArtifactRefs) == 0 || len(descriptor.ArtifactHashes) == 0 {
		return fmt.Errorf("presentation_tool_bundle_evidence_required")
	}
	return nil
}

func DefaultPresentationToolCapabilityDescriptorV0() PresentationToolCapabilityDescriptorV0 {
	return PresentationToolCapabilityDescriptorV0{
		CapabilityRef:          "orquesta.presentations.extraction",
		Version:                "v0",
		SurfaceRefs:            []PresentationToolSurfaceV0{PresentationToolSurfaceInspectV0, PresentationToolSurfaceExtractV0, PresentationToolSurfaceStatusV0},
		RequiredCapabilityRefs: []string{"presentation_source", "presentation_projector", "document_receipt"},
		SupportedFormats:       []PresentationFormatV0{PresentationFormatPPTXV0, PresentationFormatODPV0},
		AttachModes:            []PresentationToolAttachModeV0{PresentationToolAttachModeEmbeddedModuleV0, PresentationToolAttachModeLocalSidecarV0, PresentationToolAttachModeRemoteConnectorV0},
		ReceiptSchemaVersion:   PresentationExtractionContractSchemaVersionV0,
	}
}
