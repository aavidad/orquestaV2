package orquestapresentationextractiontoolcapability

import (
	"context"
	"strings"
	"testing"

	presentation "orquesta/modulos/orquesta-presentation-extraction"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

func TestProjectPresentationExtractionToolBundleV0PreservesAuthorityOwnedMetadata(t *testing.T) {
	registration := presentationExtractionToolRegistrationForTestV0()
	bundle, err := ProjectPresentationExtractionToolBundleV0(registration)
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

func TestPreparePresentationExtractionToolAttachV0RejectsDescriptorOutsideAuthority(t *testing.T) {
	registration := presentationExtractionToolRegistrationForTestV0()
	registration.Descriptor.BundleRef = "bundle:other"
	_, err := PreparePresentationExtractionToolAttachV0(context.Background(), PresentationExtractionToolAttachIntentV0{
		PresentationBundleRef: "bundle:presentation-extraction",
	}, PresentationExtractionToolCapabilityPortsV0{
		RegistrationAuthority: presentationExtractionToolRegistrationAuthorityFakeV0{registration: registration},
		SnapshotResolver:      &presentationExtractionToolSnapshotResolverFakeV0{},
	})
	if err == nil || !strings.Contains(err.Error(), "presentation_tool_registration_authority_mismatch") {
		t.Fatalf("authority mismatch err=%v", err)
	}
}

func TestPresentationExtractionToolAdapterV0DelegatesSnapshotAndEffect(t *testing.T) {
	registration := presentationExtractionToolRegistrationForTestV0()
	snapshotResolver := &presentationExtractionToolSnapshotResolverFakeV0{}
	effect := &presentationExtractionToolEffectFakeV0{}
	_, adapter, err := presentationExtractionAdapterV0(context.Background(), registration.Descriptor.BundleRef, PresentationExtractionToolCapabilityPortsV0{
		RegistrationAuthority: presentationExtractionToolRegistrationAuthorityFakeV0{registration: registration},
		SnapshotResolver:      snapshotResolver,
		Effect:                effect,
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	manifest := toolcapability.CapabilityManifestV0{ManifestRef: registration.ManifestRef}
	resolved, err := adapter.ResolveVerifiedToolBundleSnapshotV0(context.Background(), manifest, registration.Descriptor.BundleRef)
	if err != nil {
		t.Fatalf("resolve snapshot: %v", err)
	}
	if snapshotResolver.bundle.BundleRef != registration.Descriptor.BundleRef || snapshotResolver.bundle.ContentHash == "" || resolved.Bundle.ContentHash != snapshotResolver.bundle.ContentHash {
		t.Fatalf("snapshot delegation unexpected: %+v %+v", snapshotResolver.bundle, resolved)
	}
	plan := toolcapability.GeneratedAppToolAttachPlanV0{OperationRef: "operation:presentation"}
	if _, err := adapter.MaterializeToolBundleV0(context.Background(), plan); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if effect.descriptor.BundleRef != registration.Descriptor.BundleRef || effect.plan.OperationRef != plan.OperationRef {
		t.Fatalf("effect delegation unexpected: %+v %+v", effect.descriptor, effect.plan)
	}
}

func presentationExtractionToolRegistrationForTestV0() PresentationExtractionToolRegistrationV0 {
	return PresentationExtractionToolRegistrationV0{
		RegistrationRef: "registration:presentation-extraction",
		Descriptor: presentation.PresentationExtractionBundleDescriptorV0{
			BundleRef: "bundle:presentation-extraction", Capability: presentation.DefaultPresentationToolCapabilityDescriptorV0(),
			AttachMode: presentation.PresentationToolAttachModeEmbeddedModuleV0, ModuleRef: "module:presentation-extraction",
			ConfigurationRef: "config:presentation-extraction", I18NRef: "i18n:presentation-extraction",
			TestArtifactRefs: []string{"test:presentation-extraction"},
			ArtifactHashes:   []presentation.PresentationArtifactHashV0{{ArtifactRef: "module:presentation-extraction", Hash: "sha256:module"}},
		},
		ManifestRef: "manifest:presentation-extraction", ToolRef: "tool:presentation-extraction", ToolVersion: "v0",
		I18nCatalogs: []toolcapability.I18nCatalogRefV0{{Locale: "es", CatalogRef: "i18n:presentation-extraction", ContentHash: "sha256:i18n"}},
	}
}

type presentationExtractionToolRegistrationAuthorityFakeV0 struct {
	registration PresentationExtractionToolRegistrationV0
}

func (fake presentationExtractionToolRegistrationAuthorityFakeV0) ResolvePresentationExtractionToolRegistrationV0(context.Context, string) (PresentationExtractionToolRegistrationV0, error) {
	return fake.registration, nil
}

type presentationExtractionToolSnapshotResolverFakeV0 struct{ bundle toolcapability.ToolBundleV0 }

func (fake *presentationExtractionToolSnapshotResolverFakeV0) ResolveVerifiedPresentationExtractionToolSnapshotV0(_ context.Context, _ toolcapability.CapabilityManifestV0, bundle toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error) {
	fake.bundle = bundle
	return toolcapability.ToolBundleSnapshotV0{}, nil
}

type presentationExtractionToolEffectFakeV0 struct {
	descriptor presentation.PresentationExtractionBundleDescriptorV0
	plan       toolcapability.GeneratedAppToolAttachPlanV0
}

func (fake *presentationExtractionToolEffectFakeV0) MaterializePresentationExtractionToolV0(_ context.Context, descriptor presentation.PresentationExtractionBundleDescriptorV0, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	fake.descriptor, fake.plan = descriptor, plan
	return toolcapability.ToolMaterializationResultV0{}, nil
}
func (*presentationExtractionToolEffectFakeV0) UninstallPresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	return toolcapability.ToolMaterializationResultV0{}, nil
}
func (*presentationExtractionToolEffectFakeV0) RollbackPresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error) {
	return toolcapability.ToolMaterializationResultV0{}, nil
}
func (*presentationExtractionToolEffectFakeV0) ReconcilePresentationExtractionToolV0(context.Context, presentation.PresentationExtractionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error) {
	return toolcapability.ToolOperationReconciliationV0{}, nil
}
