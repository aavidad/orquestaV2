package orquestarunmemory

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type controlCommandV0 struct {
	runRef       string
	status       orquestaruncontrol.RunControlStatusV0
	requestedBy  string
	reason       string
	forced       bool
	idempotency  string
	evidenceRefs []string
}

func (store *RunMemoryStoreV0) setControlStateV0(
	_ context.Context,
	command controlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command.runRef = strings.TrimSpace(command.runRef)
	if command.runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunRefRequiredV0
	}
	state := orquestaruncontrol.RunControlStateV0{
		RunRef:       command.runRef,
		Status:       command.status,
		Forced:       command.forced,
		EvidenceRefs: compactStringsV0(command.evidenceRefs),
		Meta: orquestaruncontrol.RunControlMetaV0{
			RequestedBy:    strings.TrimSpace(command.requestedBy),
			Reason:         strings.TrimSpace(command.reason),
			IdempotencyKey: strings.TrimSpace(command.idempotency),
		},
	}
	store.mu.Lock()
	if previous, ok := store.control[command.runRef]; ok {
		state.CheckpointRecorded = previous.CheckpointRecorded
	}
	store.control[command.runRef] = cloneRunControlStateV0(state)
	store.mu.Unlock()
	return cloneRunControlStateV0(state), nil
}

func cloneRunControlStateV0(
	state orquestaruncontrol.RunControlStateV0,
) orquestaruncontrol.RunControlStateV0 {
	state.EvidenceRefs = append([]string(nil), state.EvidenceRefs...)
	return state
}
