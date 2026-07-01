package orquestaserver

import (
	"strings"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type supervisorGoalBackendActiveSnapshotV0 struct {
	Active       int
	Observed     int
	Terminal     int
	RunRefs      []string
	GoalRefs     []string
	EvidenceRefs []string
}

func supervisorGoalBackendActiveSnapshotFromStateV0(state StateV0) supervisorGoalBackendActiveSnapshotV0 {
	message := state.GoalObserverOperationalMessage
	if message == nil {
		return supervisorGoalBackendActiveSnapshotV0{}
	}
	observed := nonNegativeServerIntV0(message.Counters["observed"])
	terminal := nonNegativeServerIntV0(message.Counters["terminal"])
	active := observed - terminal
	if active < 0 {
		active = 0
	}
	return supervisorGoalBackendActiveSnapshotV0{
		Active:       active,
		Observed:     observed,
		Terminal:     terminal,
		RunRefs:      compactServerOperationalRefsV0(message.RunRefs),
		GoalRefs:     compactServerOperationalRefsV0(message.GoalRefs),
		EvidenceRefs: compactServerOperationalRefsV0(message.EvidenceRefs),
	}
}

func supervisorGoalBackendSnapshotActiveV0(snapshot supervisorGoalBackendActiveSnapshotV0) bool {
	return snapshot.Active > 0 && (len(snapshot.RunRefs) > 0 || len(snapshot.GoalRefs) > 0)
}

func supervisorQueueIdleButGoalBackendActiveV0(
	metrics supervisorResultMetricsV0,
	projection SupervisorPublicProjectionV0,
	snapshot supervisorGoalBackendActiveSnapshotV0,
) bool {
	return metrics.QueueSize == 0 &&
		strings.TrimSpace(projection.StopPublic) == "idle_no_execution" &&
		supervisorGoalBackendSnapshotActiveV0(snapshot)
}

func supervisorProjectionWithGoalBackendV0(
	projection SupervisorPublicProjectionV0,
	metrics supervisorResultMetricsV0,
	state StateV0,
) (SupervisorPublicProjectionV0, supervisorGoalBackendActiveSnapshotV0) {
	snapshot := supervisorGoalBackendActiveSnapshotFromStateV0(state)
	if !supervisorQueueIdleButGoalBackendActiveV0(metrics, projection, snapshot) {
		return projection, snapshot
	}
	projection.Status = SupervisorPublicStatusQueueIdleGoalBackendActiveV0
	projection.StopPublic = SupervisorPublicStopQueueIdleButGoalBackendActiveV0
	projection.StopCategory = SupervisorPublicCategoryGoalBackendV0
	return projection, snapshot
}

func supervisorResultQueueIdleButGoalBackendActiveForStateV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
	state StateV0,
) bool {
	metrics := collectSupervisorResultMetricsV0(result)
	projection := supervisorPublicProjectionV0(result, metrics)
	snapshot := supervisorGoalBackendActiveSnapshotFromStateV0(state)
	return supervisorQueueIdleButGoalBackendActiveV0(metrics, projection, snapshot)
}

func supervisorCountersWithGoalBackendV0(
	counters map[string]int,
	snapshot supervisorGoalBackendActiveSnapshotV0,
) map[string]int {
	out := compactServerOperationalCountersV0(counters)
	if !supervisorGoalBackendSnapshotActiveV0(snapshot) {
		return out
	}
	out["goal_backend_active"] = snapshot.Active
	out["goal_backend_observed"] = snapshot.Observed
	out["goal_backend_terminal"] = snapshot.Terminal
	return out
}
