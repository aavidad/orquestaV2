package orquestaappcodexstack

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackV0RunGlobalTickRetiraColaRunningSinRunStoreV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	runRef := "run-orphan-queue-running-001"
	setRunQueueCandidateForTestV0(
		t,
		stack,
		runRef,
		orquestarunqueue.RunStatusRunningV0,
		90,
		time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
	)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if len(result.Executions) != 0 || runtime.launchCountV0() != 0 {
		t.Fatalf("cola huerfana no debe drenarse ni lanzar agentes: result=%+v launches=%d", result, runtime.launchCountV0())
	}
	candidate := mustQueueCandidateForTestV0(t, stack, runRef)
	if candidate.Status != orquestarunqueue.RunStatusStoppedV0 ||
		!codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-orphan-executable-auto-retired") ||
		!codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-store-missing") {
		t.Fatalf("candidate=%+v", candidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-queue-orphan-executable-auto-retired") {
		t.Fatalf("state=%+v", state)
	}
}

func TestCodexStackV0ReconciliacionNoRetiraReadyConRunStoreV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-ready-with-store-001"
	taskRef := "task-ready-with-store-001"
	if err := stack.Ports.RunStore.SaveRunV0(
		context.Background(),
		codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef),
	); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		runRef,
		orquestarunqueue.RunStatusReadyV0,
		70,
		time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
	)

	if err := stack.reconcileQueuedOrphanExecutableRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("reconcileQueuedOrphanExecutableRunsV0: %v", err)
	}
	candidate := mustQueueCandidateForTestV0(t, stack, runRef)
	if candidate.Status != orquestarunqueue.RunStatusReadyV0 ||
		codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-orphan-executable-auto-retired") {
		t.Fatalf("candidate=%+v", candidate)
	}
}

func TestCodexStackV0RunGlobalTickReconciliaRunningStaleAntesDeReadyV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	staleRunRef := "run-running-stale-no-live-001"
	staleTaskRef := "task-running-stale-no-live-001"
	staleAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(staleTaskRef)
	staleRun := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(staleRunRef, staleTaskRef)
	staleRun.ProjectRef = "opes"
	staleRun.AppSpecRef = "app-spec-external-work-opes"
	staleRun.Agents = []string{staleAgentRef}
	staleRun.StartedAgents = []string{staleAgentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), staleRun); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredAutoprogrammingTaskForTestV0(staleRunRef, staleTaskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 stale: %v", err)
	}
	processRef := "process-ref-running-stale-no-live-001"
	runtime.snapshots[processRef] = orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-running-stale-no-live-001",
		LaunchRef:     "launch-ref-running-stale-no-live-001",
		Status:        orquestaruntime.ProcessRuntimeStoppedV0,
	}
	if err := stack.Stores.ProcessRegistry.RecordAgentProcessV0(
		context.Background(),
		orquestacionnucleoapp.AgentProcessRegistryRecordV0{
			RunID:          staleRunRef,
			AgentRequestID: staleAgentRef,
			ProcessRef:     processRef,
			SessionRef:     "session-ref-running-stale-no-live-001",
			LaunchRef:      "launch-ref-running-stale-no-live-001",
			ReadinessRef:   "readiness-ref-running-stale-no-live-001",
			EvidenceRefs:   []string{"evidence-ref-test-running-stale-no-live"},
		},
	); err != nil {
		t.Fatalf("RecordAgentProcessV0 stale: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		staleRunRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 26, 0, 20, 0, 0, time.UTC),
	)
	ready := postDirectorAPIWithNameV0(t, stack, "ready-after-running-stale", "Ready After Running Stale")
	setStackRunPriorityForTestV0(t, stack, ready.RunRef, ready.AppSpec.Slug, 10)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 1 || got[0] != ready.RunRef {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{ready.RunRef}, result)
	}
	staleCandidate := mustQueueCandidateForTestV0(t, stack, staleRunRef)
	if staleCandidate.Status != orquestarunqueue.RunStatusStoppedV0 ||
		!codexStackStringInSetV0(staleCandidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") ||
		!codexStackStringInSetV0(staleCandidate.EvidenceRefs, "evidence-ref-external-work-public-status-not-running-stale-no-process") {
		t.Fatalf("staleCandidate=%+v", staleCandidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: staleRunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 stale: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-control-running-stale-no-live-process") ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-external-work-public-status-not-running-stale-no-process") {
		t.Fatalf("state=%+v", state)
	}
	latest := mustLoadCodexStackRunForTestV0(t, stack, staleRunRef)
	if !codexStackStringInSetV0(latest.LostAgents, staleAgentRef) {
		t.Fatalf("run stale no reconciliada como lost: lost=%v assessments=%v", latest.LostAgents, latest.AgentAssessments)
	}
}

func TestCodexStackV0RunGlobalTickReconciliaRunningStaleGenericoConProcesoParadoV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	staleRunRef := "run-generic-running-stale-no-live-001"
	staleTaskRef := "task-generic-running-stale-no-live-001"
	staleAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(staleTaskRef)
	staleRun := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(staleRunRef, staleTaskRef)
	staleRun.AppSpecRef = "app-spec-ref-generic-running-stale-no-live"
	staleRun.Tasks = nil
	staleRun.FunctionContracts = nil
	staleRun.Agents = []string{staleAgentRef}
	staleRun.StartedAgents = []string{staleAgentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), staleRun); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	processRef := "process-ref-generic-running-stale-no-live-001"
	runtime.snapshots[processRef] = orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-generic-running-stale-no-live-001",
		LaunchRef:     "launch-ref-generic-running-stale-no-live-001",
		Status:        orquestaruntime.ProcessRuntimeStoppedV0,
	}
	if err := stack.Stores.ProcessRegistry.RecordAgentProcessV0(
		context.Background(),
		orquestacionnucleoapp.AgentProcessRegistryRecordV0{
			RunID:          staleRunRef,
			AgentRequestID: staleAgentRef,
			ProcessRef:     processRef,
			SessionRef:     "session-ref-generic-running-stale-no-live-001",
			LaunchRef:      "launch-ref-generic-running-stale-no-live-001",
			ReadinessRef:   "readiness-ref-generic-running-stale-no-live-001",
			EvidenceRefs:   []string{"evidence-ref-test-generic-running-stale-no-live"},
		},
	); err != nil {
		t.Fatalf("RecordAgentProcessV0 stale: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		staleRunRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC),
	)
	ready := postDirectorAPIWithNameV0(t, stack, "ready-after-generic-running-stale", "Ready After Generic Running Stale")
	setStackRunPriorityForTestV0(t, stack, ready.RunRef, ready.AppSpec.Slug, 10)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 1 || got[0] != ready.RunRef {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{ready.RunRef}, result)
	}
	staleCandidate := mustQueueCandidateForTestV0(t, stack, staleRunRef)
	if staleCandidate.Status != orquestarunqueue.RunStatusStoppedV0 ||
		!codexStackStringInSetV0(staleCandidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") {
		t.Fatalf("staleCandidate=%+v", staleCandidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: staleRunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 stale: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-run-control-running-stale-no-live-process") {
		t.Fatalf("state=%+v", state)
	}
	latest := mustLoadCodexStackRunForTestV0(t, stack, staleRunRef)
	if !codexStackStringInSetV0(latest.LostAgents, staleAgentRef) {
		t.Fatalf("run stale generica no reconciliada como lost: lost=%v assessments=%v", latest.LostAgents, latest.AgentAssessments)
	}
}

