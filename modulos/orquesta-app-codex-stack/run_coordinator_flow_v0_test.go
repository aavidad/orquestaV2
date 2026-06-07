package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackV0ArrancarDirectorRegistraRunEnColaGlobalV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != director.RunRef ||
		ranking.Ranked[0].PriorityScore != DefaultRunQueuePriorityScoreV0 {
		t.Fatalf("ranking=%+v director=%+v", ranking, director)
	}
}

func TestCodexStackV0RunGlobalTickEjecutaRunConMasPrioridadV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	low := postDirectorAPIWithNameV0(t, stack, "app-baja", "Agenda Baja")
	high := postDirectorAPIWithNameV0(t, stack, "app-alta", "Agenda Alta")

	setStackRunPriorityForTestV0(t, stack, low.RunRef, low.AppSpec.Slug, 10)
	setStackRunPriorityForTestV0(t, stack, high.RunRef, high.AppSpec.Slug, 90)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); !reflect.DeepEqual(got, []string{high.RunRef}) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{high.RunRef}, result)
	}
}

func TestCodexStackV0RunGlobalTickResidenteNoBloqueaEnExternalWaiterV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack.Ports.ExternalWaiter = failingCoordinatorExternalWaiterV0{t: t}
	runRef := "request-ref-autoprogramming-resident-no-wait-001"
	taskRef := "task-autoprogramming-resident-no-wait-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef)
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "project-ref-orquesta-server",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 90,
		UpdatedAt:     time.Date(2026, 5, 26, 20, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-resident-no-blocking-wait"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef: DefaultRunQueueRefV0,
		MaxRuns:  1,
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            1,
			MaxStepsPerBurst:     1,
			MaxDispatchesPerWait: 8,
			MaxCommands:          8,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     1,
		},
		OccurredAt: time.Date(2026, 5, 26, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if len(result.Executions) != 1 ||
		result.Executions[0].RunRef != runRef ||
		result.Executions[0].Outcome != string(orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackV0RunGlobalTickRespetaPausaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	paused := postDirectorAPIWithNameV0(t, stack, "app-pausada", "Agenda Pausada")
	next := postDirectorAPIWithNameV0(t, stack, "app-siguiente", "Agenda Siguiente")

	setStackRunPriorityForTestV0(t, stack, paused.RunRef, paused.AppSpec.Slug, 90)
	setStackRunPriorityForTestV0(t, stack, next.RunRef, next.AppSpec.Slug, 20)
	if _, err := stack.Stores.RunControl.PauseRunV0(context.Background(), orquestaruncontrol.PauseRunCommandV0{
		RunRef:      paused.RunRef,
		RequestedBy: "director",
		Reason:      "prueba prioridad pausada",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); !reflect.DeepEqual(got, []string{next.RunRef}) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{next.RunRef}, result)
	}
	if len(result.Skips) != 1 || result.Skips[0].RunRef != paused.RunRef {
		t.Fatalf("skips=%+v", result.Skips)
	}
}

func TestCodexStackV0RunGlobalTickNoReanudaReadyActivoParadoPorShutdownServidorV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIWithNameV0(t, stack, "app-shutdown-requeue", "Agenda Shutdown Requeue")
	setStackRunPriorityForTestV0(t, stack, director.RunRef, director.AppSpec.Slug, 90)
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         director.RunRef,
		TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:    "orquesta-director",
		Reason:         "apagado controlado solicitado por CLI al Director",
		IdempotencyKey: "idem-orquesta-server-stop",
		EvidenceRefs:   []string{"evidence-ref-test-server-shutdown"},
	}); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 0 {
		t.Fatalf("executions got %#v want empty result=%+v", got, result)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("run control reanudado durante shutdown: %+v", state)
	}
}

