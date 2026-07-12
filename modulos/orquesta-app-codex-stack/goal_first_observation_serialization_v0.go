package orquestaappcodexstack

import (
	"context"
	"strings"
	"sync"
)

// goalFirstObservationCoordinatorV0 serializa el lifecycle completo de una
// observacion por run. Observaciones de runs distintos conservan paralelismo.
type goalFirstObservationCoordinatorV0 struct {
	mu   sync.Mutex
	runs map[string]*goalFirstObservationRunGateV0
}

var goalFirstObservationCoordinatorInitMuV0 sync.Mutex

type goalFirstObservationRunGateV0 struct {
	token chan struct{}
	refs  int
}

func newGoalFirstObservationCoordinatorV0() *goalFirstObservationCoordinatorV0 {
	return &goalFirstObservationCoordinatorV0{runs: map[string]*goalFirstObservationRunGateV0{}}
}

func (coordinator *goalFirstObservationCoordinatorV0) acquireV0(
	ctx context.Context,
	runRef string,
) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runRef = strings.TrimSpace(runRef)
	if coordinator == nil || runRef == "" {
		return func() {}, nil
	}
	coordinator.mu.Lock()
	if coordinator.runs == nil {
		coordinator.runs = map[string]*goalFirstObservationRunGateV0{}
	}
	gate := coordinator.runs[runRef]
	if gate == nil {
		gate = &goalFirstObservationRunGateV0{token: make(chan struct{}, 1)}
		gate.token <- struct{}{}
		coordinator.runs[runRef] = gate
	}
	gate.refs++
	coordinator.mu.Unlock()

	select {
	case <-ctx.Done():
		coordinator.releaseReferenceV0(runRef, gate)
		return nil, ctx.Err()
	case <-gate.token:
		return func() {
			gate.token <- struct{}{}
			coordinator.releaseReferenceV0(runRef, gate)
		}, nil
	}
}

func (coordinator *goalFirstObservationCoordinatorV0) releaseReferenceV0(
	runRef string,
	gate *goalFirstObservationRunGateV0,
) {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	current := coordinator.runs[runRef]
	if current != gate {
		return
	}
	gate.refs--
	if gate.refs == 0 {
		delete(coordinator.runs, runRef)
	}
}

func (stack *StackV0) goalFirstObservationCoordinatorV0() *goalFirstObservationCoordinatorV0 {
	if stack == nil {
		return nil
	}
	// BuildStackV0 lo inicializa normalmente. El lock global conserva segura la
	// compatibilidad con composiciones manuales sin incrustar/copyar un mutex en
	// StackV0: dos primeras observaciones nunca crean gates distintos.
	goalFirstObservationCoordinatorInitMuV0.Lock()
	defer goalFirstObservationCoordinatorInitMuV0.Unlock()
	if stack.goalObservationCoordinator == nil {
		stack.goalObservationCoordinator = newGoalFirstObservationCoordinatorV0()
	}
	return stack.goalObservationCoordinator
}
