package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementStackV0RechazaCandidatoNoEjecutableV0(t *testing.T) {
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: "run-ref-1", Status: orquestarunqueue.RunStatusPausedV0,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores:   orquestaappcodexstack.StoresV0{RunQueue: queue},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: "run-ref-1"},
	)
	if result.Accepted || result.Message != "idle_self_improvement_queue_candidate_not_executable" {
		t.Fatalf("result=%+v", result)
	}
	if !queue.lastRead.IncludeNonExecutable {
		t.Fatalf("read request no pidio candidatos no ejecutables: %+v", queue.lastRead)
	}
}

func TestIdleSelfImprovementStackV0AceptaCandidatoCerradoComoAdvisoryV0(t *testing.T) {
	ctx := context.Background()
	runRef := "request-ref-autoprogramming-backlog-t45-demo"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-orquesta",
		Status:        orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseCierreV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: runRef, Status: orquestarunqueue.RunStatusClosedV0,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores: orquestaappcodexstack.StoresV0{
			RunQueue: queue,
			RunStore: runStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		ctx,
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: runRef},
	)
	if !result.Accepted ||
		result.Message != "idle_self_improvement_queue_candidate_already_closed_advisory" ||
		!containsStringForTestV0(result.NextActions, "queue_candidate_closed_after_prepare") ||
		!containsStringForTestV0(result.NextActions, "planner_should_skip_closed_backlog_ref") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-idle-self-improvement-queue-candidate-already-closed") {
		t.Fatalf("result=%+v", result)
	}
	if len(queue.priorityCommands) != 1 ||
		queue.priorityCommands[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("commands=%+v", queue.priorityCommands)
	}
}

func TestIdleSelfImprovementStackV0SincronizaCandidatoReadySiRunYaCerradoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "request-ref-autoprogramming-backlog-t46-ready-cerrado"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-orquesta",
		Status:        orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseCierreV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: runRef, Status: orquestarunqueue.RunStatusReadyV0, PriorityScore: 11,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores: orquestaappcodexstack.StoresV0{
			RunQueue: queue,
			RunStore: runStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		ctx,
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: runRef},
	)
	if !result.Accepted ||
		result.Message != "idle_self_improvement_queue_candidate_already_closed_advisory" ||
		len(queue.priorityCommands) != 1 ||
		queue.priorityCommands[0].Status != orquestarunqueue.RunStatusClosedV0 ||
		queue.priorityCommands[0].PriorityScore != 11 {
		t.Fatalf("result=%+v commands=%+v", result, queue.priorityCommands)
	}
}

func TestIdleSelfImprovementStackV0ReencolaActivoStoppedV0(t *testing.T) {
	ctx := context.Background()
	runRef := "request-ref-autoprogramming-backlog-t47-demo"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-orquesta",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: runRef, Status: orquestarunqueue.RunStatusStoppedV0, PriorityScore: 7,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores: orquestaappcodexstack.StoresV0{
			RunQueue: queue,
			RunStore: runStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		ctx,
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: runRef},
	)
	if !result.Accepted ||
		!containsStringForTestV0(result.NextActions, "queue_candidate_requeued_ready_after_prepare") ||
		len(queue.priorityCommands) != 1 ||
		queue.priorityCommands[0].Status != orquestarunqueue.RunStatusReadyV0 ||
		queue.priorityCommands[0].PriorityScore != 7 {
		t.Fatalf("result=%+v commands=%+v", result, queue.priorityCommands)
	}
}

func TestIdleSelfImprovementStackV0NoReencolaSiRunControlBloqueaV0(t *testing.T) {
	ctx := context.Background()
	runRef := "request-ref-autoprogramming-backlog-t48-control-stopped"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-orquesta",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	control := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := control.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef: runRef,
		Status: orquestaruncontrol.RunControlStatusStoppedV0,
	}); err != nil {
		t.Fatalf("PutRunControlStateV0: %v", err)
	}
	queue := &fakeIdleSelfRunQueueV0{candidates: []orquestarunqueue.RunSchedulingCandidateV0{{
		RunRef: runRef, Status: orquestarunqueue.RunStatusStoppedV0, PriorityScore: 7,
	}}}
	stack := &orquestaappcodexstack.StackV0{
		Stores: orquestaappcodexstack.StoresV0{
			RunQueue:   queue,
			RunStore:   runStore,
			RunControl: control,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		ctx,
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: runRef},
	)
	if result.Accepted ||
		result.Message != "idle_self_improvement_queue_candidate_control_blocked" ||
		!containsStringForTestV0(result.NextActions, "run_control_status=stopped") ||
		!containsStringForTestV0(result.NextActions, "planner_should_skip_control_blocked_backlog_ref") ||
		len(queue.priorityCommands) != 0 {
		t.Fatalf("result=%+v commands=%+v", result, queue.priorityCommands)
	}
}

func TestIdleSelfImprovementStackV0NoBloqueaSiCandidatoNoVisiblePeroRunPersistidoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-prepared-closed-fast"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-orquesta",
		AppSpecRef:    "app-spec-ref-orquesta",
		Status:        orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseCierreV0,
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	stack := &orquestaappcodexstack.StackV0{
		Stores: orquestaappcodexstack.StoresV0{
			RunQueue: &fakeIdleSelfRunQueueV0{},
			RunStore: runStore,
		},
		RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
	}
	result := (serverStackSupervisorV0{stack: stack}).ensureIdleSelfImprovementQueueVisibleV0(
		ctx,
		orquestaserver.IdleSelfImprovementResultV0{Accepted: true, RunRef: runRef},
	)
	if !result.Accepted ||
		result.Message != "idle_self_improvement_queue_candidate_not_visible_advisory" ||
		!containsStringForTestV0(result.NextActions, "prepared_run_persisted_not_scheduling_visible") ||
		!containsStringForTestV0(result.EvidenceRefs, "evidence-ref-idle-self-improvement-run-persisted") {
		t.Fatalf("result=%+v", result)
	}
}
