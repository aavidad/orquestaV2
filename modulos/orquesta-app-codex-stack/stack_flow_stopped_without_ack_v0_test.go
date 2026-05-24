package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestDrainRunV0ProcesoParadoSinACKNoQuedaEsperandoIndefinido(t *testing.T) {
	ctx := context.Background()
	runtime := newStoppedPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	started := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if drainRunHasPendingExternalAgentsV0(started, nil) {
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
	if drainRunHasPendingExternalAgentsV0(run, nil) {
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

func TestDrainRunV0WaitAgentRefsProcesoParadoSinACKActivaSupervision(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	agentRef := strings.TrimSpace(director.DirectorTask.AgentRequestID)
	if agentRef == "" {
		t.Fatalf("director sin agente: %+v", director)
	}
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, director.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-stopped-scoped-no-ack-001",
		WaitAgentRefs:        []string{agentRef},
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v status=%s final=%+v", err, drain.Status, drain.Final)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) {
		t.Fatalf("WaitAgentRefs no paso por supervision de progreso: assessments=%+v run=%+v", run.AgentAssessments, run)
	}
	if !codexStackHasRefV0(run.StoppedAgents, agentRef) ||
		!codexStackHasRefV0(run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("agente acotado no quedo parado/confirmado: status=%s stopped=%v confirmed=%v", drain.Status, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
}

func TestDrainRunV0ProcesoParadoPorCapacidadLimitadaNoMarcaBasura(t *testing.T) {
	ctx := context.Background()
	runtime := newCapacityLimitedPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-capacity-limited-no-ack-001",
		MaxBursts:            12,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("run sigue pendiente tras capacidad limitada: stopped=%v confirmed=%v", run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) {
		t.Fatalf("sin assessment capacity_limited stop_agent: %+v", run.AgentAssessments)
	}
	if codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictGarbageV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0) {
		t.Fatalf("capacity_limited no debe clasificarse como basura/bucle: %+v", run.AgentAssessments)
	}
}

func TestDrainRunHasPendingExternalAgentsV0StopRequestedSinConfirmarSiguePendiente(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents: []string{"agent-ref-stack-stop-pending-001"},
		StoppedAgents: []string{"agent-ref-stack-stop-pending-001"},
	}
	if !drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("stop solicitado sin confirmacion debe seguir pendiente: %+v", run)
	}

	run.ConfirmedStoppedAgents = []string{"agent-ref-stack-stop-pending-001"}
	if drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("stop confirmado debe cerrar pendiente: %+v", run)
	}
}

func TestDrainRunHasPendingExternalAgentsV0WaitRefsNoReesperaStopSolicitado(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents: []string{"agent-ref-stack-stop-scoped-001"},
		StoppedAgents: []string{"agent-ref-stack-stop-scoped-001"},
	}
	if drainRunHasPendingExternalAgentsV0(run, []string{"agent-ref-stack-stop-scoped-001"}) {
		t.Fatalf("WaitAgentRefs acotado no debe reesperar stop solicitado: %+v", run)
	}
	if !drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("scope legacy de run completo conserva stop solicitado como pendiente: %+v", run)
	}
}

func TestDrainRunHasPendingExternalAgentsV0FiltraCohorteObjetivo(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		StartedAgents:   []string{"agent-old", "agent-new"},
		DeliveredAgents: []string{"agent-new"},
	}
	if drainRunHasPendingExternalAgentsV0(run, []string{"agent-new"}) {
		t.Fatalf("cohorte nueva entregada no debe esperar agent-old: %+v", run)
	}
	if !drainRunHasPendingExternalAgentsV0(run, []string{"agent-old"}) {
		t.Fatalf("cohorte vieja pendiente debe esperar: %+v", run)
	}
	if !drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("WaitAgentRefs vacio mantiene compatibilidad legacy y debe esperar agent-old: %+v", run)
	}
}

type capacityLimitedPendingAckCodexStackRuntimeV0 struct {
	*stoppedPendingAckCodexStackRuntimeV0
}

func newCapacityLimitedPendingAckCodexStackRuntimeV0() *capacityLimitedPendingAckCodexStackRuntimeV0 {
	return &capacityLimitedPendingAckCodexStackRuntimeV0{
		stoppedPendingAckCodexStackRuntimeV0: newStoppedPendingAckCodexStackRuntimeV0(),
	}
}

func (runtime *capacityLimitedPendingAckCodexStackRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot, err := runtime.stoppedPendingAckCodexStackRuntimeV0.LaunchV0(ctx, req)
	if err != nil {
		return snapshot, err
	}
	stderrPath := filepath.Join(filepath.Dir(req.CommandPath), orquestaruntimecodex.CodexStderrFileNameV0)
	return snapshot, os.WriteFile(
		stderrPath,
		[]byte("ERROR: Selected model is at capacity. Please try a different model."),
		0o600,
	)
}

type stoppedPendingAckCodexStackRuntimeV0 struct {
	*pendingAckCodexStackRuntimeV0
}

func newStoppedPendingAckCodexStackRuntimeV0() *stoppedPendingAckCodexStackRuntimeV0 {
	return &stoppedPendingAckCodexStackRuntimeV0{
		pendingAckCodexStackRuntimeV0: newPendingAckCodexStackRuntimeV0(),
	}
}

func (runtime *pendingAckCodexStackRuntimeV0) markStoppedForTestV0(processRef string) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	stopped := runtime.snapshots[processRef]
	stopped.Status = orquestaruntime.ProcessRuntimeStoppedV0
	runtime.snapshots[processRef] = stopped
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
