package orquestagoal

import (
	"context"
	"errors"
	"sort"
	"strings"
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
		receipt: GoalLaunchReceiptV0{
			Status:          GoalStatusRunningV0,
			GoalRef:         "goal-ref-lifecycle-001",
			ExternalGoalRef: "thread-ref-goal-lifecycle-partial-001",
			EvidenceRefs:    []string{"evidence-ref-goal-lifecycle-partial-launch"},
		},
	}
	store := newGoalLifecycleStoreForTestV0()
	store.saveErr = errors.New("state_store_unavailable")

	result, err := StartGoalWorkV0(
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
	if result.Receipt.ExternalGoalRef != "thread-ref-goal-lifecycle-partial-001" ||
		result.State.ExternalGoalRef != "thread-ref-goal-lifecycle-partial-001" ||
		!goalLifecycleStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-goal-lifecycle-partial-launch") {
		t.Fatalf("result parcial sin receipt/state util: %+v", result)
	}
	if launcher.calls != 1 || store.saves != 1 || len(store.states) != 0 {
		t.Fatalf("calls=%d saves=%d states=%+v", launcher.calls, store.saves, store.states)
	}
}

func TestStartGoalWorkV0PersisteEstadoParcialSiLauncherFallaConExternalRefV0(t *testing.T) {
	launcher := &goalLifecycleLauncherForTestV0{
		receipt: GoalLaunchReceiptV0{
			Status:          GoalStatusRunningV0,
			GoalRef:         "goal-ref-lifecycle-partial-launch-error-001",
			ExternalGoalRef: "thread-ref-lifecycle-partial-launch-error-001",
			EvidenceRefs:    []string{"evidence-ref-lifecycle-partial-launch-error"},
		},
		err: errors.New("turn_start_failed"),
	}
	store := newGoalLifecycleStoreForTestV0()
	spec := validGoalLifecycleSpecForTestV0()
	spec.GoalRef = "goal-ref-lifecycle-partial-launch-error-001"

	result, err := StartGoalWorkV0(
		context.Background(),
		GoalWorkStartRequestV0{
			RunRef:       "run-ref-lifecycle-partial-launch-error-001",
			Spec:         spec,
			EvidenceRefs: []string{"evidence-ref-lifecycle-partial-request"},
		},
		GoalWorkLifecyclePortsV0{
			Launcher:   launcher,
			StateStore: store,
		},
	)

	if err == nil || err.Error() != "turn_start_failed" {
		t.Fatalf("err=%v", err)
	}
	if result.State.Status != GoalStatusInvalidV0 ||
		result.State.ExternalGoalRef != "thread-ref-lifecycle-partial-launch-error-001" ||
		result.Receipt.Status != GoalStatusInvalidV0 ||
		!goalLifecycleStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-lifecycle-partial-launch-error") {
		t.Fatalf("result parcial=%+v", result)
	}
	loaded, loadErr := store.LoadGoalWorkStateV0(context.Background(), "run-ref-lifecycle-partial-launch-error-001")
	if loadErr != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", loadErr)
	}
	if loaded.Status != GoalStatusInvalidV0 ||
		loaded.ExternalGoalRef != "thread-ref-lifecycle-partial-launch-error-001" ||
		store.saves != 1 {
		t.Fatalf("loaded=%+v saves=%d", loaded, store.saves)
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

func TestObserveGoalWorkV0ConservaIssuesDeObserverFallidoV0(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	state := mustGoalLifecycleStateForTestV0(t)
	if err := store.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := &goalLifecycleObserverForTestV0{
		result: GoalWorkResultV0{
			SchemaVersion: GoalWorkResultSchemaV0,
			Status:        GoalStatusInvalidV0,
			GoalRef:       state.GoalRef,
			Issues: []GoalWorkIssueV0{{
				Code:  "codex_app_server_control_socket_missing",
				Field: "codex_goal_backend",
			}},
		},
		err: errors.New("codex_goal_observation_rejected"),
	}

	_, err := ObserveGoalWorkV0(
		context.Background(),
		GoalWorkObserveRequestV0{RunRef: state.RunRef},
		GoalWorkLifecyclePortsV0{
			Observer:   observer,
			StateStore: store,
		},
	)
	var lifecycleIssue GoalWorkLifecycleIssueErrorV0
	if !errors.As(err, &lifecycleIssue) ||
		lifecycleIssue.Field != "goal_result" ||
		len(lifecycleIssue.Issues) != 1 ||
		lifecycleIssue.Issues[0].Code != "codex_app_server_control_socket_missing" ||
		lifecycleIssue.Issues[0].Field != "codex_goal_backend" {
		t.Fatalf("err=%v lifecycleIssue=%+v", err, lifecycleIssue)
	}
	if store.saves != 1 {
		t.Fatalf("store saves=%d", store.saves)
	}
}

func TestGoalWorkStateMatchesListRequestV0FiltraActivos(t *testing.T) {
	state := mustGoalLifecycleStateForTestV0(t)
	if !GoalWorkStateMatchesListRequestV0(state, GoalWorkStateListRequestV0{ActiveOnly: true}) {
		t.Fatalf("state running deberia coincidir: %+v", state)
	}
	state.Status = GoalStatusCompleteV0
	if !GoalWorkStateMatchesListRequestV0(state, GoalWorkStateListRequestV0{ActiveOnly: true}) {
		t.Fatalf("state complete sin cierre aceptado sigue pendiente de observacion: %+v", state)
	}
	state.LastClosure = &GoalClosureValidationV0{Status: GoalStatusAcceptedV0, Accepted: true}
	if GoalWorkStateMatchesListRequestV0(state, GoalWorkStateListRequestV0{ActiveOnly: true}) {
		t.Fatalf("state complete con cierre aceptado no deberia coincidir: %+v", state)
	}
	if !GoalWorkStateMatchesListRequestV0(state, GoalWorkStateListRequestV0{
		RunRefs:  []string{" run-ref-goal-lifecycle-001 "},
		Statuses: []string{GoalStatusCompleteV0},
	}) {
		t.Fatalf("filtro explicito deberia coincidir: %+v", state)
	}
}

func TestGoalWorkRunMarkerMatchesListRequestV0FiltraActivos(t *testing.T) {
	marker := GoalWorkRunMarkerV0{
		RunRef:       "run-ref-goal-marker-list-001",
		GoalRef:      "goal-ref-marker-list-001",
		DirectorKind: GoalDirectorKindCodexGoalV0,
		Status:       GoalStatusRunningV0,
	}
	if !GoalWorkRunMarkerMatchesListRequestV0(marker, GoalWorkRunMarkerListRequestV0{ActiveOnly: true}) {
		t.Fatalf("marker running deberia coincidir: %+v", marker)
	}
	marker.Status = GoalStatusCompleteV0
	if GoalWorkRunMarkerMatchesListRequestV0(marker, GoalWorkRunMarkerListRequestV0{ActiveOnly: true}) {
		t.Fatalf("marker terminal no deberia coincidir: %+v", marker)
	}
	if !GoalWorkRunMarkerMatchesListRequestV0(marker, GoalWorkRunMarkerListRequestV0{
		RunRefs:  []string{" run-ref-goal-marker-list-001 "},
		Statuses: []string{GoalStatusCompleteV0},
	}) {
		t.Fatalf("filtro explicito deberia coincidir: %+v", marker)
	}
}

func TestObserveActiveGoalWorksV0ListaYObservaPendientesDeObservacion(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	running := mustGoalLifecycleStateWithRefsForTestV0(
		t,
		"run-ref-goal-lifecycle-running-001",
		"goal-ref-lifecycle-running-001",
		GoalStatusRunningV0,
	)
	complete := mustGoalLifecycleStateWithRefsForTestV0(
		t,
		"run-ref-goal-lifecycle-complete-001",
		"goal-ref-lifecycle-complete-001",
		GoalStatusCompleteV0,
	)
	accepted := mustGoalLifecycleStateWithRefsForTestV0(
		t,
		"run-ref-goal-lifecycle-accepted-001",
		"goal-ref-lifecycle-accepted-001",
		GoalStatusCompleteV0,
	)
	accepted.LastClosure = &GoalClosureValidationV0{
		Status:   GoalStatusAcceptedV0,
		Accepted: true,
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), running); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 running: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), complete); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 complete: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), accepted); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 accepted: %v", err)
	}
	observer := &goalLifecycleObserverByGoalForTestV0{
		results: map[string]GoalWorkResultV0{
			running.GoalRef: {
				Status:       GoalStatusRunningV0,
				GoalRef:      running.GoalRef,
				EvidenceRefs: []string{"evidence-ref-goal-active-001"},
			},
			complete.GoalRef: {
				Status:       GoalStatusRunningV0,
				GoalRef:      complete.GoalRef,
				EvidenceRefs: []string{"evidence-ref-goal-complete-001"},
			},
		},
	}

	result, err := ObserveActiveGoalWorksV0(
		context.Background(),
		GoalWorkObserveActiveRequestV0{},
		GoalWorkLifecyclePortsV0{
			Observer:   observer,
			StateStore: store,
		},
	)
	if err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	if len(result.Observations) != 2 ||
		len(result.Issues) != 0 ||
		len(observer.requests) != 2 ||
		!goalLifecycleObservedGoalRefForTestV0(observer.requests, running.GoalRef) ||
		!goalLifecycleObservedGoalRefForTestV0(observer.requests, complete.GoalRef) ||
		goalLifecycleObservedGoalRefForTestV0(observer.requests, accepted.GoalRef) {
		t.Fatalf("result=%+v requests=%+v", result, observer.requests)
	}
	if !goalLifecycleStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-goal-active-001") {
		t.Fatalf("evidence=%v", result.EvidenceRefs)
	}
	if !goalLifecycleStringInSetForTestV0(result.EvidenceRefs, "evidence-ref-goal-complete-001") {
		t.Fatalf("evidence=%v", result.EvidenceRefs)
	}
}

