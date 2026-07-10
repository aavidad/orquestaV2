package orquestatoolcapability

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPrepareResolvesBindingAuthorityAndValidatorReceivesResolvedPlanV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	plan, err := PrepareGeneratedAppToolAttachV0(context.Background(), request, ports.asPorts())
	if err != nil || len(plan.Issues) != 0 || !plan.DryRun || !plan.EffectAuthorized {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if ports.authority.appRef != request.AppRef || ports.authority.bindingRef != request.BindingRef {
		t.Fatalf("authority lookup app=%q binding=%q", ports.authority.appRef, ports.authority.bindingRef)
	}
	validated := ports.validator.validatedPlan()
	if !reflect.DeepEqual(validated.Binding, ports.binding) || validated.PlanFingerprint == "" || validated.Snapshot.HandleRef != ports.snapshot.HandleRef {
		t.Fatalf("validated=%+v authority=%+v", validated, ports.binding)
	}
	if ports.installer.materializeCallCount() != 0 {
		t.Fatal("prepare produced an effect")
	}
}

func TestPrepareRejectsAuthorityAndResolvedRefMismatchesV0(t *testing.T) {
	tests := map[string]func(*toolCapabilityTestPortsV0){
		"binding_app": func(ports *toolCapabilityTestPortsV0) { ports.binding.AppRef = "other-app" },
		"binding_ref": func(ports *toolCapabilityTestPortsV0) { ports.binding.BindingRef = "other-binding" },
		"manifest_ref": func(ports *toolCapabilityTestPortsV0) {
			ports.manifest.ManifestRef = "other-manifest"
			ports.bundle.ManifestRef = ports.manifest.ManifestRef
			ports.bundle.ContentHash = CalculateToolBundleContentHashV0(ports.manifest, ports.bundle)
			ports.snapshot = validToolBundleSnapshotV0(ports.manifest, ports.bundle)
		},
		"bundle_ref": func(ports *toolCapabilityTestPortsV0) {
			ports.bundle.BundleRef = "other-bundle"
			ports.bundle.ContentHash = CalculateToolBundleContentHashV0(ports.manifest, ports.bundle)
			ports.snapshot = validToolBundleSnapshotV0(ports.manifest, ports.bundle)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			ports := validToolCapabilityPortsV0()
			mutate(&ports)
			plan, err := PrepareGeneratedAppToolAttachV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
			if err != nil || !hasToolCapabilityIssueV0(plan.Issues, ErrToolCapabilityAuthorityMismatchV0) {
				t.Fatalf("plan=%+v err=%v", plan, err)
			}
		})
	}
}

func TestAttachRejectsSelfAuthorizationAttemptV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	ports.binding.AllowedEffectProfileRefs = []string{"different-profile"}
	ports.binding.AllowedWriteSet = []string{"other/root"}
	receipt, err := AttachGeneratedAppToolV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusBlockedV0 || ports.installer.materializeCallCount() != 0 {
		t.Fatalf("receipt=%+v err=%v calls=%d", receipt, err, ports.installer.materializeCallCount())
	}
	if !hasToolCapabilityIssueV0(receipt.Issues, ErrToolCapabilityEffectUnauthorizedV0) ||
		!hasToolCapabilityIssueV0(receipt.Issues, ErrToolCapabilityWriteSetInvalidV0) {
		t.Fatalf("issues=%+v", receipt.Issues)
	}
}

func TestAttachDoesNotPersistUnresolvedPlanV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	callPorts := ports.asPorts()
	callPorts.BindingAuthority = nil
	if _, err := AttachGeneratedAppToolV0(context.Background(), validToolAttachRequestV0(), callPorts); err == nil || !strings.Contains(err.Error(), ErrToolCapabilityPlanUnresolvedV0) {
		t.Fatalf("err=%v", err)
	}
	if ports.store.backend.recordCount() != 0 {
		t.Fatalf("records=%d", ports.store.backend.recordCount())
	}
}

func TestSnapshotMustBeContentAddressedAndMaterializerConsumesVerifiedHandleV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	ports.snapshot.ContentHash = testToolCapabilityHashV0('f')
	receipt, err := AttachGeneratedAppToolV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusBlockedV0 || !hasToolCapabilityIssueV0(receipt.Issues, ErrToolCapabilitySnapshotInvalidV0) {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if ports.installer.materializeCallCount() != 0 {
		t.Fatal("invalid snapshot reached materializer")
	}

	ports = validToolCapabilityPortsV0()
	receipt, err = AttachGeneratedAppToolV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusAttachedV0 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	plans := ports.installer.materializedPlans()
	if len(plans) != 1 || plans[0].Snapshot.HandleRef != ports.snapshot.HandleRef || plans[0].Snapshot.SnapshotRef != ports.snapshot.SnapshotRef {
		t.Fatalf("plans=%+v snapshot=%+v", plans, ports.snapshot)
	}
}

func TestSameIdempotencyKeyWithDifferentPayloadConflictsV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	first, err := AttachGeneratedAppToolV0(context.Background(), request, ports.asPorts())
	if err != nil || first.Status != ToolReceiptStatusAttachedV0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	request.ConfigRef = "config-other"
	second, err := AttachGeneratedAppToolV0(context.Background(), request, ports.asPorts())
	if err == nil || !strings.Contains(err.Error(), ErrToolCapabilityIdempotencyConflictV0) || second.ReceiptRef != first.ReceiptRef {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if ports.installer.materializeCallCount() != 1 {
		t.Fatalf("calls=%d", ports.installer.materializeCallCount())
	}
}

