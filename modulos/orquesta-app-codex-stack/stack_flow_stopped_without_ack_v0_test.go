package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("sin assessment de revision no_ack: %+v", run.AgentAssessments)
	}
	if codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictGarbageV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) {
		t.Fatalf("no_ack no debe marcar basura ni stop_agent: %+v", run.AgentAssessments)
	}
	if len(compactStringsV0(run.LostAgents)) == 0 ||
		len(compactStringsV0(run.StoppedAgents)) != 0 ||
		len(compactStringsV0(run.ConfirmedStoppedAgents)) != 0 {
		t.Fatalf("no_ack debe reconciliar como lost sin parada runtime: lost=%v stopped=%v confirmed=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
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
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("WaitAgentRefs no paso por supervision de progreso: assessments=%+v run=%+v", run.AgentAssessments, run)
	}
	if len(compactStringsV0(run.DirectorQuestions)) == 0 {
		t.Fatalf("reconciliacion ask_director debe persistir pregunta no bloqueante para que el Director pueda decidir: %+v", run)
	}
	if !codexStackHasRefV0(run.LostAgents, agentRef) ||
		codexStackHasRefV0(run.StoppedAgents, agentRef) ||
		codexStackHasRefV0(run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("agente acotado no quedo reconciliado como lost: status=%s lost=%v stopped=%v confirmed=%v", drain.Status, run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
}

func TestDrainRunV0ProcesoParadoConACKCompletoIngiereAntesDeLost(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
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
	if _, err := runtime.StopV0(ctx, record.ProcessRef); err != nil {
		t.Fatalf("StopV0: %v", err)
	}

	descriptors := codexStackDescriptorsForTestV0(t, stack)
	var stoppedObservation orquestacionnucleoapp.AgentDeliveryObservationV0
	var ackPath string
	var descriptorFound bool
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.AgentRef) != agentRef {
			continue
		}
		stoppedObservation = drainObservationFromDescriptorForTestV0(descriptor)
		ackPath = strings.TrimSpace(descriptor.AckPath)
		descriptorFound = true
		break
	}
	if !descriptorFound {
		t.Fatalf("descriptor no encontrado agent=%s descriptors=%v", agentRef, codexStackRealSmokeDescriptorAgentsV0(descriptors))
	}
	if _, err := os.Stat(ackPath); err != nil {
		t.Fatalf("precondicion ACK completed inexistente path=%s err=%v", ackPath, err)
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-stopped-with-completed-ack-001",
		WaitAgentRefs:        []string{agentRef},
		MaxBursts:            4,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		MaxCommands:          12,
		MaxOutboxPerCycle:    4,
		MaxExternalWaits:     1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v status=%s final=%+v", err, drain.Status, drain.Final)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !drainObservationAlreadyRegisteredV0(run, stoppedObservation) {
		t.Fatalf(
			"ACK completed no se ingirio antes de lost: agents=%v started=%v delivered=%v deliveries=%v artifacts=%v lost=%v assessments=%v observation=%+v",
			run.Agents,
			run.StartedAgents,
			run.DeliveredAgents,
			run.Deliveries,
			run.PhaseArtifacts,
			run.LostAgents,
			run.AgentAssessments,
			stoppedObservation,
		)
	}
	if codexStackHasRefV0(run.LostAgents, agentRef) {
		t.Fatalf("agente con ACK completed no debe quedar lost: delivered=%v lost=%v assessments=%v", run.DeliveredAgents, run.LostAgents, run.AgentAssessments)
	}
	if codexStackRefsContainPartV0(run.AgentAssessments, "evidence-ref-no-ack") {
		t.Fatalf("ACK completed no debe pasar por reconciliacion no_ack: assessments=%v", run.AgentAssessments)
	}
}

