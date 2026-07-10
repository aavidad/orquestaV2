package orquestapresentationextractiontoolcapability

import (
	"context"
	"fmt"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// PreparePresentationExtractionToolAttachV0 delegates plan construction to the SDK.
func PreparePresentationExtractionToolAttachV0(ctx context.Context, intent PresentationExtractionToolAttachIntentV0, ports PresentationExtractionToolCapabilityPortsV0) (toolcapability.GeneratedAppToolAttachPlanV0, error) {
	registration, adapter, err := presentationExtractionAdapterV0(ctx, intent.PresentationBundleRef, ports)
	if err != nil {
		return toolcapability.GeneratedAppToolAttachPlanV0{}, err
	}
	return toolcapability.PrepareGeneratedAppToolAttachV0(ctx, presentationExtractionRequestV0(intent, registration), adapter.sdkPorts(ports, false))
}

// AttachPresentationExtractionToolV0 delegates durable claim/CAS/lease semantics to the SDK.
func AttachPresentationExtractionToolV0(ctx context.Context, intent PresentationExtractionToolAttachIntentV0, ports PresentationExtractionToolCapabilityPortsV0) (toolcapability.ToolOperationReceiptV0, error) {
	registration, adapter, err := presentationExtractionAdapterV0(ctx, intent.PresentationBundleRef, ports)
	if err != nil {
		return toolcapability.ToolOperationReceiptV0{}, err
	}
	return toolcapability.AttachGeneratedAppToolV0(ctx, presentationExtractionRequestV0(intent, registration), adapter.sdkPorts(ports, true))
}

func presentationExtractionAdapterV0(ctx context.Context, bundleRef string, ports PresentationExtractionToolCapabilityPortsV0) (PresentationExtractionToolRegistrationV0, presentationExtractionToolAdapterV0, error) {
	if ports.RegistrationAuthority == nil || ports.SnapshotResolver == nil {
		return PresentationExtractionToolRegistrationV0{}, presentationExtractionToolAdapterV0{}, fmt.Errorf("presentation_tool_capability_port_required")
	}
	registration, err := ports.RegistrationAuthority.ResolvePresentationExtractionToolRegistrationV0(ctx, bundleRef)
	if err != nil {
		return PresentationExtractionToolRegistrationV0{}, presentationExtractionToolAdapterV0{}, err
	}
	if registration.Descriptor.BundleRef != bundleRef {
		return PresentationExtractionToolRegistrationV0{}, presentationExtractionToolAdapterV0{}, fmt.Errorf("presentation_tool_registration_authority_mismatch")
	}
	return registration, presentationExtractionToolAdapterV0{registration: registration, snapshotResolver: ports.SnapshotResolver, effect: ports.Effect}, nil
}

type presentationExtractionToolAdapterV0 struct {
	registration     PresentationExtractionToolRegistrationV0
	snapshotResolver PresentationExtractionToolSnapshotResolverPortV0
	effect           PresentationExtractionToolEffectPortV0
}

func (adapter presentationExtractionToolAdapterV0) sdkPorts(ports PresentationExtractionToolCapabilityPortsV0, withEffect bool) toolcapability.ToolCapabilityPortsV0 {
	result := toolcapability.ToolCapabilityPortsV0{BindingAuthority: ports.BindingAuthority, Registry: ports.Registry, Validator: ports.Validator, ReceiptStore: ports.ReceiptStore, SnapshotResolver: adapter}
	if withEffect {
		result.Installer = adapter
	}
	return result
}

func (adapter presentationExtractionToolAdapterV0) ResolveVerifiedToolBundleSnapshotV0(ctx context.Context, manifest toolcapability.CapabilityManifestV0, bundleRef string) (toolcapability.ResolvedToolBundleSnapshotV0, error) {
	if bundleRef != adapter.registration.Descriptor.BundleRef || manifest.ManifestRef != adapter.registration.ManifestRef {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, fmt.Errorf("presentation_tool_snapshot_authority_mismatch")
	}
	bundle, err := ProjectPresentationExtractionToolBundleV0(adapter.registration)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	bundle.ContentHash = toolcapability.CalculateToolBundleContentHashV0(manifest, bundle)
	snapshot, err := adapter.snapshotResolver.ResolveVerifiedPresentationExtractionToolSnapshotV0(ctx, manifest, bundle)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	return toolcapability.ResolvedToolBundleSnapshotV0{Bundle: bundle, Snapshot: snapshot}, nil
}

func (adapter presentationExtractionToolAdapterV0) MaterializeToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("presentation_tool_effect_port_required")
	}
	return adapter.effect.MaterializePresentationExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
func (adapter presentationExtractionToolAdapterV0) UninstallToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("presentation_tool_effect_port_required")
	}
	return adapter.effect.UninstallPresentationExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
func (adapter presentationExtractionToolAdapterV0) RollbackToolBundleV0(ctx context.Context, request toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("presentation_tool_effect_port_required")
	}
	return adapter.effect.RollbackPresentationExtractionToolV0(ctx, adapter.registration.Descriptor, request)
}
func (adapter presentationExtractionToolAdapterV0) ReconcileToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolOperationReconciliationV0{}, fmt.Errorf("presentation_tool_effect_port_required")
	}
	return adapter.effect.ReconcilePresentationExtractionToolV0(ctx, adapter.registration.Descriptor, plan)
}
