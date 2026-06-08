package orquestacionnucleoapp

import (
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func scopedDirectorBriefingDispatchersV0(
	dispatchers []OutboxDispatcherBindingV0,
	targetRefs []string,
) []OutboxDispatcherBindingV0 {
	scope := newDirectorBriefingOutboxScopeV0(targetRefs)
	if !scope.enabled {
		return dispatchers
	}
	out := append([]OutboxDispatcherBindingV0(nil), dispatchers...)
	for i := range out {
		out[i].Reader = scopedDirectorBriefingOutboxReaderV0{inner: out[i].Reader, scope: scope}
	}
	return out
}

func scopedDirectorBriefingBatchDispatchersV0(
	dispatchers []OutboxBatchDispatcherBindingV0,
	targetRefs []string,
) []OutboxBatchDispatcherBindingV0 {
	scope := newDirectorBriefingOutboxScopeV0(targetRefs)
	if !scope.enabled {
		return dispatchers
	}
	out := append([]OutboxBatchDispatcherBindingV0(nil), dispatchers...)
	for i := range out {
		out[i].Reader = scopedDirectorBriefingOutboxReaderV0{inner: out[i].Reader, scope: scope}
	}
	return out
}

type directorBriefingOutboxScopeV0 struct {
	enabled bool
	ordered []string
	allowed map[string]struct{}
}

func newDirectorBriefingOutboxScopeV0(targetRefs []string) directorBriefingOutboxScopeV0 {
	ordered := compactStringsV0(targetRefs)
	scope := directorBriefingOutboxScopeV0{
		enabled: len(ordered) > 0,
		ordered: ordered,
		allowed: map[string]struct{}{},
	}
	for _, ref := range ordered {
		scope.allowed[ref] = struct{}{}
	}
	return scope
}

type scopedDirectorBriefingOutboxReaderV0 struct {
	inner orquestaoutboxdispatch.PendingOutboxReaderPortV0
	scope directorBriefingOutboxScopeV0
}

func (reader scopedDirectorBriefingOutboxReaderV0) ListPendingOutboxV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	if reader.inner == nil {
		return nil, []orquestaoutboxdispatch.DispatchIssueV0{{
			Code:    ErrDirectorBriefingExecutionInvalidV0,
			Field:   "reader",
			Message: "reader requerido",
		}}
	}
	pending, issues := reader.inner.ListPendingOutboxV0(filter)
	if !reader.scope.enabled || len(pending) == 0 {
		return pending, issues
	}
	byID := map[string]orquestaoutboxdispatch.OutboxPendingEntryV0{}
	for _, entry := range pending {
		messageID := strings.TrimSpace(entry.MessageID)
		if _, ok := reader.scope.allowed[messageID]; ok {
			byID[messageID] = entry
		}
	}
	out := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(byID))
	for _, ref := range reader.scope.ordered {
		if entry, ok := byID[ref]; ok {
			out = append(out, entry)
		}
	}
	return out, issues
}