func TestTargetLeaseBlocksOverlappingOperationWithDifferentIdempotencyV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	request.Operation = ToolOperationAttachV0
	claimed := claimPreparedPlanV0(t, request, &ports)
	if claimed.Receipt.Status != ToolReceiptStatusClaimedV0 {
		t.Fatalf("claimed=%+v", claimed)
	}
	request.IdempotencyKey = "different-idempotency"
	receipt, err := AttachGeneratedAppToolV0(context.Background(), request, ports.asPorts())
	if err == nil || !strings.Contains(err.Error(), ErrToolCapabilityTargetBusyV0) || receipt.ReceiptRef != "" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if ports.installer.materializeCallCount() != 0 {
		t.Fatal("busy target was materialized")
	}
}

func TestConcurrentClaimsAcrossLogicalStoresMaterializeExactlyOnceV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	stores := []*fakeToolCapabilityReceiptStoreV0{
		{backend: ports.store.backend}, {backend: ports.store.backend},
	}
	start := make(chan struct{})
	var wait sync.WaitGroup
	errors := make(chan error, len(stores))
	for _, store := range stores {
		wait.Add(1)
		go func(store *fakeToolCapabilityReceiptStoreV0) {
			defer wait.Done()
			<-start
			callPorts := ports.asPorts()
			callPorts.ReceiptStore = store
			_, err := AttachGeneratedAppToolV0(context.Background(), request, callPorts)
			errors <- err
		}(store)
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("attach: %v", err)
		}
	}
	if ports.installer.effectCountValue() != 1 || ports.installer.materializeCallCount() != 1 {
		t.Fatalf("effects=%d calls=%d", ports.installer.effectCountValue(), ports.installer.materializeCallCount())
	}
}

func TestResumeReconcilesCrashBeforeEffectV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	request.Operation = ToolOperationAttachV0
	claimed := claimPreparedPlanV0(t, request, &ports)
	receipt, err := ResumeGeneratedAppToolOperationV0(context.Background(), claimed.Plan.OperationRef, ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusAttachedV0 || receipt.StateVersion != 3 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if ports.installer.effectCountValue() != 1 || ports.installer.materializeCallCount() != 1 {
		t.Fatalf("effects=%d calls=%d", ports.installer.effectCountValue(), ports.installer.materializeCallCount())
	}
}

func TestReconcileCrashAfterEffectDoesNotDuplicateV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	request.Operation = ToolOperationAttachV0
	claimed := claimPreparedPlanV0(t, request, &ports)
	if _, err := ports.installer.MaterializeToolBundleV0(context.Background(), claimed.Plan); err != nil {
		t.Fatal(err)
	}
	receipt, err := ReconcileGeneratedAppToolOperationV0(context.Background(), claimed.Plan.OperationRef, ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusAttachedV0 || receipt.StateVersion != 2 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	if ports.installer.effectCountValue() != 1 || ports.installer.materializeCallCount() != 1 {
		t.Fatalf("effects=%d calls=%d", ports.installer.effectCountValue(), ports.installer.materializeCallCount())
	}
}

func TestConcurrentResumeUsesCASAndProducesOneEffectV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	request.Operation = ToolOperationAttachV0
	claimed := claimPreparedPlanV0(t, request, &ports)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, _ = ResumeGeneratedAppToolOperationV0(context.Background(), claimed.Plan.OperationRef, ports.asPorts())
		}()
	}
	close(start)
	wait.Wait()
	if ports.installer.effectCountValue() != 1 || ports.installer.materializeCallCount() != 1 {
		t.Fatalf("effects=%d calls=%d", ports.installer.effectCountValue(), ports.installer.materializeCallCount())
	}
	final, found, err := ports.store.GetToolOperationRecordV0(context.Background(), claimed.Plan.OperationRef)
	if err != nil || !found || final.Receipt.Status != ToolReceiptStatusAttachedV0 {
		t.Fatalf("final=%+v found=%v err=%v", final, found, err)
	}
}

func TestReconcileErrorTransitionsDurablyToRecoveryRequiredV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	request := validToolAttachRequestV0()
	request.Operation = ToolOperationAttachV0
	claimed := claimPreparedPlanV0(t, request, &ports)
	ports.installer.reconcileErr = errors.New("adapter unavailable")
	receipt, err := ReconcileGeneratedAppToolOperationV0(context.Background(), claimed.Plan.OperationRef, ports.asPorts())
	if err == nil || receipt.Status != ToolReceiptStatusRecoveryRequiredV0 || receipt.RecoveryRef == "" || receipt.StateVersion != 2 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	durable, found, getErr := ports.store.GetToolOperationRecordV0(context.Background(), claimed.Plan.OperationRef)
	if getErr != nil || !found || !reflect.DeepEqual(durable.Receipt, receipt) {
		t.Fatalf("durable=%+v found=%v err=%v", durable, found, getErr)
	}
}

