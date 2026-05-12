package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func TestCodexStackV0StopForzadoPorAPIDrenaAgentesYActualizaStats(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	control := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:      "stop",
		RunRef:      director.RunRef,
		RequestedBy: "director",
		Reason:      "cierre controlado de prueba",
		Forced:      true,
	})
	if control.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) || !control.Forced {
		t.Fatalf("control=%+v", control)
	}

	tick, err := stack.RunGlobalTickV0(context.Background(), orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef:       DefaultRunQueueRefV0,
		MaxRuns:        1,
		OccurredAt:     time.Date(2026, 5, 13, 8, 30, 0, 0, time.UTC),
		CorrelationID:  "corr-stack-stop-forzado-stats-001",
		DrainLimits:    stopStatsDrainLimitsV0(),
		ExcludeRunRefs: nil,
	})
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v tick=%+v", err, tick)
	}
	if len(tick.Executions) != 1 || tick.Executions[0].RunRef != director.RunRef {
		t.Fatalf("tick=%+v", tick)
	}
	if runtime.stopCountV0() != len(director.StartedAgents) {
		t.Fatalf("stops=%d want=%d", runtime.stopCountV0(), len(director.StartedAgents))
	}

	stats := postDirectorStatsStackV0(t, stack, director.RunRef)
	if stats.Stats == nil ||
		stats.Stats.Counts.AgentsStopRequested != len(director.StartedAgents) ||
		stats.Stats.Counts.AgentsStopConfirmed != len(director.StartedAgents) ||
		stats.Stats.Counts.AgentsInFlight != 0 {
		t.Fatalf("stats=%+v director=%+v", stats.Stats, director)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("state=%+v", state)
	}
}

func stopStatsDrainLimitsV0() orquestaruncoordinator.RunDrainLimitsV0 {
	return orquestaruncoordinator.RunDrainLimitsV0{
		MaxBursts:            4,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          24,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    1,
		MaxExternalWaits:     1,
	}
}

func postDirectorStatsStackV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) orquestamcp.MCPDirectorStatsToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPDirectorStatsToolInputV0{
		RunRef:               runRef,
		CorrelationID:        "corr-stack-director-stats-stop-001",
		IncludeProcessRefs:   true,
		IncludeAgentProgress: true,
		IncludeAgentUsage:    true,
	}); err != nil {
		t.Fatalf("encode stats: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/director/stats", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	return result
}
