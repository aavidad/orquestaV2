package main

import (
	"context"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestDiagnoseStartupV0CompletaStopPendienteSinAgentesYQuitaColaActiva(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	runRef := "run-shutdown-reload-stale-001"
	seedStartupPendingStopForTestV0(t, ctx, store, runRef, orquestaruncontrol.RunControlStatusStopRequestedV0)
	check := startupShutdownReloadCheckForTestV0(store, runRef)

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	assertStartupShutdownReloadClosedForTestV0(t, ctx, store, result, runRef, orquestaruncontrol.RunControlStatusStoppedV0)
}

func TestDiagnoseStartupV0CompletaCancelPendienteSinAgentesYQuitaColaActiva(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	runRef := "run-shutdown-reload-stale-002"
	seedStartupPendingStopForTestV0(t, ctx, store, runRef, orquestaruncontrol.RunControlStatusCancelRequestedV0)
	check := startupShutdownReloadCheckForTestV0(store, runRef)

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 6, 11, 8, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	assertStartupShutdownReloadClosedForTestV0(t, ctx, store, result, runRef, orquestaruncontrol.RunControlStatusCanceledV0)
}

func seedStartupPendingStopForTestV0(
	t *testing.T,
	ctx context.Context,
	store *orquestarunmemory.RunMemoryStoreV0,
	runRef string,
	status orquestaruncontrol.RunControlStatusV0,
) {
	t.Helper()
	if _, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: runRef,
		AppRef: "app-stale",
		Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	if _, err := store.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef: runRef,
		Status: status,
		Forced: true,
	}); err != nil {
		t.Fatalf("PutRunControlStateV0: %v", err)
	}
}

func startupShutdownReloadCheckForTestV0(
	store *orquestarunmemory.RunMemoryStoreV0,
	runRef string,
) serverStartupCheckV0 {
	return serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
					RunID: runRef,
				}),
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}
}

func assertStartupShutdownReloadClosedForTestV0(
	t *testing.T,
	ctx context.Context,
	store *orquestarunmemory.RunMemoryStoreV0,
	result orquestaserver.StartupCheckResultV0,
	runRef string,
	wantStatus orquestaruncontrol.RunControlStatusV0,
) {
	t.Helper()
	if !result.Ready ||
		!resultStartupEvidenceContainsV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-queue-reconciled") {
		t.Fatalf("result=%+v", result)
	}
	state, err := store.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef})
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != wantStatus {
		t.Fatalf("control status=%s want=%s", state.Status, wantStatus)
	}
	listed, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("queue still executable=%+v", listed)
	}
}
