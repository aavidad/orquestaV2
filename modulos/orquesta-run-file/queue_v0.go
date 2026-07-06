package orquestarunfile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const runFileQueueSchemaVersionV0 = "orquesta.run_file.queue.v0"

type runFileQueueSnapshotV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	Records       []runFileQueueRecordV0 `json:"records"`
}

type runFileQueueRecordV0 struct {
	RunRef    string                                    `json:"run_ref"`
	QueueRef  string                                    `json:"queue_ref,omitempty"`
	Candidate orquestarunqueue.RunSchedulingCandidateV0 `json:"candidate"`
}

func (store *RunFileStoreV0) UpsertRunSchedulingCandidateV0(
	_ context.Context,
	queueRef string,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	candidate.RunRef = strings.TrimSpace(candidate.RunRef)
	if candidate.RunRef == "" {
		return orquestarunqueue.RunSchedulingCandidateV0{}, ErrRunFileRunRefRequiredV0
	}
	entry := runFileQueueEntryV0{
		queueRef:  strings.TrimSpace(queueRef),
		candidate: cloneRunFileCandidateV0(candidate),
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	next := cloneRunFileQueueMapV0(store.queueRuns)
	next[candidate.RunRef] = entry
	if err := persistRunFileQueueV0(store.queuePath, next); err != nil {
		return orquestarunqueue.RunSchedulingCandidateV0{}, err
	}
	store.queueRuns = next
	return cloneRunFileCandidateV0(candidate), nil
}

func (store *RunFileStoreV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	filter := runFileQueueFilterV0{
		queueRef:             strings.TrimSpace(request.QueueRef),
		runRef:               strings.TrimSpace(request.RunRef),
		appRefs:              runFileStringSetV0(request.AppRefs),
		limit:                request.Limit,
		includeNonExecutable: request.IncludeNonExecutable,
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	out := make([]orquestarunqueue.RunSchedulingCandidateV0, 0, len(store.queueRuns))
	for _, key := range sortedRunFileQueueKeysV0(store.queueRuns) {
		entry := store.queueRuns[key]
		if !filter.matches(entry) {
			continue
		}
		if !filter.includeNonExecutable &&
			!orquestarunqueue.IsExecutableRunStatusV0(entry.candidate.Status) {
			continue
		}
		out = append(out, cloneRunFileCandidateV0(entry.candidate))
		if filter.limitReached(len(out)) {
			break
		}
	}
	return out, nil
}

func (store *RunFileStoreV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	command = orquestarunqueue.NormalizeRunQueuePriorityCommandV0(command)
	if len(orquestarunqueue.ValidateRunQueuePriorityCommandV0(command)) > 0 {
		return orquestarunqueue.RunSchedulingCandidateV0{}, ErrRunFileRunRefRequiredV0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureLockedV0()
	next := cloneRunFileQueueMapV0(store.queueRuns)
	entry := applyRunFilePriorityCommandV0(next[command.RunRef], command)
	next[command.RunRef] = entry
	if err := persistRunFileQueueV0(store.queuePath, next); err != nil {
		return orquestarunqueue.RunSchedulingCandidateV0{}, err
	}
	store.queueRuns = next
	return cloneRunFileCandidateV0(entry.candidate), nil
}

type runFileQueueFilterV0 struct {
	queueRef             string
	runRef               string
	appRefs              map[string]struct{}
	limit                int
	includeNonExecutable bool
}

func (filter runFileQueueFilterV0) matches(entry runFileQueueEntryV0) bool {
	if filter.queueRef != "" && entry.queueRef != filter.queueRef {
		return false
	}
	if filter.runRef != "" && entry.candidate.RunRef != filter.runRef {
		return false
	}
	if len(filter.appRefs) == 0 {
		return true
	}
	_, ok := filter.appRefs[entry.candidate.AppRef]
	return ok
}

func (filter runFileQueueFilterV0) limitReached(count int) bool {
	return filter.limit > 0 && count >= filter.limit
}

func applyRunFilePriorityCommandV0(
	entry runFileQueueEntryV0,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) runFileQueueEntryV0 {
	candidate := entry.candidate
	candidate.RunRef = command.RunRef
	if command.QueueRef != "" {
		entry.queueRef = command.QueueRef
	}
	if strings.TrimSpace(candidate.Status) == "" {
		candidate.Status = orquestarunqueue.RunStatusReadyV0
	}
	if command.Status != "" {
		candidate.Status = command.Status
	}
	if command.AppRef != "" {
		candidate.AppRef = command.AppRef
	}
	if command.FairnessGroupRef != "" {
		candidate.FairnessGroupRef = command.FairnessGroupRef
	}
	if !orquestarunqueue.RunQueueAttemptGroupEmptyV0(command.AttemptGroup) {
		candidate.AttemptGroup = command.AttemptGroup
	}
	if command.ParentRunRef != "" {
		candidate.ParentRunRef = command.ParentRunRef
	}
	if command.SupersedesRunRef != "" {
		candidate.SupersedesRunRef = command.SupersedesRunRef
	}
	if command.RescueReason != "" {
		candidate.RescueReason = command.RescueReason
	}
	if command.Reason != "" {
		candidate.Reason = command.Reason
	}
	candidate.PriorityScore = command.PriorityScore
	if !command.UpdatedAt.IsZero() {
		candidate.UpdatedAt = command.UpdatedAt
	}
	candidate.EvidenceRefs = append([]string(nil), command.EvidenceRefs...)
	candidate.WorksetClaims = cloneRunFileWorksetClaimsV0(command.WorksetClaims)
	entry.candidate = candidate
	return entry
}

func loadRunFileQueueV0(path string) (map[string]runFileQueueEntryV0, error) {
	var snapshot runFileQueueSnapshotV0
	found, err := readJSONSnapshotV0(path, &snapshot)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]runFileQueueEntryV0{}, nil
	}
	if snapshot.SchemaVersion != runFileQueueSchemaVersionV0 {
		return nil, fmt.Errorf("orquesta_run_file: queue_schema_invalid")
	}
	if len(snapshot.Records) > runFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("orquesta_run_file: queue_records_limit_exceeded")
	}
	records := map[string]runFileQueueEntryV0{}
	for _, record := range snapshot.Records {
		entry, err := normalizeLoadedRunFileQueueEntryV0(record)
		if err != nil {
			return nil, err
		}
		records[entry.candidate.RunRef] = entry
	}
	return records, nil
}

