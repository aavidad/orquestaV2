package orquestaappcodexstack

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestObserveAppDirectorGoalV0SerializaLifecycleCompletoPorRunV0(t *testing.T) {
	stack, _, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	spec := launcher.specs[0]
	observer := &goalFirstBlockingConcurrentObserverV0{
		entered: make(chan struct{}, 2),
		release: make(chan struct{}),
		result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         spec.GoalRef,
			ExternalGoalRef: started.ExternalGoalRef,
			ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
			ArtifactPaths:   goalFirstQueueTechnicalArtifactPathsV0(),
			RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
				spec,
				"evidence-ref-goal-first-serialized-required-test",
			),
			EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
		},
	}
	stack.Ports.GoalObserver = observer
	request := orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: started.Run.RunID}

	errors := make(chan error, 2)
	go func() {
		_, err := stack.ObserveAppDirectorGoalV0(context.Background(), request)
		errors <- err
	}()
	select {
	case <-observer.entered:
	case <-time.After(time.Second):
		t.Fatal("primera observacion no entro")
	}
	go func() {
		_, err := stack.ObserveAppDirectorGoalV0(context.Background(), request)
		errors <- err
	}()
	select {
	case <-observer.entered:
		t.Fatal("segunda observacion entro antes de terminar el lifecycle de la primera")
	case <-time.After(50 * time.Millisecond):
	}
	close(observer.release)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
		}
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.maxActive != 1 || observer.calls != 1 {
		t.Fatalf("calls=%d max_active=%d", observer.calls, observer.maxActive)
	}
	state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(context.Background(), started.Run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-serialized-observation-1") ||
		codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-serialized-observation-2") {
		t.Fatalf("replay terminal reobservo backend: %+v", state.EvidenceRefs)
	}
}

func TestGoalFirstObservationCoordinatorV0RunsDistintosNoSeBloqueanV0(t *testing.T) {
	coordinator := newGoalFirstObservationCoordinatorV0()
	releaseA, err := coordinator.acquireV0(context.Background(), "run-a")
	if err != nil {
		t.Fatal(err)
	}
	defer releaseA()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	releaseB, err := coordinator.acquireV0(ctx, "run-b")
	if err != nil {
		t.Fatalf("run distinto bloqueado: %v", err)
	}
	releaseB()
}

func TestGoalFirstObservationCoordinatorV0TryAcquireNoEsperaRunEnVueloV0(t *testing.T) {
	coordinator := newGoalFirstObservationCoordinatorV0()
	release, acquired := coordinator.tryAcquireV0("run-en-vuelo")
	if !acquired {
		t.Fatal("primera reserva no adquirida")
	}
	defer release()
	if second, acquired := coordinator.tryAcquireV0("run-en-vuelo"); acquired || second != nil {
		t.Fatalf("tryAcquire espero o duplico gate: acquired=%v", acquired)
	}
	if other, acquired := coordinator.tryAcquireV0("run-otro"); !acquired {
		t.Fatal("run distinto no progreso")
	} else {
		other()
	}
}

func TestObserveActiveGoalWorksV0CanceladoNoEsperaBackendDetachedNiDuplicaV0(t *testing.T) {
	stack, _, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	spec := launcher.specs[0]
	observer := &goalFirstIgnoringContextObserverV0{
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
		result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         spec.GoalRef,
			ExternalGoalRef: started.ExternalGoalRef,
			ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
			ArtifactPaths:   goalFirstQueueTechnicalArtifactPathsV0(),
			RequiredTestResults: goalFirstQueueRequiredTestResultsV0(spec,
				"evidence-ref-observer-detached-required-test"),
			EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
		},
	}
	stack.Ports.GoalObserver = observer
	ctx, cancel := context.WithCancel(context.Background())
	returned := make(chan error, 1)
	go func() {
		_, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{})
		returned <- err
	}()
	select {
	case <-observer.entered:
	case <-time.After(time.Second):
		t.Fatal("observacion no entro al backend")
	}
	cancel()
	select {
	case err := <-returned:
		if err != context.Canceled {
			t.Fatalf("cancelacion=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("el tick espero al backend que ignora ctx")
	}

	second, err := stack.ObserveActiveGoalWorksV0(context.Background(), orquestagoal.GoalWorkObserveActiveRequestV0{})
	if err != nil {
		t.Fatalf("segunda observacion: %v", err)
	}
	observer.mu.Lock()
	calls := observer.calls
	observer.mu.Unlock()
	if calls != 1 || len(second.Issues) != 1 || second.Issues[0].Code != activeGoalObservationRunInFlightIssueV0 {
		t.Fatalf("calls=%d second=%+v", calls, second)
	}
	close(observer.release)
	deadline := time.After(time.Second)
	for {
		if release, acquired := stack.goalFirstObservationCoordinatorV0().tryAcquireV0(started.Run.RunID); acquired {
			release()
			break
		}
		select {
		case <-deadline:
			t.Fatal("gate detached no se libero al retornar backend")
		case <-time.After(time.Millisecond):
		}
	}
}

type goalFirstBlockingConcurrentObserverV0 struct {
	mu        sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	result    orquestagoal.GoalWorkResultV0
	calls     int
	active    int
	maxActive int
}

type goalFirstIgnoringContextObserverV0 struct {
	mu      sync.Mutex
	entered chan struct{}
	release chan struct{}
	result  orquestagoal.GoalWorkResultV0
	calls   int
}

func (observer *goalFirstIgnoringContextObserverV0) ObserveGoalWorkV0(context.Context, orquestagoal.GoalObservationRequestV0) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	observer.calls++
	observer.mu.Unlock()
	observer.entered <- struct{}{}
	<-observer.release
	return observer.result, nil
}

func (observer *goalFirstBlockingConcurrentObserverV0) ObserveGoalWorkV0(
	ctx context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	observer.calls++
	call := observer.calls
	observer.active++
	if observer.active > observer.maxActive {
		observer.maxActive = observer.active
	}
	observer.mu.Unlock()
	observer.entered <- struct{}{}
	select {
	case <-ctx.Done():
		return orquestagoal.GoalWorkResultV0{}, ctx.Err()
	case <-observer.release:
	}
	observer.mu.Lock()
	observer.active--
	observer.mu.Unlock()
	result := observer.result
	result.EvidenceRefs = append(
		append([]string(nil), result.EvidenceRefs...),
		"evidence-ref-serialized-observation-"+strconv.Itoa(call),
	)
	return result, nil
}