func TestCodexStackV0RunGlobalTickReanudaShutdownServidorSiAutoResumeExplicitoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIWithNameV0(t, stack, "app-shutdown-autoresume", "Agenda Shutdown Auto Resume")
	setStackRunPriorityForTestV0(t, stack, director.RunRef, director.AppSpec.Slug, 90)
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         director.RunRef,
		TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:    "orquesta-director",
		Reason:         "apagado controlado solicitado por CLI al Director",
		IdempotencyKey: "idem-orquesta-server-stop",
		EvidenceRefs: []string{
			orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0,
			"evidence-ref-test-server-shutdown-autoresume",
		},
	}); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); !reflect.DeepEqual(got, []string{director.RunRef}) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{director.RunRef}, result)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusRunningV0 {
		t.Fatalf("run control no reanudado tras auto-resume explicito: %+v", state)
	}
}

func TestCodexStackV0RunGlobalTickNoResucitaRunParadaPorControlHumanoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIWithNameV0(t, stack, "app-human-stop-requeue", "Agenda Human Stop Requeue")
	setStackRunPriorityForTestV0(t, stack, director.RunRef, director.AppSpec.Slug, 90)
	if _, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       director.RunRef,
		RequestedBy:  "operador-humano",
		Reason:       "parada solicitada desde panel",
		EvidenceRefs: []string{"evidence-ref-test-human-stop"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         director.RunRef,
		TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:    "orquesta-run-control",
		Reason:         "agentes drenados por control de run",
		IdempotencyKey: "idem-run-control-complete-" + codexStackOperationalClosureSafeRefV0(director.RunRef) + "-stopped",
		EvidenceRefs:   []string{"evidence-ref-test-human-stop-complete"},
	}); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 0 {
		t.Fatalf("executions got %#v want empty result=%+v", got, result)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("run control reanudado tras parada humana: %+v", state)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("run parada sigue ejecutable en cola: %+v", visible)
	}
}

func TestCodexStackV0RecoverNoReanudaStoppedSiRetryCerradoExisteV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	baseRef := "request-ref-autoprogramming-backlog-tx-stopped-001"
	retryRef := baseRef + "-retry-closed"
	taskRef := "task-autoprogramming-tx-stopped-001"
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(baseRef, taskRef)
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	markRunControlStoppedAutoResumeForTestV0(t, stack, baseRef)
	setRunQueueCandidateForTestV0(t, stack, baseRef, orquestarunqueue.RunStatusStoppedV0, 60, time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC))
	setRunQueueCandidateForTestV0(t, stack, retryRef, orquestarunqueue.RunStatusClosedV0, 10, time.Date(2026, 5, 27, 11, 0, 0, 0, time.UTC))

	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}

	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: baseRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("base reanudada aunque existe retry cerrado: %+v", state)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("no debe reactivar stopped con retry cerrado: %+v", visible)
	}
}

func TestCodexStackV0RecoverReanudaAppChangeAutoplanBloqueadoSinMicrotareaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	change := postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	refs := appChangeAutoPlanRefsForRecoveryTestV0(t, stack, run)
	run = codexStackRunWithActivePhaseForTestV0(run, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.DirectorAnsweredQuestions = []string{change.DirectorQuestionRef}
	run.DirectorAnswers = nil
	run.Decisions = []string{refs.DecisionRef}
	run.FunctionContracts = nil
	run.Tasks = nil
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-" + refs.TaskDecisionRef,
	}
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 blocked: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		director.RunRef,
		orquestarunqueue.RunStatusReadyV0,
		90,
		time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
	)

	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}
	recovered, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	if recovered.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 || len(recovered.Blockers) != 0 {
		t.Fatalf("run no recuperada: status=%s blockers=%v", recovered.Status, recovered.Blockers)
	}

	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-recover-app-change-autoplan",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	drained, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 drained: %v", err)
	}
	if !codexStackStringInSetForTestV0(drained.Tasks, refs.TaskRef) {
		t.Fatalf("microtarea no materializada: tasks=%v want=%s blockers=%v", drained.Tasks, refs.TaskRef, drained.Blockers)
	}
}

