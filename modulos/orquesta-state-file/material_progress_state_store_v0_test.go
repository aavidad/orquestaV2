package orquestastatefile

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

func TestMaterialProgressStateStoreV0CreatesLoadsAndRecovers(t *testing.T) {
	root := t.TempDir()
	state := materialProgressStateForStoreV0()
	store := mustStoreV0(t, root)
	saved, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state)
	if err != nil || saved.StoreVersion != 1 {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
	reopened := mustStoreV0(t, root)
	loaded, err := reopened.LoadMaterialProgressStateV0(context.Background(), state.RunRef, state.GoalRef)
	if err != nil || loaded.StoreVersion != 1 || loaded.LastCheckpointRef != state.LastCheckpointRef {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestMaterialProgressStateStoreV0ReplayAndDivergence(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	state := materialProgressStateForStoreV0()
	first, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state)
	if err != nil || replay.StoreVersion != first.StoreVersion {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	divergent := state
	divergent.ObservedAt = divergent.ObservedAt.Add(time.Second)
	_, err = store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, divergent)
	assertMaterialProgressStateConflictV0(t, err)
}

func TestMaterialProgressStateStoreV0RejectsRegressionAndCheckpointDivergence(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	state := materialProgressStateForStoreV0()
	if _, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state); err != nil {
		t.Fatal(err)
	}
	next := state
	next.Segment.StartSequence = 1
	next.Segment.StartTokensAccumulated = 100
	next.LastCheckpoint.Sequence = 1
	next.LastCheckpoint.TokensAccumulated = 100
	next.LastDecision = orquestaautoprogramming.DecideMaterialProgressV0(orquestaautoprogramming.MaterialProgressInputV0{
		Policy: next.Policy, Segment: next.Segment, Checkpoint: next.LastCheckpoint,
	})
	refreshMaterialProgressStateRefsV0(&next)
	_, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 1, next)
	assertMaterialProgressStateConflictV0(t, err)
	sameSequence := state
	sameSequence.LastCheckpoint.EvidenceRefs = []string{"evidence-ref-store-other"}
	sameSequence.Segment.EvidenceRefs = []string{"evidence-ref-store-other"}
	sameSequence.EvidenceRefs = []string{"evidence-ref-store-other"}
	sameSequence.LastDecision = orquestaautoprogramming.DecideMaterialProgressV0(orquestaautoprogramming.MaterialProgressInputV0{
		Policy: sameSequence.Policy, Segment: sameSequence.Segment, Checkpoint: sameSequence.LastCheckpoint,
	})
	refreshMaterialProgressStateRefsV0(&sameSequence)
	_, err = store.CompareAndSwapMaterialProgressStateV0(context.Background(), 1, sameSequence)
	assertMaterialProgressStateConflictV0(t, err)
}

func TestMaterialProgressStateStoreV0CongelaContratoYExigeSecuenciaNueva(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	state := materialProgressStateForStoreV0()
	if _, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*orquestaautoprogramming.MaterialProgressStateV0){
		func(value *orquestaautoprogramming.MaterialProgressStateV0) { value.Policy.WarningAfterTokens++ },
		func(value *orquestaautoprogramming.MaterialProgressStateV0) { value.BaselineRef = "baseline-ref-other" },
		func(value *orquestaautoprogramming.MaterialProgressStateV0) {
			value.ObservedAt = value.ObservedAt.Add(time.Second)
		},
	} {
		changed := state
		mutate(&changed)
		refreshMaterialProgressStateRefsV0(&changed)
		_, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 1, changed)
		assertMaterialProgressStateConflictV0(t, err)
	}
}

func TestMaterialProgressStateStoreV0AceptaSiguienteSecuenciaMonotona(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	state := materialProgressStateForStoreV0()
	if _, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 0, state); err != nil {
		t.Fatal(err)
	}
	next := state
	next.LastCheckpoint.Sequence++
	next.LastCheckpoint.TokensAccumulated += 5
	next.LastCheckpoint.MaterialClass = orquestaautoprogramming.MaterialProgressClassNoneV0
	next.LastCheckpoint.EvidenceRefs = nil
	next.LastDecision = orquestaautoprogramming.DecideMaterialProgressV0(
		orquestaautoprogramming.MaterialProgressInputV0{
			Policy: next.Policy, Segment: next.Segment, Checkpoint: next.LastCheckpoint,
		},
	)
	refreshMaterialProgressStateRefsV0(&next)
	saved, err := store.CompareAndSwapMaterialProgressStateV0(context.Background(), 1, next)
	if err != nil || saved.StoreVersion != 2 || saved.LastCheckpoint.Sequence != next.LastCheckpoint.Sequence {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
}

