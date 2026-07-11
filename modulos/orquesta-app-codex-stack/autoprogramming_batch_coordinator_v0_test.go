package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestAutoprogrammingBatchPrelaunchPersistePlanYRunningSoloTrasReceiptV0(t *testing.T) {
	ctx := context.Background()
	store := newAutoprogrammingBatchStoreForTestV0()
	batch := autoprogrammingBatchForStackTestV0(t)
	mustSaveAutoprogrammingBatchForStackTestV0(t, ctx, store, 0, batch)
	delegate := &batchPrelaunchDelegateForTestV0{store: store, batchRef: batch.BatchRef}
	launcher := autoprogrammingBatchGoalLauncherV0{Delegate: delegate, Store: store, BatchRef: batch.BatchRef}
	spec := orquestagoal.GoalWorkSpecV0{GoalRef: batch.Members[0].GoalRef, ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: autoprogrammingBatchTaskContextKindV0, Ref: batch.Members[0].TaskRef}}}
	if _, err := launcher.LaunchGoalWorkV0(ctx, spec); err != nil {
		t.Fatalf("LaunchGoalWorkV0: %v", err)
	}
	got := mustLoadAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef)
	if !delegate.sawPrepared || got.Members[0].FocalStatus != orquestaautoprogramming.AutoprogrammingBatchFocalRunningV0 || got.Members[1].FocalStatus != orquestaautoprogramming.AutoprogrammingBatchFocalPendingV0 {
		t.Fatalf("prelaunch=%v batch=%+v", delegate.sawPrepared, got)
	}

	failedBatch := autoprogrammingBatchForStackTestV0(t)
	failedBatch.BatchRef += "-failed"
	failedBatch.PlanHash = ""
	validation := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef: failedBatch.BatchRef, RequestRef: failedBatch.RequestRef, ProjectRef: failedBatch.ProjectRef,
		BaseRevision: failedBatch.BaseRevision, Members: failedBatch.Members, FrozenTests: failedBatch.FrozenTests,
	})
	mustSaveAutoprogrammingBatchForStackTestV0(t, ctx, store, 0, validation.Batch)
	failing := autoprogrammingBatchGoalLauncherV0{Delegate: batchFailingLauncherForTestV0{}, Store: store, BatchRef: validation.Batch.BatchRef}
	_, _ = failing.LaunchGoalWorkV0(ctx, spec)
	blocked := mustLoadAutoprogrammingBatchForStackTestV0(t, ctx, store, validation.Batch.BatchRef)
	if blocked.Status != orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 || blocked.Members[0].FocalStatus != orquestaautoprogramming.AutoprogrammingBatchFocalPendingV0 {
		t.Fatalf("failed launch batch=%+v", blocked)
	}
}