func TestCodexStackV0RecoverNoReanudaAppChangeConBlockerAjenoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	change := postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	refs := appChangeAutoPlanRefsForRecoveryTestV0(t, stack, run)
	run = codexStackRunWithActivePhaseForTestV0(run, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.DirectorAnsweredQuestions = []string{change.DirectorQuestionRef}
	run.Decisions = []string{refs.DecisionRef}
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-" + refs.TaskDecisionRef,
		"security-sensitive-payload-blocked",
	}
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 blocked: %v", err)
	}

	recovered, ok, err := stack.recoverBlockedAppChangeAutoPlanRunV0(
		context.Background(),
		globalTickCommandForTestV0(),
		orquestaruncontrol.DefaultRunControlStateV0(director.RunRef),
		run,
	)
	if err != nil {
		t.Fatalf("recoverBlockedAppChangeAutoPlanRunV0: %v", err)
	}
	if ok || recovered.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("no debe recuperar blockers mixtos: ok=%v recovered=%+v", ok, recovered)
	}
}

func TestCodexStackV0RecoverNoReanudaAppChangeConStopHumanoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	change := postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	refs := appChangeAutoPlanRefsForRecoveryTestV0(t, stack, run)
	run = codexStackRunWithActivePhaseForTestV0(run, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.DirectorAnsweredQuestions = []string{change.DirectorQuestionRef}
	run.Decisions = []string{refs.DecisionRef}
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-" + refs.TaskDecisionRef,
	}
	stopped, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:      director.RunRef,
		RequestedBy: "operador-humano",
		Reason:      "parada manual",
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}

	recovered, ok, err := stack.recoverBlockedAppChangeAutoPlanRunV0(
		context.Background(),
		globalTickCommandForTestV0(),
		stopped,
		run,
	)
	if err != nil {
		t.Fatalf("recoverBlockedAppChangeAutoPlanRunV0: %v", err)
	}
	if ok || recovered.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("no debe recuperar stop humano: ok=%v recovered=%+v", ok, recovered)
	}
}

func TestCodexStackV0RecoverReanudaSoloUltimoStoppedUtilV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	baseRef := "request-ref-autoprogramming-backlog-tx-latest-001"
	retryRef := baseRef + "-retry-latest"
	baseTaskRef := "task-autoprogramming-tx-latest-base-001"
	retryTaskRef := "task-autoprogramming-tx-latest-retry-001"
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), codexStackAutoprogrammingRunForCoordinatorRepairTestV0(baseRef, baseTaskRef)); err != nil {
		t.Fatalf("SaveRunV0 base: %v", err)
	}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), codexStackAutoprogrammingRunForCoordinatorRepairTestV0(retryRef, retryTaskRef)); err != nil {
		t.Fatalf("SaveRunV0 retry: %v", err)
	}
	markRunControlStoppedAutoResumeForTestV0(t, stack, baseRef)
	markRunControlStoppedAutoResumeForTestV0(t, stack, retryRef)
	setRunQueueCandidateForTestV0(t, stack, baseRef, orquestarunqueue.RunStatusStoppedV0, 40, time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC))
	setRunQueueCandidateForTestV0(t, stack, retryRef, orquestarunqueue.RunStatusStoppedV0, 70, time.Date(2026, 5, 27, 11, 0, 0, 0, time.UTC))

	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}

	baseState, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: baseRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 base: %v", err)
	}
	retryState, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: retryRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 retry: %v", err)
	}
	if baseState.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		retryState.Status != orquestaruncontrol.RunControlStatusRunningV0 {
		t.Fatalf("states base=%+v retry=%+v", baseState, retryState)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(visible) != 1 ||
		visible[0].RunRef != retryRef ||
		visible[0].Status != orquestarunqueue.RunStatusReadyV0 {
		t.Fatalf("solo el ultimo stopped util debe quedar ready: %+v", visible)
	}
}