func TestUpgradeValidatesPreviousTerminalAndRestoresExactSnapshotV0(t *testing.T) {
	ports, previous := validUpgradeToolCapabilityPortsV0(t)
	ports.installer.materializeErr = errors.New("upgrade failed")
	request := validToolAttachRequestV0()
	request.IdempotencyKey = "upgrade-idempotency"
	request.PreviousReceiptRef = previous.Receipt.ReceiptRef
	receipt, err := UpgradeGeneratedAppToolV0(context.Background(), request, ports.asPorts())
	if err != nil || receipt.Status != ToolReceiptStatusRolledBackV0 {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
	rollbacks := ports.installer.rollbackRequests()
	if len(rollbacks) != 1 || rollbacks[0].PreviousReceipt.ReceiptRef != previous.Receipt.ReceiptRef ||
		rollbacks[0].PreviousPlan.Snapshot.HandleRef != previous.Plan.Snapshot.HandleRef ||
		rollbacks[0].PreviousPlan.Snapshot.ContentHash != previous.Plan.Snapshot.ContentHash {
		t.Fatalf("rollbacks=%+v previous=%+v", rollbacks, previous)
	}
}

func TestUpgradeRejectsMissingUpgradeFromAndInadequatePreviousStateV0(t *testing.T) {
	t.Run("upgrade_from", func(t *testing.T) {
		ports, previous := validUpgradeToolCapabilityPortsV0(t)
		ports.bundle.Compatibility.UpgradeFromRefs = nil
		ports.bundle.ContentHash = CalculateToolBundleContentHashV0(ports.manifest, ports.bundle)
		ports.snapshot = validToolBundleSnapshotV0(ports.manifest, ports.bundle)
		request := validToolAttachRequestV0()
		request.IdempotencyKey = "upgrade-no-compat"
		request.PreviousReceiptRef = previous.Receipt.ReceiptRef
		receipt, err := UpgradeGeneratedAppToolV0(context.Background(), request, ports.asPorts())
		if err != nil || receipt.Status != ToolReceiptStatusBlockedV0 || !hasToolCapabilityIssueV0(receipt.Issues, ErrToolCapabilityUpgradeIncompatibleV0) {
			t.Fatalf("receipt=%+v err=%v", receipt, err)
		}
	})

	t.Run("previous_state", func(t *testing.T) {
		ports, previous := validUpgradeToolCapabilityPortsV0(t)
		ports.store.backend.replaceStatus(previous.Plan.OperationRef, ToolReceiptStatusRolledBackV0)
		request := validToolAttachRequestV0()
		request.IdempotencyKey = "upgrade-bad-state"
		request.PreviousReceiptRef = previous.Receipt.ReceiptRef
		receipt, err := UpgradeGeneratedAppToolV0(context.Background(), request, ports.asPorts())
		if err != nil || receipt.Status != ToolReceiptStatusBlockedV0 || !hasToolCapabilityIssueV0(receipt.Issues, ErrToolCapabilityPreviousReceiptInvalidV0) {
			t.Fatalf("receipt=%+v err=%v", receipt, err)
		}
	})
}

func TestStatusRequiresAppAndToolScopeV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	for _, filter := range []ToolOperationReceiptFilterV0{{}, {AppRef: "app-ref"}, {ToolRef: "tool-ref"}, {ReceiptRef: "receipt-ref"}} {
		if _, err := StatusGeneratedAppToolV0(context.Background(), filter, ports.store); err == nil || !strings.Contains(err.Error(), ErrToolCapabilityStatusScopeRequiredV0) {
			t.Fatalf("filter=%+v err=%v", filter, err)
		}
	}
	receipt, err := AttachGeneratedAppToolV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
	if err != nil {
		t.Fatal(err)
	}
	found, err := StatusGeneratedAppToolV0(context.Background(), ToolOperationReceiptFilterV0{AppRef: receipt.AppRef, ToolRef: receipt.ToolRef}, ports.store)
	if err != nil || len(found) != 1 || found[0].ReceiptRef != receipt.ReceiptRef {
		t.Fatalf("found=%+v err=%v", found, err)
	}
}

func TestPlanFingerprintCoversImmutableAuthorityAndPayloadV0(t *testing.T) {
	ports := validToolCapabilityPortsV0()
	plan, err := PrepareGeneratedAppToolAttachV0(context.Background(), validToolAttachRequestV0(), ports.asPorts())
	if err != nil || len(plan.Issues) != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	base := plan.PlanFingerprint
	mutations := map[string]func(*GeneratedAppToolAttachPlanV0){
		"manifest":    func(value *GeneratedAppToolAttachPlanV0) { value.Manifest.ManifestRef = "manifest-other" },
		"bundle_hash": func(value *GeneratedAppToolAttachPlanV0) { value.Bundle.ContentHash = testToolCapabilityHashV0('e') },
		"config":      func(value *GeneratedAppToolAttachPlanV0) { value.ConfigRef = "config-other" },
		"write_set":   func(value *GeneratedAppToolAttachPlanV0) { value.WriteSet = []string{"internal/adapters/other"} },
		"binding":     func(value *GeneratedAppToolAttachPlanV0) { value.Binding.AppVersion = "2.0.0" },
		"mode":        func(value *GeneratedAppToolAttachPlanV0) { value.IntegrationMode = IntegrationModeLocalSidecarV0 },
		"operation":   func(value *GeneratedAppToolAttachPlanV0) { value.Operation = ToolOperationUpgradeV0 },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := plan
			mutate(&changed)
			if got := CalculateGeneratedAppToolPlanFingerprintV0(changed); got == base {
				t.Fatalf("mutation %s did not change fingerprint", name)
			}
		})
	}
}

