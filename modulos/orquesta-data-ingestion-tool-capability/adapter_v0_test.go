package orquestadataingestiontoolcapability

import (
	"context"
	"strings"
	"testing"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

const dataIngestionToolTestHashV0 = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestProjectDataIngestionToolBundleV0PreservesAuthorityOwnedMetadata(t *testing.T) {
	registration := dataIngestionToolRegistrationForTestV0()
	bundle, err := ProjectDataIngestionToolBundleV0(registration)
	if err != nil {
		t.Fatalf("project bundle: %v", err)
	}
	if bundle.BundleRef != registration.Descriptor.BundleRef ||
		bundle.ManifestRef != registration.ManifestRef ||
		bundle.ToolRef != registration.ToolRef ||
		bundle.CapabilityRef != registration.Descriptor.Capability.CapabilityRef ||
		len(bundle.ArtifactRefs) != 1 || bundle.ArtifactRefs[0].ContentHash != dataIngestionToolTestHashV0 ||
		len(bundle.I18nCatalogs) != 1 || bundle.I18nCatalogs[0].Locale != "es" {
		t.Fatalf("bundle projection unexpected: %+v", bundle)
	}
}

func TestPrepareDataIngestionToolAttachV0RejectsDescriptorOutsideAuthority(t *testing.T) {
	registration := dataIngestionToolRegistrationForTestV0()
	registration.Descriptor.BundleRef = "bundle:other"
	_, err := PrepareDataIngestionToolAttachV0(context.Background(), DataIngestionToolAttachIntentV0{
		DataBundleRef: "bundle:data-ingestion",
	}, DataIngestionToolCapabilityPortsV0{
		RegistrationAuthority: dataIngestionToolRegistrationAuthorityFakeV0{registration: registration},
		SnapshotResolver:      &dataIngestionToolSnapshotResolverFakeV0{},
	})
	if err == nil || !strings.Contains(err.Error(), "data_ingestion_tool_registration_authority_mismatch") {
		t.Fatalf("authority mismatch err=%v", err)
	}
}

func TestPrepareAndAttachDataIngestionToolV0DelegateThroughSDK(t *testing.T) {
	registration := dataIngestionToolRegistrationForTestV0()
	manifest := dataIngestionToolManifestForTestV0(registration)
	snapshot := &dataIngestionToolSnapshotResolverFakeV0{}
	effect := &dataIngestionToolEffectFakeV0{}
	ports := DataIngestionToolCapabilityPortsV0{
		RegistrationAuthority: dataIngestionToolRegistrationAuthorityFakeV0{registration: registration},
		SnapshotResolver:      snapshot,
		BindingAuthority:      dataIngestionToolBindingAuthorityFakeV0{binding: dataIngestionToolBindingForTestV0()},
		Registry:              dataIngestionToolRegistryFakeV0{manifest: manifest},
		Validator:             dataIngestionToolValidatorFakeV0{},
		Effect:                effect,
		ReceiptStore:          &dataIngestionToolReceiptStoreFakeV0{},
	}
	intent := DataIngestionToolAttachIntentV0{
		RequestRef: "request:data-ingestion", IdempotencyKey: "idempotency:data-ingestion",
		RequestedBy: "user:operator", AppRef: "app:generated", BindingRef: "binding:data-ingestion",
		DataBundleRef: "bundle:data-ingestion", WriteSet: []string{"internal/adapters/data-ingestion"},
	}
	plan, err := PrepareDataIngestionToolAttachV0(context.Background(), intent, ports)
	if err != nil || len(plan.Issues) != 0 || plan.Operation != toolcapability.ToolOperationPrepareV0 {
		t.Fatalf("prepare plan=%+v err=%v", plan, err)
	}
	if snapshot.calls != 1 || plan.Bundle.BundleRef != registration.Descriptor.BundleRef || plan.Snapshot.HandleRef != "handle:data-ingestion" {
		t.Fatalf("prepare did not delegate snapshot: calls=%d plan=%+v", snapshot.calls, plan)
	}
	receipt, err := AttachDataIngestionToolV0(context.Background(), intent, ports)
	if err != nil || receipt.Status != toolcapability.ToolReceiptStatusAttachedV0 {
		t.Fatalf("attach receipt=%+v err=%v", receipt, err)
	}
	if effect.materialized == nil || effect.materialized.OperationRef != receipt.OperationRef || effect.materialized.Snapshot.HandleRef != "handle:data-ingestion" {
		t.Fatalf("effect did not receive SDK plan: %+v", effect.materialized)
	}
}

func dataIngestionToolRegistrationForTestV0() DataIngestionToolRegistrationV0 {
	return DataIngestionToolRegistrationV0{
		RegistrationRef: "registration:data-ingestion",
		Descriptor: ingestion.DataIngestionBundleDescriptorV0{
			BundleRef: "bundle:data-ingestion", Capability: ingestion.DefaultDataIngestionToolCapabilityDescriptorV0(),
			AttachMode: ingestion.DataIngestionToolAttachModeEmbeddedModuleV0, ModuleRef: "module:data-ingestion",
			ConfigurationRef: "config:data-ingestion", I18NRef: "i18n:data-ingestion",
			TestArtifactRefs: []string{"test:data-ingestion"},
			ArtifactHashes:   []ingestion.DataIngestionArtifactHashV0{{ArtifactRef: "module:data-ingestion", Hash: dataIngestionToolTestHashV0}},
		},
		ManifestRef: "manifest:data-ingestion", ToolRef: "tool:data-ingestion", ToolVersion: "1.0.0",
		I18nCatalogs:  []toolcapability.I18nCatalogRefV0{{Locale: "es", CatalogRef: "i18n:data-ingestion", ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		Compatibility: toolcapability.ToolCompatibilityV0{AppContractRefs: []string{"contract:generated-app"}},
	}
}

func dataIngestionToolManifestForTestV0(registration DataIngestionToolRegistrationV0) toolcapability.CapabilityManifestV0 {
	return toolcapability.CapabilityManifestV0{
		SchemaVersion: toolcapability.CapabilityManifestSchemaV0, ManifestRef: registration.ManifestRef,
		CapabilityRef: registration.Descriptor.Capability.CapabilityRef, ToolRef: registration.ToolRef, Version: registration.ToolVersion,
		InputSchemaRef: "schema:input", OutputSchemaRef: "schema:output", Locales: []string{"es"},
		EffectProfile:   toolcapability.CapabilityEffectProfileV0{ProfileRef: registration.Descriptor.Capability.PermissionProfileRef, EffectRefs: []string{"effect:data-ingestion"}},
		Budgets:         toolcapability.CapabilityBudgetsV0{BudgetRefs: []string{"budget:data-ingestion"}},
		Idempotency:     toolcapability.CapabilityIdempotencyV0{KeySchemaRef: "schema:idempotency", ScopeRef: "scope:app"},
		ConfigSchemaRef: "schema:config", TestRefs: []string{"test:data-ingestion"}, ReceiptSchemaRef: ingestion.DataIngestionReceiptSchemaVersionV0,
		IntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0}, Compatibility: toolcapability.ToolCompatibilityV0{AppContractRefs: []string{"contract:generated-app"}},
	}
}

func dataIngestionToolBindingForTestV0() toolcapability.GeneratedAppToolBindingV0 {
	return toolcapability.GeneratedAppToolBindingV0{
		SchemaVersion: toolcapability.GeneratedAppToolBindingSchemaV0, BindingRef: "binding:data-ingestion", AppRef: "app:generated", AppVersion: "1.0.0",
		ArchitectureContractRef: "contract:generated-app", PortRefs: []string{"port:data-ingestion"}, AdapterRefs: []string{"adapter:data-ingestion"},
		CompositionRef: "composition:generated", SupportedIntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0},
		ConfigSchemaRefs: []string{"schema:config"}, I18nCatalogRefs: []string{"i18n:data-ingestion"}, AllowedEffectProfileRefs: []string{"data_ingestion"},
		AllowedWriteSet: []string{"internal/adapters"}, CompatibilityRefs: []string{"contract:generated-app"},
	}
}

type dataIngestionToolRegistrationAuthorityFakeV0 struct {
	registration DataIngestionToolRegistrationV0
}

func (fake dataIngestionToolRegistrationAuthorityFakeV0) ResolveDataIngestionToolRegistrationV0(context.Context, string) (DataIngestionToolRegistrationV0, error) {
	return fake.registration, nil
}

type dataIngestionToolSnapshotResolverFakeV0 struct{ calls int }

func (fake *dataIngestionToolSnapshotResolverFakeV0) ResolveVerifiedDataIngestionToolSnapshotV0(_ context.Context, manifest toolcapability.CapabilityManifestV0, bundle toolcapability.ToolBundleV0) (toolcapability.ToolBundleSnapshotV0, error) {
	fake.calls++
	return toolcapability.ToolBundleSnapshotV0{
		SchemaVersion: toolcapability.ToolBundleSnapshotSchemaV0, SnapshotRef: toolcapability.ToolBundleSnapshotRefV0(bundle.ContentHash),
		HandleRef: "handle:data-ingestion", VerificationRef: "verification:data-ingestion", ManifestRef: manifest.ManifestRef,
		BundleRef: bundle.BundleRef, ContentHash: bundle.ContentHash,
	}, nil
}

type dataIngestionToolBindingAuthorityFakeV0 struct {
	binding toolcapability.GeneratedAppToolBindingV0
}

func (fake dataIngestionToolBindingAuthorityFakeV0) ResolveGeneratedAppToolBindingV0(context.Context, string, string) (toolcapability.GeneratedAppToolBindingV0, error) {
	return fake.binding, nil
}

type dataIngestionToolRegistryFakeV0 struct {
	manifest toolcapability.CapabilityManifestV0
}

func (fake dataIngestionToolRegistryFakeV0) GetCapabilityManifestV0(context.Context, string) (toolcapability.CapabilityManifestV0, error) {
	return fake.manifest, nil
}

type dataIngestionToolValidatorFakeV0 struct{}

func (dataIngestionToolValidatorFakeV0) ValidateGeneratedAppToolAttachPlanV0(context.Context, toolcapability.GeneratedAppToolAttachPlanV0) []toolcapability.ToolCapabilityIssueV0 {
	return nil
}

type dataIngestionToolEffectFakeV0 struct {
	materialized *toolcapability.GeneratedAppToolAttachPlanV0
}

func (fake *dataIngestionToolEffectFakeV0) MaterializeDataIngestionToolV0(_ context.Context, _ ingestion.DataIngestionBundleDescriptorV0, plan toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	fake.materialized = &plan
	return toolcapability.ToolMaterializationResultV0{Status: "materialized", EvidenceRefs: []string{"evidence:data-ingestion"}}, nil
}
func (*dataIngestionToolEffectFakeV0) UninstallDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolMaterializationResultV0, error) {
	return toolcapability.ToolMaterializationResultV0{}, nil
}
func (*dataIngestionToolEffectFakeV0) RollbackDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.ToolRollbackRequestV0) (toolcapability.ToolMaterializationResultV0, error) {
	return toolcapability.ToolMaterializationResultV0{}, nil
}
func (*dataIngestionToolEffectFakeV0) ReconcileDataIngestionToolV0(context.Context, ingestion.DataIngestionBundleDescriptorV0, toolcapability.GeneratedAppToolAttachPlanV0) (toolcapability.ToolOperationReconciliationV0, error) {
	return toolcapability.ToolOperationReconciliationV0{}, nil
}

