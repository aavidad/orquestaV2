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
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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

func TestCodexStackV0RunSupervisorStopForzadoQuedaPendingSiRuntimeNoConfirmaV0(t *testing.T) {
	runtime := newFirstStopPendingCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	control := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:      "stop",
		RunRef:      director.RunRef,
		RequestedBy: "director",
		Reason:      "parada pendiente de confirmacion runtime",
		Forced:      true,
	})
	if control.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) {
		t.Fatalf("control=%+v", control)
	}

	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-stop-pending-runtime-001",
		CorrelationID:        "corr-stop-pending-runtime-001",
		RunRef:               director.RunRef,
		MaxTicks:             2,
		MaxBursts:            2,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 8,
		MaxCommands:          16,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    1,
		MaxExternalWaits:     1,
	})
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.StopReason != string(CodexSupervisorStopPendingV0) ||
		supervisor.Last.Status != string(CodexSupervisorRuntimeStopPendingV0) ||
		!codexStackRefsContainPartV0(supervisor.Last.EvidenceRefs, "evidence-ref-codex-supervisor-stop-pending-runtime-not-confirmed") {
		t.Fatalf("supervisor=%+v", supervisor)
	}
	if runtime.stopCountV0() == 0 || runtime.ignoredProcessRefV0() == "" {
		t.Fatalf("runtime no recibio stop pendiente: stops=%d ignored=%q", runtime.stopCountV0(), runtime.ignoredProcessRefV0())
	}
	snapshot := runtime.ignoredSnapshotV0()
	if snapshot.Status != orquestaruntime.ProcessRuntimeRunningV0 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStopRequestedV0 {
		t.Fatalf("state=%+v", state)
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.ConfirmedStoppedAgents) >= len(run.StartedAgents) {
		t.Fatalf("run no debe estar confirmado parado: %+v", run)
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

type firstStopPendingCodexStackRuntimeV0 struct {
	*pendingAckCodexStackRuntimeV0

	ignored bool
	ref     string
}

func newFirstStopPendingCodexStackRuntimeV0() *firstStopPendingCodexStackRuntimeV0 {
	return &firstStopPendingCodexStackRuntimeV0{
		pendingAckCodexStackRuntimeV0: newPendingAckCodexStackRuntimeV0(),
	}
}

func (runtime *firstStopPendingCodexStackRuntimeV0) StopV0(
	_ context.Context,
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	snapshot := runtime.snapshots[processRef]
	runtime.stops = append(runtime.stops, processRef)
	if !runtime.ignored {
		runtime.ignored = true
		runtime.ref = processRef
		snapshot.Status = orquestaruntime.ProcessRuntimeRunningV0
		runtime.snapshots[processRef] = snapshot
		return snapshot, nil
	}
	snapshot.Status = orquestaruntime.ProcessRuntimeStoppedV0
	snapshot.StopRef = "stop-ref-" + processRef
	runtime.snapshots[processRef] = snapshot
	return snapshot, nil
}

func (runtime *firstStopPendingCodexStackRuntimeV0) ignoredProcessRefV0() string {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.ref
}

func (runtime *firstStopPendingCodexStackRuntimeV0) ignoredSnapshotV0() orquestaruntime.ProcessRuntimeSnapshotV0 {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.snapshots[runtime.ref]
}