func TestToolBundleContentHashAndDuplicateValidationV0(t *testing.T) {
	manifest := validToolCapabilityManifestV0()
	bundle := validToolBundleV0(manifest)
	original := bundle.ContentHash
	bundle.ArtifactRefs[0].ContentHash = testToolCapabilityHashV0('e')
	if issues := ValidateToolBundleV0(manifest, bundle); !hasToolCapabilityIssueV0(issues, ErrToolCapabilityHashMismatchV0) {
		t.Fatalf("issues=%+v", issues)
	}
	bundle = validToolBundleV0(manifest)
	bundle.ArtifactRefs = append(bundle.ArtifactRefs, HashedArtifactV0{ArtifactRef: "artifact-other", ContentHash: bundle.ArtifactRefs[0].ContentHash})
	bundle.ContentHash = CalculateToolBundleContentHashV0(manifest, bundle)
	if issues := ValidateToolBundleV0(manifest, bundle); !hasToolCapabilityIssueV0(issues, ErrToolCapabilityDuplicateV0) {
		t.Fatalf("issues=%+v", issues)
	}
	if original == "" {
		t.Fatal("empty content hash")
	}
}

func TestToolBundleContentHashEquivalentOrderAndGoldenV0(t *testing.T) {
	manifest := validToolCapabilityManifestV0()
	manifest.Locales = []string{"en-US", "es-ES"}
	manifest.TestRefs = []string{"test-b", "test-a"}
	manifest.ConnectorSlots = []ConnectorSlotV0{
		{SlotRef: "slot-b", ContractRef: "contract-b", CapabilityRefs: []string{"cap-b", "cap-a"}},
		{SlotRef: "slot-a", ContractRef: "contract-a", Required: true},
	}
	bundle := validToolBundleV0(manifest)
	bundle.ArtifactRefs = []HashedArtifactV0{
		{ArtifactRef: "artifact-b", ContentHash: testToolCapabilityHashV0('c')},
		{ArtifactRef: "artifact-a", ContentHash: testToolCapabilityHashV0('a')},
	}
	bundle.I18nCatalogs = []I18nCatalogRefV0{
		{Locale: "es-ES", CatalogRef: "catalog-es", ContentHash: testToolCapabilityHashV0('b')},
		{Locale: "en-US", CatalogRef: "catalog-en", ContentHash: testToolCapabilityHashV0('d')},
	}
	bundle.MigrationRefs = []string{"migration-b", "migration-a"}
	bundle.TestRefs = []string{"test-b", "test-a"}
	hash := CalculateToolBundleContentHashV0(manifest, bundle)

	reorderedManifest := manifest
	reorderedManifest.Locales = []string{"es-ES", "en-US"}
	reorderedManifest.TestRefs = []string{"test-a", "test-b"}
	reorderedManifest.ConnectorSlots = []ConnectorSlotV0{manifest.ConnectorSlots[1], manifest.ConnectorSlots[0]}
	reorderedManifest.ConnectorSlots[1].CapabilityRefs = []string{"cap-a", "cap-b"}
	reorderedBundle := bundle
	reorderedBundle.ArtifactRefs = []HashedArtifactV0{bundle.ArtifactRefs[1], bundle.ArtifactRefs[0]}
	reorderedBundle.I18nCatalogs = []I18nCatalogRefV0{bundle.I18nCatalogs[1], bundle.I18nCatalogs[0]}
	reorderedBundle.MigrationRefs = []string{"migration-a", "migration-b"}
	reorderedBundle.TestRefs = []string{"test-a", "test-b"}
	if got := CalculateToolBundleContentHashV0(reorderedManifest, reorderedBundle); got != hash {
		t.Fatalf("equivalent order changed hash: got=%s want=%s", got, hash)
	}

	baseManifest := validToolCapabilityManifestV0()
	baseBundle := validToolBundleV0(baseManifest)
	const expected = "sha256:bf7841924c26bf59dd737a89cb2136f827a20c07330354e0333cf8a6fd2f0d3e"
	if baseBundle.ContentHash != expected {
		t.Fatalf("content hash contract changed: got=%s want=%s", baseBundle.ContentHash, expected)
	}
}

