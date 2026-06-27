package orquestamcp

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const mcpAutoprogrammingQueuedNotDispatchedV0 = "queued_not_dispatched"

func diagnosticsFromQueuedNotDispatchedMCPAutoprogrammingV0(
	queue *MCPRunQueuePriorityToolResultV0,
	goalStatesByRunRef map[string]struct{},
	observedRuns ...*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if queue == nil {
		return nil
	}
	observed := mcpAutoprogrammingObservedRunStatsSetV0(observedRuns...)
	out := []MCPAutoprogrammingDiagnosticV0{}
	for _, candidate := range queue.Ranked {
		runRef := strings.TrimSpace(candidate.RunRef)
		_, goalFirst := goalStatesByRunRef[runRef]
		if runRef == "" ||
			!mcpAutoprogrammingQueuedNotDispatchedStatusV0(candidate.Status) ||
			goalFirst ||
			mcpAutoprogrammingObservedRunHasDispatchSignalV0(observed[runRef]) {
			continue
		}
		diagnostic := mcpAutoprogrammingDiagnosticV0(
			mcpAutoprogrammingQueuedNotDispatchedV0,
			"run:"+runRef,
			"run en cola sin agente/proceso observado; action=supervise_or_enable_resident_director",
		)
		diagnostic.EvidenceRefs = compactStringsMCPV0(append(
			[]string{"evidence-ref-run-queue-queued-not-dispatched"},
			candidate.EvidenceRefs...,
		))
		out = append(out, diagnostic)
	}
	return out
}

func countMCPAutoprogrammingQueuedNotDispatchedV0(
	queue *MCPRunQueuePriorityToolResultV0,
	goalStatesByRunRef map[string]struct{},
	observedRuns ...*MCPDirectorStatsToolResultV0,
) int {
	return len(diagnosticsFromQueuedNotDispatchedMCPAutoprogrammingV0(
		queue,
		goalStatesByRunRef,
		observedRuns...,
	))
}

func mcpAutoprogrammingQueuedNotDispatchedStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "ready", "queued", "pending":
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingObservedRunStatsSetV0(
	observedRuns ...*MCPDirectorStatsToolResultV0,
) map[string]*MCPDirectorStatsToolResultV0 {
	out := map[string]*MCPDirectorStatsToolResultV0{}
	for _, observed := range observedRuns {
		if observed == nil || observed.Stats == nil {
			continue
		}
		runRef := strings.TrimSpace(observed.Stats.RunRef)
		if runRef == "" {
			runRef = strings.TrimSpace(observed.RunRef)
		}
		if runRef != "" {
			out[runRef] = observed
		}
	}
	return out
}

func mcpAutoprogrammingObservedRunHasDispatchSignalV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Stats == nil {
		return false
	}
	counts := observed.Stats.Counts
	return counts.AgentsRequested > 0 ||
		counts.AgentsStarted > 0 ||
		counts.AgentsInFlight > 0 ||
		counts.AgentsDelivered > 0 ||
		counts.Deliveries > 0 ||
		mcpAutoprogrammingRunStatsHasLiveSignalV0(*observed.Stats)
}

func mcpAutoprogrammingGoalRunRefSetV0(
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
) map[string]struct{} {
	out := map[string]struct{}{}
	for runRef := range goalStatesByRunRef {
		runRef = strings.TrimSpace(runRef)
		if runRef != "" {
			out[runRef] = struct{}{}
		}
	}
	return out
}
