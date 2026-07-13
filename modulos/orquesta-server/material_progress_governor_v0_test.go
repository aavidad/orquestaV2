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
	runtime, progress := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	_ = runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(80, false))
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(105, false))

	observation := result.Observations[0]
	if observation.Terminal || observation.NeedsRework || !materialProgressResultHasIssueForTestV0(observation.Result, materialProgressWarningCodeV0) {
		t.Fatalf("observation=%+v", observation)
	}
	if progress.state.LastDecision.Action != orquestaautoprogramming.MaterialProgressActionWarningV0 {
		t.Fatalf("progress=%+v", progress.state)
	}
}

func TestMaterialProgressGovernorV0PrimeraMuestraAltaFijaBaselineV0(t *testing.T) {
	runtime, progress := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(80, false))

	observation := result.Observations[0]
	if observation.Terminal || observation.NeedsRework || progress.state.StoreVersion != 1 ||
		progress.state.Segment.StartSequence != 1 || progress.state.Segment.StartTokensAccumulated != 80 ||
		progress.state.LastCheckpoint.Sequence != 1 || progress.state.LastDecision.TokensWithoutMaterial != 0 ||
		progress.state.LastDecision.Action != orquestaautoprogramming.MaterialProgressActionContinueV0 ||
		progress.state.LastDecision.MaterialProgressed {
		t.Fatalf("observation=%+v progress=%+v", observation, progress.state)
	}
}

func TestMaterialProgressGovernorV0SegundaMuestraSinDiffSigueAdvisoryV0(t *testing.T) {
	runtime, _ := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	first := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))
	if first.Observations[0].Terminal || first.Observations[0].NeedsRework {
		t.Fatalf("first=%+v", first)
	}
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(100, false))

	observation := result.Observations[0]
	if observation.Terminal || observation.NeedsRework || observation.State.Status != orquestagoal.GoalStatusRunningV0 ||
		observation.State.LastClosure != nil ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressNoDiffStopCodeV0) {
		t.Fatalf("observation=%+v", observation)
	}
}

func TestMaterialProgressGovernorV0ContextoOmitidoSigueAdvisoryV0(t *testing.T) {
	runtime, _ := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	runtime.materialProgressEvidence = materialProgressEvidenceForTestV0{
		class: orquestaautoprogramming.MaterialProgressClassNoneV0, omitContext: true,
	}
	_ = runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(100, false))

	observation := result.Observations[0]
	if observation.Terminal || observation.NeedsRework || observation.State.Status != orquestagoal.GoalStatusRunningV0 ||
		observation.State.LastClosure != nil || !materialProgressResultHasIssueForTestV0(observation.Result, materialProgressNoDiffStopCodeV0) ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressContextMissingCodeV0) {
		t.Fatalf("observation=%+v", observation)
	}
}

func TestMaterialProgressGovernorV0ReincidenciaSigueAdvisoryV0(t *testing.T) {
	runtime, _ := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	_ = runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, true))
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(100, true))

	observation := result.Observations[0]
	if observation.Terminal || observation.NeedsRework || observation.State.Status != orquestagoal.GoalStatusRunningV0 ||
		observation.State.LastClosure != nil ||
		!materialProgressResultHasIssueForTestV0(observation.Result, materialProgressHardStopCodeV0) {
		t.Fatalf("observation=%+v", observation)
	}
}

func TestMaterialProgressGovernorV0DiffVerificadoRenuevaTramoV0(t *testing.T) {
	runtime, progress := materialProgressRuntimeForTestV0(
		orquestaautoprogramming.MaterialProgressClassDiffV0,
		[]string{"evidence-ref-diff-verified"},
	)
	result := runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(80, false))

	if result.Observations[0].Terminal || result.Observations[0].NeedsRework ||
		!progress.state.LastDecision.MaterialProgressed || progress.state.Segment.StartTokensAccumulated != 80 {
		t.Fatalf("result=%+v progress=%+v", result, progress.state)
	}
}

func TestMaterialProgressGovernorV0ReplayConservaTelemetriaAdvisoryV0(t *testing.T) {
	runtime, _ := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	_ = runtime.reconcileMaterialProgressV0(context.Background(), materialProgressActiveResultForTestV0(50, false))
	input := materialProgressActiveResultForTestV0(100, false)
	first := runtime.reconcileMaterialProgressV0(context.Background(), input)
	second := runtime.reconcileMaterialProgressV0(context.Background(), input)

	if first.Observations[0].Terminal || second.Observations[0].Terminal ||
		first.Observations[0].NeedsRework || second.Observations[0].NeedsRework ||
		!materialProgressResultHasIssueForTestV0(first.Observations[0].Result, materialProgressNoDiffStopCodeV0) ||
		!materialProgressResultHasIssueForTestV0(second.Observations[0].Result, materialProgressNoDiffStopCodeV0) {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}

func TestMaterialProgressGovernorV0SinUsoTipadoConservaFallbackV0(t *testing.T) {
	runtime, progress := materialProgressRuntimeForTestV0(orquestaautoprogramming.MaterialProgressClassNoneV0, nil)
	input := materialProgressActiveResultForTestV0(0, false)
	input.Observations[0].Result.UsageObservation = orquestagoal.GoalUsageObservationV0{}
	result := runtime.reconcileMaterialProgressV0(context.Background(), input)
	if goalObservationHasEvidenceV0(result.Observations[0], materialProgressGovernedEvidenceV0) ||
		progress.state.StoreVersion != 0 {
		t.Fatalf("result=%+v progress=%+v", result, progress.state)
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
	class       orquestaautoprogramming.MaterialProgressClassV0
	refs        []string
	omitContext bool
}

func (source materialProgressEvidenceForTestV0) ClassifyMaterialProgressV0(context.Context, orquestaautoprogramming.MaterialProgressEvidenceRequestV0) (orquestaautoprogramming.MaterialProgressEvidenceV0, error) {
	return orquestaautoprogramming.MaterialProgressEvidenceV0{
		Verified: true, MaterialClass: source.class, EvidenceRefs: source.refs,
		BaselineRef: "baseline-ref-material-progress", WriteSetSHA256: strings.Repeat("a", 64),
		ContextRevisionRef: source.contextRevisionRef(),
	}, nil
}

func (source materialProgressEvidenceForTestV0) contextRevisionRef() string {
	if source.omitContext {
		return ""
	}
	return "context-ref-material-progress"
}

type materialProgressGoalStoreForTestV0 struct {
	state orquestagoal.GoalWorkStateV0
}

func (store *materialProgressGoalStoreForTestV0) LoadGoalWorkStateV0(context.Context, string) (orquestagoal.GoalWorkStateV0, error) {
	return store.state, nil
}

func (store *materialProgressGoalStoreForTestV0) SaveGoalWorkStateV0(_ context.Context, state orquestagoal.GoalWorkStateV0) error {
	store.state = state
	return nil
}

func materialProgressRuntimeForTestV0(class orquestaautoprogramming.MaterialProgressClassV0, refs []string) (*RuntimeV0, *materialProgressStoreForTestV0) {
	progress := &materialProgressStoreForTestV0{}
	return &RuntimeV0{
		config:         ConfigV0{AutoprogrammingGoalProgressPolicy: AutoprogrammingGoalProgressPolicyConfigV0{CheckpointOnlyHighConsumptionTokens: 100}},
		goalStateStore: &materialProgressGoalStoreForTestV0{state: materialProgressGoalStateForTestV0(false)}, materialProgressStore: progress,
		materialProgressEvidence: materialProgressEvidenceForTestV0{class: class, refs: refs},
	}, progress
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
