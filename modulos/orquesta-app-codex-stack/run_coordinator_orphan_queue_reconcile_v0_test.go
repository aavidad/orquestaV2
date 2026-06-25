package orquestaappcodexstack

import (
	"context"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
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
		t.Fatalf("run stale no reconciliada como lost: lost=%v assessments=%v", latest.LostAgents, latest.AgentAssessments)
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
