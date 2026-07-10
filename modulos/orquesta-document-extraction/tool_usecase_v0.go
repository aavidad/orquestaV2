package orquestadocumentextraction

import (
	"context"
	"fmt"
	"strings"
)

// ExecuteDocumentToolV0 is the single asynchronous use case behind all
// document surfaces. MCP, HTTP and CLI adapters only deserialize requests and
// delegate here; workers consume the opaque command through the operation port.
func ExecuteDocumentToolV0(
	ctx context.Context,
	command DocumentToolCommandV0,
	store DocumentToolOperationStorePortV0,
) (DocumentToolOperationV0, error) {
	if store == nil {
		return DocumentToolOperationV0{}, fmt.Errorf("document_tool_operation_store_required")
	}
	if err := ValidateDocumentToolCommandV0(command); err != nil {
		return DocumentToolOperationV0{}, err
	}
	if command.Surface == DocumentToolSurfaceStatusV0 {
		return store.LoadDocumentToolOperationV0(ctx, command.OperationRef)
	}
	return store.SubmitDocumentToolCommandV0(ctx, command)
}

func ValidateDocumentToolCommandV0(command DocumentToolCommandV0) error {
	if blankDocumentValueV0(command.CommandRef) || blankDocumentValueV0(command.IdempotencyKey) || blankDocumentValueV0(command.ConfigurationRef) {
		return fmt.Errorf("document_tool_command_invalid")
	}
	if err := validateDocumentToolBudgetV0(command.Budget); err != nil {
		return err
	}
	if err := validateDocumentToolEffectProfileV0(command.EffectProfile); err != nil {
		return err
	}
	switch command.Surface {
	case DocumentToolSurfaceInspectV0, DocumentToolSurfaceExtractV0:
		if blankDocumentValueV0(command.DocumentRef) {
			return fmt.Errorf("document_tool_document_ref_required")
		}
		if command.Surface == DocumentToolSurfaceExtractV0 && blankDocumentValueV0(command.SchemaRef) {
			return fmt.Errorf("document_tool_schema_ref_required")
		}
	case DocumentToolSurfaceStatusV0:
		if blankDocumentValueV0(command.OperationRef) {
			return fmt.Errorf("document_tool_operation_ref_required")
		}
	case DocumentToolSurfaceReviewV0:
		if blankDocumentValueV0(command.OperationRef) || blankDocumentValueV0(command.ReviewRef) {
			return fmt.Errorf("document_tool_review_ref_required")
		}
	case DocumentToolSurfaceExportV0:
		if blankDocumentValueV0(command.OperationRef) || blankDocumentValueV0(command.ExportRef) {
			return fmt.Errorf("document_tool_export_ref_required")
		}
	default:
		return fmt.Errorf("document_tool_surface_invalid")
	}
	return nil
}

func ValidateDocumentToolCapabilityDescriptorV0(descriptor DocumentToolCapabilityDescriptorV0) error {
	if blankDocumentValueV0(descriptor.CapabilityRef) || blankDocumentValueV0(descriptor.Version) || blankDocumentValueV0(descriptor.ReceiptSchemaVersion) || len(descriptor.SurfaceRefs) == 0 || len(descriptor.AttachModes) == 0 {
		return fmt.Errorf("document_tool_capability_descriptor_invalid")
	}
	if err := validateDocumentToolEffectProfileV0(descriptor.PermissionProfile); err != nil {
		return err
	}
	return nil
}

func ValidateDocumentExtractionBundleDescriptorV0(descriptor DocumentExtractionBundleDescriptorV0) error {
	if blankDocumentValueV0(descriptor.BundleRef) || blankDocumentValueV0(descriptor.ConfigurationRef) || blankDocumentValueV0(descriptor.I18NRef) {
		return fmt.Errorf("document_tool_bundle_descriptor_invalid")
	}
	if err := ValidateDocumentToolCapabilityDescriptorV0(descriptor.Capability); err != nil {
		return err
	}
	switch descriptor.AttachMode {
	case DocumentToolAttachModeEmbeddedModuleV0:
		if blankDocumentValueV0(descriptor.ModuleRef) {
			return fmt.Errorf("document_tool_bundle_module_ref_required")
		}
	case DocumentToolAttachModeLocalSidecarV0, DocumentToolAttachModeRemoteConnectorV0:
		if blankDocumentValueV0(descriptor.ConnectorRef) {
			return fmt.Errorf("document_tool_bundle_connector_ref_required")
		}
	default:
		return fmt.Errorf("document_tool_bundle_attach_mode_invalid")
	}
	if len(descriptor.TestArtifactRefs) == 0 || len(descriptor.ArtifactHashes) == 0 {
		return fmt.Errorf("document_tool_bundle_evidence_required")
	}
	return nil
}

func DefaultDocumentToolCapabilityDescriptorV0() DocumentToolCapabilityDescriptorV0 {
	return DocumentToolCapabilityDescriptorV0{
		CapabilityRef: "orquesta.documents.extraction", Version: "v0",
		SurfaceRefs:            []DocumentToolSurfaceV0{DocumentToolSurfaceInspectV0, DocumentToolSurfaceExtractV0, DocumentToolSurfaceStatusV0, DocumentToolSurfaceReviewV0, DocumentToolSurfaceExportV0},
		RequiredCapabilityRefs: []string{"document_source", "document_parser", "document_evidence", "document_receipt"},
		PermissionProfile:      DocumentToolEffectProfileV0{EffectRef: "document_extraction", PermissionRefs: []string{"document.read", "document.receipt.write"}, DataHandlingMode: DocumentDataHandlingLocalV0},
		AttachModes:            []DocumentToolAttachModeV0{DocumentToolAttachModeEmbeddedModuleV0, DocumentToolAttachModeLocalSidecarV0, DocumentToolAttachModeRemoteConnectorV0},
		ReceiptSchemaVersion:   DocumentExtractionReceiptSchemaVersionV0,
	}
}

func validateDocumentToolBudgetV0(budget DocumentToolBudgetV0) error {
	if blankDocumentValueV0(budget.BudgetRef) || budget.MaxPages < 0 || budget.MaxBytes < 0 || budget.MaxDurationSeconds < 0 || budget.MaxCostMinor < 0 {
		return fmt.Errorf("document_tool_budget_invalid")
	}
	return nil
}

func validateDocumentToolEffectProfileV0(profile DocumentToolEffectProfileV0) error {
	if blankDocumentValueV0(profile.EffectRef) || len(profile.PermissionRefs) == 0 {
		return fmt.Errorf("document_tool_permission_profile_invalid")
	}
	if profile.DataHandlingMode == "" {
		return nil
	}
	if profile.DataHandlingMode != DocumentDataHandlingLocalV0 && profile.DataHandlingMode != DocumentDataHandlingCloudV0 {
		return fmt.Errorf("document_tool_permission_mode_invalid")
	}
	return nil
}

func documentToolPublicErrorV0(code string, retryable bool) DocumentToolPublicErrorV0 {
	code = strings.TrimSpace(code)
	return DocumentToolPublicErrorV0{Code: code, MessageKey: code, Retryable: retryable}
}