func TestCodexStackV0RecoverReencolaRunEntregadaConStopCheckpointParaDrenarV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIWithNameV0(t, stack, "app-delivered-stop-drain", "Agenda Delivered Stop Drain")
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        director.RunRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        director.AppSpec.Slug,
		Status:        orquestarunqueue.RunStatusDeliveredV0,
		PriorityScore: 80,
		UpdatedAt:     time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-delivered-before-stop"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       director.RunRef,
		RequestedBy:  "operador-humano",
		Reason:       "parada solicitada desde panel",
		EvidenceRefs: []string{"evidence-ref-test-delivered-stop-requested"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.RecordRunCheckpointV0(context.Background(), orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:       director.RunRef,
		RequestedBy:  "orquesta-mcp-run-control",
		Reason:       "checkpoint de stop ya registrado",
		EvidenceRefs: []string{"evidence-ref-test-delivered-stop-checkpoint"},
	}); err != nil {
		t.Fatalf("RecordRunCheckpointV0: %v", err)
	}
	trackedQueue := &trackingStackRunQueuePortV0{inner: stack.Stores.RunQueue}
	stack.Stores.RunQueue = trackedQueue

	command := globalTickCommandForTestV0()
	command.QueueLimit = 1
	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), command); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}
	if trackedQueue.lastRead.Limit != 0 || !trackedQueue.lastRead.IncludeNonExecutable {
		t.Fatalf("reconciliacion debe revisar toda la cola no ejecutable: read=%+v", trackedQueue.lastRead)
	}

	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(visible) != 1 ||
		visible[0].RunRef != director.RunRef ||
		visible[0].Status != orquestarunqueue.RunStatusReadyV0 ||
		!codexStackStringInSetV0(visible[0].EvidenceRefs, "evidence-ref-run-queue-stop-requested-ready-for-drain") {
		t.Fatalf("visible=%+v", visible)
	}
}

func TestCodexStackV0RecoverTerminalizaRunRunningConStopCheckpointSinAgentesPendientesV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "request-ref-autoprogramming-running-stop-no-pending-001"
	taskRef := "task-autoprogramming-running-stop-no-pending-001"
	agentRef := "agent-ref-running-stop-no-pending-001"
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef)
	run.StartedAgents = []string{agentRef}
	run.ConfirmedStoppedAgents = []string{agentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "project-ref-orquesta-server",
		Status:        orquestarunqueue.RunStatusRunningV0,
		PriorityScore: 80,
		UpdatedAt:     time.Date(2026, 5, 27, 14, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-running-before-stop-sync"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "orquesta-director",
		Reason:       "reinicio ordenado",
		EvidenceRefs: []string{"evidence-ref-test-running-stop-requested"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.RecordRunCheckpointV0(context.Background(), orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:       runRef,
		RequestedBy:  "orquesta-mcp-run-control",
		Reason:       "checkpoint de stop ya registrado",
		EvidenceRefs: []string{"evidence-ref-test-running-stop-checkpoint"},
	}); err != nil {
		t.Fatalf("RecordRunCheckpointV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 0 {
		t.Fatalf("run terminalizado por reconciliacion no debe drenarse: executions=%+v result=%+v", got, result)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-control-complete-no-pending-agents") {
		t.Fatalf("state=%+v", state)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("run parado no debe quedar ejecutable: %+v", visible)
	}
	all, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0, IncludeNonExecutable: true},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 all: %v", err)
	}
	if len(all) != 1 ||
		all[0].RunRef != runRef ||
		all[0].Status != orquestarunqueue.RunStatusStoppedV0 ||
		!codexStackStringInSetV0(all[0].EvidenceRefs, "evidence-ref-run-queue-run-control-blocked-reconciled") {
		t.Fatalf("queue all=%+v", all)
	}
}

