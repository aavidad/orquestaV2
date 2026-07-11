package orquestaserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestMaterialProgressGovernorV0WarningNoDetieneV0(t *testing.T) {
	runtime, progress, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, true)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(25, false))

	observation := result.Observations[0]
	if observation.Terminal || stopper.uniqueStops != 0 || !materialProgressResultHasIssueForTestV0(observation.Result, materialProgressWarningCodeV0) {
		t.Fatalf("observation=%+v stopper=%+v", observation, stopper)
	}
	if progress.state.LastDecision.Action != orquestaautoprogramming.MaterialProgressActionWarningV0 {
		t.Fatalf("progress=%+v", progress.state)
	}
}

func TestMaterialProgressGovernorV0SinDiffConfirmadoNoRelanzaV0(t *testing.T) {
	runtime, _, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, true)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))

	observation := result.Observations[0]
	if !observation.Terminal || observation.NeedsRework || stopper.uniqueStops != 1 ||
		observation.State.LastClosure == nil || observation.State.LastClosure.NeedsRework ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressNoDiffStopCodeV0) {
		t.Fatalf("observation=%+v stopper=%+v", observation, stopper)
	}
}

func TestMaterialProgressGovernorV0ReincidenciaExigeHardStopSinReworkV0(t *testing.T) {
	runtime, _, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, true)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, true))

	observation := result.Observations[0]
	if !observation.Terminal || observation.NeedsRework || stopper.uniqueStops != 1 ||
		observation.State.LastClosure == nil || observation.State.LastClosure.NeedsRework ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressHardStopCodeV0) {
		t.Fatalf("observation=%+v stopper=%+v", observation, stopper)
	}
}

func TestMaterialProgressGovernorV0DiffVerificadoRenuevaTramoV0(t *testing.T) {
	runtime, progress, stopper := materialProgressRuntimeForTestV0(
		orquestaautoprogramming.MaterialProgressClassDiffV0,
		[]string{"evidence-ref-diff-verified"},
		true,
	)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(80, false))

	if result.Observations[0].Terminal || stopper.uniqueStops != 0 ||
		!progress.state.LastDecision.MaterialProgressed || progress.state.Segment.StartTokensAccumulated != 80 {
		t.Fatalf("result=%+v progress=%+v stopper=%+v", result, progress.state, stopper)
	}
}

func TestMaterialProgressGovernorV0StopNoConfirmadoNoPublicaTerminalV0(t *testing.T) {
	runtime, _, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, false)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))

	observation := result.Observations[0]
	if observation.Terminal || observation.State.Status != orquestagoal.GoalStatusRunningV0 || stopper.uniqueStops != 1 ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressNoDiffStopCodeV0+"_stop_unconfirmed") {
		t.Fatalf("observation=%+v stopper=%+v", observation, stopper)
	}
}

func TestMaterialProgressGovernorV0ReplayReutilizaIdempotenciaV0(t *testing.T) {
	runtime, _, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, false)
	input := materialProgressActiveResultForTestV0(50, false)
	first := runtime.reconcileMaterialProgressV0(context.Background(), input)
	second := runtime.reconcileMaterialProgressV0(context.Background(), input)

	if first.Observations[0].Terminal || second.Observations[0].Terminal || stopper.uniqueStops != 1 || stopper.calls != 2 {
		t.Fatalf("first=%+v second=%+v stopper=%+v", first, second, stopper)
	}
}

func TestMaterialProgressGovernorV0SinUsoTipadoConservaFallbackV0(t *testing.T) {
	runtime, progress, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, true)
	input := materialProgressActiveResultForTestV0(0, false)
	input.Observations[0].Result.UsageObservation = orquestagoal.GoalUsageObservationV0{}
	result := runtime.reconcileMaterialProgressV0(context.Background(), input)
	if goalObservationHasEvidenceV0(result.Observations[0], materialProgressGovernedEvidenceV0) ||
		progress.state.StoreVersion != 0 || stopper.calls != 0 {
		t.Fatalf("result=%+v progress=%+v stopper=%+v", result, progress.state, stopper)
	}
}

func TestMaterialProgressGovernorV0FalloPersistenciaGoalNoPublicaTerminalV0(t *testing.T) {
	runtime, _, stopper := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil, true)
	runtime.goalStateStore.(*materialProgressGoalStoreForTestV0).saveErr = errors.New("save_failed")
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))
	observation := result.Observations[0]
	if observation.Terminal || stopper.uniqueStops != 1 ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressNoDiffStopCodeV0+"_state_save_failed") {
		t.Fatalf("observation=%+v stopper=%+v", observation, stopper)
	}
}

type materialProgressStoreForTestV0 struct {
	state orquestaautoprogramming.MaterialProgressStateV0
}

func (store *materialProgressStoreForTestV0) LoadMaterialProgressStateV0(context.Context, string, string) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	if store.state.StoreVersion == 0 {
		return orquestaautoprogramming.MaterialProgressStateV0{}, errors.New("not_found")
	}
	return store.state, nil
}

