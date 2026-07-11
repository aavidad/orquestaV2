package orquestastatefile

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestAutoprogrammingBatchStoreV0CreatesLoadsAndReplays(t *testing.T) {
	root := t.TempDir()
	store := mustStoreV0(t, root)
	batch := autoprogrammingBatchForStoreV0(t)

	saved, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch)
	if err != nil || saved.StoreVersion != 1 {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
	replay, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch)
	if err != nil || !reflect.DeepEqual(replay, saved) {
		t.Fatalf("replay=%+v saved=%+v err=%v", replay, saved, err)
	}
	if path := store.autoprogrammingBatchPathV0(batch.BatchRef); filepath.Dir(path) != filepath.Join(root, autoprogrammingBatchesDirV0) || strings.Contains(filepath.Base(path), batch.BatchRef) {
		t.Fatalf("path=%q", path)
	}

	reopened := mustStoreV0(t, root)
	loaded, err := reopened.LoadAutoprogrammingBatchV0(context.Background(), batch.BatchRef)
	if err != nil || !reflect.DeepEqual(loaded, saved) {
		t.Fatalf("loaded=%+v saved=%+v err=%v", loaded, saved, err)
	}
}

func TestAutoprogrammingBatchStoreV0RejectsDivergentCASAndPlanMutation(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	batch := autoprogrammingBatchForStoreV0(t)
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch); err != nil {
		t.Fatal(err)
	}

	launchedA := mustAutoprogrammingBatchTransitionV0(t, orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-a", "task-a"))
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 1, launchedA); err != nil {
		t.Fatal(err)
	}
	launchedB := mustAutoprogrammingBatchTransitionV0(t, orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, 1, "launch-b", "task-b"))
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 1, launchedB); err == nil {
		t.Fatal("CAS divergente aceptado")
	}

	mutated := batch
	mutated.Members[0].WriteSet = []string{"modulos/orquesta-state-file/other.go"}
	mutated.PlanHash = orquestaautoprogramming.AutoprogrammingBatchPlanHashV0(mutated)
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 2, mutated); err == nil {
		t.Fatal("mutacion de plan congelado aceptada")
	}
}

func TestAutoprogrammingBatchStoreV0SerializesConcurrentCreates(t *testing.T) {
	root := t.TempDir()
	first, second := mustStoreV0(t, root), mustStoreV0(t, root)
	batches := []orquestaautoprogramming.AutoprogrammingBatchV0{autoprogrammingBatchForStoreV0(t), autoprogrammingBatchForStoreV0(t)}
	batches[1].RequestRef = "request-ref-other"
	batches[1].PlanHash = orquestaautoprogramming.AutoprogrammingBatchPlanHashV0(batches[1])
	stores := []*StoreV0{first, second}
	errs := make([]error, len(stores))
	var group sync.WaitGroup
	for index := range stores {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, errs[index] = stores[index].CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batches[index])
		}(index)
	}
	group.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes=%d errs=%v", successes, errs)
	}
}

func TestAutoprogrammingBatchStoreV0RejectsInvalidAggregate(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	batch := autoprogrammingBatchForStoreV0(t)
	batch.PlanHash = "invalid"
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch); err == nil {
		t.Fatal("batch invalido aceptado")
	}
}

func TestAutoprogrammingBatchStoreV0RejectsIncorrectCandidateVersion(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	batch := autoprogrammingBatchForStoreV0(t)
	batch.StoreVersion = 2
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch); err == nil {
		t.Fatal("candidate con version incorrecta aceptado")
	}
}

func TestAutoprogrammingBatchStoreV0RejectsDirectStateJump(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	batch := autoprogrammingBatchForStoreV0(t)
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch); err != nil {
		t.Fatal(err)
	}
	jump := orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(batch)
	jump.StoreVersion = 2
	jump.Status = orquestaautoprogramming.AutoprogrammingBatchStatusGoalsRunningV0
	for index := range jump.Members {
		jump.Members[index].FocalStatus = orquestaautoprogramming.AutoprogrammingBatchFocalRunningV0
	}
	appendFabricatedBatchActionV0(&jump, "fabricated-jump")
	assertAutoprogrammingBatchAggregateValidV0(t, jump)
	if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 1, jump); err == nil {
		t.Fatal("salto directo aceptado")
	}
}

