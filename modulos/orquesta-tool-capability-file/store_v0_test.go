package orquestatoolcapabilityfile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

const (
	fileStoreHelperEnvV0       = "TEST_TOOL_FILE_STORE_HELPER"
	fileStoreHelperRootEnvV0   = "TEST_TOOL_FILE_STORE_ROOT"
	fileStoreHelperResultEnvV0 = "TEST_TOOL_FILE_STORE_RESULT"
)

func TestToolOperationFileStorePersistsPlanCASAndLeaseV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	store, err := NewToolOperationFileStoreV0(root)
	if err != nil {
		t.Fatal(err)
	}
	record := validFileToolOperationRecordV0("operation-one", "idempotency-one")
	claimed, acquired, err := store.ClaimToolOperationV0(context.Background(), record)
	if err != nil || !acquired {
		t.Fatalf("claimed=%+v acquired=%v err=%v", claimed, acquired, err)
	}

	reopened, err := NewToolOperationFileStoreV0(root)
	if err != nil {
		t.Fatal(err)
	}
	durable, found, err := reopened.GetToolOperationRecordV0(context.Background(), record.Plan.OperationRef)
	if err != nil || !found || durable.Plan.PlanFingerprint != record.Plan.PlanFingerprint || durable.Plan.Snapshot.HandleRef != record.Plan.Snapshot.HandleRef {
		t.Fatalf("durable=%+v found=%v err=%v", durable, found, err)
	}

	overlap := validFileToolOperationRecordV0("operation-two", "idempotency-two")
	if _, acquired, err := reopened.ClaimToolOperationV0(context.Background(), overlap); err == nil || acquired || !strings.Contains(err.Error(), toolcapability.ErrToolCapabilityTargetBusyV0) {
		t.Fatalf("overlap acquired=%v err=%v", acquired, err)
	}

	completed := durable.Receipt
	completed.Status = toolcapability.ToolReceiptStatusAttachedV0
	completed.StateVersion++
	completed.RecoveryRef = ""
	updated, swapped, err := reopened.CompareAndSwapToolOperationV0(context.Background(), durable.Plan.OperationRef, durable.Receipt.StateVersion, completed)
	if err != nil || !swapped || updated.Receipt.Status != toolcapability.ToolReceiptStatusAttachedV0 {
		t.Fatalf("updated=%+v swapped=%v err=%v", updated, swapped, err)
	}
	if stale, swapped, err := reopened.CompareAndSwapToolOperationV0(context.Background(), durable.Plan.OperationRef, durable.Receipt.StateVersion, completed); err != nil || swapped || stale.Receipt.StateVersion != completed.StateVersion {
		t.Fatalf("stale=%+v swapped=%v err=%v", stale, swapped, err)
	}
	if _, acquired, err := reopened.ClaimToolOperationV0(context.Background(), overlap); err != nil || !acquired {
		t.Fatalf("lease not released acquired=%v err=%v", acquired, err)
	}
}

func TestToolOperationFileStoreMultiprocessClaimV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	const workers = 8
	start := make(chan struct{})
	results := make([]string, workers)
	errorsByWorker := make([]error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			resultPath := filepath.Join(root, fmt.Sprintf("result-%d", index))
			command := exec.Command(os.Args[0], "-test.run=^TestToolOperationFileStoreClaimHelperV0$")
			command.Env = append(os.Environ(),
				fileStoreHelperEnvV0+"=1",
				fileStoreHelperRootEnvV0+"="+root,
				fileStoreHelperResultEnvV0+"="+resultPath,
			)
			errorsByWorker[index] = command.Run()
			if errorsByWorker[index] == nil {
				content, readErr := os.ReadFile(resultPath)
				errorsByWorker[index] = readErr
				results[index] = string(content)
			}
		}()
	}
	close(start)
	wait.Wait()
	acquired := 0
	for index := range results {
		if errorsByWorker[index] != nil {
			t.Fatalf("worker %d: %v", index, errorsByWorker[index])
		}
		if results[index] == "acquired" {
			acquired++
		} else if results[index] != "existing" {
			t.Fatalf("worker %d result=%q", index, results[index])
		}
	}
	if acquired != 1 {
		t.Fatalf("acquired=%d results=%v", acquired, results)
	}
	store, err := NewToolOperationFileStoreV0(root)
	if err != nil {
		t.Fatal(err)
	}
	record, found, err := store.GetToolOperationRecordV0(context.Background(), "operation-shared")
	if err != nil || !found || record.Receipt.Status != toolcapability.ToolReceiptStatusClaimedV0 {
		t.Fatalf("record=%+v found=%v err=%v", record, found, err)
	}
}

