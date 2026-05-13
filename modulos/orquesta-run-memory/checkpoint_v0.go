package orquestarunmemory

import (
	"context"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func (store *RunMemoryStoreV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command = orquestaruncontrol.NormalizeRecordRunCheckpointCommandV0(command)
	if command.RunRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunRefRequiredV0
	}
	store.mu.Lock()
	previous, ok := store.control[command.RunRef]
	if !ok {
		previous = orquestaruncontrol.DefaultRunControlStateV0(command.RunRef)
	}
	previous.CheckpointRecorded = true
	previous.EvidenceRefs = compactStringsV0(append(previous.EvidenceRefs, command.EvidenceRefs...))
	previous.Meta = orquestaruncontrol.RunControlMetaV0{
		RequestedBy:    command.RequestedBy,
		Reason:         command.Reason,
		IdempotencyKey: command.IdempotencyKey,
	}
	store.control[command.RunRef] = cloneRunControlStateV0(previous)
	store.mu.Unlock()
	return cloneRunControlStateV0(previous), nil
}
