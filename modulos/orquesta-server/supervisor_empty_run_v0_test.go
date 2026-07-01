package orquestaserver

import (
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	stopreason "orquesta/modulos/orquesta-run-supervisor/stopreason"
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

func TestSupervisorProjectionV0DistingueOutboxExternalWaitYProcesoVerificadoV0(t *testing.T) {
	cases := []struct {
		name         string
		result       orquestarunsupervisor.RunSupervisorResultV0
		wantStatus   string
		wantStop     string
		wantCategory string
	}{{
		name: "outbox pendiente",
		result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: "wait_unhandled_outbox",
			StopProjection: stopreason.ProjectionV0{
				PublicReason: "wait_unhandled_outbox",
				Category:     "wait_outbox",
			},
		},
		wantStatus:   SupervisorPublicStatusWaitingOutboxV0,
		wantStop:     SupervisorPublicStopWaitingOutboxV0,
		wantCategory: SupervisorPublicCategoryWaitOutboxV0,
	}, {
		name: "espera externa",
		result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: "wait_external",
			StopProjection: stopreason.ProjectionV0{
				PublicReason: "wait_external",
				Category:     "wait_external",
			},
		},
		wantStatus:   SupervisorPublicStatusWaitingExternalV0,
		wantStop:     SupervisorPublicStopWaitingExternalV0,
		wantCategory: SupervisorPublicCategoryWaitExternalV0,
	}, {
		name: "proceso externo verificado",
		result: orquestarunsupervisor.RunSupervisorResultV0{
			StopReason: "running",
			StopProjection: stopreason.ProjectionV0{
				PublicReason: "running",
				Category:     "external_process",
				EvidenceRefs: []string{"process_live"},
			},
		},
		wantStatus:   SupervisorPublicStatusRunningLiveV0,
		wantStop:     SupervisorPublicStopRunningLiveV0,
		wantCategory: SupervisorPublicCategoryExternalProcessV0,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projection := supervisorPublicProjectionV0(tc.result, collectSupervisorResultMetricsV0(tc.result))
			if projection.Status != tc.wantStatus ||
				projection.StopPublic != tc.wantStop ||
				projection.StopCategory != tc.wantCategory {
				t.Fatalf("projection=%+v", projection)
			}
		})
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