func TestAutoprogrammingBatchDosGoalsIntegraEncadenadoGateUnicoPromueveYReplayV0(t *testing.T) {
	ctx := context.Background()
	store := newAutoprogrammingBatchStoreForTestV0()
	batch := autoprogrammingBatchReadyForIntegrationStackTestV0(t, ctx, store)
	states := newGoalFirstQueueStateStoreForTestV0()
	seedAutoprogrammingBatchGoalStatesForStackTestV0(t, ctx, states, batch)
	integration := &batchIntegrationPortForTestV0{store: store, batchRef: batch.BatchRef}
	runner := &batchTestRunnerForTestV0{store: store, batchRef: batch.BatchRef}
	promotion := &batchPromotionPortForTestV0{store: store, batchRef: batch.BatchRef}
	finalizer := &batchPromotionFinalizerForTestV0{store: store, batchRef: batch.BatchRef}
	stack := StackV0{
		Stores: StoresV0{AutoprogrammingBatchStore: store, AppGoalStateStore: states},
		Ports:  mustBatchGoalStatePortsForTestV0(states),
		Codex:  CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{
			Enabled: true, Port: promotion, BatchPromotionFinalizer: finalizer, GoalWorkspaceProvisioner: &fakeGoalWorkspaceProvisionerForStackTestV0{root: t.TempDir()},
			GoalWorkspaceIntegration: integration, BatchTestRunner: runner, GoalWorkspaceRoot: t.TempDir(),
			BatchIntegrationReceiptDir: t.TempDir(), BatchPromotionReceiptDir: t.TempDir(), CommitMessage: "test: integrate batch",
		},
	}
	closed, _, err := stack.advanceAutoprogrammingBatchV0(ctx, batch)
	if err != nil {
		t.Fatalf("advanceAutoprogrammingBatchV0: %v", err)
	}
	if closed.Status != orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0 || closed.IntegratedRevision != "revision-integrated-2" ||
		closed.Members[0].ParentRevision != batch.BaseRevision || closed.Members[1].ParentRevision != "revision-integrated-1" ||
		integration.calls != 2 || runner.calls != 1 || finalizer.calls != 1 || promotion.promotions != 0 || promotion.archives != 0 {
		t.Fatalf("closed=%+v calls integration=%d gate=%d finalizer=%d generic=%d/%d", closed, integration.calls, runner.calls, finalizer.calls, promotion.promotions, promotion.archives)
	}
	if runner.lastGeneration != 1 {
		t.Fatalf("gate generation=%d", runner.lastGeneration)
	}

	restarted := stack
	replayed, _, err := restarted.advanceAutoprogrammingBatchV0(ctx, mustLoadAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef))
	if err != nil || replayed.Status != orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0 || integration.calls != 2 || runner.calls != 1 || finalizer.calls != 1 || promotion.promotions != 0 || promotion.archives != 0 {
		t.Fatalf("replay=%+v err=%v calls=%d/%d/%d/%d/%d", replayed, err, integration.calls, runner.calls, finalizer.calls, promotion.promotions, promotion.archives)
	}
}

func TestAutoprogrammingBatchClaimsHuerfanosNoReejecutanCiegamenteV0(t *testing.T) {
	ctx := context.Background()
	t.Run("gate", func(t *testing.T) {
		store := newAutoprogrammingBatchStoreForTestV0()
		batch := autoprogrammingBatchReadyForGateStackTestV0(t, ctx, store)
		testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.ClaimAutoprogrammingBatchTestV0(current, current.StoreVersion, "claim-orphan-test", current.IntegratedRevision, testHash, "claim-ref-orphan-test")
		})
		runner := &batchTestRunnerForTestV0{}
		stack := StackV0{Stores: StoresV0{AutoprogrammingBatchStore: store}, AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{BatchTestRunner: runner}}
		blocked, _, err := stack.advanceAutoprogrammingBatchV0(ctx, batch)
		if err != nil || blocked.Status != orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 || runner.calls != 0 || runner.reconciles != 1 {
			t.Fatalf("blocked=%+v err=%v calls=%d reconcile=%d", blocked, err, runner.calls, runner.reconciles)
		}
	})

	t.Run("promotion", func(t *testing.T) {
		store := newAutoprogrammingBatchStoreForTestV0()
		batch := autoprogrammingBatchGatePassedStackTestV0(t, ctx, store)
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.ClaimAutoprogrammingBatchPromotionV0(current, current.StoreVersion, "claim-orphan-promotion", "claim-ref-orphan-promotion", current.IntegratedRevision)
		})
		states := newGoalFirstQueueStateStoreForTestV0()
		seedAutoprogrammingBatchGoalStatesForStackTestV0(t, ctx, states, batch)
		promotion := &batchPromotionPortForTestV0{}
		reconciler := &batchPromotionReconcilerForTestV0{found: true}
		stack := StackV0{Stores: StoresV0{AutoprogrammingBatchStore: store, AppGoalStateStore: states}, Ports: mustBatchGoalStatePortsForTestV0(states), Codex: CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()}, AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{Enabled: true, Port: promotion, BatchPromotionReconciler: reconciler, BatchPromotionReceiptDir: t.TempDir()}}
		closed, _, err := stack.advanceAutoprogrammingBatchV0(ctx, batch)
		if err != nil || closed.Status != orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0 || promotion.promotions != 0 || promotion.archives != 0 || reconciler.calls != 1 {
			t.Fatalf("closed=%+v err=%v effects=%d/%d reconcile=%d", closed, err, promotion.promotions, promotion.archives, reconciler.calls)
		}
	})
}