func (store *materialProgressStoreForTestV0) CompareAndSwapMaterialProgressStateV0(_ context.Context, expected uint64, state orquestaautoprogramming.MaterialProgressStateV0) (orquestaautoprogramming.MaterialProgressStateV0, error) {
	if store.state.StoreVersion != expected {
		return orquestaautoprogramming.MaterialProgressStateV0{}, errors.New("cas_conflict")
	}
	state.StoreVersion = expected + 1
	store.state = state
	return state, nil
}

type materialProgressEvidenceForTestV0 struct {
	class orquestaautoprogramming.MaterialProgressClassV0
	refs  []string
}

func (source materialProgressEvidenceForTestV0) ClassifyMaterialProgressV0(context.Context, orquestaautoprogramming.MaterialProgressEvidenceRequestV0) (orquestaautoprogramming.MaterialProgressEvidenceV0, error) {
	return orquestaautoprogramming.MaterialProgressEvidenceV0{
		Verified: true, MaterialClass: source.class, EvidenceRefs: source.refs,
		BaselineRef: "baseline-ref-material-progress", WriteSetSHA256: strings.Repeat("a", 64),
		ContextRevisionRef: "context-ref-material-progress",
	}, nil
}

type materialProgressGoalStoreForTestV0 struct {
	state   orquestagoal.GoalWorkStateV0
	saveErr error
}

func (store *materialProgressGoalStoreForTestV0) LoadGoalWorkStateV0(context.Context, string) (orquestagoal.GoalWorkStateV0, error) {
	return store.state, nil
}

func (store *materialProgressGoalStoreForTestV0) SaveGoalWorkStateV0(_ context.Context, state orquestagoal.GoalWorkStateV0) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.state = state
	return nil
}

type materialProgressStopperForTestV0 struct {
	requested   bool
	calls       int
	uniqueStops int
	keys        map[string]bool
}

func (stopper *materialProgressStopperForTestV0) RequestGoalCooperativeStopV0(_ context.Context, request GoalCooperativeStopRequestV0) (GoalCooperativeStopResultV0, error) {
	stopper.calls++
	if !stopper.keys[request.IdempotencyKey] {
		stopper.keys[request.IdempotencyKey] = true
		stopper.uniqueStops++
	}
	return GoalCooperativeStopResultV0{Requested: stopper.requested, EvidenceRefs: []string{"evidence-ref-stop-confirmed"}}, nil
}

func materialProgressRuntimeForTestV0(class orquestaautoprogramming.MaterialProgressClassV0, refs []string, stopConfirmed bool) (*RuntimeV0, *materialProgressStoreForTestV0, *materialProgressStopperForTestV0) {
	progress := &materialProgressStoreForTestV0{}
	goalStore := &materialProgressGoalStoreForTestV0{state: materialProgressGoalStateForTestV0(false)}
	stopper := &materialProgressStopperForTestV0{requested: stopConfirmed, keys: map[string]bool{}}
	return &RuntimeV0{
		config:         ConfigV0{AutoprogrammingGoalProgressPolicy: AutoprogrammingGoalProgressPolicyConfigV0{CheckpointOnlyHighConsumptionTokens: 100}},
		goalStateStore: goalStore, goalStopper: stopper, materialProgressStore: progress,
		materialProgressEvidence: materialProgressEvidenceForTestV0{class: class, refs: refs},
	}, progress, stopper
}

func materialProgressActiveResultForTestV0(tokens int64, rework bool) orquestagoal.GoalWorkObserveActiveResultV0 {
	state := materialProgressGoalStateForTestV0(rework)
	return orquestagoal.GoalWorkObserveActiveResultV0{Observations: []orquestagoal.GoalWorkObserveResultV0{{
		State: state,
		Result: orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0, Status: orquestagoal.GoalStatusRunningV0,
			GoalRef: state.GoalRef, ExternalGoalRef: state.ExternalGoalRef,
			UsageObservation: orquestagoal.GoalUsageObservationV0{
				TokensAccumulated: tokens, ObservedAt: "2026-07-11T12:00:00Z",
				SourceRef: "source-ref-material-progress", EvidenceRefs: []string{"evidence-ref-usage"},
			},
		},
	}}}
}

func materialProgressGoalStateForTestV0(rework bool) orquestagoal.GoalWorkStateV0 {
	contextRefs := []orquestagoal.GoalContextRefV0{{Kind: "worktree_baseline", Ref: "baseline-ref-material-progress"}}
	if rework {
		contextRefs = append(contextRefs, orquestagoal.GoalContextRefV0{Kind: "source_goal", Ref: "goal-ref-source"})
	}
	return orquestagoal.GoalWorkStateV0{
		SchemaVersion: orquestagoal.GoalWorkStateSchemaV0, RunRef: "run-ref-material-progress", GoalRef: "goal-ref-material-progress",
		ExternalGoalRef: "external-goal-ref-material-progress", Status: orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{ReworkPolicy: orquestagoal.GoalReworkPolicyV0{MaxReworkGoals: 1}, ContextRefs: contextRefs},
	}
}

func materialProgressResultHasIssueForTestV0(result orquestagoal.GoalWorkResultV0, code string) bool {
	for _, issue := range result.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