func TestCodexStackV0RecoverNoTerminalizaStopConDomainWorkPendienteV0(t *testing.T) {
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	stack := mustBuildCodexStackWithDomainWorkForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		&fakeCodexStackDomainWorkExecutorV0{},
	)
	stack.DomainDelivery.Ledger = ledger
	runRef := "request-ref-domain-work-stop-pending-001"
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(context.Background(), DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-domain-work-stop-pending-001",
		Status:         DomainWorkArtifactSubmissionStatusSubmittingV0,
		RunRef:         runRef,
		TaskRef:        "task-ref-domain-work-stop-pending-001",
		DeliveryRef:    "delivery-ref-domain-work-stop-pending-001",
		DomainRef:      "opes",
		JobRef:         "job-ref-domain-work-stop-pending-001",
		ArtifactRef:    "artifact-ref-domain-work-stop-pending-001",
		ArtifactType:   "visual_asset",
	}); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	state, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "orquesta-director",
		Reason:       "reinicio ordenado con domain work pendiente",
		EvidenceRefs: []string{"evidence-ref-test-domain-work-pending-stop"},
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}

	if err := stack.completeQueuedRunControlIfStopHasNoPendingAgentsV0(
		context.Background(),
		globalTickCommandForTestV0(),
		orquestarunqueue.RunSchedulingCandidateV0{
			RunRef:        runRef,
			AppRef:        "project-ref-orquesta-server",
			Status:        orquestarunqueue.RunStatusRunningV0,
			PriorityScore: 80,
			UpdatedAt:     time.Date(2026, 5, 27, 14, 30, 0, 0, time.UTC),
		},
		state,
		orquestacoreworkflow.OrchestrationRunV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
			RunID:         runRef,
			Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		},
	); err != nil {
		t.Fatalf("completeQueuedRunControlIfStopHasNoPendingAgentsV0: %v", err)
	}
	state, err = stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status == orquestaruncontrol.RunControlStatusStoppedV0 ||
		codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-control-complete-no-pending-agents") {
		t.Fatalf("run-control terminalizado con domain work pendiente: %+v", state)
	}
}

func TestCodexStackV0RecoverTerminalizaStopSinRunActivaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "request-ref-autoprogramming-stop-without-active-run-001"
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "project-ref-orquesta-server",
		Status:        orquestarunqueue.RunStatusClosedV0,
		PriorityScore: 0,
		UpdatedAt:     time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-closed-before-stop"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.StopRunV0(context.Background(), orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "operador-humano",
		Reason:       "stop sobre run historica",
		EvidenceRefs: []string{"evidence-ref-test-stop-without-active-run"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.RecordRunCheckpointV0(context.Background(), orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:       runRef,
		RequestedBy:  "orquesta-mcp-run-control",
		Reason:       "checkpoint de stop sobre run historica",
		EvidenceRefs: []string{"evidence-ref-test-stop-without-active-run-checkpoint"},
	}); err != nil {
		t.Fatalf("RecordRunCheckpointV0: %v", err)
	}

	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}

	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-control-complete-without-active-run") {
		t.Fatalf("state=%+v", state)
	}
	all, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0, IncludeNonExecutable: true},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(all) != 1 || all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue debe conservar closed si ya estaba archivada: %+v", all)
	}
}

func TestCodexStackV0RecoverReencolaRunActivoNoEjecutableConTrabajoAbiertoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "request-ref-autoprogramming-requeue-active-001"
	taskRef := "task-autoprogramming-requeue-active-001"
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef)
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "project-ref-orquesta-server",
		Status:        orquestarunqueue.RunStatusCanceledV0,
		PriorityScore: 70,
		UpdatedAt:     time.Date(2026, 5, 25, 16, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-stale-canceled-queue"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	visibleBefore, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 before: %v", err)
	}
	if len(visibleBefore) != 0 {
		t.Fatalf("visibleBefore=%+v", visibleBefore)
	}

	if err := stack.recoverQueuedStoppedActiveRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("recoverQueuedStoppedActiveRunsV0: %v", err)
	}

	visibleAfter, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: DefaultRunQueueRefV0},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 after: %v", err)
	}
	if len(visibleAfter) != 1 ||
		visibleAfter[0].RunRef != runRef ||
		visibleAfter[0].Status != orquestarunqueue.RunStatusReadyV0 ||
		!codexStackStringInSetV0(visibleAfter[0].EvidenceRefs, "evidence-ref-run-queue-non-executable-active-reconciled") {
		t.Fatalf("visibleAfter=%+v", visibleAfter)
	}
}