func TestAutoprogrammingBatchPromotionResultadoDivergenteOBasuraBloqueaV0(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name  string
		alter func(*AutoprogrammingBatchPromotionResultV0)
	}{
		{name: "revision-mismatch", alter: func(result *AutoprogrammingBatchPromotionResultV0) { result.IntegratedRevision = "revision-other" }},
		{name: "dirty", alter: func(result *AutoprogrammingBatchPromotionResultV0) { result.CanonicalClean = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newAutoprogrammingBatchStoreForTestV0()
			batch := autoprogrammingBatchGatePassedStackTestV0(t, ctx, store)
			generic := &batchPromotionPortForTestV0{}
			finalizer := &batchPromotionFinalizerForTestV0{alter: test.alter}
			stack := StackV0{Stores: StoresV0{AutoprogrammingBatchStore: store}, Codex: CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()}, AutoprogrammingPromotion: AutoprogrammingPromotionConfigV0{Port: generic, BatchPromotionFinalizer: finalizer, BatchPromotionReceiptDir: t.TempDir()}}
			blocked, _, err := stack.advanceAutoprogrammingBatchV0(ctx, batch)
			if err != nil || blocked.Status != orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 || generic.promotions != 0 || generic.archives != 0 || finalizer.calls != 1 {
				t.Fatalf("blocked=%+v err=%v generic=%d/%d finalizer=%d", blocked, err, generic.promotions, generic.archives, finalizer.calls)
			}
		})
	}
}

type autoprogrammingBatchStoreForTestV0 struct {
	mu      sync.Mutex
	batches map[string]orquestaautoprogramming.AutoprogrammingBatchV0
}

func newAutoprogrammingBatchStoreForTestV0() *autoprogrammingBatchStoreForTestV0 {
	return &autoprogrammingBatchStoreForTestV0{batches: map[string]orquestaautoprogramming.AutoprogrammingBatchV0{}}
}

func (store *autoprogrammingBatchStoreForTestV0) LoadAutoprogrammingBatchV0(_ context.Context, ref string) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	batch, ok := store.batches[ref]
	if !ok {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, fmt.Errorf("not found")
	}
	return cloneAutoprogrammingBatchForStackTestV0(batch), nil
}

func (store *autoprogrammingBatchStoreForTestV0) CompareAndSwapAutoprogrammingBatchV0(_ context.Context, expected uint64, batch orquestaautoprogramming.AutoprogrammingBatchV0) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	current, found := store.batches[batch.BatchRef]
	if (!found && expected != 0) || (found && current.StoreVersion != expected) || batch.StoreVersion != expected+1 {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, fmt.Errorf("cas")
	}
	store.batches[batch.BatchRef] = cloneAutoprogrammingBatchForStackTestV0(batch)
	return cloneAutoprogrammingBatchForStackTestV0(batch), nil
}

func cloneAutoprogrammingBatchForStackTestV0(batch orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	body, _ := json.Marshal(batch)
	var clone orquestaautoprogramming.AutoprogrammingBatchV0
	_ = json.Unmarshal(body, &clone)
	return clone
}

type batchPrelaunchDelegateForTestV0 struct {
	store       *autoprogrammingBatchStoreForTestV0
	batchRef    string
	sawPrepared bool
}

func (delegate *batchPrelaunchDelegateForTestV0) LaunchGoalWorkV0(ctx context.Context, spec orquestagoal.GoalWorkSpecV0) (orquestagoal.GoalLaunchReceiptV0, error) {
	batch, _ := delegate.store.LoadAutoprogrammingBatchV0(ctx, delegate.batchRef)
	delegate.sawPrepared = batch.Status == orquestaautoprogramming.AutoprogrammingBatchStatusPreparedV0 && batch.Members[0].FocalStatus == orquestaautoprogramming.AutoprogrammingBatchFocalPendingV0
	return orquestagoal.GoalLaunchReceiptV0{SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0, Status: orquestagoal.GoalStatusRunningV0, GoalRef: spec.GoalRef, ExternalGoalRef: "external-goal-ref-batch"}, nil
}