func TestCodexStackV0NoParaRunningGenericoSinRegistroProcesoV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	staleRunRef := "run-generic-running-no-process-001"
	staleTaskRef := "task-generic-running-no-process-001"
	staleAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(staleTaskRef)
	staleRun := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(staleRunRef, staleTaskRef)
	staleRun.Agents = []string{staleAgentRef}
	staleRun.StartedAgents = []string{staleAgentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), staleRun); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredAutoprogrammingTaskForTestV0(staleRunRef, staleTaskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 stale: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		staleRunRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 28, 9, 10, 0, 0, time.UTC),
	)

	if err := stack.reconcileQueuedRunningStaleRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("reconcileQueuedRunningStaleRunsV0: %v", err)
	}
	candidate := mustQueueCandidateForTestV0(t, stack, staleRunRef)
	if candidate.Status != orquestarunqueue.RunStatusRunningV0 ||
		codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") {
		t.Fatalf("candidate=%+v", candidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: staleRunRef},
	)
	if err == nil && state.Status == orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("state stopped sin registro de proceso: %+v", state)
	}
}

func TestCodexStackV0NoReconciliaRunningGenericoConProcessRefNoEncontradoV0(t *testing.T) {
	runtime := missingSnapshotCodexStackRuntimeForTestV0{fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0()}
	stack := mustBuildCodexStackForTestV0(t, runtime)
	runRef := "run-generic-running-process-ref-missing-001"
	taskRef := "task-generic-running-process-ref-missing-001"
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
	if err := stack.Stores.ProcessRegistry.RecordAgentProcessV0(
		context.Background(),
		orquestacionnucleoapp.AgentProcessRegistryRecordV0{
			RunID:          runRef,
			AgentRequestID: agentRef,
			ProcessRef:     "process-ref-generic-running-process-ref-missing-001",
			SessionRef:     "session-ref-generic-running-process-ref-missing-001",
			LaunchRef:      "launch-ref-generic-running-process-ref-missing-001",
			ReadinessRef:   "readiness-ref-generic-running-process-ref-missing-001",
			EvidenceRefs:   []string{"evidence-ref-process-registry-present"},
		},
	); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		runRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 28, 9, 20, 0, 0, time.UTC),
	)

	liveness, err := stack.queuedRunningStaleProcessLivenessV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("queuedRunningStaleProcessLivenessV0: %v", err)
	}
	if liveness.Verifiable || liveness.Live || liveness.RecordCount != 1 {
		t.Fatalf("liveness=%+v", liveness)
	}
	if err := stack.reconcileQueuedRunningStaleRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("reconcileQueuedRunningStaleRunsV0: %v", err)
	}
	candidate := mustQueueCandidateForTestV0(t, stack, runRef)
	if candidate.Status != orquestarunqueue.RunStatusRunningV0 ||
		codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") {
		t.Fatalf("candidate=%+v", candidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil && state.Status == orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("state stopped con snapshot no verificable: %+v", state)
	}
}

