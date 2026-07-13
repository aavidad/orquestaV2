package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestObserveActiveGoalWorksV0BloqueadoNoImpideOtroYRetornaDeadlineV0(t *testing.T) {
	store := goalFirstQueueStateFileStoreForTestV0(t, t.TempDir())
	launcher := &goalFirstQueueLauncherForTestV0{}
	stack := goalFirstQueueStateFileStackForTestV0(store, launcher, &goalFirstQueueObserverForTestV0{}, nil)
	started := make([]orquestaappdirectorservice.StartAppDirectorResultV0, 0, 2)
	for index := 0; index < 2; index++ {
		request := goalFirstQueueStartRequestForTestV0()
		request.RunRef = fmt.Sprintf("run-goal-first-deadline-%02d", index)
		request.CorrelationID = fmt.Sprintf("corr-goal-first-deadline-%02d", index)
		request.AppSpecRequest.RequestID = fmt.Sprintf("request-ref-goal-first-deadline-%02d", index)
		result, err := orquestaappdirectorservice.StartAppDirectorV0(context.Background(), request, stack.Ports)
		if err != nil {
			t.Fatalf("start %d: %v", index, err)
		}
		started = append(started, result)
	}
	observer := &goalFirstDeadlineIsolationObserverV0{
		blockedGoalRef:  started[0].GoalRef,
		enteredBlocked:  make(chan struct{}, 1),
		enteredProgress: make(chan struct{}, 4),
		releaseBlocked:  make(chan struct{}),
		calls:           map[string]int{},
	}
	stack.Ports.GoalObserver = observer
	t.Cleanup(func() { observer.releaseOnce.Do(func() { close(observer.releaseBlocked) }) })

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	resultCh := make(chan orquestagoal.GoalWorkObserveActiveResultV0, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{})
		resultCh <- result
		errCh <- err
	}()
	select {
	case <-observer.enteredBlocked:
	case <-time.After(time.Second):
		t.Fatal("run bloqueado no entro")
	}
	select {
	case <-observer.enteredProgress:
	case <-time.After(time.Second):
		t.Fatal("segundo run no progreso mientras el primero estaba bloqueado")
	}
	var first orquestagoal.GoalWorkObserveActiveResultV0
	select {
	case first = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("batch no retorno al vencer el deadline padre")
	}
	if err := <-errCh; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if !activeGoalObservationHasRunV0(first.Observations, started[1].Run.RunID) {
		t.Fatalf("resultado parcial perdio run progresado: %+v", first.Observations)
	}
	if !activeGoalObservationHasIssueV0(first.Issues, started[0].Run.RunID, activeGoalObservationDeadlineIssueV0) {
		t.Fatalf("falta issue deadline para run bloqueado: %+v", first.Issues)
	}

	second, err := stack.ObserveActiveGoalWorksV0(context.Background(), orquestagoal.GoalWorkObserveActiveRequestV0{})
	if err != nil {
		t.Fatalf("segundo tick: %v", err)
	}
	if !activeGoalObservationHasIssueV0(second.Issues, started[0].Run.RunID, activeGoalObservationRunInFlightIssueV0) {
		t.Fatalf("falta issue in-flight exacto: %+v", second.Issues)
	}
	observer.mu.Lock()
	blockedCalls := observer.calls[started[0].GoalRef]
	progressCalls := observer.calls[started[1].GoalRef]
	observer.mu.Unlock()
	if blockedCalls != 1 || progressCalls < 2 {
		t.Fatalf("calls bloqueado=%d progreso=%d", blockedCalls, progressCalls)
	}
	observer.releaseOnce.Do(func() { close(observer.releaseBlocked) })
}

