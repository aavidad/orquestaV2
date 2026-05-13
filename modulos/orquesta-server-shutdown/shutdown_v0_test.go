package orquestaservershutdown

import (
	"context"
	"reflect"
	"testing"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestShutdownServerV0SolicitaStopDrenaYQuedaReady(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-a", AppRef: "app-a", Status: "ready"},
		{RunRef: "run-b", AppRef: "app-b", Status: "ready"},
	})
	deps.stats.stats["run-a"] = RunShutdownStatsV0{RunRef: "run-a"}
	deps.stats.stats["run-b"] = RunShutdownStatsV0{RunRef: "run-b"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		QueueRef:      "global",
		Forced:        true,
		RequestedBy:   "operator",
		Reason:        "apagado controlado",
		CorrelationID: "corr-shutdown-test-001",
		OccurredAt:    time.Date(2026, 5, 13, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.RunsRequested != 2 ||
		result.RunsStopped != 2 ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if got := deps.control.stopped; !reflect.DeepEqual(got, []string{"run-a", "run-b"}) {
		t.Fatalf("stopped=%v", got)
	}
}

func TestShutdownServerV0NoDrenaSiFaltaCheckpoint(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-checkpoint", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.pending["run-checkpoint"] = []string{"agent-ref-a"}
	deps.checkpoint.evidence["run-checkpoint"] = []string{"shutdown-checkpoint-issue-pending-agent_ack"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy: "operator",
		Reason:      "apagado no forzado",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusWaitingCheckpointV0 ||
		result.CheckpointsPending != 1 ||
		result.CheckpointAgentsPending != 1 ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if len(result.Runs) != 1 ||
		!reflect.DeepEqual(result.Runs[0].PendingCheckpointAgentRefs, []string{"agent-ref-a"}) ||
		!reflect.DeepEqual(result.Runs[0].CheckpointEvidenceRefs, []string{"shutdown-checkpoint-issue-pending-agent_ack"}) {
		t.Fatalf("pending checkpoint no expuesto: %+v", result.Runs)
	}
	if len(deps.control.stopped) != 0 {
		t.Fatalf("stop requests=%+v", deps.control.stopped)
	}
}

func TestShutdownServerV0PreparaCheckpointAntesDeStopNoForzado(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-graceful", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.recorded["run-graceful"] = true
	deps.stats.stats["run-graceful"] = RunShutdownStatsV0{RunRef: "run-graceful"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:   "operator",
		Reason:        "apagado graceful",
		CorrelationID: "corr-shutdown-test-002",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.CheckpointsPending != 0 ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
	if !deps.control.states["run-graceful"].CheckpointRecorded ||
		!reflect.DeepEqual(deps.control.stopped, []string{"run-graceful"}) {
		t.Fatalf("control=%+v", deps.control)
	}
	if got := shutdownEventsForTestV0(deps.events); !reflect.DeepEqual(got, []string{
		"prepare:run-graceful",
		"record_checkpoint:run-graceful",
		"stop:run-graceful",
	}) {
		t.Fatalf("orden shutdown=%v", got)
	}
}

func TestShutdownServerV0NoPideStopSiCheckpointNoEstaListo(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-pending-before-stop", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.pending["run-pending-before-stop"] = []string{"agent-ref-a"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy: "operator",
		Reason:      "apagado no forzado",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.Status != ServerShutdownStatusWaitingCheckpointV0 ||
		result.Runs[0].StopRequested ||
		len(deps.control.stopped) != 0 ||
		!reflect.DeepEqual(shutdownEventsForTestV0(deps.events), []string{"prepare:run-pending-before-stop"}) {
		t.Fatalf("result=%+v stopped=%v events=%v", result, deps.control.stopped, shutdownEventsForTestV0(deps.events))
	}
}

func TestShutdownServerV0FuerzaStopSiDeadlineCheckpointExpirado(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-deadline", AppRef: "app-a", Status: "ready"},
	})
	deps.checkpoint.pending["run-deadline"] = []string{"agent-ref-a"}
	deps.checkpoint.evidence["run-deadline"] = []string{"shutdown-checkpoint-issue-pending-agent_ack"}
	deps.stats.stats["run-deadline"] = RunShutdownStatsV0{RunRef: "run-deadline"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:          "operator",
		Reason:               "apagado no forzado con deadline",
		OccurredAt:           time.Date(2026, 5, 13, 12, 5, 0, 0, time.UTC),
		CheckpointDeadlineAt: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.CheckpointsPending != 0 ||
		result.CheckpointDeadlinesExpired != 1 ||
		!result.Runs[0].ForcedAfterCheckpointDeadline ||
		!result.Runs[0].CheckpointDeadlineExpired ||
		!deps.control.states["run-deadline"].Forced ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v control=%+v supervisor=%+v", result, deps.control, deps.supervisor)
	}
	if !reflect.DeepEqual(shutdownEventsForTestV0(deps.events), []string{
		"prepare:run-deadline",
		"stop:run-deadline",
	}) {
		t.Fatalf("orden shutdown=%v", shutdownEventsForTestV0(deps.events))
	}
}

func TestShutdownServerV0SinRunsQuedaReady(t *testing.T) {
	deps := newServerShutdownDepsForTestV0(nil)

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.RunsRequested != 0 ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
}
