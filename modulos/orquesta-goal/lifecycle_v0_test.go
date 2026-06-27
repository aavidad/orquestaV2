package orquestagoal

import (
	"context"
	"errors"
	"testing"
)

func TestStartGoalWorkV0LanzaYGuardaEstadoNeutral(t *testing.T) {
	launcher := &goalLifecycleLauncherForTestV0{
		receipt: GoalLaunchReceiptV0{
			Status:       GoalStatusAcceptedV0,
			EvidenceRefs: []string{"evidence-ref-goal-launch-001"},
		},
	}
	store := newGoalLifecycleStoreForTestV0()
	spec := validGoalLifecycleSpecForTestV0()
	spec.EvidenceRefs = []string{"evidence-ref-goal-spec-001"}

	result, err := StartGoalWorkV0(
		context.Background(),
		GoalWorkStartRequestV0{
			RunRef:       "run-ref-goal-lifecycle-001",
			Spec:         spec,
			EvidenceRefs: []string{"evidence-ref-goal-state-001", "evidence-ref-goal-spec-001"},
		},
		GoalWorkLifecyclePortsV0{
			Launcher:   launcher,
			StateStore: store,
		},
	)
	if err != nil {
		t.Fatalf("StartGoalWorkV0: %v", err)
	}
	if launcher.calls != 1 || len(launcher.specs) != 1 {
		t.Fatalf("launcher calls=%d specs=%d", launcher.calls, len(launcher.specs))
	}
	if result.Receipt.Status != GoalStatusAcceptedV0 ||
		result.State.Status != GoalStatusRunningV0 ||
		result.State.GoalRef != spec.GoalRef ||
		result.State.ExternalGoalRef != spec.GoalRef ||
		result.State.Spec.RunRef != "run-ref-goal-lifecycle-001" {
		t.Fatalf("result=%+v", result)
	}
	if !goalLifecycleStringInSetForTestV0(result.State.EvidenceRefs, "evidence-ref-goal-state-001") ||
		!goalLifecycleStringInSetForTestV0(result.State.EvidenceRefs, "evidence-ref-goal-spec-001") ||
		!goalLifecycleStringInSetForTestV0(result.State.EvidenceRefs, "evidence-ref-goal-launch-001") {
		t.Fatalf("evidence=%v", result.State.EvidenceRefs)
	}
	loaded, err := store.LoadGoalWorkStateV0(context.Background(), "run-ref-goal-lifecycle-001")
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if loaded.GoalRef != spec.GoalRef || store.saves != 1 {
		t.Fatalf("loaded=%+v saves=%d", loaded, store.saves)
	}
}

func TestStartGoalWorkV0DevuelveErrorSiStoreFallaSinRelanzar(t *testing.T) {
	launcher := &goalLifecycleLauncherForTestV0{
		receipt: GoalLaunchReceiptV0{Status: GoalStatusRunningV0},
	}
	store := newGoalLifecycleStoreForTestV0()
	store.saveErr = errors.New("state_store_unavailable")

	_, err := StartGoalWorkV0(
		context.Background(),
		GoalWorkStartRequestV0{
			RunRef: "run-ref-goal-lifecycle-001",
			Spec:   validGoalLifecycleSpecForTestV0(),
		},
		GoalWorkLifecyclePortsV0{
			Launcher:   launcher,
			StateStore: store,
		},
	)
	if err == nil || err.Error() != "state_store_unavailable" {
		t.Fatalf("err=%v", err)
	}
	if launcher.calls != 1 || store.saves != 1 || len(store.states) != 0 {
		t.Fatalf("calls=%d saves=%d states=%+v", launcher.calls, store.saves, store.states)
	}
}

func TestStartGoalWorkV0RechazaRunRefAusenteAntesDeLanzar(t *testing.T) {
	launcher := &goalLifecycleLauncherForTestV0{
		receipt: GoalLaunchReceiptV0{Status: GoalStatusRunningV0},
	}
	_, err := StartGoalWorkV0(
		context.Background(),
		GoalWorkStartRequestV0{Spec: validGoalLifecycleSpecForTestV0()},
		GoalWorkLifecyclePortsV0{
			Launcher:   launcher,
			StateStore: newGoalLifecycleStoreForTestV0(),
		},
	)
	if err == nil || err.Error() != "goal_work_lifecycle_invalid: run_ref" {
		t.Fatalf("err=%v", err)
	}
	if launcher.calls != 0 {
		t.Fatalf("launcher calls=%d", launcher.calls)
	}
}

func TestObserveGoalWorkV0RunningNoExigeClosureValidator(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	state := mustGoalLifecycleStateForTestV0(t)
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := &goalLifecycleObserverForTestV0{
		result: GoalWorkResultV0{
			Status:       GoalStatusRunningV0,
			GoalRef:      state.GoalRef,
			EvidenceRefs: []string{"evidence-ref-goal-running-001"},
		},
	}

	result, err := ObserveGoalWorkV0(
		context.Background(),
		GoalWorkObserveRequestV0{RunRef: state.RunRef},
		GoalWorkLifecyclePortsV0{
			Observer:   observer,
			StateStore: store,
		},
	)
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if result.Terminal ||
		result.ClosureEvaluated ||
		result.State.LastResult == nil ||
		result.State.LastClosure != nil ||
		result.State.Status != GoalStatusRunningV0 ||
		observer.requests[0].GoalRef != state.GoalRef ||
		observer.requests[0].ExternalGoalRef != state.ExternalGoalRef {
		t.Fatalf("result=%+v requests=%+v", result, observer.requests)
	}
	if !goalLifecycleStringInSetForTestV0(result.State.EvidenceRefs, "evidence-ref-goal-running-001") {
		t.Fatalf("evidence=%v", result.State.EvidenceRefs)
	}
}

