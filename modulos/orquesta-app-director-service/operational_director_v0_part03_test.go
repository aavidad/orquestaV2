package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueOperationalDirectorPlanStatePostLoopV0ConsumeWaitAntesDeExpirarMaxWaitCero(t *testing.T) {
	runRef := "run-app-director-operational-plan-state-max-wait-zero-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	ports := StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	}

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T16:20:00Z",
		CorrelationID:           "corr-app-director-operational-plan-state-max-wait-zero-first",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if len(first.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", first.Run.Tasks)
	}
	initialState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 initial: %v", err)
	}
	initialWaitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, initialState, "step-wait-subagents")
	if len(initialWaitStep.WaitRefs) != 1 {
		t.Fatalf("initial wait refs=%+v", initialWaitStep.WaitRefs)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(first.Run.Tasks[0])
	deliveredRun := first.Run
	deliveredRun.DeliveredAgents = []string{agentRef}
	if err := store.SaveRunV0(context.Background(), deliveredRun); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T16:20:01Z",
		CorrelationID:              "corr-app-director-operational-plan-state-max-wait-zero-post",
		OperationalDirectorPlanRef: plan.PlanRef,
		MaxBursts:                  4,
		MaxStepsPerBurst:           4,
		MaxDispatchesPerWait:       4,
		MaxCommands:                20,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           0,
	}
	_, err = continueOperationalDirectorPlanStatePostLoopV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:                deliveredRun,
			PendingOutboxCount: 1,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef: runRef,
		},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				Run:    deliveredRun,
			},
			Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
				AttemptNumber: 1,
				Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					Run:    deliveredRun,
				},
			}},
		},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0 pending outbox: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 pending outbox: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason == "external-wait-exhausted" {
		t.Fatalf("state avanzo con outbox pendiente: state=%+v wait=%+v", state, waitStep)
	}
	_, err = continueOperationalDirectorPlanStatePostLoopV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    deliveredRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef: runRef,
		},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				Run:    deliveredRun,
			},
			Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
				AttemptNumber: 1,
				Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					Run:    deliveredRun,
				},
			}},
		},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0 delivered: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 final: %v", err)
	}
	waitStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-review-deliveries" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		state.ClosureReason != "" {
		t.Fatalf("state expiro en vez de avanzar: state=%+v wait=%+v review=%+v", state, waitStep, reviewStep)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, initialWaitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 final: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusContinuedV0 ||
		serviceStringInSetV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") {
		t.Fatalf("waitState=%+v", waitState)
	}
}

func TestContinueOperationalDirectorPlanStatePostLoopV0AbreRevisionConReviewYaActivo(t *testing.T) {
	runRef := "run-app-director-operational-plan-state-review-active-reentry"
	planRef := "plan-ref-app-director-operational-plan-state-review-active-reentry"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-service-operational-closure-001"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Tasks = []string{taskRef}
	run.DeliveredTasks = []string{taskRef}
	run.DeliveredAgents = []string{agentRef}
	run.Deliveries = []string{deliveryRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = "step-review-deliveries"
	state.PendingAgentRefs = nil
	state.ClosureReason = ""
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{taskRef}
			state.Steps[index].AgentRefs = []string{agentRef}
			state.Steps[index].DeliveryRefs = []string{deliveryRef}
			state.Steps[index].Reason = "closure-open-tasks-review-followups"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].Reason = ""
			state.Steps[index].BlockerRefs = nil
		}
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	ports := StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		EventReader:                sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	_, err := continueOperationalDirectorPlanStatePostLoopV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T16:21:00Z",
			CorrelationID:              "corr-app-director-operational-plan-state-review-active-reentry",
			OperationalDirectorPlanRef: planRef,
			MaxBursts:                  4,
			MaxStepsPerBurst:           4,
			MaxDispatchesPerWait:       4,
			MaxCommands:                20,
			MaxOutboxPerCycle:          8,
			MaxExternalWaits:           0,
		},
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{RunRef: runRef},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0: %v", err)
	}
	updatedRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if updatedRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		!operationalDirectorRunPhaseActiveV0(updatedRun, orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("run no abrio revision: %+v", updatedRun)
	}
}

func TestEnsureOperationalDirectorReviewPhaseV0ReabreRevisionConNuevoDeliveryMismoIntento(t *testing.T) {
	runRef := "run-app-director-operational-review-reopen-new-delivery"
	planRef := "plan-ref-app-director-operational-review-reopen-new-delivery"
	taskRef := "task-ref-app-director-operational-review-reopen-new-delivery"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-app-director-operational-review-reopen-new-delivery"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	oldSuffix := appDirectorSafeRefPartV0(runRef + "-" + planRef + "-step-review-deliveries-replan-attempt-3")
	oldOpen, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-review-" + oldSuffix,
			RunID:          runRef,
			IdempotencyKey: "idem-open-review-" + oldSuffix,
			CorrelationID:  "corr-old-review-open",
			RequestedBy:    "orquesta-app-codex-stack-drain",
			OccurredAt:     "2026-05-22T16:22:00Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			Reason:  "operational-director-review-deliveries",
		},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0 old review: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(context.Background(), store, sink, oldOpen); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 old review: %v", err)
	}
	reopenProgramacion, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-open-programacion-after-old-review",
			RunID:          runRef,
			IdempotencyKey: "idem-open-programacion-after-old-review",
			CorrelationID:  "corr-programacion-after-old-review",
			RequestedBy:    "orquesta-app-codex-stack-drain",
			OccurredAt:     "2026-05-22T16:22:01Z",
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Reason:  "Reabrir programacion tras retrabajo.",
		},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0 programacion: %v", err)
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(context.Background(), store, sink, reopenProgramacion); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0 programacion: %v", err)
	}
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0
	state.ActiveStepID = "step-review-deliveries"
	state.ActiveWaveRef = ""
	state.ActiveCohortRef = ""
	state.ActiveParentTaskRef = ""
	state.PendingAgentRefs = []string{agentRef}
	state.ReplanAttempts = 3
	state.ClosureReason = ""
	for index := range state.Steps {
		switch state.Steps[index].StepID {
		case "step-review-deliveries":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].WaveRef = ""
			state.Steps[index].CohortRef = ""
			state.Steps[index].ParentTaskRef = ""
			state.Steps[index].TaskRefs = []string{taskRef}
			state.Steps[index].AgentRefs = []string{agentRef}
			state.Steps[index].DeliveryRefs = []string{deliveryRef}
			state.Steps[index].Reason = "closure-open-tasks-review-followups"
		case "step-replan-or-close":
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].Reason = ""
			state.Steps[index].BlockerRefs = nil
		}
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	ports := StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	opened, err := ensureOperationalDirectorReviewPhaseV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T16:22:02Z",
			CorrelationID:              "corr-new-review-open",
			RequestedBy:                "orquesta-app-codex-stack-drain",
			OperationalDirectorPlanRef: planRef,
		},
		ports,
	)
	if err != nil {
		t.Fatalf("ensureOperationalDirectorReviewPhaseV0: %v", err)
	}
	if !opened {
		t.Fatalf("review phase no marcada como abierta")
	}
	updatedRun, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if updatedRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		!operationalDirectorRunPhaseActiveV0(updatedRun, orquestacoreworkflow.OrchestrationPhaseRevisionV0) {
		t.Fatalf("run no reabrio revision con nuevo delivery: %+v", updatedRun)
	}
	if got := serviceCountEventsByTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseOpenedV0); got != 3 {
		t.Fatalf("PhaseOpened got=%d want=3 events=%+v", got, sink.EventsV0())
	}
}