type batchFailingLauncherForTestV0 struct{}

func (batchFailingLauncherForTestV0) LaunchGoalWorkV0(context.Context, orquestagoal.GoalWorkSpecV0) (orquestagoal.GoalLaunchReceiptV0, error) {
	return orquestagoal.GoalLaunchReceiptV0{}, fmt.Errorf("launch failed")
}

type batchIntegrationPortForTestV0 struct {
	store    *autoprogrammingBatchStoreForTestV0
	batchRef string
	calls    int
}

func (port *batchIntegrationPortForTestV0) IntegrateGoalWorkspaceV0(ctx context.Context, request orquestaruntimeworktree.GoalWorkspaceIntegrationRequestV0) (orquestaruntimeworktree.GoalWorkspaceIntegrationResultV0, []orquestaruntimeworktree.WorktreeIssueV0) {
	batch, _ := port.store.LoadAutoprogrammingBatchV0(ctx, port.batchRef)
	if _, claimed, matches := autoprogrammingBatchIntegrationClaimRefV0(batch, batch.Members[port.calls].TaskRef, request.ExpectedParentRevision); !claimed || !matches {
		return orquestaruntimeworktree.GoalWorkspaceIntegrationResultV0{}, []orquestaruntimeworktree.WorktreeIssueV0{{Code: orquestaruntimeworktree.WorktreeIssueInvalidRequestV0}}
	}
	port.calls++
	return orquestaruntimeworktree.GoalWorkspaceIntegrationResultV0{SchemaVersion: orquestaruntimeworktree.GoalWorkspaceIntegrationSchemaVersionV0, Status: orquestaruntimeworktree.GoalWorkspaceIntegrationStatusIntegratedV0, IntegrationRef: request.IntegrationRef, BaseRevision: request.BaseRevision, ExpectedParentRevision: request.ExpectedParentRevision, SourceCommit: fmt.Sprintf("source-revision-%d", port.calls), IntegratedCommit: fmt.Sprintf("revision-integrated-%d", port.calls)}, nil
}

type batchTestRunnerForTestV0 struct {
	store          *autoprogrammingBatchStoreForTestV0
	batchRef       string
	calls          int
	reconciles     int
	lastGeneration uint64
}

func (runner *batchTestRunnerForTestV0) RunAutoprogrammingBatchTestV0(ctx context.Context, request AutoprogrammingBatchTestRunRequestV0) (AutoprogrammingBatchTestRunResultV0, error) {
	if runner.store != nil {
		batch, _ := runner.store.LoadAutoprogrammingBatchV0(ctx, runner.batchRef)
		if _, claimed := autoprogrammingBatchClaimRefV0(batch, request.GateGeneration, request.Revision, request.TestHash); !claimed {
			return AutoprogrammingBatchTestRunResultV0{}, fmt.Errorf("claim missing")
		}
	}
	runner.calls++
	runner.lastGeneration = request.GateGeneration
	return AutoprogrammingBatchTestRunResultV0{ReceiptRef: "batch-test-receipt", Status: orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0}, nil
}

func (runner *batchTestRunnerForTestV0) ReconcileAutoprogrammingBatchTestClaimV0(context.Context, AutoprogrammingBatchTestRunRequestV0) (AutoprogrammingBatchTestRunResultV0, bool, error) {
	runner.reconciles++
	return AutoprogrammingBatchTestRunResultV0{}, false, nil
}

type batchPromotionPortForTestV0 struct {
	store      *autoprogrammingBatchStoreForTestV0
	batchRef   string
	promotions int
	archives   int
}

type batchPromotionFinalizerForTestV0 struct {
	store    *autoprogrammingBatchStoreForTestV0
	batchRef string
	calls    int
	alter    func(*AutoprogrammingBatchPromotionResultV0)
}

