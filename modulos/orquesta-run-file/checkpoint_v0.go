package orquestarunfile

import (
	"context"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func (store *RunFileStoreV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command = orquestaruncontrol.NormalizeRecordRunCheckpointCommandV0(command)
	if command.RunRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunFileRunRefRequiredV0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	state, ok := store.control[command.RunRef]
	if !ok {
		state = orquestaruncontrol.DefaultRunControlStateV0(command.RunRef)
	}
	state.CheckpointRecorded = true
	state.EvidenceRefs = compactRunFileStringsV0(append(state.EvidenceRefs, command.EvidenceRefs...))
	state.Meta = orquestaruncontrol.RunControlMetaV0{
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		IdempotencyKey: command.IdempotencyKey,
	}
	next := cloneRunFileControlMapV0(store.control)
	next[command.RunRef] = cloneRunFileControlStateV0(state)
	if err := persistRunFileControlV0(store.controlPath, next); err != nil {
		return orquestaruncontrol.RunControlStateV0{}, err
	}
	store.control = next
	return cloneRunFileControlStateV0(state), nil
}
