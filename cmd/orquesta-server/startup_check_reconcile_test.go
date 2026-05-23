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

func TestDiagnoseStartupV0ReconciliaColaTerminalSinForcedStop(t *testing.T) {
	ctx := context.Background()
	store := orquestarunmemory.NewRunMemoryStoreV0()
	_, err := store.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: "run-terminal-queue-ready",
		AppRef: "app-001",
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	_, err = store.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-terminal-queue-ready",
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:  "test",
	})
	if err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunQueue:   store,
				RunControl: store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{
				QueueRef: "queue-main",
			},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready ||
		!resultStartupEvidenceContainsV0(result.EvidenceRefs, "evidence-ref-orquesta-startup-queue-reconciled") {
		t.Fatalf("result=%+v", result)
	}
	listed, err := store.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("queue still executable=%+v", listed)
	}
}

func TestDiagnoseStartupV0ReconciliaRunCerradaEnColaReady(t *testing.T) {
	ctx := context.Background()
	queueControl := orquestarunmemory.NewRunMemoryStoreV0()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:  "run-closed-queue-ready",
		Status: orquestacoreworkflow.OrchestrationRunStatusClosedV0,
	})
	_, err := queueControl.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: "run-closed-queue-ready",
		AppRef: "app-001",
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunStore:   runStore,
				RunQueue:   queueControl,
				RunControl: queueControl,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 5, 23, 11, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready {
		t.Fatalf("result=%+v", result)
	}
	listed, err := queueControl.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("queue still executable=%+v", listed)
	}
}

func TestDiagnoseStartupV0ReconciliaRunEntregadaEnColaReady(t *testing.T) {
	ctx := context.Background()
	queueControl := orquestarunmemory.NewRunMemoryStoreV0()
	taskRef := "task-ref-startup-delivered-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:           "run-delivered-queue-ready",
		Tasks:           []string{taskRef},
		Agents:          []string{agentRef},
		StartedAgents:   []string{agentRef},
		DeliveredAgents: []string{agentRef},
		Deliveries:      []string{"delivery-ref-startup-delivered-001"},
		DeliveredTasks:  []string{taskRef},
	})
	_, err := queueControl.UpsertRunSchedulingCandidateV0(ctx, "queue-main", orquestarunqueue.RunSchedulingCandidateV0{
		RunRef: "run-delivered-queue-ready",
		AppRef: "app-001",
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpsertRunSchedulingCandidateV0: %v", err)
	}
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunStore:   runStore,
				RunQueue:   queueControl,
				RunControl: queueControl,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
		Mode: startupCleanupModeDiagnoseV0,
	}

	result, err := check.PrepareStartupV0(ctx, orquestaserver.StartupCheckCommandV0{
		OccurredAt: time.Date(2026, 5, 23, 11, 7, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PrepareStartupV0: %v", err)
	}
	if !result.Ready {
		t.Fatalf("result=%+v", result)
	}
	listed, err := queueControl.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("queue still executable=%+v", listed)
	}
}