func TestToolOperationFileStoreClaimHelperV0(t *testing.T) {
	if os.Getenv(fileStoreHelperEnvV0) != "1" {
		t.Skip("multiprocess helper")
	}
	store, err := NewToolOperationFileStoreV0(os.Getenv(fileStoreHelperRootEnvV0))
	if err != nil {
		t.Fatal(err)
	}
	_, acquired, err := store.ClaimToolOperationV0(context.Background(), validFileToolOperationRecordV0("operation-shared", "idempotency-shared"))
	if err != nil {
		t.Fatal(err)
	}
	result := "existing"
	if acquired {
		result = "acquired"
	}
	if err := os.WriteFile(os.Getenv(fileStoreHelperResultEnvV0), []byte(result), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestToolOperationFileStoreRejectsSymlinkRootAndStateV0(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkRoot := filepath.Join(parent, "linked")
	if err := os.Symlink(realRoot, symlinkRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := NewToolOperationFileStoreV0(symlinkRoot); err == nil {
		t.Fatal("accepted symlink root")
	}
	store, err := NewToolOperationFileStoreV0(realRoot)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(realRoot, toolOperationFileStateNameV0)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetToolOperationRecordV0(context.Background(), "missing"); err == nil {
		t.Fatal("followed symlink state")
	}
}

func TestToolOperationFileStoreRejectsPermissiveRootAndTrailingStateV0(t *testing.T) {
	permissive := filepath.Join(t.TempDir(), "permissive")
	if err := os.Mkdir(permissive, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewToolOperationFileStoreV0(permissive); err == nil {
		t.Fatal("accepted group/world accessible root")
	}

	root := filepath.Join(t.TempDir(), "store")
	store, err := NewToolOperationFileStoreV0(root)
	if err != nil {
		t.Fatal(err)
	}
	record := validFileToolOperationRecordV0("operation-trailing", "idempotency-trailing")
	if _, _, err := store.ClaimToolOperationV0(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	state, err := os.OpenFile(filepath.Join(root, toolOperationFileStateNameV0), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.WriteString("{}\n"); err != nil {
		state.Close()
		t.Fatal(err)
	}
	if err := state.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetToolOperationRecordV0(context.Background(), record.Plan.OperationRef); err == nil {
		t.Fatal("accepted state with trailing JSON")
	}
}

func validFileToolOperationRecordV0(operationRef, idempotencyKey string) toolcapability.ToolOperationRecordV0 {
	manifest := toolcapability.CapabilityManifestV0{
		SchemaVersion: toolcapability.CapabilityManifestSchemaV0,
		ManifestRef:   "manifest-ref", CapabilityRef: "capability-ref", ToolRef: "tool-ref", Version: "1.0.0",
		InputSchemaRef: "input-schema", OutputSchemaRef: "output-schema", Locales: []string{"es-ES"},
		EffectProfile:   toolcapability.CapabilityEffectProfileV0{ProfileRef: "effect-profile"},
		Idempotency:     toolcapability.CapabilityIdempotencyV0{KeySchemaRef: "key-schema", ScopeRef: "scope-ref"},
		ConfigSchemaRef: "config-schema", TestRefs: []string{"test-ref"}, ReceiptSchemaRef: "receipt-schema",
		IntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0},
	}
	bundle := toolcapability.ToolBundleV0{
		SchemaVersion: toolcapability.ToolBundleSchemaV0,
		BundleRef:     "bundle-ref", ManifestRef: manifest.ManifestRef, CapabilityRef: manifest.CapabilityRef,
		ToolRef: manifest.ToolRef, Version: manifest.Version,
		ArtifactRefs: []toolcapability.HashedArtifactV0{{ArtifactRef: "artifact-ref", ContentHash: fileTestHashV0('a')}},
		I18nCatalogs: []toolcapability.I18nCatalogRefV0{{Locale: "es-ES", CatalogRef: "catalog-ref", ContentHash: fileTestHashV0('b')}},
		TestRefs:     []string{"test-ref"},
	}
	bundle.ContentHash = toolcapability.CalculateToolBundleContentHashV0(manifest, bundle)
	binding := toolcapability.GeneratedAppToolBindingV0{
		SchemaVersion: toolcapability.GeneratedAppToolBindingSchemaV0,
		BindingRef:    "binding-ref", AppRef: "app-ref", AppVersion: "1.0.0",
		ArchitectureContractRef: "architecture-contract", PortRefs: []string{"port-ref"}, AdapterRefs: []string{"adapter-ref"},
		CompositionRef: "composition-ref", SupportedIntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0},
		ConfigSchemaRefs: []string{"config-schema"}, I18nCatalogRefs: []string{"catalog-ref"},
		AllowedEffectProfileRefs: []string{"effect-profile"}, AllowedWriteSet: []string{"internal/adapters"},
	}
	snapshot := toolcapability.ToolBundleSnapshotV0{
		SchemaVersion: toolcapability.ToolBundleSnapshotSchemaV0,
		SnapshotRef:   toolcapability.ToolBundleSnapshotRefV0(bundle.ContentHash), HandleRef: "safe-handle-ref",
		VerificationRef: "verification-ref", ManifestRef: manifest.ManifestRef, BundleRef: bundle.BundleRef, ContentHash: bundle.ContentHash,
	}
	plan := toolcapability.GeneratedAppToolAttachPlanV0{
		SchemaVersion: toolcapability.ToolAttachPlanSchemaV0,
		OperationRef:  operationRef, TargetRef: "shared-target-ref", RequestRef: "request-ref-" + operationRef,
		Operation: toolcapability.ToolOperationAttachV0, IdempotencyKey: idempotencyKey,
		Manifest: manifest, Bundle: bundle, Snapshot: snapshot, Binding: binding,
		IntegrationMode: toolcapability.IntegrationModeEmbeddedModuleV0, ConfigRef: "config-ref",
		WriteSet: []string{"internal/adapters/tool"}, EffectAuthorized: true,
	}
	plan.PlanFingerprint = toolcapability.CalculateGeneratedAppToolPlanFingerprintV0(plan)
	plan.PlanRef = toolcapability.ToolAttachPlanRefV0(plan.PlanFingerprint)
	receipt := toolcapability.ToolOperationReceiptV0{
		SchemaVersion: toolcapability.ToolOperationReceiptSchemaV0,
		ReceiptRef:    operationRef + "-receipt", OperationRef: operationRef, TargetRef: plan.TargetRef,
		PlanRef: plan.PlanRef, PlanFingerprint: plan.PlanFingerprint, Operation: plan.Operation,
		Status: toolcapability.ToolReceiptStatusClaimedV0, StateVersion: 1, IdempotencyKey: idempotencyKey,
		AppRef: binding.AppRef, BindingRef: binding.BindingRef, ToolRef: manifest.ToolRef, CapabilityRef: manifest.CapabilityRef,
		ManifestRef: manifest.ManifestRef, BundleRef: bundle.BundleRef, ContentHash: bundle.ContentHash,
		ConfigRef: plan.ConfigRef, SnapshotRef: snapshot.SnapshotRef, SnapshotHandleRef: snapshot.HandleRef,
		RecoveryRef: operationRef + "-recovery",
	}
	return toolcapability.ToolOperationRecordV0{SchemaVersion: toolcapability.ToolOperationRecordSchemaV0, Plan: plan, Receipt: receipt}
}

func fileTestHashV0(value byte) string {
	return "sha256:" + strings.Repeat(string(value), 64)
}
