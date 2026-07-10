package orquestadocumentextraction_test

import (
	"context"
	"testing"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
	orquestadocumentextractionfake "orquesta/modulos/orquesta-document-extraction-fake"
)

func TestExecuteDocumentToolV0IsIdempotentAndStatusUsesSameUseCase(t *testing.T) {
	store := orquestadocumentextractionfake.NewInMemoryDocumentToolOperationStoreV0(orquestadocumentextraction.DocumentAdapterIdentityV0{AdapterRef: "operation-store", Version: "1"})
	command := documentToolCommandV0(orquestadocumentextraction.DocumentToolSurfaceExtractV0)
	first, err := orquestadocumentextraction.ExecuteDocumentToolV0(context.Background(), command, store)
	if err != nil {
		t.Fatalf("first command: %v", err)
	}
	second, err := orquestadocumentextraction.ExecuteDocumentToolV0(context.Background(), command, store)
	if err != nil || first.OperationRef != second.OperationRef || first.State != orquestadocumentextraction.DocumentToolOperationQueuedV0 {
		t.Fatalf("idempotency = %#v %#v %v", first, second, err)
	}
	statusCommand := documentToolCommandV0(orquestadocumentextraction.DocumentToolSurfaceStatusV0)
	statusCommand.OperationRef = first.OperationRef
	status, err := orquestadocumentextraction.ExecuteDocumentToolV0(context.Background(), statusCommand, store)
	if err != nil || status.OperationRef != first.OperationRef {
		t.Fatalf("status = %#v, %v", status, err)
	}
}

func TestDocumentCapabilityAndProvisionalBundleDescriptor(t *testing.T) {
	capability := orquestadocumentextraction.DefaultDocumentToolCapabilityDescriptorV0()
	if err := orquestadocumentextraction.ValidateDocumentToolCapabilityDescriptorV0(capability); err != nil {
		t.Fatalf("capability: %v", err)
	}
	bundle := orquestadocumentextraction.DocumentExtractionBundleDescriptorV0{BundleRef: "bundle:document-extraction", Capability: capability, AttachMode: orquestadocumentextraction.DocumentToolAttachModeEmbeddedModuleV0, ModuleRef: "module:document-extraction", ConfigurationRef: "config:1", I18NRef: "i18n:1", TestArtifactRefs: []string{"test:acceptance"}, ArtifactHashes: []orquestadocumentextraction.DocumentArtifactHashV0{{ArtifactRef: "module:document-extraction", Hash: "sha256:module"}}, ReceiptRef: "receipt:1"}
	if err := orquestadocumentextraction.ValidateDocumentExtractionBundleDescriptorV0(bundle); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if len(capability.SurfaceRefs) != 5 {
		t.Fatalf("surfaces = %#v", capability.SurfaceRefs)
	}
}

func documentToolCommandV0(surface orquestadocumentextraction.DocumentToolSurfaceV0) orquestadocumentextraction.DocumentToolCommandV0 {
	return orquestadocumentextraction.DocumentToolCommandV0{CommandRef: "command:1", Surface: surface, IdempotencyKey: "idem:1", DocumentRef: "document:1", SchemaRef: "schema:1", ConfigurationRef: "config:1", Budget: orquestadocumentextraction.DocumentToolBudgetV0{BudgetRef: "budget:1", MaxPages: 1}, EffectProfile: orquestadocumentextraction.DocumentToolEffectProfileV0{EffectRef: "effect:document", PermissionRefs: []string{"document.read"}, DataHandlingMode: orquestadocumentextraction.DocumentDataHandlingLocalV0}}
}