func TestCapabilityContractsRejectAllAmbiguousDuplicatesV0(t *testing.T) {
	manifestCases := map[string]func(*CapabilityManifestV0){
		"locales": func(value *CapabilityManifestV0) { value.Locales = []string{"es-ES", "es-ES"} },
		"modes": func(value *CapabilityManifestV0) {
			value.IntegrationModes = []string{IntegrationModeEmbeddedModuleV0, IntegrationModeEmbeddedModuleV0}
		},
		"refs": func(value *CapabilityManifestV0) { value.TestRefs = []string{"test-ref", "test-ref"} },
		"slots": func(value *CapabilityManifestV0) {
			value.ConnectorSlots = append(value.ConnectorSlots, value.ConnectorSlots[0])
		},
	}
	for name, mutate := range manifestCases {
		t.Run("manifest_"+name, func(t *testing.T) {
			manifest := validToolCapabilityManifestV0()
			mutate(&manifest)
			if issues := ValidateCapabilityManifestV0(manifest); !hasToolCapabilityIssueV0(issues, ErrToolCapabilityDuplicateV0) {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}

	manifest := validToolCapabilityManifestV0()
	bundleCases := map[string]func(*ToolBundleV0){
		"artifacts": func(value *ToolBundleV0) { value.ArtifactRefs = append(value.ArtifactRefs, value.ArtifactRefs[0]) },
		"catalogs":  func(value *ToolBundleV0) { value.I18nCatalogs = append(value.I18nCatalogs, value.I18nCatalogs[0]) },
		"hashes": func(value *ToolBundleV0) {
			value.ArtifactRefs = append(value.ArtifactRefs, HashedArtifactV0{ArtifactRef: "artifact-other", ContentHash: value.ArtifactRefs[0].ContentHash})
		},
	}
	for name, mutate := range bundleCases {
		t.Run("bundle_"+name, func(t *testing.T) {
			bundle := validToolBundleV0(manifest)
			mutate(&bundle)
			bundle.ContentHash = CalculateToolBundleContentHashV0(manifest, bundle)
			if issues := ValidateToolBundleV0(manifest, bundle); !hasToolCapabilityIssueV0(issues, ErrToolCapabilityDuplicateV0) {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}
	binding := validToolBindingV0()
	binding.PortRefs = []string{"port-ref", "port-ref"}
	if issues := ValidateGeneratedAppToolBindingV0(binding); !hasToolCapabilityIssueV0(issues, ErrToolCapabilityDuplicateV0) {
		t.Fatalf("binding issues=%+v", issues)
	}
}

func TestToolCapabilityWriteSetRejectsAdversarialPathsV0(t *testing.T) {
	invalid := []string{
		"https://example.invalid/tool", "file:relative", "C:relative", "C:/absolute", `C:\absolute`,
		`\\server\share`, "//server/share", "~", "~/tool", "internal/\x00tool", "../tool",
		"internal/../tool", "internal/./tool", "/absolute", "internal//tool", "internal/tool/", " internal/tool",
	}
	for _, value := range invalid {
		if safeToolCapabilityWriteSetV0(value) {
			t.Errorf("accepted %q", value)
		}
	}
	if toolCapabilityWriteSetWithinV0([]string{"internal/adapter/tool"}, []string{"internal/adapt"}) {
		t.Fatal("string prefix treated as segment prefix")
	}
}

type toolCapabilityTestPortsV0 struct {
	manifest  CapabilityManifestV0
	bundle    ToolBundleV0
	snapshot  ToolBundleSnapshotV0
	binding   GeneratedAppToolBindingV0
	authority *fakeToolBindingAuthorityV0
	validator *fakeToolCapabilityValidatorV0
	installer *fakeToolCapabilityInstallerV0
	store     *fakeToolCapabilityReceiptStoreV0
}

func validToolCapabilityPortsV0() toolCapabilityTestPortsV0 {
	manifest := validToolCapabilityManifestV0()
	bundle := validToolBundleV0(manifest)
	binding := validToolBindingV0()
	return toolCapabilityTestPortsV0{
		manifest:  manifest,
		bundle:    bundle,
		snapshot:  validToolBundleSnapshotV0(manifest, bundle),
		binding:   binding,
		authority: &fakeToolBindingAuthorityV0{},
		validator: &fakeToolCapabilityValidatorV0{},
		installer: newFakeToolCapabilityInstallerV0(),
		store:     &fakeToolCapabilityReceiptStoreV0{backend: newFakeToolCapabilityReceiptBackendV0()},
	}
}

func (ports toolCapabilityTestPortsV0) asPorts() ToolCapabilityPortsV0 {
	ports.authority.setBinding(ports.binding)
	return ToolCapabilityPortsV0{
		Catalog:          fakeToolCapabilityCatalogV0{manifests: []CapabilityManifestV0{ports.manifest}},
		BindingAuthority: ports.authority,
		Registry:         fakeToolCapabilityRegistryV0{manifest: ports.manifest},
		SnapshotResolver: fakeToolBundleSnapshotResolverV0{resolved: ResolvedToolBundleSnapshotV0{Bundle: ports.bundle, Snapshot: ports.snapshot}},
		Validator:        ports.validator,
		Installer:        ports.installer,
		ReceiptStore:     ports.store,
	}
}

func claimPreparedPlanV0(t *testing.T, request GeneratedAppToolAttachRequestV0, ports *toolCapabilityTestPortsV0) ToolOperationRecordV0 {
	t.Helper()
	plan, err := buildGeneratedAppToolAttachPlanV0(context.Background(), request, ports.asPorts(), false)
	if err != nil || len(plan.Issues) != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	record := newToolOperationRecordV0(plan, ToolReceiptStatusClaimedV0)
	claimed, acquired, err := ports.store.ClaimToolOperationV0(context.Background(), record)
	if err != nil || !acquired {
		t.Fatalf("claimed=%+v acquired=%v err=%v", claimed, acquired, err)
	}
	return claimed
}

func validUpgradeToolCapabilityPortsV0(t *testing.T) (toolCapabilityTestPortsV0, ToolOperationRecordV0) {
	t.Helper()
	oldPorts := validToolCapabilityPortsV0()
	oldPorts.manifest.ManifestRef = "manifest-old"
	oldPorts.manifest.Version = "1.1.0"
	oldPorts.bundle = validToolBundleV0(oldPorts.manifest)
	oldPorts.bundle.BundleRef = "bundle-old"
	oldPorts.bundle.ContentHash = CalculateToolBundleContentHashV0(oldPorts.manifest, oldPorts.bundle)
	oldPorts.snapshot = validToolBundleSnapshotV0(oldPorts.manifest, oldPorts.bundle)
	oldRequest := validToolAttachRequestV0()
	oldRequest.Operation = ToolOperationAttachV0
	oldRequest.ManifestRef = oldPorts.manifest.ManifestRef
	oldRequest.BundleRef = oldPorts.bundle.BundleRef
	oldRequest.IdempotencyKey = "old-idempotency"
	oldPlan, err := buildGeneratedAppToolAttachPlanV0(context.Background(), oldRequest, oldPorts.asPorts(), false)
	if err != nil || len(oldPlan.Issues) != 0 {
		t.Fatalf("old plan=%+v err=%v", oldPlan, err)
	}
	previous := newToolOperationRecordV0(oldPlan, ToolReceiptStatusAttachedV0)

	ports := validToolCapabilityPortsV0()
	ports.store = oldPorts.store
	ports.bundle.Compatibility.UpgradeFromRefs = []string{previous.Receipt.BundleRef}
	ports.bundle.ContentHash = CalculateToolBundleContentHashV0(ports.manifest, ports.bundle)
	ports.snapshot = validToolBundleSnapshotV0(ports.manifest, ports.bundle)
	ports.store.backend.seed(previous)
	return ports, previous
}

type fakeToolCapabilityCatalogV0 struct{ manifests []CapabilityManifestV0 }

func (fake fakeToolCapabilityCatalogV0) ListCapabilityManifestsV0(_ context.Context, _ CapabilityManifestFilterV0) ([]CapabilityManifestV0, error) {
	return append([]CapabilityManifestV0(nil), fake.manifests...), nil
}

type fakeToolBindingAuthorityV0 struct {
	mu         sync.Mutex
	binding    GeneratedAppToolBindingV0
	appRef     string
	bindingRef string
	err        error
}

func (fake *fakeToolBindingAuthorityV0) setBinding(binding GeneratedAppToolBindingV0) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.binding = binding
}

func (fake *fakeToolBindingAuthorityV0) ResolveGeneratedAppToolBindingV0(_ context.Context, appRef, bindingRef string) (GeneratedAppToolBindingV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.appRef = appRef
	fake.bindingRef = bindingRef
	return fake.binding, fake.err
}

type fakeToolCapabilityRegistryV0 struct{ manifest CapabilityManifestV0 }

func (fake fakeToolCapabilityRegistryV0) GetCapabilityManifestV0(_ context.Context, _ string) (CapabilityManifestV0, error) {
	return fake.manifest, nil
}

type fakeToolBundleSnapshotResolverV0 struct {
	resolved ResolvedToolBundleSnapshotV0
	err      error
}

func (fake fakeToolBundleSnapshotResolverV0) ResolveVerifiedToolBundleSnapshotV0(_ context.Context, _ CapabilityManifestV0, _ string) (ResolvedToolBundleSnapshotV0, error) {
	return fake.resolved, fake.err
}

type fakeToolCapabilityValidatorV0 struct {
	mu     sync.Mutex
	issues []ToolCapabilityIssueV0
	plan   GeneratedAppToolAttachPlanV0
}

func (fake *fakeToolCapabilityValidatorV0) ValidateGeneratedAppToolAttachPlanV0(_ context.Context, plan GeneratedAppToolAttachPlanV0) []ToolCapabilityIssueV0 {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.plan = plan
	return append([]ToolCapabilityIssueV0(nil), fake.issues...)
}

func (fake *fakeToolCapabilityValidatorV0) validatedPlan() GeneratedAppToolAttachPlanV0 {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.plan
}

type fakeToolCapabilityInstallerV0 struct {
	mu                sync.Mutex
	applied           map[string]bool
	materializeCalls  int
	effectCount       int
	materializeErr    error
	reconcileErr      error
	forcedDisposition string
	plans             []GeneratedAppToolAttachPlanV0
	rollbacks         []ToolRollbackRequestV0
}

func newFakeToolCapabilityInstallerV0() *fakeToolCapabilityInstallerV0 {
	return &fakeToolCapabilityInstallerV0{applied: make(map[string]bool)}
}

func (fake *fakeToolCapabilityInstallerV0) MaterializeToolBundleV0(_ context.Context, plan GeneratedAppToolAttachPlanV0) (ToolMaterializationResultV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.materializeCalls++
	fake.plans = append(fake.plans, plan)
	if fake.materializeErr != nil {
		return ToolMaterializationResultV0{}, fake.materializeErr
	}
	if !fake.applied[plan.OperationRef] {
		fake.effectCount++
		fake.applied[plan.OperationRef] = true
	}
	return ToolMaterializationResultV0{Status: "materialized", EvidenceRefs: []string{"evidence-materialized"}}, nil
}

func (fake *fakeToolCapabilityInstallerV0) UninstallToolBundleV0(_ context.Context, plan GeneratedAppToolAttachPlanV0) (ToolMaterializationResultV0, error) {
	return fake.MaterializeToolBundleV0(context.Background(), plan)
}

func (fake *fakeToolCapabilityInstallerV0) RollbackToolBundleV0(_ context.Context, request ToolRollbackRequestV0) (ToolMaterializationResultV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.rollbacks = append(fake.rollbacks, request)
	fake.applied[request.FailedPlan.OperationRef] = true
	return ToolMaterializationResultV0{Status: "rolled_back", EvidenceRefs: []string{"evidence-rollback"}}, nil
}

func (fake *fakeToolCapabilityInstallerV0) ReconcileToolBundleV0(_ context.Context, plan GeneratedAppToolAttachPlanV0) (ToolOperationReconciliationV0, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.reconcileErr != nil {
		return ToolOperationReconciliationV0{}, fake.reconcileErr
	}
	disposition := fake.forcedDisposition
	if disposition == "" {
		disposition = ToolReconcileDispositionNotAppliedV0
		if fake.applied[plan.OperationRef] {
			disposition = ToolReconcileDispositionAppliedV0
		}
	}
	return ToolOperationReconciliationV0{Disposition: disposition, EvidenceRefs: []string{"evidence-reconcile"}}, nil
}

func (fake *fakeToolCapabilityInstallerV0) materializeCallCount() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.materializeCalls
}

func (fake *fakeToolCapabilityInstallerV0) effectCountValue() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.effectCount
}

func (fake *fakeToolCapabilityInstallerV0) materializedPlans() []GeneratedAppToolAttachPlanV0 {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return append([]GeneratedAppToolAttachPlanV0(nil), fake.plans...)
}

func (fake *fakeToolCapabilityInstallerV0) rollbackRequests() []ToolRollbackRequestV0 {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return append([]ToolRollbackRequestV0(nil), fake.rollbacks...)
}

type fakeToolCapabilityReceiptBackendV0 struct {
	mu          sync.Mutex
	byOperation map[string]ToolOperationRecordV0
	byReceipt   map[string]string
	leases      map[string]string
}

func newFakeToolCapabilityReceiptBackendV0() *fakeToolCapabilityReceiptBackendV0 {
	return &fakeToolCapabilityReceiptBackendV0{
		byOperation: make(map[string]ToolOperationRecordV0),
		byReceipt:   make(map[string]string),
		leases:      make(map[string]string),
	}
}

func (backend *fakeToolCapabilityReceiptBackendV0) seed(record ToolOperationRecordV0) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	backend.byOperation[record.Plan.OperationRef] = record
	backend.byReceipt[record.Receipt.ReceiptRef] = record.Plan.OperationRef
	if !ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
		backend.leases[record.Plan.TargetRef] = record.Plan.OperationRef
	}
}

func (backend *fakeToolCapabilityReceiptBackendV0) replaceStatus(operationRef, status string) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	record := backend.byOperation[operationRef]
	record.Receipt.Status = status
	backend.byOperation[operationRef] = record
	if ToolReceiptStatusIsTargetTerminalV0(status) {
		delete(backend.leases, record.Plan.TargetRef)
	}
}

