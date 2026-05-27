package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateSinTaskStoreEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-no-task-store"
	planRef := "plan-ref-service-operational-closure-plan-state-no-task-store"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            "task-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-plan-state-no-task-store",
			ClosureRef:        "closure-ref-service-operational-closure-plan-state-no-task-store",
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T11:05:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if source.Called {
		t.Fatalf("closure source no debe invocarse sin task store")
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin task store: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-task-store-unavailable" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-task-store-unavailable") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-task-store-unavailable" ||
		serviceCountStringV0(step.BlockerRefs, "operational-closure-task-store-unavailable") != 1 {
		t.Fatalf("plan state no bloqueado por falta de task store: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateConOutboxPendienteEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-outbox-pending"
	planRef := "plan-ref-service-operational-closure-plan-state-outbox-pending"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            "task-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-plan-state-outbox-pending",
			ClosureRef:        "closure-ref-service-operational-closure-plan-state-outbox-pending",
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:00:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			PendingOutboxCount: 2,
			Run:                run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if source.Called {
		t.Fatalf("closure source no debe invocarse con outbox pendiente")
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar con outbox pendiente: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-outbox-pending" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-outbox-pending") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-outbox-pending" ||
		serviceCountStringV0(step.BlockerRefs, "operational-closure-outbox-pending") != 1 {
		t.Fatalf("plan state no bloqueado por outbox pendiente: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0NoCierraConPlanStatePostWaitActivo(t *testing.T) {
	for _, stepKind := range []orquestadirectoroperativo.OperationalDirectorStepKindV0{
		orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
	} {
		t.Run(string(stepKind), func(t *testing.T) {
			runRef := "run-service-operational-closure-plan-state-" + string(stepKind)
			planRef := "plan-ref-service-operational-closure-plan-state-" + string(stepKind)
			run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
			run.Tasks = []string{"task-ref-service-operational-closure-001"}
			run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
			run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
			waitRefs := []string(nil)
			pendingAgentRefs := []string(nil)
			if stepKind == orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 {
				waitRefs = []string{"wait-ref-service-operational-closure-plan-state"}
				pendingAgentRefs = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")}
			}
			source := &serviceOperationalClosureSourceForTestV0{
				Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
					TaskID:                   "task-ref-service-operational-closure-001",
					DeliveryRef:              "delivery-ref-service-operational-closure-001",
					AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
					ValidationRef:            "validation-ref-service-operational-closure-plan-state",
					ClosureRef:               "closure-ref-service-operational-closure-plan-state",
					RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
					EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
				},
			}
			state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
				SchemaVersion: orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
				StateRef:      "state-ref-service-operational-closure-plan-state-" + string(stepKind),
				PlanRef:       planRef,
				RequestRef:    "request-ref-service-operational-closure-plan-state",
				RunRef:        runRef,
				ProjectRef:    "orquesta",
				Mode:          orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
				Status:        orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
				ActiveStepID:  "step-active-post-wait",
				ObservedAt:    "2026-05-17T14:35:00Z",
				Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
					StepID:           "step-active-post-wait",
					Kind:             stepKind,
					Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
					TaskRefs:         []string{"task-ref-service-operational-closure-001"},
					AgentRefs:        []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
					PendingAgentRefs: pendingAgentRefs,
					WaitRefs:         waitRefs,
				}},
			}
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
			loop, issues, err := maybeCloseOperationalDirectorV0(
				context.Background(),
				ContinueAppDirectorRequestV0{
					RunRef:                     runRef,
					OccurredAt:                 "2026-05-17T14:35:00Z",
					OperationalDirectorPlanRef: planRef,
				},
				StartAppDirectorPortsV0{
					RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
					EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
					EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
					DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
					RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
					OperationalClosureSource:   source,
					OperationalPlanStateStore:  planStateStore,
					OperationalPlanStateWriter: planStateStore,
				},
				orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
					Run:    run,
				},
				orquestacionnucleoapp.ProgressiveLoopRequestV0{},
			)
			if err != nil || len(issues) != 0 {
				t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
			}
			if source.Called {
				t.Fatalf("closure source no debe invocarse con plan state activo en %s", stepKind)
			}
			if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
				t.Fatalf("run cerrado con plan state activo: %+v", loop.Run)
			}
		})
	}
}
