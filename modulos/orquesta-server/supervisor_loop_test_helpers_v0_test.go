package orquestaserver

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type fakeSupervisorV0 struct {
	calls                  int
	selfCalls              int
	planCalls              int
	lastCommand            orquestarunsupervisor.RunSupervisorCommandV0
	lastSelfRequest        IdleSelfImprovementRequestV0
	lastPlanRequest        IdleSelfImprovementPlanRequestV0
	results                []fakeSupervisorResultV0
	planRequests           []IdleSelfImprovementRequestV0
	planErr                error
	blocker                IdleSelfImprovementBlockerResultV0
	retryableRunRefs       []string
	retryableRequestRefs   []string
	retryableEvidenceRefs  []string
	filterCalls            int
	lastFilterRequest      IdleSelfImprovementRequestFilterRequestV0
	filterRequests         []IdleSelfImprovementRequestV0
	filterErr              error
	selfRequestRefs        []string
	selfResults            []IdleSelfImprovementResultV0
	selfErrs               []error
	selfStarted            chan struct{}
	selfRelease            chan struct{}
	goalObservationCalls   int
	lastGoalObservation    orquestagoal.GoalWorkObserveActiveRequestV0
	goalObservationResult  []orquestagoal.GoalWorkObserveActiveResultV0
	goalObservationErrs    []error
	goalStarted            chan struct{}
	goalRelease            chan struct{}
	materializedGoalCalls  int
	materializedGoalResult IdleSelfImprovementMaterializedGoalResultV0
	materializedGoalErr    error
}

func (fake *fakeSupervisorV0) PrepareIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementRequestV0,
) (IdleSelfImprovementResultV0, error) {
	fake.selfCalls++
	index := fake.selfCalls - 1
	fake.lastSelfRequest = request
	fake.selfRequestRefs = append(fake.selfRequestRefs, request.RequestRef)
	if fake.selfStarted != nil {
		fake.selfStarted <- struct{}{}
	}
	if fake.selfRelease != nil {
		<-fake.selfRelease
	}
	if index < len(fake.selfErrs) && fake.selfErrs[index] != nil {
		return IdleSelfImprovementResultV0{}, fake.selfErrs[index]
	}
	if index < len(fake.selfResults) {
		return fake.selfResults[index], nil
	}
	return IdleSelfImprovementResultV0{
		Accepted: true, RunRef: "run-ref-idle-self-improvement-001", RequestRef: request.RequestRef, Status: "ok",
	}, nil
}

func (fake *fakeSupervisorV0) IdleSelfImprovementBlockersV0(
	context.Context,
	IdleSelfImprovementBlockerRequestV0,
) (IdleSelfImprovementBlockerResultV0, error) {
	return fake.blocker, nil
}

func (fake *fakeSupervisorV0) RetryableIdleSelfImprovementRunRefsV0(
	context.Context,
	IdleSelfImprovementRunFreshnessRequestV0,
) (IdleSelfImprovementRunFreshnessResultV0, error) {
	return IdleSelfImprovementRunFreshnessResultV0{
		RetryableRunRefs: append([]string(nil), fake.retryableRunRefs...),
		RetryableRequestRefs: append(
			[]string(nil),
			fake.retryableRequestRefs...,
		),
		EvidenceRefs: append([]string(nil), fake.retryableEvidenceRefs...),
	}, nil
}

func (fake *fakeSupervisorV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request IdleSelfImprovementRequestFilterRequestV0,
) (IdleSelfImprovementRequestFilterResultV0, error) {
	fake.filterCalls++
	fake.lastFilterRequest = request
	if fake.filterErr != nil {
		return IdleSelfImprovementRequestFilterResultV0{}, fake.filterErr
	}
	if fake.filterRequests != nil {
		return IdleSelfImprovementRequestFilterResultV0{Requests: fake.filterRequests}, nil
	}
	return IdleSelfImprovementRequestFilterResultV0{Requests: request.Requests}, nil
}

type fakeSupervisorResultV0 struct {
	result orquestarunsupervisor.RunSupervisorResultV0
	err    error
}