func (backend *fakeToolCapabilityReceiptBackendV0) recordCount() int {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return len(backend.byOperation)
}

type fakeToolCapabilityReceiptStoreV0 struct {
	backend *fakeToolCapabilityReceiptBackendV0
}

func (fake *fakeToolCapabilityReceiptStoreV0) ClaimToolOperationV0(_ context.Context, record ToolOperationRecordV0) (ToolOperationRecordV0, bool, error) {
	if issues := ValidateToolOperationRecordV0(record); len(issues) > 0 {
		return ToolOperationRecordV0{}, false, fmt.Errorf("invalid record: %+v", issues)
	}
	fake.backend.mu.Lock()
	defer fake.backend.mu.Unlock()
	if existing, found := fake.backend.byOperation[record.Plan.OperationRef]; found {
		return existing, false, nil
	}
	if operationRef, leased := fake.backend.leases[record.Plan.TargetRef]; leased && operationRef != record.Plan.OperationRef {
		return ToolOperationRecordV0{}, false, fmt.Errorf("%s", ErrToolCapabilityTargetBusyV0)
	}
	fake.backend.byOperation[record.Plan.OperationRef] = record
	fake.backend.byReceipt[record.Receipt.ReceiptRef] = record.Plan.OperationRef
	if !ToolReceiptStatusIsTargetTerminalV0(record.Receipt.Status) {
		fake.backend.leases[record.Plan.TargetRef] = record.Plan.OperationRef
	}
	return record, true, nil
}

