package orquestadocumentextractiontoolcapability

import (
	"context"
	"fmt"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// PrepareDocumentExtractionToolAttachV0 resolves descriptor authority before
// invoking the generic plan use case.
func PrepareDocumentExtractionToolAttachV0(ctx context.Context, intent DocumentExtractionToolAttachIntentV0, ports DocumentExtractionToolCapabilityPortsV0) (toolcapability.GeneratedAppToolAttachPlanV0, error) {
	registration, adapter, err := documentExtractionAdapterV0(ctx, intent.DocumentBundleRef, ports)
	if err != nil {
		return toolcapability.GeneratedAppToolAttachPlanV0{}, err
	}
	return toolcapability.PrepareGeneratedAppToolAttachV0(ctx, documentExtractionRequestV0(intent, registration), adapter.sdkPorts(ports, false))
}

// AttachDocumentExtractionToolV0 uses SDK durable claim/CAS/lease semantics.
func AttachDocumentExtractionToolV0(ctx context.Context, intent DocumentExtractionToolAttachIntentV0, ports DocumentExtractionToolCapabilityPortsV0) (toolcapability.ToolOperationReceiptV0, error) {
	registration, adapter, err := documentExtractionAdapterV0(ctx, intent.DocumentBundleRef, ports)
	if err != nil {
		return toolcapability.ToolOperationReceiptV0{}, err
	}
	return toolcapability.AttachGeneratedAppToolV0(ctx, documentExtractionRequestV0(intent, registration), adapter.sdkPorts(ports, true))
}

func documentExtractionAdapterV0(ctx context.Context, bundleRef string, ports DocumentExtractionToolCapabilityPortsV0) (DocumentExtractionToolRegistrationV0, documentExtractionToolAdapterV0, error) {
	if ports.RegistrationAuthority == nil || ports.SnapshotResolver == nil {
		return DocumentExtractionToolRegistrationV0{}, documentExtractionToolAdapterV0{}, fmt.Errorf("document_tool_capability_port_required")
	}
	registration, err := ports.RegistrationAuthority.ResolveDocumentExtractionToolRegistrationV0(ctx, bundleRef)
	if err != nil {
		return DocumentExtractionToolRegistrationV0{}, documentExtractionToolAdapterV0{}, err
	}
	if registration.Descriptor.BundleRef != bundleRef {
		return DocumentExtractionToolRegistrationV0{}, documentExtractionToolAdapterV0{}, fmt.Errorf("document_tool_registration_authority_mismatch")
	}
	return registration, documentExtractionToolAdapterV0{registration: registration, snapshotResolver: ports.SnapshotResolver, effect: ports.Effect}, nil
}

type documentExtractionToolAdapterV0 struct {
	registration     DocumentExtractionToolRegistrationV0
	snapshotResolver DocumentExtractionToolSnapshotResolverPortV0
	effect           DocumentExtractionToolEffectPortV0
}

func (adapter documentExtractionToolAdapterV0) sdkPorts(ports DocumentExtractionToolCapabilityPortsV0, withEffect bool) toolcapability.ToolCapabilityPortsV0 {
	result := toolcapability.ToolCapabilityPortsV0{BindingAuthority: ports.BindingAuthority, Registry: ports.Registry, Validator: ports.Validator, ReceiptStore: ports.ReceiptStore, SnapshotResolver: adapter}
	if withEffect {
		result.Installer = adapter
	}
	return result
}

func (adapter documentExtractionToolAdapterV0) ResolveVerifiedToolBundleSnapshotV0(ctx context.Context, manifest toolcapability.CapabilityManifestV0, bundleRef string) (toolcapability.ResolvedToolBundleSnapshotV0, error) {
	if bundleRef != adapter.registration.Descriptor.BundleRef || manifest.ManifestRef != adapter.registration.ManifestRef {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, fmt.Errorf("document_tool_snapshot_authority_mismatch")
	}
	bundle, err := ProjectDocumentExtractionToolBundleV0(adapter.registration)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	bundle.ContentHash = toolcapability.CalculateToolBundleContentHashV0(manifest, bundle)
	snapshot, err := adapter.snapshotResolver.ResolveVerifiedDocumentExtractionToolSnapshotV0(ctx, manifest, bundle)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	return toolcapability.ResolvedToolBundleSnapshotV0{Bundle: bundle, Snapshot: snapshot}, nil
}

func (adapter documentExtractionToolAdapterV0) MaterializeToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("document_tool_effect_port_required")
	}
	return adapter.effect.MaterializeDocumentExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
func (adapter documentExtractionToolAdapterV0) UninstallToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("document_tool_effect_port_required")
	}
	return adapter.effect.UninstallDocumentExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
func (adapter documentExtractionToolAdapterV0) RollbackToolBundleV0(ctx context.Context, request toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("document_tool_effect_port_required")
	}
	return adapter.effect.RollbackDocumentExtractionToolV0(ctx, adapter.registration.Descriptor, request)
}
func (adapter documentExtractionToolAdapterV0) ReconcileToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolOperationReconciliationV0{}, fmt.Errorf("document_tool_effect_port_required")
	}
	return adapter.effect.ReconcileDocumentExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