func TestCodexStackV0RunGlobalTickMarcaRunCerradaComoTerminalEnColaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if len(result.Executions) != 1 ||
		result.Executions[0].RunRef != director.RunRef ||
		result.Executions[0].QueueStatus != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("result=%+v", result)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("ranking=%+v", ranking)
	}
}

func TestCodexStackV0RepairAutoprogrammingNoBloqueaTaskLegacyInmutableV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "request-ref-autoprogramming-legacy-immutable-001"
	taskRef := "task-autoprogramming-legacy-immutable-001"
	task := stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef)
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef)
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(context.Background(), task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}

	err := stack.repairQueuedAutoprogrammingWorkflowTasksV0(context.Background(), runRef)

	if err != nil {
		t.Fatalf("repairQueuedAutoprogrammingWorkflowTasksV0: %v", err)
	}
	stored, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(stored) != 1 || codexStackWorkflowTaskLooksOperationalDirectorV0(stored[0]) {
		t.Fatalf("task legacy no debe mutarse in-place: %+v", stored)
	}
}

func TestCodexStackV0WaitRefsColaReparaTasksEntregadasYAgentesArrancadosPendientesV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "request-ref-autoprogramming-wait-repair-001"
	deliveredTaskRef := "task-autoprogramming-wait-repair-001"
	reworkTaskRef := "task-ref-review-rework-task-autoprogramming-wait-repair-001"
	ghostTaskRef := "task-ref-review-rework-task-autoprogramming-wait-repair-ghost-001"
	deliveredAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(deliveredTaskRef)
	reworkAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(reworkTaskRef)

	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, deliveredTaskRef)
	run.Tasks = []string{deliveredTaskRef, reworkTaskRef, ghostTaskRef}
	run.Agents = []string{deliveredAgentRef, reworkAgentRef}
	run.StartedAgents = []string{deliveredAgentRef, reworkAgentRef}
	run.DeliveredAgents = []string{deliveredAgentRef}
	run.DeliveredTasks = []string{deliveredTaskRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredOperationalTaskForTestV0(runRef, ghostTaskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 ghost: %v", err)
	}

	refs, err := stack.queuedOperationalDirectorWaitAgentRefsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("queuedOperationalDirectorWaitAgentRefsV0: %v", err)
	}
	if !codexStackStringInSetV0(refs, reworkAgentRef) {
		t.Fatalf("wait refs no incluyen agente arrancado pendiente: refs=%v want=%s", refs, reworkAgentRef)
	}
	if codexStackStringInSetV0(refs, deliveredAgentRef) {
		t.Fatalf("wait refs no deben reabrir agente ya entregado: refs=%v delivered=%s", refs, deliveredAgentRef)
	}
	if codexStackStringInSetV0(stackDrainOpenTaskRefsV0(run), deliveredTaskRef) {
		t.Fatalf("open task refs no deben incluir tarea ya entregada: open=%v", stackDrainOpenTaskRefsV0(run))
	}
}

func TestCodexStackV0DrainQueueStatusMarcaRunEntregadaComoTerminalEnColaV0(t *testing.T) {
	taskRef := "task-ref-stack-delivered-queue-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-stack-delivered-queue-001"
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Tasks:           []string{taskRef},
				Agents:          []string{agentRef},
				StartedAgents:   []string{agentRef},
				DeliveredAgents: []string{agentRef},
				Deliveries:      []string{deliveryRef},
				DeliveredTasks:  []string{taskRef},
			},
		},
	}
	if got := stackDrainQueueStatusV0(result); got != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("queue_status=%q want %q", got, orquestarunqueue.RunStatusDeliveredV0)
	}
}