func (fake *fakeToolCapabilityReceiptStoreV0) GetToolOperationRecordV0(_ context.Context, operationRef string) (ToolOperationRecordV0, bool, error) {
	fake.backend.mu.Lock()
	defer fake.backend.mu.Unlock()
	record, found := fake.backend.byOperation[operationRef]
	return record, found, nil
}

func (fake *fakeToolCapabilityReceiptStoreV0) GetToolOperationRecordByReceiptRefV0(_ context.Context, receiptRef string) (ToolOperationRecordV0, bool, error) {
	fake.backend.mu.Lock()
	defer fake.backend.mu.Unlock()
	operationRef, found := fake.backend.byReceipt[receiptRef]
	if !found {
		return ToolOperationRecordV0{}, false, nil
	}
	return fake.backend.byOperation[operationRef], true, nil
}

func (fake *fakeToolCapabilityReceiptStoreV0) CompareAndSwapToolOperationV0(_ context.Context, operationRef string, expectedVersion uint64, receipt ToolOperationReceiptV0) (ToolOperationRecordV0, bool, error) {
	fake.backend.mu.Lock()
	defer fake.backend.mu.Unlock()
	record, found := fake.backend.byOperation[operationRef]
	if !found || record.Receipt.StateVersion != expectedVersion {
		return record, false, nil
	}
	if receipt.StateVersion != expectedVersion+1 || receipt.OperationRef != operationRef || receipt.PlanFingerprint != record.Plan.PlanFingerprint {
		return record, false, errors.New("invalid CAS transition")
	}
	next := record
	next.Receipt = receipt
	if issues := ValidateToolOperationRecordV0(next); len(issues) > 0 {
		return record, false, fmt.Errorf("invalid record: %+v", issues)
	}
	fake.backend.byOperation[operationRef] = next
	if ToolReceiptStatusIsTargetTerminalV0(receipt.Status) {
		delete(fake.backend.leases, record.Plan.TargetRef)
	} else {
		fake.backend.leases[record.Plan.TargetRef] = operationRef
	}
	return next, true, nil
}

