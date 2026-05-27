package orquestarunfile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const runFileControlSchemaVersionV0 = "orquesta.run_file.control.v0"

type runFileControlSnapshotV0 struct {
	SchemaVersion string                   `json:"schema_version"`
	Records       []runFileControlRecordV0 `json:"records"`
}

type runFileControlRecordV0 struct {
	RunRef string                               `json:"run_ref"`
	State  orquestaruncontrol.RunControlStateV0 `json:"state"`
}

type runFileControlCommandV0 struct {
	runRef       string
	status       orquestaruncontrol.RunControlStatusV0
	requestedBy  string
	reason       string
	forced       bool
	idempotency  string
	evidenceRefs []string
}

func (store *RunFileStoreV0) PutRunControlStateV0(
	_ context.Context,
	state orquestaruncontrol.RunControlStateV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state.RunRef = strings.TrimSpace(state.RunRef)
	if state.RunRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunFileRunRefRequiredV0
	}
	state = cloneRunFileControlStateV0(state)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	next := cloneRunFileControlMapV0(store.control)
	next[state.RunRef] = state
	if err := persistRunFileControlV0(store.controlPath, next); err != nil {
		return orquestaruncontrol.RunControlStateV0{}, err
	}
	store.control = next
	return cloneRunFileControlStateV0(state), nil
}

func (store *RunFileStoreV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	runRef := strings.TrimSpace(request.RunRef)
	if runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunFileRunRefRequiredV0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	state, ok := store.control[runRef]
	if !ok {
		return orquestaruncontrol.RunControlStateV0{},
			orquestaruncontrol.RunControlStateNotFoundErrorV0{RunRef: runRef}
	}
	return cloneRunFileControlStateV0(state), nil
}

func (store *RunFileStoreV0) PauseRunV0(
	ctx context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, runFileControlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusPausedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunFileStoreV0) ResumeRunV0(
	ctx context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, runFileControlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusRunningV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunFileStoreV0) StopRunV0(
	ctx context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, runFileControlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusStopRequestedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		forced:       command.Forced,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunFileStoreV0) CancelRunV0(
	ctx context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	return store.setControlStateV0(ctx, runFileControlCommandV0{
		runRef:       command.RunRef,
		status:       orquestaruncontrol.RunControlStatusCancelRequestedV0,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		forced:       command.Forced,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunFileStoreV0) CompleteRunControlV0(
	ctx context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command = orquestaruncontrol.NormalizeCompleteRunControlCommandV0(command)
	if !orquestaruncontrol.IsCompleteRunControlTargetV0(command.TargetStatus) {
		return orquestaruncontrol.RunControlStateV0{},
			orquestaruncontrol.RunControlCompletionTargetErrorV0{TargetStatus: command.TargetStatus}
	}
	return store.setControlStateV0(ctx, runFileControlCommandV0{
		runRef:       command.RunRef,
		status:       command.TargetStatus,
		requestedBy:  command.RequestedBy,
		reason:       command.Reason,
		idempotency:  command.IdempotencyKey,
		evidenceRefs: command.EvidenceRefs,
	})
}

func (store *RunFileStoreV0) setControlStateV0(
	_ context.Context,
	command runFileControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	command.runRef = strings.TrimSpace(command.runRef)
	if command.runRef == "" {
		return orquestaruncontrol.RunControlStateV0{}, ErrRunFileRunRefRequiredV0
	}
	state := runFileControlStateFromCommandV0(command)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	if previous, ok := store.control[command.runRef]; ok {
		state.CheckpointRecorded = previous.CheckpointRecorded
		state.EvidenceRefs = compactRunFileStringsV0(append(previous.EvidenceRefs, state.EvidenceRefs...))
	}
	next := cloneRunFileControlMapV0(store.control)
	next[command.runRef] = cloneRunFileControlStateV0(state)
	if err := persistRunFileControlV0(store.controlPath, next); err != nil {
		return orquestaruncontrol.RunControlStateV0{}, err
	}
	store.control = next
	return cloneRunFileControlStateV0(state), nil
}

func runFileControlStateFromCommandV0(
	command runFileControlCommandV0,
) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef:       command.runRef,
		Status:       command.status,
		Forced:       command.forced,
		EvidenceRefs: compactRunFileStringsV0(command.evidenceRefs),
		Meta: orquestaruncontrol.RunControlMetaV0{
			RequestedBy:    strings.TrimSpace(command.requestedBy),
			Reason:         strings.TrimSpace(command.reason),
			IdempotencyKey: strings.TrimSpace(command.idempotency),
		},
	}
}

func loadRunFileControlV0(
	path string,
) (map[string]orquestaruncontrol.RunControlStateV0, error) {
	var snapshot runFileControlSnapshotV0
	found, err := readJSONSnapshotV0(path, &snapshot)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]orquestaruncontrol.RunControlStateV0{}, nil
	}
	if snapshot.SchemaVersion != runFileControlSchemaVersionV0 {
		return nil, fmt.Errorf("orquesta_run_file: control_schema_invalid")
	}
	if len(snapshot.Records) > runFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("orquesta_run_file: control_records_limit_exceeded")
	}
	records := map[string]orquestaruncontrol.RunControlStateV0{}
	for _, record := range snapshot.Records {
		state, err := normalizeLoadedRunFileControlStateV0(record)
		if err != nil {
			return nil, err
		}
		records[state.RunRef] = state
	}
	return records, nil
}

func normalizeLoadedRunFileControlStateV0(
	record runFileControlRecordV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	state := cloneRunFileControlStateV0(record.State)
	state.RunRef = strings.TrimSpace(state.RunRef)
	if state.RunRef == "" {
		state.RunRef = strings.TrimSpace(record.RunRef)
	}
	if state.RunRef == "" {
		return orquestaruncontrol.RunControlStateV0{},
			fmt.Errorf("orquesta_run_file: control_record_invalid")
	}
	return state, nil
}

func persistRunFileControlV0(
	path string,
	records map[string]orquestaruncontrol.RunControlStateV0,
) error {
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	snapshot := runFileControlSnapshotV0{
		SchemaVersion: runFileControlSchemaVersionV0,
		Records:       make([]runFileControlRecordV0, 0, len(keys)),
	}
	for _, key := range keys {
		state := cloneRunFileControlStateV0(records[key])
		snapshot.Records = append(snapshot.Records, runFileControlRecordV0{
			RunRef: state.RunRef,
			State:  state,
		})
	}
	return writeAtomicJSONSnapshotV0(path, snapshot)
}

func cloneRunFileControlMapV0(
	records map[string]orquestaruncontrol.RunControlStateV0,
) map[string]orquestaruncontrol.RunControlStateV0 {
	next := make(map[string]orquestaruncontrol.RunControlStateV0, len(records))
	for key, state := range records {
		next[key] = cloneRunFileControlStateV0(state)
	}
	return next
}

func cloneRunFileControlStateV0(
	state orquestaruncontrol.RunControlStateV0,
) orquestaruncontrol.RunControlStateV0 {
	state.EvidenceRefs = append([]string(nil), state.EvidenceRefs...)
	return state
}