func TestObserveActiveGoalWorksV0ConservaIncidenciaPorGoalYContinua(t *testing.T) {
	store := newGoalLifecycleStoreForTestV0()
	first := mustGoalLifecycleStateWithRefsForTestV0(
		t,
		"run-ref-goal-lifecycle-active-001",
		"goal-ref-lifecycle-active-001",
		GoalStatusRunningV0,
	)
	second := mustGoalLifecycleStateWithRefsForTestV0(
		t,
		"run-ref-goal-lifecycle-active-002",
		"goal-ref-lifecycle-active-002",
		GoalStatusRunningV0,
	)
	if err := store.SaveGoalWorkStateV0(context.Background(), first); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 first: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(context.Background(), second); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 second: %v", err)
	}
	observer := &goalLifecycleObserverByGoalForTestV0{
		results: map[string]GoalWorkResultV0{
			second.GoalRef: {
				Status:       GoalStatusRunningV0,
				GoalRef:      second.GoalRef,
				EvidenceRefs: []string{"evidence-ref-goal-active-002"},
			},
		},
		errs: map[string]error{
			first.GoalRef: errors.New("goal_observer_unavailable"),
		},
	}

	result, err := ObserveActiveGoalWorksV0(
		context.Background(),
		GoalWorkObserveActiveRequestV0{},
		GoalWorkLifecyclePortsV0{
			Observer:   observer,
			StateStore: store,
		},
	)
	if err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	if len(result.Observations) != 1 ||
		result.Observations[0].State.RunRef != second.RunRef ||
		len(result.Issues) != 1 ||
		result.Issues[0].RunRef != first.RunRef ||
		result.Issues[0].Code != "observe_goal_failed" ||
		len(observer.requests) != 2 {
		t.Fatalf("result=%+v requests=%+v", result, observer.requests)
	}
}