func TestCodexStackV0NoParaOPESRunningSinRegistroProcesoNiEvidenciaTerminalV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	staleRunRef := "run-opes-running-no-process-no-terminal-001"
	staleTaskRef := "task-opes-running-no-process-no-terminal-001"
	staleAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(staleTaskRef)
	staleRun := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(staleRunRef, staleTaskRef)
	staleRun.ProjectRef = "opes"
	staleRun.AppSpecRef = "app-spec-external-work-opes"
	staleRun.Agents = []string{staleAgentRef}
	staleRun.StartedAgents = []string{staleAgentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), staleRun); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredAutoprogrammingTaskForTestV0(staleRunRef, staleTaskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 stale: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		staleRunRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 26, 2, 40, 0, 0, time.UTC),
	)

	if err := stack.reconcileQueuedRunningStaleRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("reconcileQueuedRunningStaleRunsV0: %v", err)
	}
	candidate := mustQueueCandidateForTestV0(t, stack, staleRunRef)
	if candidate.Status != orquestarunqueue.RunStatusRunningV0 ||
		codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") {
		t.Fatalf("candidate=%+v", candidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: staleRunRef},
	)
	if err == nil && state.Status == orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("state stopped sin evidencia terminal: %+v", state)
	}
}

func TestCodexStackV0NoReconciliaOPESRunningConProcessRefNoEncontradoV0(t *testing.T) {
	runtime := missingSnapshotCodexStackRuntimeForTestV0{fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0()}
	stack := mustBuildCodexStackForTestV0(t, runtime)
	runRef := "run-opes-running-process-ref-missing-001"
	taskRef := "task-opes-running-process-ref-missing-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(runRef, taskRef)
	run.ProjectRef = "opes"
	run.AppSpecRef = "app-spec-external-work-opes"
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
	if err := stack.Stores.ProcessRegistry.RecordAgentProcessV0(
		context.Background(),
		orquestacionnucleoapp.AgentProcessRegistryRecordV0{
			RunID:          runRef,
			AgentRequestID: agentRef,
			ProcessRef:     "process-ref-opes-running-process-ref-missing-001",
			SessionRef:     "session-ref-opes-running-process-ref-missing-001",
			LaunchRef:      "launch-ref-opes-running-process-ref-missing-001",
			ReadinessRef:   "readiness-ref-opes-running-process-ref-missing-001",
			EvidenceRefs:   []string{"evidence-ref-process-registry-present"},
		},
	); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		runRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC),
	)

	liveness, err := stack.queuedRunningStaleProcessLivenessV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("queuedRunningStaleProcessLivenessV0: %v", err)
	}
	if liveness.Verifiable || liveness.Live || liveness.RecordCount != 1 {
		t.Fatalf("liveness=%+v", liveness)
	}
	if err := stack.reconcileQueuedRunningStaleRunsV0(context.Background(), globalTickCommandForTestV0()); err != nil {
		t.Fatalf("reconcileQueuedRunningStaleRunsV0: %v", err)
	}
	candidate := mustQueueCandidateForTestV0(t, stack, runRef)
	if candidate.Status != orquestarunqueue.RunStatusRunningV0 ||
		codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-run-queue-running-stale-no-live-process-reconciled") {
		t.Fatalf("candidate=%+v", candidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil && state.Status == orquestaruncontrol.RunControlStatusStoppedV0 {
		t.Fatalf("state stopped con snapshot no verificable: %+v", state)
	}
}

