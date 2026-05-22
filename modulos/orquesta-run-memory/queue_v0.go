package orquestarunmemory

import (
	"context"
	"strings"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func (store *RunMemoryStoreV0) ListRunSchedulingCandidatesV0(
	_ context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	filter := runQueueFilterV0{
		queueRef: strings.TrimSpace(request.QueueRef),
		appRefs:  stringSetV0(request.AppRefs),
		limit:    request.Limit,
	}
	store.mu.RLock()
	out := make([]orquestarunqueue.RunSchedulingCandidateV0, 0, len(store.queueRuns))
	for _, entry := range store.queueRuns {
		if !filter.matches(entry) {
			continue
		}
		if !orquestarunqueue.IsExecutableRunStatusV0(entry.candidate.Status) {
			continue
		}
		out = append(out, cloneRunSchedulingCandidateV0(entry.candidate))
		if filter.limitReached(len(out)) {
			break
		}
	}
	store.mu.RUnlock()
	return out, nil
}

func (store *RunMemoryStoreV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	command = orquestarunqueue.NormalizeRunQueuePriorityCommandV0(command)
	if len(orquestarunqueue.ValidateRunQueuePriorityCommandV0(command)) > 0 {
		return orquestarunqueue.RunSchedulingCandidateV0{}, ErrRunRefRequiredV0
	}
	store.mu.Lock()
	entry := store.queueRuns[command.RunRef]
	entry = applyPriorityCommandV0(entry, command)
	store.queueRuns[command.RunRef] = entry
	store.mu.Unlock()
	return cloneRunSchedulingCandidateV0(entry.candidate), nil
}

type runQueueFilterV0 struct {
	queueRef string
	appRefs  map[string]struct{}
	limit    int
}

func (filter runQueueFilterV0) matches(entry runQueueEntryV0) bool {
	if filter.queueRef != "" && entry.queueRef != filter.queueRef {
		return false
	}
	if len(filter.appRefs) == 0 {
		return true
	}
	_, ok := filter.appRefs[entry.candidate.AppRef]
	return ok
}

func (filter runQueueFilterV0) limitReached(count int) bool {
	return filter.limit > 0 && count >= filter.limit
}

func applyPriorityCommandV0(
	entry runQueueEntryV0,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) runQueueEntryV0 {
	candidate := entry.candidate
	candidate.RunRef = command.RunRef
	if command.QueueRef != "" {
		entry.queueRef = command.QueueRef
	}
	if strings.TrimSpace(candidate.Status) == "" {
		candidate.Status = "ready"
	}
	if command.Status != "" {
		candidate.Status = command.Status
	}
	if command.AppRef != "" {
		candidate.AppRef = command.AppRef
	}
	candidate.PriorityScore = command.PriorityScore
	if !command.UpdatedAt.IsZero() {
		candidate.UpdatedAt = command.UpdatedAt
	}
	candidate.EvidenceRefs = append([]string(nil), command.EvidenceRefs...)
	entry.candidate = candidate
	return entry
}

func cloneRunSchedulingCandidateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) orquestarunqueue.RunSchedulingCandidateV0 {
	candidate.EvidenceRefs = append([]string(nil), candidate.EvidenceRefs...)
	return candidate
}