func TestObserveGoalWorkV0TerminalAceptadoPersisteClosure(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	state := mustGoalLifecycleStateForTestV0(t)
	state.Spec.RequiredTests = []GoalRequiredTestV0{{TestRef: "test-ref-goal-lifecycle-001"}}
	state.Spec.ArtifactContracts = []GoalArtifactContractV0{{
		ArtifactRef:  "artifact-ref-goal-lifecycle-001",
		ArtifactType: "source_tree",
		Required:     true,
	}}
	state.Spec.ClosurePolicy = GoalClosurePolicyV0{
		RequireRequiredTests: true,
		RequireArtifacts:     true,
		RequiredEvidenceRefs: []string{"evidence-ref-goal-required-001"},
	}
	state.Spec = NormalizeGoalWorkSpecV0(state.Spec)
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := &goalLifecycleObserverForTestV0{
		result: GoalWorkResultV0{
			Status:       GoalStatusCompleteV0,
			GoalRef:      state.GoalRef,
			ArtifactRefs: []string{"artifact-ref-goal-lifecycle-001"},
			RequiredTestResults: []GoalRequiredTestResultV0{{
				TestRef:      "test-ref-goal-lifecycle-001",
				Status:       "passed",
				EvidenceRefs: []string{"evidence-ref-goal-test-001"},
			}},
			EvidenceRefs: []string{"evidence-ref-goal-required-001"},
		},
	}

	result, err := ObserveGoalWorkV0(
		context.Background(),
		GoalWorkObserveRequestV0{RunRef: state.RunRef},
		GoalWorkLifecyclePortsV0{
			Observer:         observer,
			ClosureValidator: DefaultGoalWorkClosureValidatorV0{},
			StateStore:       store,
		},
	)
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if !result.Terminal ||
		!result.ClosureEvaluated ||
		!result.Accepted ||
		result.NeedsRework ||
		result.Closure.Status != GoalStatusAcceptedV0 ||
		result.State.Status != GoalStatusCompleteV0 ||
		result.State.LastClosure == nil ||
		!result.State.LastClosure.Accepted {
		t.Fatalf("result=%+v", result)
	}
}

func TestObserveGoalWorkV0TerminalBloqueadoNoDecideRun(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	state := mustGoalLifecycleStateForTestV0(t)
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := &goalLifecycleObserverForTestV0{
		result: GoalWorkResultV0{
			Status:       GoalStatusInvalidV0,
			GoalRef:      state.GoalRef,
			EvidenceRefs: []string{"evidence-ref-goal-invalid-001"},
		},
	}

	result, err := ObserveGoalWorkV0(
		context.Background(),
		GoalWorkObserveRequestV0{RunRef: state.RunRef},
		GoalWorkLifecyclePortsV0{
			Observer:         observer,
			ClosureValidator: DefaultGoalWorkClosureValidatorV0{},
			StateStore:       store,
		},
	)
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	if !result.Terminal ||
		!result.ClosureEvaluated ||
		result.Accepted ||
		!result.NeedsRework ||
		result.State.Status != GoalStatusInvalidV0 ||
		result.State.LastClosure == nil ||
		!result.State.LastClosure.NeedsRework {
		t.Fatalf("result=%+v", result)
	}
}

func validGoalLifecycleSpecForTestV0() GoalWorkSpecV0 {
	return GoalWorkSpecV0{
		GoalRef:      "goal-ref-lifecycle-001",
		Objective:    "Implementar una tarea acotada con evidencias.",
		DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet:     []GoalWriteScopeV0{{Path: "docs"}},
	}
}

func mustGoalLifecycleStateForTestV0(t *testing.T) GoalWorkStateV0 {
	t.Helper()
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef: "run-ref-goal-lifecycle-001",
		Spec:   validGoalLifecycleSpecForTestV0(),
		LaunchReceipt: GoalLaunchReceiptV0{
			Status:          GoalStatusRunningV0,
			ExternalGoalRef: "external-goal-ref-lifecycle-001",
		},
		EvidenceRefs: []string{"evidence-ref-goal-state-001"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	return state
}

type goalLifecycleLauncherForTestV0 struct {
	calls   int
	specs   []GoalWorkSpecV0
	receipt GoalLaunchReceiptV0
	err     error
}

func (launcher *goalLifecycleLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec GoalWorkSpecV0,
) (GoalLaunchReceiptV0, error) {
	launcher.calls++
	launcher.specs = append(launcher.specs, spec)
	if launcher.err != nil {
		return GoalLaunchReceiptV0{}, launcher.err
	}
	return launcher.receipt, nil
}

type goalLifecycleObserverForTestV0 struct {
	requests []GoalObservationRequestV0
	result   GoalWorkResultV0
	err      error
}

func (observer *goalLifecycleObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request GoalObservationRequestV0,
) (GoalWorkResultV0, error) {
	observer.requests = append(observer.requests, request)
	if observer.err != nil {
		return GoalWorkResultV0{}, observer.err
	}
	return observer.result, nil
}

type goalLifecycleStoreForTestV0 struct {
	states  map[string]GoalWorkStateV0
	saves   int
	saveErr error
}

func newGoalLifecycleStoreForTestV0() *goalLifecycleStoreForTestV0 {
	return &goalLifecycleStoreForTestV0{states: map[string]GoalWorkStateV0{}}
}

func (store *goalLifecycleStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state GoalWorkStateV0,
) error {
	store.saves++
	if store.saveErr != nil {
		return store.saveErr
	}
	normalized, err := NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *goalLifecycleStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (GoalWorkStateV0, error) {
	state, ok := store.states[runRef]
	if !ok {
		return GoalWorkStateV0{}, errors.New("goal_state_not_found")
	}
	return state, nil
}

func goalLifecycleStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