func (finalizer *batchPromotionFinalizerForTestV0) FinalizeAutoprogrammingBatchPromotionV0(ctx context.Context, request AutoprogrammingBatchPromotionRequestV0) (AutoprogrammingBatchPromotionResultV0, error) {
	if finalizer.store != nil {
		batch, _ := finalizer.store.LoadAutoprogrammingBatchV0(ctx, finalizer.batchRef)
		if claimRef, claimed := autoprogrammingBatchPromotionClaimRefV0(batch); !claimed || claimRef != request.ClaimRef {
			return AutoprogrammingBatchPromotionResultV0{}, fmt.Errorf("promotion claim missing")
		}
	}
	finalizer.calls++
	result := AutoprogrammingBatchPromotionResultV0{BatchRef: request.BatchRef, GateGeneration: request.GateGeneration, ClaimRef: request.ClaimRef, IntegratedRevision: request.IntegratedRevision, CanonicalClean: true, ReceiptRef: "batch-promotion-receipt"}
	if finalizer.alter != nil {
		finalizer.alter(&result)
	}
	return result, nil
}

func (port *batchPromotionPortForTestV0) PromoteAutoprogrammingStagingV0(ctx context.Context, command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	if port.store != nil {
		batch, _ := port.store.LoadAutoprogrammingBatchV0(ctx, port.batchRef)
		if _, claimed := autoprogrammingBatchPromotionClaimRefV0(batch); !claimed {
			return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{}, fmt.Errorf("promotion claim missing")
		}
	}
	port.promotions++
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{Status: orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0, IntegrationStatus: orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0, IntegrationReceiptRef: "batch-promotion-receipt"}, nil
}

func (port *batchPromotionPortForTestV0) ArchiveAutoprogrammingStagingV0(context.Context, orquestaautoprogramming.AutoprogrammingStagingCleanupCommandV0) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	port.archives++
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{Status: orquestaautoprogramming.AutoprogrammingStagingEffectArchivedV0}, nil
}

type batchPromotionReconcilerForTestV0 struct {
	found bool
	calls int
}

func (reconciler *batchPromotionReconcilerForTestV0) ReconcileAutoprogrammingBatchPromotionV0(_ context.Context, request AutoprogrammingBatchPromotionRequestV0) (AutoprogrammingBatchPromotionResultV0, bool, error) {
	reconciler.calls++
	return AutoprogrammingBatchPromotionResultV0{BatchRef: request.BatchRef, GateGeneration: request.GateGeneration, ClaimRef: request.ClaimRef, IntegratedRevision: request.IntegratedRevision, CanonicalClean: true, ReceiptRef: "batch-promotion-reconciled-receipt"}, reconciler.found, nil
}

func autoprogrammingBatchForStackTestV0(t *testing.T) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	result := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef: "batch-ref-stack-test", RequestRef: "request-ref-stack-test", ProjectRef: "project-ref-stack-test", BaseRevision: "base-revision-test",
		Members: []orquestaautoprogramming.AutoprogrammingBatchMemberV0{
			{TaskRef: "task-a", GoalRef: "goal-a", RunRef: "run-a", WorkspaceRef: "workspace-goal-a", WriteSet: []string{"a.go"}},
			{TaskRef: "task-b", GoalRef: "goal-b", RunRef: "run-b", WorkspaceRef: "workspace-goal-b", WriteSet: []string{"b.go"}},
		},
		FrozenTests: []orquestaautoprogramming.AutoprogrammingBatchTestV0{{Command: "go test ./...", SHA256: strings.Repeat("a", 64)}},
	})
	if !result.Accepted {
		t.Fatalf("NewAutoprogrammingBatchV0: %+v", result.Issues)
	}
	return result.Batch
}

func autoprogrammingBatchReadyForIntegrationStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := autoprogrammingBatchForStackTestV0(t)
	mustSaveAutoprogrammingBatchForStackTestV0(t, ctx, store, 0, batch)
	for _, member := range batch.Members {
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(current, current.StoreVersion, "launch-"+member.TaskRef, member.TaskRef)
		})
	}
	for _, member := range batch.Members {
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.RegisterAutoprogrammingBatchFocalCloseV0(current, current.StoreVersion, "close-"+member.TaskRef, member.TaskRef)
		})
	}
	return batch
}

func autoprogrammingBatchReadyForGateStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := autoprogrammingBatchReadyForIntegrationStackTestV0(t, ctx, store)
	parents := []string{batch.BaseRevision, "revision-integrated-1"}
	for index, member := range batch.Members {
		claimRef := fmt.Sprintf("integration-claim-%d", index+1)
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.ClaimAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, "claim-integration-"+member.TaskRef, claimRef, member.TaskRef, parents[index])
		})
		batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
			return orquestaautoprogramming.RegisterAutoprogrammingBatchIntegrationV0(current, current.StoreVersion, "receipt-integration-"+member.TaskRef, claimRef, member.TaskRef, fmt.Sprintf("source-revision-%d", index+1), parents[index], fmt.Sprintf("revision-integrated-%d", index+1), fmt.Sprintf("integration-receipt-%d", index+1))
		})
	}
	return batch
}

func autoprogrammingBatchGatePassedStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := autoprogrammingBatchReadyForGateStackTestV0(t, ctx, store)
	hash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	batch = mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.ClaimAutoprogrammingBatchTestV0(current, current.StoreVersion, "claim-test", current.IntegratedRevision, hash, "claim-ref-test")
	})
	return mustTransitionAutoprogrammingBatchForStackTestV0(t, ctx, store, batch.BatchRef, func(current orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0 {
		return orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(current, current.StoreVersion, "receipt-test", current.IntegratedRevision, hash, "claim-ref-test", "receipt-ref-test", orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0)
	})
}

func mustSaveAutoprogrammingBatchForStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0, expected uint64, batch orquestaautoprogramming.AutoprogrammingBatchV0) {
	t.Helper()
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(ctx, expected, batch); err != nil {
		t.Fatalf("CompareAndSwapAutoprogrammingBatchV0: %v", err)
	}
}

func mustLoadAutoprogrammingBatchForStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0, ref string) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch, err := store.LoadAutoprogrammingBatchV0(ctx, ref)
	if err != nil {
		t.Fatalf("LoadAutoprogrammingBatchV0: %v", err)
	}
	return batch
}

func mustTransitionAutoprogrammingBatchForStackTestV0(t *testing.T, ctx context.Context, store *autoprogrammingBatchStoreForTestV0, ref string, transition func(orquestaautoprogramming.AutoprogrammingBatchV0) orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch, err := transitionStoredAutoprogrammingBatchV0(ctx, store, ref, transition)
	if err != nil {
		t.Fatalf("transitionStoredAutoprogrammingBatchV0: %v", err)
	}
	return batch
}

func seedAutoprogrammingBatchGoalStatesForStackTestV0(t *testing.T, _ context.Context, states *goalFirstQueueStateStoreForTestV0, batch orquestaautoprogramming.AutoprogrammingBatchV0) {
	t.Helper()
	for _, member := range batch.Members {
		states.states[member.RunRef] = orquestagoal.GoalWorkStateV0{
			RunRef: member.RunRef, GoalRef: member.GoalRef,
			Spec: orquestagoal.GoalWorkSpecV0{RunRef: member.RunRef, RequestRef: batch.RequestRef, ProjectRef: batch.ProjectRef, GoalRef: member.GoalRef, WorkKind: orquestaautoprogramming.AutoprogrammingGoalWorkKindV0,
				ContextRefs: []orquestagoal.GoalContextRefV0{{Kind: "goal_workspace", Ref: member.WorkspaceRef}, {Kind: "worktree", Ref: "worktree-ref-batch"}, {Kind: "branch", Ref: "branch-ref-batch"}}},
		}
	}
}

func mustBatchGoalStatePortsForTestV0(states *goalFirstQueueStateStoreForTestV0) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	return orquestaappdirectorservice.StartAppDirectorPortsV0{GoalStateStore: states}
}
