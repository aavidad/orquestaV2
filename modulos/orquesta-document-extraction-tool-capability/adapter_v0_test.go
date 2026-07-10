package orquestadocumentextractiontoolcapability

import (
	"context"
	"strings"
	"testing"

	document "orquesta/modulos/orquesta-document-extraction"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

func TestProjectDocumentExtractionToolBundleV0PreservesAuthorityOwnedMetadata(t *testing.T) {
	registration := documentExtractionToolRegistrationForTestV0()
	bundle, err := ProjectDocumentExtractionToolBundleV0(registration)
	if err != nil {
		t.Fatalf("project bundle: %v", err)
	}
	if bundle.BundleRef != registration.Descriptor.BundleRef ||
		bundle.ManifestRef != registration.ManifestRef ||
		bundle.ToolRef != registration.ToolRef ||
		bundle.CapabilityRef != registration.Descriptor.Capability.CapabilityRef ||
		len(bundle.ArtifactRefs) != 1 || bundle.ArtifactRefs[0].ContentHash != "sha256:module" ||
		len(bundle.I18nCatalogs) != 1 || bundle.I18nCatalogs[0].Locale != "es" {
		t.Fatalf("bundle projection unexpected: %+v", bundle)
	}
}

func TestPrepareDocumentExtractionToolAttachV0RejectsDescriptorOutsideAuthority(t *testing.T) {
	registration := documentExtractionToolRegistrationForTestV0()
	registration.Descriptor.BundleRef = "bundle:other"
	_, err := PrepareDocumentExtractionToolAttachV0(context.Background(), DocumentExtractionToolAttachIntentV0{
		DocumentBundleRef: "bundle:document-extraction",
	}, DocumentExtractionToolCapabilityPortsV0{
		RegistrationAuthority: documentExtractionToolRegistrationAuthorityFakeV0{registration: registration},
		SnapshotResolver:      documentExtractionToolSnapshotResolverFakeV0{},
	})
	if err == nil || !strings.Contains(err.Error(), "document_tool_registration_authority_mismatch") {
		t.Fatalf("authority mismatch err=%v", err)
	}
}

func documentExtractionToolRegistrationForTestV0() DocumentExtractionToolRegistrationV0 {
	return DocumentExtractionToolRegistrationV0{
		RegistrationRef: "registration:document-extraction",
		Descriptor: document.DocumentExtractionBundleDescriptorV0{
			BundleRef:        "bundle:document-extraction",
			Capability:       document.DefaultDocumentToolCapabilityDescriptorV0(),
			AttachMode:       document.DocumentToolAttachModeEmbeddedModuleV0,
			ModuleRef:        "module:document-extraction",
			ConfigurationRef: "config:document-extraction",
			I18NRef:          "i18n:document-extraction",
			TestArtifactRefs: []string{"test:document-extraction"},
			ArtifactHashes:   []document.DocumentArtifactHashV0{{ArtifactRef: "module:document-extraction", Hash: "sha256:module"}},
		},
		ManifestRef:  "manifest:document-extraction",
		ToolRef:      "tool:document-extraction",
		ToolVersion:  "v0",
		I18nCatalogs: []toolcapability.I18nCatalogRefV0{{Locale: "es", CatalogRef: "i18n:document-extraction", ContentHash: "sha256:i18n"}},
	}
}

type documentExtractionToolRegistrationAuthorityFakeV0 struct {
	registration DocumentExtractionToolRegistrationV0
}

func (fake documentExtractionToolRegistrationAuthorityFakeV0) ResolveDocumentExtractionToolRegistrationV0(context.Context, string) (DocumentExtractionToolRegistrationV0, error) {
	return fake.registration, nil
}

type documentExtractionToolSnapshotResolverFakeV0 struct{}

func (documentExtractionToolSnapshotResolverFakeV0) ResolveVerifiedDocumentExtractionToolSnapshotV0(context.Context, toolcapability.CapabilityManifestV0, toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error) {
	return toolcapability.ToolBundleSnapshotV0{}, nil
}