func TestObserveActiveGoalWorksV0NoIniciaJobsTrasDeadlineV0(t *testing.T) {
	store := goalFirstQueueStateFileStoreForTestV0(t, t.TempDir())
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstAllBlockingObserverV0{entered: make(chan string, 8), release: make(chan struct{})}
	stack := goalFirstQueueStateFileStackForTestV0(store, launcher, &goalFirstQueueObserverForTestV0{}, nil)
	stack.Ports.GoalObserver = observer
	for index := 0; index < activeGoalObservationFanoutV0+1; index++ {
		request := goalFirstQueueStartRequestForTestV0()
		request.RunRef = fmt.Sprintf("run-goal-first-no-start-%02d", index)
		request.CorrelationID = fmt.Sprintf("corr-goal-first-no-start-%02d", index)
		request.AppSpecRequest.RequestID = fmt.Sprintf("request-ref-goal-first-no-start-%02d", index)
		if _, err := orquestaappdirectorservice.StartAppDirectorV0(context.Background(), request, stack.Ports); err != nil {
			t.Fatalf("start %d: %v", index, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{})
		done <- err
	}()
	for index := 0; index < activeGoalObservationFanoutV0; index++ {
		select {
		case <-observer.entered:
		case <-time.After(time.Second):
			t.Fatalf("solo entraron %d workers", index)
		}
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("batch no retorno con todos los workers bloqueados")
	}
	select {
	case ref := <-observer.entered:
		t.Fatalf("job extra inicio tras deadline: %s", ref)
	default:
	}
	close(observer.release)
	time.Sleep(20 * time.Millisecond)
	observer.mu.Lock()
	calls := observer.calls
	observer.mu.Unlock()
	if calls != activeGoalObservationFanoutV0 {
		t.Fatalf("calls=%d want=%d", calls, activeGoalObservationFanoutV0)
	}
}

func TestObserveActiveGoalWorksV0RespetaVentanaLargaDelParentV0(t *testing.T) {
	store := goalFirstQueueStateFileStoreForTestV0(t, t.TempDir())
	launcher := &goalFirstQueueLauncherForTestV0{}
	stack := goalFirstQueueStateFileStackForTestV0(store, launcher, &goalFirstQueueObserverForTestV0{}, nil)
	stack.Ports.GoalObserver = goalFirstRespectingDelayObserverV0{delay: 120 * time.Millisecond}
	request := goalFirstQueueStartRequestForTestV0()
	request.RunRef = "run-goal-first-parent-window"
	request.CorrelationID = "corr-goal-first-parent-window"
	request.AppSpecRequest.RequestID = "request-ref-goal-first-parent-window"
	if _, err := orquestaappdirectorservice.StartAppDirectorV0(context.Background(), request, stack.Ports); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	startedAt := time.Now()
	result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{})
	if err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed < 100*time.Millisecond {
		t.Fatalf("observer largo fue cortado por timeout hijo: %s", elapsed)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations=%d", len(result.Observations))
	}
}

func TestObserveActiveGoalWorksV0FanoutAcotadoYOrdenDeterministaV0(t *testing.T) {
	store := goalFirstQueueStateFileStoreForTestV0(t, t.TempDir())
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstFanoutObserverV0{entered: make(chan struct{}, 5), release: make(chan struct{})}
	stack := goalFirstQueueStateFileStackForTestV0(store, launcher, &goalFirstQueueObserverForTestV0{}, nil)
	stack.Ports.GoalObserver = observer
	started := make([]orquestaappdirectorservice.StartAppDirectorResultV0, 0, 5)
	for index := 0; index < 5; index++ {
		request := goalFirstQueueStartRequestForTestV0()
		request.RunRef = fmt.Sprintf("run-goal-first-fanout-%02d", index)
		request.CorrelationID = fmt.Sprintf("corr-goal-first-fanout-%02d", index)
		request.AppSpecRequest.RequestID = fmt.Sprintf("request-ref-goal-first-fanout-%02d", index)
		result, err := orquestaappdirectorservice.StartAppDirectorV0(context.Background(), request, stack.Ports)
		if err != nil {
			t.Fatalf("start %d: %v", index, err)
		}
		started = append(started, result)
	}
	resultCh := make(chan orquestagoal.GoalWorkObserveActiveResultV0, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := stack.ObserveActiveGoalWorksV0(context.Background(), orquestagoal.GoalWorkObserveActiveRequestV0{})
		resultCh <- result
		errCh <- err
	}()
	for index := 0; index < activeGoalObservationFanoutV0; index++ {
		select {
		case <-observer.entered:
		case <-time.After(time.Second):
			t.Fatalf("fanout no alcanzo %d", activeGoalObservationFanoutV0)
		}
	}
	observer.mu.Lock()
	if observer.maxActive != activeGoalObservationFanoutV0 {
		t.Fatalf("max_active=%d", observer.maxActive)
	}
	observer.mu.Unlock()
	close(observer.release)
	if err := <-errCh; err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	result := <-resultCh
	if len(result.Observations) != len(started) {
		t.Fatalf("observations=%d", len(result.Observations))
	}
	for index, observation := range result.Observations {
		if observation.State.RunRef != started[index].Run.RunID {
			t.Fatalf("orden no determinista index=%d got=%s want=%s", index, observation.State.RunRef, started[index].Run.RunID)
		}
	}
}