func TestMaterialProgressStateStoreV0ConcurrentCASAndPaths(t *testing.T) {
	root := t.TempDir()
	first, second := mustStoreV0(t, root), mustStoreV0(t, root)
	state := materialProgressStateForStoreV0()
	if path := first.materialProgressStatePathV0(state.RunRef, state.GoalRef); filepath.Dir(path) != filepath.Join(root, materialProgressStatesDirV0) || strings.Contains(filepath.Base(path), state.RunRef) {
		t.Fatalf("path=%q", path)
	}
	states := []orquestaautoprogramming.MaterialProgressStateV0{state, state}
	states[1].ObservedAt = states[1].ObservedAt.Add(time.Second)
	stores := []*StoreV0{first, second}
	errs := make([]error, len(states))
	var group sync.WaitGroup
	for i := range states {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			_, errs[i] = stores[i].CompareAndSwapMaterialProgressStateV0(context.Background(), 0, states[i])
		}(i)
	}
	group.Wait()
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		assertMaterialProgressStateConflictV0(t, err)
	}
	if successes != 1 {
		t.Fatalf("successes=%d errs=%v", successes, errs)
	}
}

func materialProgressStateForStoreV0() orquestaautoprogramming.MaterialProgressStateV0 {
	segment := orquestaautoprogramming.MaterialProgressSegmentV0{StartSequence: 2, StartTokensAccumulated: 110, ContextRevisionRef: "context-ref-store", EvidenceRefs: []string{"evidence-ref-store"}}
	checkpoint := orquestaautoprogramming.MaterialProgressCheckpointV0{Sequence: 2, TokensAccumulated: 110, ContextRevisionRef: "context-ref-store", MaterialClass: orquestaautoprogramming.MaterialProgressClassDiffV0, EvidenceRefs: []string{"evidence-ref-store"}}
	state := orquestaautoprogramming.MaterialProgressStateV0{SchemaVersion: orquestaautoprogramming.MaterialProgressStateSchemaVersionV0, RunRef: "run-ref-store", GoalRef: "goal-ref-store", Policy: orquestaautoprogramming.MaterialProgressPolicyV0{WarningAfterTokens: 10, ReplanRequiredAfterTokens: 20, HardStopRequiredAfterTokens: 30, MaxReplans: 1}, Segment: segment, LastCheckpoint: checkpoint, LastDecision: orquestaautoprogramming.MaterialProgressDecisionV0{Accepted: true, Action: orquestaautoprogramming.MaterialProgressActionContinueV0, MaterialProgressed: true, Segment: segment}, BaselineRef: "baseline-ref-store", WriteSetSHA256: strings.Repeat("a", 64), ContextRevisionRef: "context-ref-store", ObservedAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC), EvidenceRefs: []string{"evidence-ref-store"}}
	refreshMaterialProgressStateRefsV0(&state)
	return state
}

func refreshMaterialProgressStateRefsV0(state *orquestaautoprogramming.MaterialProgressStateV0) {
	state.LastCheckpointRef = orquestaautoprogramming.MaterialProgressCheckpointRefV0(state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, state.LastCheckpoint)
	state.LastActionIdempotencyKey = orquestaautoprogramming.MaterialProgressActionIdempotencyKeyV0(state.RunRef, state.GoalRef, state.LastCheckpointRef, state.LastDecision.Action)
}

func assertMaterialProgressStateConflictV0(t *testing.T, err error) {
	t.Helper()
	var conflict orquestaautoprogramming.MaterialProgressStateCASConflictErrorV0
	if !errors.As(err, &conflict) {
		t.Fatalf("err=%v, expected CAS conflict", err)
	}
}