func (fake *fakeSupervisorV0) RunGlobalSupervisorV0(
	_ context.Context,
	command orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	fake.calls++
	fake.lastCommand = command
	if len(fake.results) >= fake.calls {
		next := fake.results[fake.calls-1]
		return next.result, next.err
	}
	return orquestarunsupervisor.RunSupervisorResultV0{
		Ticks:           []orquestarunsupervisor.RunSupervisorTickSummaryV0{{TickNumber: 1}},
		TotalExecutions: 1,
		TotalSkips:      2,
		StopReason:      orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *fakeSupervisorV0) ObserveActiveGoalWorksV0(
	_ context.Context,
	request orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	fake.goalObservationCalls++
	index := fake.goalObservationCalls - 1
	fake.lastGoalObservation = request
	if fake.goalStarted != nil {
		fake.goalStarted <- struct{}{}
	}
	if fake.goalRelease != nil {
		<-fake.goalRelease
	}
	if index < len(fake.goalObservationErrs) && fake.goalObservationErrs[index] != nil {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, fake.goalObservationErrs[index]
	}
	if index < len(fake.goalObservationResult) {
		return fake.goalObservationResult[index], nil
	}
	return orquestagoal.GoalWorkObserveActiveResultV0{}, nil
}

func (fake *fakeSupervisorV0) LoadIdleSelfImprovementMaterializedGoalResultV0(
	context.Context,
	IdleSelfImprovementMaterializedGoalResultRequestV0,
) (IdleSelfImprovementMaterializedGoalResultV0, error) {
	fake.materializedGoalCalls++
	if fake.materializedGoalErr != nil {
		return IdleSelfImprovementMaterializedGoalResultV0{}, fake.materializedGoalErr
	}
	return fake.materializedGoalResult, nil
}

type memoryStateStoreV0 struct {
	mu   sync.Mutex
	last StateV0
}

func (store *memoryStateStoreV0) SaveServerStateV0(_ context.Context, state StateV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.last = state
	return nil
}
func (store *memoryStateStoreV0) LoadServerStateV0(context.Context) (StateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.last, nil
}

type memoryGoalStateStoreV0 struct {
	mu     sync.Mutex
	states map[string]orquestagoal.GoalWorkStateV0
}

func newMemoryGoalStateStoreV0() *memoryGoalStateStoreV0 {
	return &memoryGoalStateStoreV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
}

func (store *memoryGoalStateStoreV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *memoryGoalStateStoreV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[runRef]
	if !ok {
		return orquestagoal.GoalWorkStateV0{}, errors.New("goal state not found")
	}
	return state, nil
}

func (store *memoryGoalStateStoreV0) ListGoalWorkStatesV0(
	_ context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(store.states))
	for _, state := range store.states {
		if orquestagoal.GoalWorkStateMatchesListRequestV0(state, request) {
			out = append(out, state)
		}
	}
	return out, nil
}

type fixedClockV0 struct{ now time.Time }

func (clock fixedClockV0) Now() time.Time { return clock.now }

func markNoExecutionSinceForTestV0(runtime *RuntimeV0, now time.Time) {
	markNoExecutionObservedAtForTestV0(runtime, now.Add(-61*time.Second))
}

func markNoExecutionObservedAtForTestV0(runtime *RuntimeV0, now time.Time) {
	runtime.tracker.MarkSupervisorV0(runtime.config.SupervisorCommand, orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, now)
}

func resetNoExecutionWindowForTestV0(runtime *RuntimeV0) {
	runtime.tracker.mu.Lock()
	defer runtime.tracker.mu.Unlock()
	runtime.tracker.supervisorNoExecutionSince = time.Time{}
}

func markNoExecutionWindowStartForTestV0(runtime *RuntimeV0, idleSince time.Time) {
	runtime.tracker.MarkSupervisorV0(runtime.config.SupervisorCommand, orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, idleSince)
}

func waitRuntimeAsyncWorkForTestV0(t *testing.T, runtime *RuntimeV0) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !runtime.waitAsyncWorkV0(ctx) {
		t.Fatalf("runtime async work sigue en vuelo")
	}
}

func containsStringForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
