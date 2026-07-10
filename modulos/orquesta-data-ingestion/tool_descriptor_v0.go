package orquestadataingestion

import "fmt"

type DataIngestionToolSurfaceV0 string

const (
	DataIngestionToolSurfaceInspectV0 DataIngestionToolSurfaceV0 = "data.ingestion.inspect.v0"
	DataIngestionToolSurfaceIngestV0  DataIngestionToolSurfaceV0 = "data.ingestion.ingest.v0"
	DataIngestionToolSurfaceStatusV0  DataIngestionToolSurfaceV0 = "data.ingestion.status.v0"
)

type DataIngestionToolAttachModeV0 string

const (
	DataIngestionToolAttachModeEmbeddedModuleV0  DataIngestionToolAttachModeV0 = "embedded_module"
	DataIngestionToolAttachModeLocalSidecarV0    DataIngestionToolAttachModeV0 = "local_sidecar"
	DataIngestionToolAttachModeRemoteConnectorV0 DataIngestionToolAttachModeV0 = "remote_connector"
)

// DataIngestionToolCapabilityDescriptorV0 is the neutral capability contract.
// Runtime, connector, parser and transport choices remain outside this module.
type DataIngestionToolCapabilityDescriptorV0 struct {
	CapabilityRef          string                          `json:"capability_ref"`
	Version                string                          `json:"version"`
	SurfaceRefs            []DataIngestionToolSurfaceV0    `json:"surface_refs"`
	RequiredCapabilityRefs []string                        `json:"required_capability_refs"`
	PermissionProfileRef   string                          `json:"permission_profile_ref"`
	AttachModes            []DataIngestionToolAttachModeV0 `json:"attach_modes"`
	ReceiptSchemaVersion   string                          `json:"receipt_schema_version"`
}

// DataIngestionBundleDescriptorV0 is the provisional ingestion-specific
// attachment contract consumed by the generic ToolBundle/AttachPlan adapter.
type DataIngestionBundleDescriptorV0 struct {
	BundleRef        string                                  `json:"bundle_ref"`
	Capability       DataIngestionToolCapabilityDescriptorV0 `json:"capability"`
	AttachMode       DataIngestionToolAttachModeV0           `json:"attach_mode"`
	ModuleRef        string                                  `json:"module_ref,omitempty"`
	ConnectorRef     string                                  `json:"connector_ref,omitempty"`
	ConfigurationRef string                                  `json:"configuration_ref"`
	I18NRef          string                                  `json:"i18n_ref"`
	TestArtifactRefs []string                                `json:"test_artifact_refs"`
	ArtifactHashes   []DataIngestionArtifactHashV0           `json:"artifact_hashes"`
	ReceiptRef       string                                  `json:"receipt_ref,omitempty"`
}

type DataIngestionArtifactHashV0 struct {
	ArtifactRef string `json:"artifact_ref"`
	Hash        string `json:"hash"`
}

func ValidateDataIngestionToolCapabilityDescriptorV0(descriptor DataIngestionToolCapabilityDescriptorV0) error {
	if blankDataValueV0(descriptor.CapabilityRef) || blankDataValueV0(descriptor.Version) ||
		blankDataValueV0(descriptor.PermissionProfileRef) || blankDataValueV0(descriptor.ReceiptSchemaVersion) ||
		len(descriptor.SurfaceRefs) == 0 || len(descriptor.AttachModes) == 0 {
		return fmt.Errorf("data_ingestion_tool_capability_descriptor_invalid")
	}
	return nil
}

func ValidateDataIngestionBundleDescriptorV0(descriptor DataIngestionBundleDescriptorV0) error {
	if blankDataValueV0(descriptor.BundleRef) || blankDataValueV0(descriptor.ConfigurationRef) || blankDataValueV0(descriptor.I18NRef) {
		return fmt.Errorf("data_ingestion_tool_bundle_descriptor_invalid")
	}
	if err := ValidateDataIngestionToolCapabilityDescriptorV0(descriptor.Capability); err != nil {
		return err
	}
	switch descriptor.AttachMode {
	case DataIngestionToolAttachModeEmbeddedModuleV0:
		if blankDataValueV0(descriptor.ModuleRef) {
			return fmt.Errorf("data_ingestion_tool_bundle_module_ref_required")
		}
	case DataIngestionToolAttachModeLocalSidecarV0, DataIngestionToolAttachModeRemoteConnectorV0:
		if blankDataValueV0(descriptor.ConnectorRef) {
			return fmt.Errorf("data_ingestion_tool_bundle_connector_ref_required")
		}
	default:
		return fmt.Errorf("data_ingestion_tool_bundle_attach_mode_invalid")
	}
	if len(descriptor.TestArtifactRefs) == 0 || len(descriptor.ArtifactHashes) == 0 {
		return fmt.Errorf("data_ingestion_tool_bundle_evidence_required")
	}
	return nil
}

func DefaultDataIngestionToolCapabilityDescriptorV0() DataIngestionToolCapabilityDescriptorV0 {
	return DataIngestionToolCapabilityDescriptorV0{
		CapabilityRef: "orquesta.data.ingestion", Version: "v0",
		SurfaceRefs:            []DataIngestionToolSurfaceV0{DataIngestionToolSurfaceInspectV0, DataIngestionToolSurfaceIngestV0, DataIngestionToolSurfaceStatusV0},
		RequiredCapabilityRefs: []string{"data_source", "data_profile", "data_mapping", "data_validation", "data_receipt"},
		PermissionProfileRef:   "data_ingestion",
		AttachModes:            []DataIngestionToolAttachModeV0{DataIngestionToolAttachModeEmbeddedModuleV0, DataIngestionToolAttachModeLocalSidecarV0, DataIngestionToolAttachModeRemoteConnectorV0},
		ReceiptSchemaVersion:   DataIngestionReceiptSchemaVersionV0,
	}
}