type goalFirstFanoutObserverV0 struct {
	mu        sync.Mutex
	entered   chan struct{}
	release   chan struct{}
	active    int
	maxActive int
}

type goalFirstDeadlineIsolationObserverV0 struct {
	mu              sync.Mutex
	blockedGoalRef  string
	enteredBlocked  chan struct{}
	enteredProgress chan struct{}
	releaseBlocked  chan struct{}
	releaseOnce     sync.Once
	calls           map[string]int
}

func (observer *goalFirstDeadlineIsolationObserverV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	observer.calls[request.GoalRef]++
	observer.mu.Unlock()
	if request.GoalRef == observer.blockedGoalRef {
		select {
		case observer.enteredBlocked <- struct{}{}:
		default:
		}
		<-observer.releaseBlocked
	} else {
		observer.enteredProgress <- struct{}{}
	}
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		EvidenceRefs:    []string{"evidence-ref-goal-observation-progress"},
	}, nil
}

type goalFirstAllBlockingObserverV0 struct {
	mu      sync.Mutex
	entered chan string
	release chan struct{}
	calls   int
}

type goalFirstRespectingDelayObserverV0 struct{ delay time.Duration }

func (observer goalFirstRespectingDelayObserverV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	timer := time.NewTimer(observer.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return orquestagoal.GoalWorkResultV0{}, ctx.Err()
	case <-timer.C:
	}
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
	}, nil
}

func (observer *goalFirstAllBlockingObserverV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	observer.calls++
	observer.mu.Unlock()
	observer.entered <- request.ExternalGoalRef
	<-observer.release
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
	}, nil
}

func activeGoalObservationHasRunV0(observations []orquestagoal.GoalWorkObserveResultV0, runRef string) bool {
	for _, observation := range observations {
		if observation.State.RunRef == runRef {
			return true
		}
	}
	return false
}

func activeGoalObservationHasIssueV0(issues []orquestagoal.GoalWorkObserveActiveIssueV0, runRef, code string) bool {
	for _, issue := range issues {
		if issue.RunRef == runRef && issue.Code == code {
			return true
		}
	}
	return false
}

func (observer *goalFirstFanoutObserverV0) ObserveGoalWorkV0(_ context.Context, request orquestagoal.GoalObservationRequestV0) (orquestagoal.GoalWorkResultV0, error) {
	observer.mu.Lock()
	observer.active++
	if observer.active > observer.maxActive {
		observer.maxActive = observer.active
	}
	observer.mu.Unlock()
	observer.entered <- struct{}{}
	<-observer.release
	observer.mu.Lock()
	observer.active--
	observer.mu.Unlock()
	return orquestagoal.GoalWorkResultV0{SchemaVersion: orquestagoal.GoalWorkResultSchemaV0, Status: orquestagoal.GoalStatusRunningV0, GoalRef: request.GoalRef, ExternalGoalRef: request.ExternalGoalRef}, nil
}
