package orquestaservershutdown

import (
	"context"
	"reflect"
	"testing"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestShutdownServerV0NoForzadoBloqueaConGoalActivo(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-goal-active", AppRef: "app-a", Status: "ready"},
	})
	deps.active.works = []ActiveShutdownWorkV0{{
		Kind:            "goal_first",
		RunRef:          "run-goal-active",
		WorkRef:         "goal-ref-001",
		ExternalWorkRef: "external-goal-ref-001",
		Status:          "running",
		EvidenceRefs:    []string{"goal-state-ref-run-goal-active"},
	}}
	deps.active.evidence = []string{"goal-active-list-ref"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:   "orquesta-director",
		Reason:        "apagado no forzado con goal activo",
		CorrelationID: "corr-active-goal-shutdown",
		EvidenceRefs:  []string{"operator-shutdown-request"},
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusActiveGoalsPresentV0 ||
		result.ActiveWorkCount != 1 ||
		len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].RunRef != "run-goal-active" ||
		result.ActiveWorks[0].WorkRef != "goal-ref-001" ||
		len(deps.control.stopped) != 0 ||
		deps.supervisor.calls != 0 {
		t.Fatalf("result=%+v stopped=%v supervisor=%+v", result, deps.control.stopped, deps.supervisor)
	}
	if !serverShutdownStringsContainForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-active-goals-present") {
		t.Fatalf("evidence_refs no incluye bloqueo active goals: %+v", result.EvidenceRefs)
	}
}

func TestShutdownServerV0ForzadoNoBloqueaConGoalActivo(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-goal-active-force", AppRef: "app-a", Status: "ready"},
	})
	deps.active.works = []ActiveShutdownWorkV0{{
		Kind:    "goal_first",
		RunRef:  "run-goal-active-force",
		WorkRef: "goal-ref-force",
		Status:  "running",
	}}
	deps.stats.stats["run-goal-active-force"] = RunShutdownStatsV0{RunRef: "run-goal-active-force"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		Forced:      true,
		RequestedBy: "orquesta-director",
		Reason:      "apagado forzado con goal activo",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if !result.ShutdownReady ||
		result.Status != ServerShutdownStatusReadyV0 ||
		!reflect.DeepEqual(deps.control.stopped, []string{"run-goal-active-force"}) ||
		deps.active.calls != 1 {
		t.Fatalf("result=%+v stopped=%v active_calls=%d", result, deps.control.stopped, deps.active.calls)
	}
}

func TestShutdownServerV0ForzadoBloqueaConBackendStillRunning(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-goal-backend-active-timeout-force", AppRef: "app-a", Status: "ready"},
	})
	deps.active.works = []ActiveShutdownWorkV0{{
		Kind:            "goal_backend",
		RunRef:          "run-goal-backend-active-timeout-force",
		WorkRef:         "goal-ref-timeout-force",
		ExternalWorkRef: "external-goal-ref-timeout-force",
		Status:          ServerShutdownStatusBackendStillRunningV0,
		EvidenceRefs:    []string{"goal-timeout-force-ref"},
	}}
	deps.active.evidence = []string{"goal-backend-active-force-list-ref"}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		Forced:        true,
		RequestedBy:   "orquesta-director",
		Reason:        "apagado forzado con backend vivo",
		CorrelationID: "corr-backend-still-running-forced-shutdown",
		EvidenceRefs:  []string{"operator-forced-shutdown-request"},
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusBackendStillRunningV0 ||
		result.ActiveWorkCount != 1 ||
		len(result.ActiveWorks) != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" ||
		len(deps.control.stopped) != 0 ||
		deps.supervisor.calls != 0 ||
		deps.active.calls != 1 ||
		!serverShutdownStringsContainForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-backend-still-running") {
		t.Fatalf("result=%+v stopped=%v supervisor=%+v active_calls=%d", result, deps.control.stopped, deps.supervisor, deps.active.calls)
	}
}

func TestShutdownServerV0NoForzadoBloqueaConBackendStillRunning(t *testing.T) {
	deps := newServerShutdownDepsForTestV0([]orquestarunqueue.RunSchedulingCandidateV0{
		{RunRef: "run-goal-backend-active-timeout", AppRef: "app-a", Status: "ready"},
	})
	deps.active.works = []ActiveShutdownWorkV0{{
		Kind:            "goal_backend",
		RunRef:          "run-goal-backend-active-timeout",
		WorkRef:         "goal-ref-timeout",
		ExternalWorkRef: "external-goal-ref-timeout",
		Status:          ServerShutdownStatusBackendStillRunningV0,
		EvidenceRefs:    []string{"goal-timeout-ref"},
	}}

	result, err := ShutdownServerV0(context.Background(), deps.deps(), ServerShutdownCommandV0{
		RequestedBy:   "orquesta-director",
		Reason:        "apagado no forzado con backend vivo",
		CorrelationID: "corr-backend-still-running-shutdown",
	})
	if err != nil {
		t.Fatalf("ShutdownServerV0: %v", err)
	}
	if result.ShutdownReady ||
		result.Status != ServerShutdownStatusBackendStillRunningV0 ||
		result.ActiveWorkCount != 1 ||
		result.ActiveWorks[0].Kind != "goal_backend" ||
		len(deps.control.stopped) != 0 ||
		deps.supervisor.calls != 0 ||
		!serverShutdownStringsContainForTestV0(result.EvidenceRefs, "evidence-ref-shutdown-backend-still-running") {
		t.Fatalf("result=%+v stopped=%v supervisor=%+v", result, deps.control.stopped, deps.supervisor)
	}
}
