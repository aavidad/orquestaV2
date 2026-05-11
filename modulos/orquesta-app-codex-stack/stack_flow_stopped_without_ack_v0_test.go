package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestDrainRunV0ProcesoParadoSinACKNoQuedaEsperandoIndefinido(t *testing.T) {
	ctx := context.Background()
	runtime := newStoppedPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	started := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if drainRunHasPendingExternalAgentsV0(started) {
		t.Fatalf("postDirector dejo agentes parados sin cerrar: started=%v stopped=%v confirmed=%v", started.StartedAgents, started.StoppedAgents, started.ConfirmedStoppedAgents)
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-stopped-no-ack-001",
		MaxBursts:            12,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     2,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if drainRunHasPendingExternalAgentsV0(run) {
		t.Fatalf("run sigue con agentes externos pendientes: status=%s started=%v stopped=%v confirmed=%v", drain.Status, run.StartedAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) {
		t.Fatalf("sin assessment stop_agent: %+v", run.AgentAssessments)
	}
	if len(compactStringsV0(run.StoppedAgents)) == 0 ||
		len(compactStringsV0(run.ConfirmedStoppedAgents)) == 0 {
		t.Fatalf("sin parada confirmada: stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
}

type stoppedPendingAckCodexStackRuntimeV0 struct {
	*pendingAckCodexStackRuntimeV0
}

func newStoppedPendingAckCodexStackRuntimeV0() *stoppedPendingAckCodexStackRuntimeV0 {
	return &stoppedPendingAckCodexStackRuntimeV0{
		pendingAckCodexStackRuntimeV0: newPendingAckCodexStackRuntimeV0(),
	}
}

func (runtime *stoppedPendingAckCodexStackRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot, err := runtime.pendingAckCodexStackRuntimeV0.LaunchV0(ctx, req)
	if err != nil {
		return snapshot, err
	}
	runtime.mu.Lock()
	stopped := runtime.snapshots[snapshot.ProcessRef]
	stopped.Status = orquestaruntime.ProcessRuntimeStoppedV0
	runtime.snapshots[snapshot.ProcessRef] = stopped
	runtime.mu.Unlock()
	return snapshot, nil
}