func TestCodexStackV0RunGlobalTickReconciliaOPESUsageLimitSinProcesoRegistradoV0(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	staleRunRef := "run-opes-usage-limit-no-process-001"
	staleTaskRef := "task-opes-usage-limit-no-process-001"
	staleAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(staleTaskRef)
	staleRun := codexStackAutoprogrammingRunForCoordinatorRepairTestV0(staleRunRef, staleTaskRef)
	staleRun.ProjectRef = "opes"
	staleRun.AppSpecRef = "app-spec-external-work-opes"
	staleRun.Agents = []string{staleAgentRef}
	staleRun.StartedAgents = []string{staleAgentRef}
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), staleRun); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if err := stack.Ports.DirectorTaskStore.SaveWorkflowTaskV0(
		context.Background(),
		stackDeliveredAutoprogrammingTaskForTestV0(staleRunRef, staleTaskRef),
	); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 stale: %v", err)
	}
	runtimeDir, err := orquestaruntimecodexdelivery.CodexReceiptAgentRuntimeDirV0(
		stack.CodexRuntimeWorkDir,
		staleRunRef,
		staleAgentRef,
	)
	if err != nil {
		t.Fatalf("CodexReceiptAgentRuntimeDirV0: %v", err)
	}
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll runtimeDir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStderrFileNameV0),
		[]byte("ERROR: You've hit your usage limit. Visit settings/usage to purchase more credits or try again later.\n"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile stderr: %v", err)
	}
	if err := stack.Stores.ReceiptStore.RecordCodexReceiptDescriptorV0(
		context.Background(),
		orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
			RunID:    staleRunRef,
			AgentRef: staleAgentRef,
			Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
				RequestID: staleAgentRef,
				AgentPacket: orquestaruntime.AgentStartPacketV0{
					RequestID: staleAgentRef,
					Task: orquestaruntime.AgentStartTaskV0{
						TaskRef: staleTaskRef,
					},
				},
			},
			AckPath: filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
	); err != nil {
		t.Fatalf("RecordCodexReceiptDescriptorV0 stale: %v", err)
	}
	setRunQueueCandidateForTestV0(
		t,
		stack,
		staleRunRef,
		orquestarunqueue.RunStatusRunningV0,
		100,
		time.Date(2026, 6, 26, 2, 50, 0, 0, time.UTC),
	)
	ready := postDirectorAPIWithNameV0(t, stack, "ready-after-usage-limit", "Ready After Usage Limit")
	setStackRunPriorityForTestV0(t, stack, ready.RunRef, ready.AppSpec.Slug, 10)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); len(got) != 1 || got[0] != ready.RunRef {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{ready.RunRef}, result)
	}
	staleCandidate := mustQueueCandidateForTestV0(t, stack, staleRunRef)
	if staleCandidate.Status != orquestarunqueue.RunStatusStoppedV0 ||
		!codexStackStringInSetV0(staleCandidate.EvidenceRefs, "evidence-ref-provider-usage-limit-retry-after") ||
		!codexStackStringInSetV0(staleCandidate.EvidenceRefs, "evidence-ref-codex-usage-quota-exhausted") {
		t.Fatalf("staleCandidate=%+v", staleCandidate)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: staleRunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0 stale: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusStoppedV0 ||
		state.Meta.Reason != "provider_usage_limit_retry_after" ||
		!codexStackStringInSetV0(state.EvidenceRefs, "evidence-ref-provider-usage-limit-retry-after") {
		t.Fatalf("state=%+v", state)
	}
	latest := mustLoadCodexStackRunForTestV0(t, stack, staleRunRef)
	if !codexStackStringInSetV0(latest.LostAgents, staleAgentRef) {
		t.Fatalf("run stale no reconciliada como lost: lost=%v assessments=%v", latest.LostAgents, latest.AgentAssessments)
	}
}

type missingSnapshotCodexStackRuntimeForTestV0 struct {
	*fakeCodexStackRuntimeV0
}

func (runtime missingSnapshotCodexStackRuntimeForTestV0) SnapshotV0(
	processRef string,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	return orquestaruntime.ProcessRuntimeSnapshotV0{}, orquestaruntime.ProcessRuntimeErrorV0{
		Code:       orquestaruntime.ProcessRuntimeNoEncontradoV0,
		MessageKey: "orquesta.runtime.process.process_runtime_no_encontrado",
		Field:      "process_ref",
	}
}

func mustQueueCandidateForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
) orquestarunqueue.RunSchedulingCandidateV0 {
	t.Helper()
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             DefaultRunQueueRefV0,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	for _, candidate := range candidates {
		if candidate.RunRef == runRef {
			return candidate
		}
	}
	t.Fatalf("candidate %s no encontrado en %+v", runRef, candidates)
	return orquestarunqueue.RunSchedulingCandidateV0{}
}
