package orquestaserver

import (
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestSupervisorProjectionV0ExternalWorkEmptyRunNoEsOKV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	state := (&StatusTrackerV0{}).MarkSupervisorV0(
		orquestarunsupervisor.RunSupervisorCommandV0{QueueRef: "queue-ref-srv-task-021"},
		supervisorResultWithExternalEmptyRunForTestV0("run-ref-srv-task-021"),
		now,
	)

	if state.LastSupervisorStatus != SupervisorPublicStatusExternalEmptyRunV0 ||
		state.LastSupervisorStatus == SupervisorPublicStatusOKV0 ||
		state.LastSupervisorStopPublic != SupervisorPublicStopExternalEmptyRunV0 ||
		state.LastSupervisorStopCategory != SupervisorPublicCategoryExternalProcessV0 {
		t.Fatalf("state=%+v", state)
	}
	if got := state.LastSupervisorOperationalMessage.Counters["external_work_empty_run"]; got != 1 {
		t.Fatalf("external_work_empty_run=%d counters=%v", got, state.LastSupervisorOperationalMessage.Counters)
	}
}

func supervisorResultWithExternalEmptyRunForTestV0(runRef string) orquestarunsupervisor.RunSupervisorResultV0 {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason:      "done",
		TotalExecutions: 1,
		Ticks: []orquestarunsupervisor.RunSupervisorTickSummaryV0{{
			TickNumber: 1,
			Result: orquestaruncoordinator.RunCoordinatorTickResultV0{
				Ranked: []orquestaruncoordinator.RankedRunSummaryV0{{RunRef: runRef, Rank: 1}},
				Executions: []orquestaruncoordinator.RunExecutionSummaryV0{{
					RunRef:      runRef,
					Outcome:     "quiescent",
					QueueStatus: "done",
					EvidenceRefs: []string{
						"quiescent",
						"projection-tasks-0",
						"projection-open-tasks-0",
						"projection-requested-agents-0",
					},
				}},
			},
		}},
	}
}
