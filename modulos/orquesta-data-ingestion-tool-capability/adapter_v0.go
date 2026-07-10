package orquestadataingestiontoolcapability

import (
	"context"
	"fmt"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

// PrepareDataIngestionToolAttachV0 resolves descriptor authority before using
// the generic SDK plan use case.
func PrepareDataIngestionToolAttachV0(ctx context.Context, intent DataIngestionToolAttachIntentV0, ports DataIngestionToolCapabilityPortsV0) (toolcapability.GeneratedAppToolAttachPlanV0, error) {
	registration, adapter, err := dataIngestionAdapterV0(ctx, intent.DataBundleRef, ports)
	if err != nil {
		return toolcapability.GeneratedAppToolAttachPlanV0{}, err
	}
	return toolcapability.PrepareGeneratedAppToolAttachV0(ctx, dataIngestionRequestV0(intent, registration), adapter.sdkPorts(ports, false))
}

// AttachDataIngestionToolV0 uses SDK durable claim/CAS/lease semantics.
func AttachDataIngestionToolV0(ctx context.Context, intent DataIngestionToolAttachIntentV0, ports DataIngestionToolCapabilityPortsV0) (toolcapability.ToolOperationReceiptV0, error) {
	registration, adapter, err := dataIngestionAdapterV0(ctx, intent.DataBundleRef, ports)
	if err != nil {
		return toolcapability.ToolOperationReceiptV0{}, err
	}
	return toolcapability.AttachGeneratedAppToolV0(ctx, dataIngestionRequestV0(intent, registration), adapter.sdkPorts(ports, true))
}

func dataIngestionAdapterV0(ctx context.Context, bundleRef string, ports DataIngestionToolCapabilityPortsV0) (DataIngestionToolRegistrationV0, dataIngestionToolAdapterV0, error) {
	if ports.RegistrationAuthority == nil || ports.SnapshotResolver == nil {
		return DataIngestionToolRegistrationV0{}, dataIngestionToolAdapterV0{}, fmt.Errorf("data_ingestion_tool_capability_port_required")
	}
	registration, err := ports.RegistrationAuthority.ResolveDataIngestionToolRegistrationV0(ctx, bundleRef)
	if err != nil {
		return DataIngestionToolRegistrationV0{}, dataIngestionToolAdapterV0{}, err
	}
	if registration.Descriptor.BundleRef != bundleRef {
		return DataIngestionToolRegistrationV0{}, dataIngestionToolAdapterV0{}, fmt.Errorf("data_ingestion_tool_registration_authority_mismatch")
	}
	return registration, dataIngestionToolAdapterV0{registration: registration, snapshotResolver: ports.SnapshotResolver, effect: ports.Effect}, nil
}

type dataIngestionToolAdapterV0 struct {
	registration     DataIngestionToolRegistrationV0
	snapshotResolver DataIngestionToolSnapshotResolverPortV0
	effect           DataIngestionToolEffectPortV0
}

func (adapter dataIngestionToolAdapterV0) sdkPorts(ports DataIngestionToolCapabilityPortsV0, withEffect bool) toolcapability.ToolCapabilityPortsV0 {
	result := toolcapability.ToolCapabilityPortsV0{BindingAuthority: ports.BindingAuthority, Registry: ports.Registry, Validator: ports.Validator, ReceiptStore: ports.ReceiptStore, SnapshotResolver: adapter}
	if withEffect {
		result.Installer = adapter
	}
	return result
}

func (adapter dataIngestionToolAdapterV0) ResolveVerifiedToolBundleSnapshotV0(ctx context.Context, manifest toolcapability.CapabilityManifestV0, bundleRef string) (toolcapability.ResolvedToolBundleSnapshotV0, error) {
	if bundleRef != adapter.registration.Descriptor.BundleRef || manifest.ManifestRef != adapter.registration.ManifestRef {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, fmt.Errorf("data_ingestion_tool_snapshot_authority_mismatch")
	}
	bundle, err := ProjectDataIngestionToolBundleV0(adapter.registration)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	bundle.ContentHash = toolcapability.CalculateToolBundleContentHashV0(manifest, bundle)
	snapshot, err := adapter.snapshotResolver.ResolveVerifiedDataIngestionToolSnapshotV0(ctx, manifest, bundle)
	if err != nil {
		return toolcapability.ResolvedToolBundleSnapshotV0{}, err
	}
	return toolcapability.ResolvedToolBundleSnapshotV0{Bundle: bundle, Snapshot: snapshot}, nil
}

func (adapter dataIngestionToolAdapterV0) MaterializeToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("data_ingestion_tool_effect_port_required")
	}
	return adapter.effect.MaterializeDataIngestionToolV0(ctx, adapter.registration.Descriptor, plan)
}

func (adapter dataIngestionToolAdapterV0) UninstallToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("data_ingestion_tool_effect_port_required")
	}
	return adapter.effect.UninstallDataIngestionToolV0(ctx, adapter.registration.Descriptor, plan)
}

func (adapter dataIngestionToolAdapterV0) RollbackToolBundleV0(ctx context.Context, request toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolMaterializationResultV0{}, fmt.Errorf("data_ingestion_tool_effect_port_required")
	}
	return adapter.effect.RollbackDataIngestionToolV0(ctx, adapter.registration.Descriptor, request)
}

func (adapter dataIngestionToolAdapterV0) ReconcileToolBundleV0(ctx context.Context, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error) {
	if adapter.effect == nil {
		return toolcapability.ToolOperationReconciliationV0{}, fmt.Errorf("data_ingestion_tool_effect_port_required")
	}
	return adapter.effect.ReconcileDataIngestionToolV0(ctx, adapter.registration.Descriptor, plan)
}
