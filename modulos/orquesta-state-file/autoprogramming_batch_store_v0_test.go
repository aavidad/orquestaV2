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