func normalizeLoadedRunFileQueueEntryV0(
	record runFileQueueRecordV0,
) (runFileQueueEntryV0, error) {
	candidate := cloneRunFileCandidateV0(record.Candidate)
	candidate.RunRef = strings.TrimSpace(candidate.RunRef)
	if candidate.RunRef == "" {
		candidate.RunRef = strings.TrimSpace(record.RunRef)
	}
	if candidate.RunRef == "" {
		return runFileQueueEntryV0{}, fmt.Errorf("orquesta_run_file: queue_record_invalid")
	}
	return runFileQueueEntryV0{
		queueRef:  strings.TrimSpace(record.QueueRef),
		candidate: candidate,
	}, nil
}

func persistRunFileQueueV0(
	path string,
	records map[string]runFileQueueEntryV0,
) error {
	keys := sortedRunFileQueueKeysV0(records)
	snapshot := runFileQueueSnapshotV0{
		SchemaVersion: runFileQueueSchemaVersionV0,
		Records:       make([]runFileQueueRecordV0, 0, len(keys)),
	}
	for _, key := range keys {
		entry := records[key]
		candidate := cloneRunFileCandidateV0(entry.candidate)
		snapshot.Records = append(snapshot.Records, runFileQueueRecordV0{
			RunRef:    candidate.RunRef,
			QueueRef:  entry.queueRef,
			Candidate: candidate,
		})
	}
	return writeAtomicJSONSnapshotV0(path, snapshot)
}

func cloneRunFileQueueMapV0(
	records map[string]runFileQueueEntryV0,
) map[string]runFileQueueEntryV0 {
	next := make(map[string]runFileQueueEntryV0, len(records))
	for key, entry := range records {
		next[key] = runFileQueueEntryV0{
			queueRef:  entry.queueRef,
			candidate: cloneRunFileCandidateV0(entry.candidate),
		}
	}
	return next
}

func sortedRunFileQueueKeysV0(records map[string]runFileQueueEntryV0) []string {
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneRunFileCandidateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) orquestarunqueue.RunSchedulingCandidateV0 {
	candidate.AttemptGroup = orquestarunqueue.NormalizeRunQueueAttemptGroupV0(candidate.AttemptGroup)
	candidate.EvidenceRefs = append([]string(nil), candidate.EvidenceRefs...)
	candidate.WorksetClaims = cloneRunFileWorksetClaimsV0(candidate.WorksetClaims)
	return candidate
}

func cloneRunFileWorksetClaimsV0(
	claims []orquestarunqueue.WorksetClaimV0,
) []orquestarunqueue.WorksetClaimV0 {
	out := make([]orquestarunqueue.WorksetClaimV0, 0, len(claims))
	for _, claim := range claims {
		claim.ReadSet = append([]orquestarunqueue.ScopeRefV0(nil), claim.ReadSet...)
		claim.WriteSet = append([]orquestarunqueue.ScopeRefV0(nil), claim.WriteSet...)
		claim.DependsOn = append([]string(nil), claim.DependsOn...)
		claim.EvidenceRefs = append([]string(nil), claim.EvidenceRefs...)
		out = append(out, claim)
	}
	if out == nil {
		return []orquestarunqueue.WorksetClaimV0{}
	}
	return out
}
