package orquestaappcodexstack

import (
	"context"
	"testing"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
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