func TestCodexStackV0DrainQueueStatusMantieneActivoArranqueDirectorSinTareasV0(t *testing.T) {
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run: orquestacoreworkflow.OrchestrationRunV0{
				Status:         orquestacoreworkflow.OrchestrationRunStatusActiveV0,
				CurrentPhase:   orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
				StartedAgents:  []string{"agent-ref-stack-director-queue-001"},
				PhaseArtifacts: []string{"artifact-ref-stack-director-queue-001"},
				Decisions:      []string{"decision-ref-stack-director-parcial-001"},
			},
		},
	}
	if got := stackDrainQueueStatusV0(result); got != "" {
		t.Fatalf("queue_status=%q want activo para permitir decisiones tardias", got)
	}
}

func TestCodexStackV0DrainQueueStatusMantieneActivoArranqueDirectorConACKSinArtifactV0(t *testing.T) {
	agentRef := "agent-ref-stack-director-queue-ack-001"
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run: orquestacoreworkflow.OrchestrationRunV0{
				Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
				CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
				StartedAgents:   []string{agentRef},
				DeliveredAgents: []string{agentRef},
			},
		},
	}
	if got := stackDrainQueueStatusV0(result); got != "" {
		t.Fatalf("queue_status=%q want activo con ACK antes de decision file", got)
	}
}

func codexStackAutoprogrammingRunForCoordinatorRepairTestV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	for index := range phases {
		if phases[index].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		}
	}
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             runRef,
		ProjectRef:        "project-ref-orquesta-server",
		AppSpecRef:        codexStackAutoprogrammingAppSpecPrefixV0 + "legacy-immutable",
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:      orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:            phases,
		Tasks:             []string{taskRef},
		FunctionContracts: []string{"BuildAutoprogrammingProgrammableWorkV0"},
	}
}

func postDirectorAPIWithNameV0(
	t *testing.T,
	stack StackV0,
	ref string,
	name string,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	spec := codexStackAppSpecRequestV0()
	spec.RequestID = "request-ref-" + ref
	spec.Nombre = name
	spec.Objetivo = "Gestionar " + name + " con API REST y web."
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:      "request-ref-http-" + ref,
		CorrelationID:  "corr-" + ref,
		AppSpecRequest: spec,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("director status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode director: %v", err)
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 {
		t.Fatalf("director result=%+v", result)
	}
	return result
}

func setStackRunPriorityForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	appRef string,
	priority int,
) {
	t.Helper()
	result := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:        "set_priority",
		QueueRef:      DefaultRunQueueRefV0,
		RunRef:        runRef,
		AppRef:        appRef,
		PriorityScore: priority,
		OccurredAt:    "2026-05-11T12:00:00Z",
		RequestedBy:   "stack-test",
	})
	if result.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		t.Fatalf("priority result=%+v", result)
	}
}

type appChangeAutoPlanRefsForRecoveryV0 struct {
	AnswerRef       string
	DecisionRef     string
	ContractRef     string
	TaskDecisionRef string
	TaskRef         string
}

func appChangeAutoPlanRefsForRecoveryTestV0(
	t *testing.T,
	stack StackV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) appChangeAutoPlanRefsForRecoveryV0 {
	t.Helper()
	decisions, err := (orquestaappchangedirectorsource.AppChangeDirectorDecisionSourceV0{
		Store: stack.Stores.AppChangeStore,
	}).ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	refs := appChangeAutoPlanRefsForRecoveryV0{}
	for _, decision := range decisions {
		switch decision.CommandType {
		case orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0:
			if decision.AnswerQuestion != nil {
				refs.AnswerRef = decision.AnswerQuestion.AnswerID
			}
		case orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0:
			if decision.AcceptDecision != nil {
				refs.DecisionRef = decision.AcceptDecision.DecisionRef
			}
		case orquestadirectoragent.DirectorAgentCommandPublishContractV0:
			if decision.PublishContract != nil {
				refs.ContractRef = decision.PublishContract.ContractRef
			}
		case orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0:
			refs.TaskDecisionRef = decision.DecisionRef
			if decision.CreateMicrotask != nil {
				refs.TaskRef = decision.CreateMicrotask.Task.TaskID
			}
		}
	}
	if refs.AnswerRef == "" ||
		refs.DecisionRef == "" ||
		refs.ContractRef == "" ||
		refs.TaskDecisionRef == "" ||
		refs.TaskRef == "" {
		t.Fatalf("refs incompletas desde decisiones: refs=%+v decisions=%+v", refs, decisions)
	}
	return refs
}

