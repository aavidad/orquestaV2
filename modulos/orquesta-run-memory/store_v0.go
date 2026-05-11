package orquestarunmemory

import (
	"context"
	"errors"
	"strings"
	"sync"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

var (
	ErrRunRefRequiredV0 = errors.New("run_ref required")
)

type RunMemoryStoreV0 struct {
	mu        sync.RWMutex
	control   map[string]orquestaruncontrol.RunControlStateV0
	queueRuns map[string]runQueueEntryV0
}

type runQueueEntryV0 struct {
	queueRef  string
	candidate orquestarunqueue.RunSchedulingCandidateV0
}

var _ orquestaruncontrol.RunControlPortV0 = (*RunMemoryStoreV0)(nil)
var _ orquestarunqueue.RunQueuePortV0 = (*RunMemoryStoreV0)(nil)

func NewRunMemoryStoreV0() *RunMemoryStoreV0 {
	return &RunMemoryStoreV0{
		control:   map[string]orquestaruncontrol.RunControlStateV0{},
		queueRuns: map[string]runQueueEntryV0{},
	}
}

func (store *RunMemoryStoreV0) PutRunControlStateV0(
	_ context.Context,
	state orquestaruncontrol.RunControlStateV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state.RunRef = strings.TrimSpace(state.RunRef)
	if state.RunRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunRefRequiredV0
	}
	state = cloneRunControlStateV0(state)
	store.mu.Lock()
	store.control[state.RunRef] = state
	store.mu.Unlock()
	return cloneRunControlStateV0(state), nil
}

func (store *RunMemoryStoreV0) UpsertRunSchedulingCandidateV0(
	_ context.Context,
	queueRef string,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	candidate.RunRef = strings.TrimSpace(candidate.RunRef)
	if candidate.RunRef == "" {
		return orquestarunqueue.RunSchedulingCandidateV0{}, ErrRunRefRequiredV0
	}
	entry := runQueueEntryV0{
		queueRef:  strings.TrimSpace(queueRef),
		candidate: cloneRunSchedulingCandidateV0(candidate),
	}
	store.mu.Lock()
	store.queueRuns[candidate.RunRef] = entry
	store.mu.Unlock()
	return cloneRunSchedulingCandidateV0(candidate), nil
}

func (store *RunMemoryStoreV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	runRef := strings.TrimSpace(request.RunRef)
	if runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunRefRequiredV0
	}
	store.mu.RLock()
	state, ok := store.control[runRef]
	store.mu.RUnlock()
	if !ok {
		return orquestaruncontrol.RunControlStateV0{},
			orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: runRef}
	}
	return cloneRunControlStateV0(state), nil
}

func (store *RunMemoryStoreV0) PauseRunV0(
	ctx context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, controlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusPausedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunMemoryStoreV0) ResumeRunV0(
	ctx context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, controlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusRunningV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunMemoryStoreV0) StopRunV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, controlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusStopRequestedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		forced:       command.Forced,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunMemoryStoreV0) CancelRunV0(
	ctx context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, controlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusCancelRequestedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		forced:       command.Forced,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}