type dataIngestionToolReceiptStoreFakeV0 struct {
	record *toolcapability.ToolOperationRecordV0
}

func (fake *dataIngestionToolReceiptStoreFakeV0) ClaimToolOperationV0(_ context.Context, record toolcapability.ToolOperationRecordV0) (toolcapability.ToolOperationRecordV0, bool, error) {
	if fake.record != nil {
		return *fake.record, false, nil
	}
	fake.record = &record
	return record, true, nil
}
func (fake *dataIngestionToolReceiptStoreFakeV0) GetToolOperationRecordV0(context.Context, string) (toolcapability.ToolOperationRecordV0, bool, error) {
	if fake.record == nil {
		return toolcapability.ToolOperationRecordV0{}, false, nil
	}
	return *fake.record, true, nil
}
func (fake *dataIngestionToolReceiptStoreFakeV0) GetToolOperationRecordByReceiptRefV0(context.Context, string) (toolcapability.ToolOperationRecordV0, bool, error) {
	if fake.record == nil {
		return toolcapability.ToolOperationRecordV0{}, false, nil
	}
	return *fake.record, true, nil
}
func (fake *dataIngestionToolReceiptStoreFakeV0) CompareAndSwapToolOperationV0(_ context.Context, _ string, _ uint64, receipt toolcapability.ToolOperationReceiptV0) (toolcapability.ToolOperationRecordV0, bool, error) {
	if fake.record == nil {
		return toolcapability.ToolOperationRecordV0{}, false, nil
	}
	fake.record.Receipt = receipt
	return *fake.record, true, nil
}
func (*dataIngestionToolReceiptStoreFakeV0) ListToolOperationReceiptsV0(context.Context, toolcapability.ToolOperationReceiptFilterV0) ([]toolcapability.ToolOperationReceiptV0, error) {
	return nil, nil
}