func TestObserveActiveGoalWorksV0ExigeLister(t *testing.T) {
	_, err := ObserveActiveGoalWorksV0(
		context.Background(),
		GoalWorkObserveActiveRequestV0{},
		GoalWorkLifecyclePortsV0{
			Observer:   &goalLifecycleObserverForTestV0{},
			StateStore: goalLifecycleStoreWithoutListForTestV0{},
		},
	)
	if err == nil || err.Error() != "goal_work_lifecycle_invalid: ports.goal_state_lister" {
		t.Fatalf("err=%v", err)
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

func mustGoalLifecycleStateWithRefsForTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	status string,
) GoalWorkStateV0 {
	t.Helper()
	spec := validGoalLifecycleSpecForTestV0()
	spec.GoalRef = goalRef
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef: runRef,
		Spec:   spec,
		LaunchReceipt: GoalLaunchReceiptV0{
			Status:          GoalStatusRunningV0,
			ExternalGoalRef: "external-" + goalRef,
		},
		EvidenceRefs: []string{"evidence-ref-goal-state-" + goalRef},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	state.Status = status
	state, err = NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
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
		return launcher.receipt, launcher.err
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
		return observer.result, observer.err
	}
	return observer.result, nil
}

type goalLifecycleObserverByGoalForTestV0 struct {
	requests []GoalObservationRequestV0
	results  map[string]GoalWorkResultV0
	errs     map[string]error
}

func (observer *goalLifecycleObserverByGoalForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request GoalObservationRequestV0,
) (GoalWorkResultV0, error) {
	observer.requests = append(observer.requests, request)
	if err := observer.errs[request.GoalRef]; err != nil {
		return GoalWorkResultV0{}, err
	}
	if result, ok := observer.results[request.GoalRef]; ok {
		return result, nil
	}
	return GoalWorkResultV0{}, errors.New("goal_result_not_configured")
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

func (store *goalLifecycleStoreForTestV0) ListGoalWorkStatesV0(
	_ context.Context,
	request GoalWorkStateListRequestV0,
) ([]GoalWorkStateV0, error) {
	request = NormalizeGoalWorkStateListRequestV0(request)
	refs := append([]string(nil), request.RunRefs...)
	if len(refs) == 0 {
		for runRef := range store.states {
			refs = append(refs, runRef)
		}
	}
	sort.Strings(refs)
	var out []GoalWorkStateV0
	for _, ref := range refs {
		state, ok := store.states[strings.TrimSpace(ref)]
		if !ok {
			continue
		}
		normalized, err := NewGoalWorkStateV0(state)
		if err != nil {
			return nil, err
		}
		if !GoalWorkStateMatchesListRequestV0(normalized, request) {
			continue
		}
		out = append(out, normalized)
		if request.MaxItems > 0 && len(out) >= request.MaxItems {
			break
		}
	}
	return out, nil
}

type goalLifecycleStoreWithoutListForTestV0 struct{}

func (goalLifecycleStoreWithoutListForTestV0) SaveGoalWorkStateV0(
	context.Context,
	GoalWorkStateV0,
) error {
	return nil
}

func (goalLifecycleStoreWithoutListForTestV0) LoadGoalWorkStateV0(
	context.Context,
	string,
) (GoalWorkStateV0, error) {
	return GoalWorkStateV0{}, errors.New("goal_state_not_found")
}

func goalLifecycleStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func goalLifecycleObservedGoalRefForTestV0(
	requests []GoalObservationRequestV0,
	want string,
) bool {
	for _, request := range requests {
		if request.GoalRef == want {
			return true
		}
	}
	return false
}