func TestAutoprogrammingBatchStoreV0RejectsClosedRollbackAndBlockedMutation(t *testing.T) {
	t.Run("closed_to_prepared", func(t *testing.T) {
		store := mustStoreV0(t, t.TempDir())
		closed := persistClosedAutoprogrammingBatchV0(t, store)
		replay, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), closed.StoreVersion-1, closed)
		if err != nil || !reflect.DeepEqual(replay, closed) {
			t.Fatalf("replay=%+v err=%v", replay, err)
		}
		rollback := orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(closed)
		rollback.StoreVersion++
		rollback.Status = orquestaautoprogramming.AutoprogrammingBatchStatusPreparedV0
		appendFabricatedBatchActionV0(&rollback, "fabricated-rollback")
		assertAutoprogrammingBatchAggregateValidV0(t, rollback)
		if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), closed.StoreVersion, rollback); err == nil {
			t.Fatal("rollback closed -> prepared aceptado")
		}
	})

	t.Run("blocked_is_terminal", func(t *testing.T) {
		store := mustStoreV0(t, t.TempDir())
		batch := autoprogrammingBatchForStoreV0(t)
		if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch); err != nil {
			t.Fatal(err)
		}
		blocked := persistAutoprogrammingBatchTransitionV0(t, store, batch,
			orquestaautoprogramming.BlockAutoprogrammingBatchV0(batch, batch.StoreVersion, "block", "block-ref"))
		replay, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), blocked.StoreVersion-1, blocked)
		if err != nil || !reflect.DeepEqual(replay, blocked) {
			t.Fatalf("replay=%+v err=%v", replay, err)
		}
		mutation := orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(blocked)
		mutation.StoreVersion++
		mutation.Status = orquestaautoprogramming.AutoprogrammingBatchStatusPreparedV0
		mutation.BlockRef = ""
		appendFabricatedBatchActionV0(&mutation, "fabricated-unblock")
		assertAutoprogrammingBatchAggregateValidV0(t, mutation)
		if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), blocked.StoreVersion, mutation); err == nil {
			t.Fatal("mutacion de batch bloqueado aceptada")
		}
	})
}

func TestAutoprogrammingBatchStoreV0RejectsReceiptRemovalAndModification(t *testing.T) {
	t.Run("claim_removal", func(t *testing.T) {
		store := mustStoreV0(t, t.TempDir())
		claimed := persistAutoprogrammingBatchThroughClaimV0(t, store)
		candidate := orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(claimed)
		candidate.StoreVersion++
		candidate.TestClaims = nil
		appendFabricatedBatchActionV0(&candidate, "fabricated-claim-removal")
		assertAutoprogrammingBatchAggregateValidV0(t, candidate)
		if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), claimed.StoreVersion, candidate); err == nil {
			t.Fatal("eliminacion de claim aceptada")
		}
	})

	for _, testCase := range []struct {
		name   string
		mutate func(*orquestaautoprogramming.AutoprogrammingBatchV0)
	}{
		{name: "removal", mutate: func(batch *orquestaautoprogramming.AutoprogrammingBatchV0) { batch.TestReceipts = nil }},
		{name: "modification", mutate: func(batch *orquestaautoprogramming.AutoprogrammingBatchV0) {
			batch.TestReceipts[0].ReceiptRef = "receipt-ref-modified"
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := mustStoreV0(t, t.TempDir())
			failed := persistFailedReceiptAutoprogrammingBatchV0(t, store)
			candidate := orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(failed)
			candidate.StoreVersion++
			testCase.mutate(&candidate)
			appendFabricatedBatchActionV0(&candidate, "fabricated-receipt-"+testCase.name)
			assertAutoprogrammingBatchAggregateValidV0(t, candidate)
			if _, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), failed.StoreVersion, candidate); err == nil {
				t.Fatalf("%s de receipt aceptada", testCase.name)
			}
		})
	}
}

func autoprogrammingBatchForStoreV0(t *testing.T) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	result := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef: "batch-ref-store", RequestRef: "request-ref-store", ProjectRef: "project-ref-store", BaseRevision: "base-revision-store",
		Members: []orquestaautoprogramming.AutoprogrammingBatchMemberV0{
			{TaskRef: "task-a", GoalRef: "goal-a", RunRef: "run-a", WorkspaceRef: "workspace-a", WriteSet: []string{"modulos/orquesta-state-file/a.go"}},
			{TaskRef: "task-b", GoalRef: "goal-b", RunRef: "run-b", WorkspaceRef: "workspace-b", WriteSet: []string{"modulos/orquesta-state-file/b.go"}},
		},
		FrozenTests: []orquestaautoprogramming.AutoprogrammingBatchTestV0{
			{Command: "go test -count=1 ./modulos/orquesta-state-file", SHA256: strings.Repeat("a", 64)},
		},
	})
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	return result.Batch
}