func TestDrainRunV0SnapshotRuntimePerdidoNoBloqueaSupervisorV0(t *testing.T) {
	ctx := context.Background()
	runtime := newSnapshotMissingPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	agentRef := strings.TrimSpace(director.DirectorTask.AgentRequestID)
	if agentRef == "" {
		t.Fatalf("director sin agente: %+v", director)
	}

	drain, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-missing-runtime-snapshot-001",
		WaitAgentRefs:        []string{agentRef},
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxExternalWaits:     1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0 no debe bloquear por process_runtime_no_encontrado: %v status=%s final=%+v", err, drain.Status, drain.Final)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackHasRefV0(run.LostAgents, agentRef) ||
		codexStackHasRefV0(run.StoppedAgents, agentRef) ||
		codexStackHasRefV0(run.ConfirmedStoppedAgents, agentRef) {
		t.Fatalf("snapshot perdido debe reconciliar como lost: status=%s lost=%v stopped=%v confirmed=%v", drain.Status, run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("sin assessment de director tras snapshot perdido: %+v", run.AgentAssessments)
	}
}

func TestRunGlobalTickV0SnapshotRuntimePerdidoNoBloqueaSupervisorV0(t *testing.T) {
	ctx := context.Background()
	runtime := newSnapshotMissingPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	result, err := stack.RunGlobalTickV0(ctx, globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 no debe bloquear por process_runtime_no_encontrado: %v result=%+v", err, result)
	}
	if len(result.Executions) == 0 {
		t.Fatalf("tick sin ejecuciones tras recuperar process_runtime_no_encontrado: %+v", result)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if !codexStackHasRefV0(run.LostAgents, strings.TrimSpace(director.DirectorTask.AgentRequestID)) {
		t.Fatalf("run no reconciliado como lost: %+v", run)
	}
}

func TestDrainRunV0ProcesoParadoConTextoCapacidadNoDisparaFiltroDeCapacidad(t *testing.T) {
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
		t.Fatalf("run sigue pendiente tras texto de capacidad: lost=%v stopped=%v confirmed=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("texto de capacidad debe quedar como revision recuperable: %+v", run.AgentAssessments)
	}
	if codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictGarbageV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0) {
		t.Fatalf("texto de capacidad no debe clasificar fuerte: %+v", run.AgentAssessments)
	}
	if len(compactStringsV0(run.LostAgents)) == 0 ||
		len(compactStringsV0(run.StoppedAgents)) != 0 ||
		len(compactStringsV0(run.ConfirmedStoppedAgents)) != 0 {
		t.Fatalf("texto de capacidad debe reconciliar como lost sin stop runtime: lost=%v stopped=%v confirmed=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
}

func TestDrainRunV0ProcesoParadoPorAuthInvalidaPreguntaDirectorSinBasura(t *testing.T) {
	ctx := context.Background()
	runtime := newAuthInvalidPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	if _, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-auth-invalid-no-ack-001",
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
		t.Fatalf("run sigue pendiente tras auth invalida: lost=%v stopped=%v confirmed=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
	}
	if !codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#severity:"+orquestacoreworkflow.AgentAssessmentSeverityCriticalV0) {
		t.Fatalf("sin assessment auth ask_director critical: %+v", run.AgentAssessments)
	}
	if codexStackRefsContainPartV0(run.AgentAssessments, "#verdict:"+orquestacoreworkflow.AgentAssessmentVerdictGarbageV0) ||
		codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionStopAgentV0) {
		t.Fatalf("auth invalida no debe marcar basura ni stop_agent: %+v", run.AgentAssessments)
	}
	if len(compactStringsV0(run.LostAgents)) == 0 ||
		len(compactStringsV0(run.StoppedAgents)) != 0 ||
		len(compactStringsV0(run.ConfirmedStoppedAgents)) != 0 {
		t.Fatalf("auth invalida debe reconciliar como lost sin parada runtime: lost=%v stopped=%v confirmed=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents)
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

type authInvalidPendingAckCodexStackRuntimeV0 struct {
	*stoppedPendingAckCodexStackRuntimeV0
}

func newAuthInvalidPendingAckCodexStackRuntimeV0() *authInvalidPendingAckCodexStackRuntimeV0 {
	return &authInvalidPendingAckCodexStackRuntimeV0{
		stoppedPendingAckCodexStackRuntimeV0: newStoppedPendingAckCodexStackRuntimeV0(),
	}
}

func (runtime *authInvalidPendingAckCodexStackRuntimeV0) LaunchV0(
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
		[]byte("ERROR 401 token_invalidated refresh_token_reused"),
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
