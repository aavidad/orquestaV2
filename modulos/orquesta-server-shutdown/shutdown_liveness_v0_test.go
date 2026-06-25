package orquestaservershutdown

import (
	"context"
	"testing"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestShutdownServerV0ReadyConLivenessObservadaSinProcesosVivos(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-stale-agents", AppRef: "app-a", Status: "ready"},
	})
	deps.stats.stats["run-stale-agents"] = RunShutdownStatsV0{
		RunRef:                  "run-stale-agents",
		AgentsInFlight:          3,
		ProcessLivenessObserved: true,
		AgentsRunningStale:      2,
		AgentsLost:              1,
	}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		Forced:      true,
		RequestedBy: "orquesta-director",
		Reason:      "apagado con agentes logicos obsoletos",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		result.AgentsInFlight != 3 ||
		result.AgentsRunningLive != 0 ||
		result.AgentsRunningStale != 2 ||
		result.AgentsLost != 1 ||
		!result.ProcessLivenessObserved ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
}

func TestShutdownServerV0EsperaConLivenessObservadaYProcesoVivo(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-live-agent", AppRef: "app-a", Status: "ready"},
	})
	deps.stats.stats["run-live-agent"] = RunShutdownStatsV0{
		RunRef:                  "run-live-agent",
		AgentsInFlight:          3,
		ProcessLivenessObserved: true,
		AgentsRunningLive:       1,
		AgentsRunningStale:      2,
	}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		Forced:      true,
		RequestedBy: "orquesta-director",
		Reason:      "apagado con un proceso vivo",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusWaitingDrainV0 ||
		result.AgentsRunningLive != 1 ||
		deps.supervisor.calls != 1 {
		t.Fatalf("result=%+v supervisor=%+v", result, deps.supervisor)
	}
}