func (fake *fakeToolCapabilityReceiptStoreV0) ListToolOperationReceiptsV0(_ context.Context, filter ToolOperationReceiptFilterV0) ([]ToolOperationReceiptV0, error) {
	fake.backend.mu.Lock()
	defer fake.backend.mu.Unlock()
	var receipts []ToolOperationReceiptV0
	for _, record := range fake.backend.byOperation {
		receipt := record.Receipt
		if (filter.ReceiptRef == "" || receipt.ReceiptRef == filter.ReceiptRef) &&
			(filter.AppRef == "" || receipt.AppRef == filter.AppRef) &&
			(filter.BindingRef == "" || receipt.BindingRef == filter.BindingRef) &&
			(filter.ToolRef == "" || receipt.ToolRef == filter.ToolRef) &&
			(filter.Operation == "" || receipt.Operation == filter.Operation) &&
			(filter.IdempotencyKey == "" || receipt.IdempotencyKey == filter.IdempotencyKey) {
			receipts = append(receipts, receipt)
		}
	}
	return receipts, nil
}

func validToolCapabilityManifestV0() CapabilityManifestV0 {
	return CapabilityManifestV0{
		SchemaVersion: CapabilityManifestSchemaV0, ManifestRef: "manifest-ref", CapabilityRef: "capability-ref", ToolRef: "tool-ref", Version: "1.2.3",
		InputSchemaRef: "input-schema-ref", OutputSchemaRef: "output-schema-ref", Locales: []string{"es-ES"},
		EffectProfile: CapabilityEffectProfileV0{ProfileRef: "effect-profile-approved", EffectRefs: []string{"effect-ref"}},
		Budgets:       CapabilityBudgetsV0{BudgetRefs: []string{"budget-ref"}}, Idempotency: CapabilityIdempotencyV0{KeySchemaRef: "idem-schema", ScopeRef: "idem-scope"},
		ConnectorSlots:  []ConnectorSlotV0{{SlotRef: "connector-slot", ContractRef: "connector-contract", Required: true}},
		ConfigSchemaRef: "config-schema-ref", SecretRefs: []string{"secret-ref"}, DependencyRefs: []string{"dependency-ref"}, LicenseRefs: []string{"license-ref"},
		TestRefs: []string{"test-ref"}, HealthCheckRefs: []string{"health-ref"}, ReceiptSchemaRef: "receipt-schema-ref",
		IntegrationModes: []string{IntegrationModeEmbeddedModuleV0, IntegrationModeLocalSidecarV0}, Compatibility: ToolCompatibilityV0{AppContractRefs: []string{"app-contract-ref"}},
	}
}

func validToolBundleV0(manifest CapabilityManifestV0) ToolBundleV0 {
	bundle := ToolBundleV0{
		SchemaVersion: ToolBundleSchemaV0, BundleRef: "bundle-ref", ManifestRef: manifest.ManifestRef,
		CapabilityRef: manifest.CapabilityRef, ToolRef: manifest.ToolRef, Version: manifest.Version,
		ArtifactRefs:  []HashedArtifactV0{{ArtifactRef: "artifact-ref", ContentHash: testToolCapabilityHashV0('a')}},
		I18nCatalogs:  []I18nCatalogRefV0{{Locale: "es-ES", CatalogRef: "catalog-ref", ContentHash: testToolCapabilityHashV0('b')}},
		MigrationRefs: []string{"migration-ref"}, TestRefs: []string{"test-ref"}, Compatibility: ToolCompatibilityV0{AppContractRefs: []string{"app-contract-ref"}},
	}
	bundle.ContentHash = CalculateToolBundleContentHashV0(manifest, bundle)
	return bundle
}

func validToolBundleSnapshotV0(manifest CapabilityManifestV0, bundle ToolBundleV0) ToolBundleSnapshotV0 {
	return ToolBundleSnapshotV0{
		SchemaVersion: ToolBundleSnapshotSchemaV0, SnapshotRef: ToolBundleSnapshotRefV0(bundle.ContentHash),
		HandleRef: "verified-handle-ref", VerificationRef: "verification-ref",
		ManifestRef: manifest.ManifestRef, BundleRef: bundle.BundleRef, ContentHash: bundle.ContentHash,
	}
}

func validToolBindingV0() GeneratedAppToolBindingV0 {
	return GeneratedAppToolBindingV0{
		SchemaVersion: GeneratedAppToolBindingSchemaV0, BindingRef: "binding-ref", AppRef: "app-ref", AppVersion: "1.0.0",
		ArchitectureContractRef: "hexagonal-contract-ref", PortRefs: []string{"port-ref"}, AdapterRefs: []string{"adapter-ref"},
		CompositionRef: "composition-ref", SupportedIntegrationModes: []string{IntegrationModeEmbeddedModuleV0},
		ConfigSchemaRefs: []string{"config-schema-ref"}, I18nCatalogRefs: []string{"catalog-ref"},
		AllowedEffectProfileRefs: []string{"effect-profile-approved"}, AllowedWriteSet: []string{"internal/adapters"},
	}
}

func validToolAttachRequestV0() GeneratedAppToolAttachRequestV0 {
	return GeneratedAppToolAttachRequestV0{
		RequestRef: "request-ref", IdempotencyKey: "idem-ref", RequestedBy: "test-operator",
		AppRef: "app-ref", BindingRef: "binding-ref", IntegrationMode: IntegrationModeEmbeddedModuleV0,
		ManifestRef: "manifest-ref", BundleRef: "bundle-ref", ConfigRef: "config-ref", WriteSet: []string{"internal/adapters/tool"},
	}
}

func testToolCapabilityHashV0(value byte) string {
	return "sha256:" + strings.Repeat(string(value), 64)
}