func mustAutoprogrammingBatchTransitionV0(
	t *testing.T,
	result orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0,
) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	if !result.Accepted {
		t.Fatalf("issues=%+v", result.Issues)
	}
	return result.Batch
}

func persistAutoprogrammingBatchTransitionV0(
	t *testing.T,
	store *StoreV0,
	current orquestaautoprogramming.AutoprogrammingBatchV0,
	result orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0,
) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	next := mustAutoprogrammingBatchTransitionV0(t, result)
	saved, err := store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), current.StoreVersion, next)
	if err != nil {
		t.Fatalf("persist transition: %v", err)
	}
	return saved
}

func persistAutoprogrammingBatchThroughClaimV0(t *testing.T, store *StoreV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := autoprogrammingBatchForStoreV0(t)
	var err error
	batch, err = store.CompareAndSwapAutoprogrammingBatchV0(context.Background(), 0, batch)
	if err != nil {
		t.Fatal(err)
	}
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, batch.StoreVersion, "launch-a", "task-a"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, batch.StoreVersion, "launch-b", "task-b"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchFocalCloseV0(batch, batch.StoreVersion, "close-a", "task-a"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchFocalCloseV0(batch, batch.StoreVersion, "close-b", "task-b"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.ClaimAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "claim-integration-a", "integration-claim-a", "task-a", "source-a", batch.BaseRevision))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "integrate-a", "integration-claim-a", "task-a", "source-a", batch.BaseRevision, "revision-a", "integration-receipt-a"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.ClaimAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "claim-integration-b", "integration-claim-b", "task-b", "source-b", "revision-a"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchIntegrationV0(batch, batch.StoreVersion, "integrate-b", "integration-claim-b", "task-b", "source-b", "revision-a", "revision-final", "integration-receipt-b"))
	testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	return persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.ClaimAutoprogrammingBatchTestV0(batch, batch.StoreVersion, "claim", "revision-final", testHash, "claim-ref"))
}

func persistFailedReceiptAutoprogrammingBatchV0(t *testing.T, store *StoreV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := persistAutoprogrammingBatchThroughClaimV0(t, store)
	testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	return persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(batch, batch.StoreVersion, "receipt-failed", "revision-final", testHash, "claim-ref", "receipt-ref", orquestaautoprogramming.AutoprogrammingBatchTestReceiptFailedV0))
}

func persistClosedAutoprogrammingBatchV0(t *testing.T, store *StoreV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch := persistAutoprogrammingBatchThroughClaimV0(t, store)
	testHash := orquestaautoprogramming.AutoprogrammingBatchTestHashV0(batch.FrozenTests[0])
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RecordAutoprogrammingBatchTestReceiptV0(batch, batch.StoreVersion, "receipt-passed", "revision-final", testHash, "claim-ref", "receipt-ref", orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.ClaimAutoprogrammingBatchPromotionV0(batch, batch.StoreVersion, "claim-promotion", "promotion-claim", "revision-final"))
	batch = persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.RegisterAutoprogrammingBatchPromotionV0(batch, batch.StoreVersion, "promotion", "promotion-claim", "revision-final", "promotion-receipt"))
	return persistAutoprogrammingBatchTransitionV0(t, store, batch, orquestaautoprogramming.CloseAutoprogrammingBatchV0(batch, batch.StoreVersion, "close"))
}

func appendFabricatedBatchActionV0(batch *orquestaautoprogramming.AutoprogrammingBatchV0, key string) {
	batch.ActionReceipts = append(batch.ActionReceipts, orquestaautoprogramming.AutoprogrammingBatchActionReceiptV0{
		IdempotencyKey: key, ActionHash: strings.Repeat("c", 64), StoreVersion: batch.StoreVersion,
	})
}

func assertAutoprogrammingBatchAggregateValidV0(t *testing.T, batch orquestaautoprogramming.AutoprogrammingBatchV0) {
	t.Helper()
	if validation := orquestaautoprogramming.ValidateAutoprogrammingBatchV0(batch); !validation.Accepted {
		t.Fatalf("fabricated aggregate unexpectedly invalid: %+v", validation.Issues)
	}
}