func codexStackRunWithActivePhaseForTestV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	run.CurrentPhase = phase
	hasTarget := false
	hasProgramming := false
	for index := range run.Phases {
		if run.Phases[index].ID == phase {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			hasTarget = true
			continue
		}
		if run.Phases[index].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			hasProgramming = true
		}
		if run.Phases[index].Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
		}
	}
	if !hasTarget {
		run.Phases = append(run.Phases, orquestacoreworkflow.OrchestrationPhaseV0{
			ID:     phase,
			Status: orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		})
	}
	if !hasProgramming {
		run.Phases = append(run.Phases, orquestacoreworkflow.OrchestrationPhaseV0{
			ID:     orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status: orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
		})
	}
	return run
}

func setRunQueueCandidateForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	status string,
	priority int,
	updatedAt time.Time,
) {
	t.Helper()
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(context.Background(), orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "project-ref-orquesta-server",
		Status:        status,
		PriorityScore: priority,
		UpdatedAt:     updatedAt,
		EvidenceRefs:  []string{"evidence-ref-test-queue-candidate"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 %s: %v", runRef, err)
	}
}

func markRunControlStoppedAutoResumeForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) {
	t.Helper()
	if _, err := stack.Stores.RunControl.CompleteRunControlV0(context.Background(), orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         runRef,
		TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:    "orquesta-run-control",
		Reason:         "agentes drenados por control de run",
		IdempotencyKey: "idem-run-control-complete-" + codexStackOperationalClosureSafeRefV0(runRef) + "-stopped",
		EvidenceRefs: []string{
			orquestaruncontrol.RunControlEvidenceAutoResumeAllowedV0,
			"evidence-ref-test-auto-resume",
		},
	}); err != nil {
		t.Fatalf("CompleteRunControlV0 %s: %v", runRef, err)
	}
}

func globalTickCommandForTestV0() orquestaruncoordinator.RunCoordinatorTickCommandV0 {
	now := time.Date(2026, 5, 11, 12, 5, 0, 0, time.UTC)
	return orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef:   DefaultRunQueueRefV0,
		MaxRuns:    1,
		OccurredAt: now,
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            2,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          4,
			MaxOutboxPerCycle:    2,
			MaxDecisionCycles:    1,
			MaxExternalWaits:     1,
		},
	}
}

func executionRefsForStackCoordinatorTestV0(
	result orquestaruncoordinator.RunCoordinatorTickResultV0,
) []string {
	refs := make([]string, 0, len(result.Executions))
	for _, execution := range result.Executions {
		refs = append(refs, execution.RunRef)
	}
	return refs
}

type failingCoordinatorExternalWaiterV0 struct {
	t *testing.T
}

type trackingStackRunQueuePortV0 struct {
	inner    orquestarunqueue.RunQueuePortV0
	lastRead orquestarunqueue.RunQueueReadRequestV0
}

func (queue *trackingStackRunQueuePortV0) ListRunSchedulingCandidatesV0(
	ctx context.Context,
	request orquestarunqueue.RunQueueReadRequestV0,
) ([]orquestarunqueue.RunSchedulingCandidateV0, error) {
	queue.lastRead = request
	return queue.inner.ListRunSchedulingCandidatesV0(ctx, request)
}

func (queue *trackingStackRunQueuePortV0) SetRunPriorityV0(
	ctx context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	return queue.inner.SetRunPriorityV0(ctx, command)
}

func (waiter failingCoordinatorExternalWaiterV0) WaitExternalProgressV0(
	context.Context,
	orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	waiter.t.Fatalf("RunGlobalTickV0 residente no debe bloquear esperando progreso externo")
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{}, nil
}
