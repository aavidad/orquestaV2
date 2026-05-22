package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestContinueAppDirectorV0MaterializaOperationalDirectorPlanYEsperaOla(t *testing.T) {
	runRef := "run-app-director-operational-plan-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:45:00Z",
		CorrelationID:           "corr-app-director-operational-plan-001",
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
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", result.LoopStatus, result)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	taskRef := result.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
		t.Fatalf("started=%v want=%s", result.Run.StartedAgents, agentRef)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || tasks[0].WaveRef == "" || tasks[0].CohortRef == "" {
		t.Fatalf("operational task sin metadata: %+v", tasks)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: tasks[0].WaveRef, CohortRef: tasks[0].CohortRef},
		"corr-app-director-operational-plan-001",
	)
	state, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(state.AgentRefs, agentRef) ||
		!serviceStringInSetV0(state.PendingAgentRefs, agentRef) {
		t.Fatalf("wait state=%+v agent=%s", state, agentRef)
	}
	planState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if planState.ActiveStepID != "step-wait-subagents" ||
		planState.ActiveWaveRef != tasks[0].WaveRef ||
		!serviceStringInSetV0(planState.PendingAgentRefs, agentRef) ||
		len(planState.Steps) == 0 {
		t.Fatalf("plan state=%+v", planState)
	}
	var waitStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0
	for _, step := range planState.Steps {
		if step.StepID == planState.ActiveStepID {
			waitStep = step
			break
		}
	}
	if !serviceStringInSetV0(waitStep.WaitRefs, waitRef) {
		t.Fatalf("plan state wait refs=%+v want=%s", waitStep.WaitRefs, waitRef)
	}
}

func TestContinueAppDirectorV0MaterializaAgentesParalelosDelPlanOperativo(t *testing.T) {
	runRef := "run-app-director-operational-plan-parallel-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanWithParallelLaunchesForTestV0(t, runRef, 3)

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T21:25:00Z",
		CorrelationID:           "corr-app-director-operational-plan-parallel-001",
		MaxBursts:               6,
		MaxStepsPerBurst:        6,
		MaxDispatchesPerWait:    6,
		MaxCommands:             30,
		MaxOutboxPerCycle:       12,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		len(result.Run.Tasks) != 3 {
		t.Fatalf("result=%+v", result)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, result.Run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	waveRef := tasks[0].WaveRef
	cohortRef := tasks[0].CohortRef
	for _, task := range tasks {
		agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID)
		if task.WaveRef != waveRef ||
			task.CohortRef != cohortRef ||
			!serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
			t.Fatalf("task=%+v started=%v", task, result.Run.StartedAgents)
		}
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: waveRef, CohortRef: cohortRef},
		"corr-app-director-operational-plan-parallel-001",
	)
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if len(waitState.PendingAgentRefs) != 3 {
		t.Fatalf("wait state=%+v", waitState)
	}
	planState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if planState.ActiveStepID != "step-wait-subagents" ||
		len(planState.PendingAgentRefs) != 3 ||
		planState.ActiveWaveRef != waveRef ||
		planState.ActiveCohortRef != cohortRef {
		t.Fatalf("plan state=%+v", planState)
	}
}

func TestContinueAppDirectorV0ReentraDesdeOperationalDirectorPlanState(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	dispatchers := []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		serviceCapacityDispatcherForTestV0(store, sink, ledger),
		serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
	}

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:55:00Z",
		CorrelationID:           "corr-app-director-operational-plan-reentry-first",
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
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers:                dispatchers,
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if len(first.Run.Tasks) != 1 {
		t.Fatalf("first tasks=%v", first.Run.Tasks)
	}
	taskRef := first.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}

	second, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T13:56:00Z",
		CorrelationID:              "corr-app-director-operational-plan-reentry-second",
		MaxBursts:                  2,
		MaxStepsPerBurst:           2,
		MaxDispatchesPerWait:       2,
		MaxCommands:                10,
		MaxOutboxPerCycle:          4,
		OperationalDirectorPlanRef: plan.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  store,
		EventSink:                 sink,
		OutboxLedger:              ledger,
		DirectorTaskStore:         taskStore,
		WaitStateWriter:           waitStore,
		OperationalPlanStateStore: planStateStore,
		Dispatchers:               dispatchers,
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 second: %v", err)
	}
	if second.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", second.LoopStatus, second)
	}
	if len(second.Run.Tasks) != 1 || second.Run.Tasks[0] != taskRef {
		t.Fatalf("second tasks=%v want=%s", second.Run.Tasks, taskRef)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: tasks[0].WaveRef, CohortRef: tasks[0].CohortRef},
		"corr-app-director-operational-plan-reentry-second",
	)
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(waitState.AgentRefs, agentRef) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, agentRef) {
		t.Fatalf("wait state from plan state=%+v agent=%s", waitState, agentRef)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaAReviewTrasWaitConsumido(t *testing.T) {
	runRef := "run-app-director-operational-plan-state-review-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T14:05:00Z",
		CorrelationID:           "corr-app-director-operational-plan-state-review-first",
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
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		OperationalPlanStateStore:  planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
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
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T14:06:00Z",
		OperationalDirectorPlanRef: plan.PlanRef,
	}, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" || len(state.PendingAgentRefs) != 0 {
		t.Fatalf("state=%+v", state)
	}
	var waitAccepted, reviewRunning bool
	for _, step := range state.Steps {
		switch step.StepID {
		case "step-wait-subagents":
			waitAccepted = step.Status == orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 &&
				len(step.PendingAgentRefs) == 0
		case "step-review-deliveries":
			reviewRunning = step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
				serviceStringInSetV0(step.AgentRefs, agentRef)
		}
	}
	if !waitAccepted || !reviewRunning {
		t.Fatalf("state=%+v waitAccepted=%v reviewRunning=%v", state, waitAccepted, reviewRunning)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, initialWaitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusContinuedV0 ||
		len(waitState.PendingAgentRefs) != 0 ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-continued-v0") != 1 {
		t.Fatalf("waitState=%+v", waitState)
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T14:06:01Z",
		OperationalDirectorPlanRef: plan.PlanRef,
	}, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}
	replayedWaitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, initialWaitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 replay: %v", err)
	}
	if replayedWaitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusContinuedV0 ||
		serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-continued-v0") != 1 {
		t.Fatalf("replayedWaitState=%+v", replayedWaitState)
	}
}

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

func TestContinueAppDirectorV0BloqueaWaitSubagentsCuandoManagedWaitExpira(t *testing.T) {
	runRef := "run-app-director-operational-wait-expired-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	waiter := &serviceContinueWaiterForTestV0{Continue: true}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T16:10:00Z",
		CorrelationID:           "corr-app-director-operational-wait-expired-001",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		MaxExternalWaits:        2,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		ExternalWaiter:             waiter,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		waiter.Calls != 2 {
		t.Fatalf("result=%+v waiter_calls=%d", result, waiter.Calls)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		!serviceStringInSetV0(state.BlockerRefs, "external-wait-exhausted") ||
		state.ClosureReason != "external-wait-exhausted" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason != "external-wait-exhausted" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "external-wait-exhausted") ||
		len(waitStep.PendingAgentRefs) != 0 ||
		len(state.PendingAgentRefs) != 0 {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
	if len(waitStep.WaitRefs) != 1 {
		t.Fatalf("wait refs=%+v", waitStep.WaitRefs)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusExpiredV0 ||
		waitState.Attempt != 3 ||
		len(waitState.PendingAgentRefs) != 0 ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("waitState=%+v", waitState)
	}
	changed, err := applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T16:10:01Z",
		OperationalDirectorPlanRef: plan.PlanRef,
		MaxExternalWaits:           2,
	}, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
	}, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		Final:  orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0, Run: result.Run},
		Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{
			{AttemptNumber: 1, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
			{AttemptNumber: 2, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
			{AttemptNumber: 3, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
		},
		ExternalWaits: []orquestacionnucleoapp.ManagedProgressiveLoopExternalWaitV0{
			{WaitNumber: 1, Continue: true},
			{WaitNumber: 2, Continue: true},
		},
	})
	if err != nil {
		t.Fatalf("applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0 replay: %v", err)
	}
	if changed {
		t.Fatalf("replay changed blocked state")
	}
	replayedWaitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 replay: %v", err)
	}
	if serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("replayedWaitState=%+v", replayedWaitState)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RecuperaDeliveryTardiaTrasWaitExpirado(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	fixture.Run.Reviews = nil
	fixture.Run.ReviewResults = nil
	fixture.Run.AcceptedReviews = nil
	state := fixture.State
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"external-wait-exhausted"}
	state.ClosureReason = "external-wait-exhausted"
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			step.PendingAgentRefs = nil
			step.BlockerRefs = []string{"external-wait-exhausted"}
			step.Reason = "external-wait-exhausted"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)

	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:24:59Z",
		CorrelationID:              "corr-app-director-late-delivery-after-expired-wait-no-writer",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  runStore,
		OperationalPlanStateStore: planStore,
	})
	if err == nil {
		t.Fatalf("err nil sin writer")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "ports.operational_plan_state_writer" {
		t.Fatalf("err=%T %#v", err, err)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:25:00Z",
		CorrelationID:              "corr-app-director-late-delivery-after-expired-wait",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		OperationalPlanStateStore:  planStore,
		OperationalPlanStateWriter: planStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v agent=%s", reentered, fixture.AgentRef)
	}
	recovered, err := planStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		recovered.ClosureReason != "" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(recovered.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-expired-late-delivery-v0") {
		t.Fatalf("recovered=%+v wait=%+v review=%+v", recovered, waitStep, reviewStep)
	}
}

func TestOperationalDirectorPlanStatePostLoopScopeV0RecargaRunStoreAntesDeCerrarWait(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	state := fixture.State
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = []string{fixture.AgentRef}
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.PendingAgentRefs = []string{fixture.AgentRef}
			step.Reason = "wait-subagents-running"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	latest := fixture.Run
	latest.DeliveredAgents = nil
	latest.DeliveredTasks = nil
	latest.Deliveries = nil
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(latest)
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	staleLoop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		Run: orquestacoreworkflow.OrchestrationRunV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
			RunID:         fixture.RunRef,
		},
	}

	got, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStore},
		staleLoop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{RunRef: fixture.RunRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 pending: %v", err)
	}
	if got.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("scope cerro con run stale: got=%+v latest=%+v", got, latest)
	}

	latest.DeliveredAgents = []string{fixture.AgentRef}
	if err := runStore.SaveRunV0(context.Background(), latest); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	got, err = operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStore},
		staleLoop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{RunRef: fixture.RunRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 delivered: %v", err)
	}
	if got.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		!serviceStringInSetV0(got.Run.DeliveredAgents, fixture.AgentRef) {
		t.Fatalf("scope no cerro con delivery latest: got=%+v", got)
	}
}

func TestContinueAppDirectorV0AvanzaDeWaitAReviewAbriendoRevision(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	run := serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.ProjectRef = fixture.Run.ProjectRef
	run.AppSpecRef = fixture.Run.AppSpecRef
	run.Tasks = []string{fixture.TaskRef}
	run.Agents = []string{fixture.AgentRef}
	run.StartedAgents = []string{fixture.AgentRef}
	run.DeliveredAgents = []string{fixture.AgentRef}
	run.DeliveredTasks = []string{fixture.TaskRef}
	run.Deliveries = []string{fixture.DeliveryRef}
	run.LastEventID = "event-ref-delivery-review-autofollow-001"
	run.LastSequence = 1
	state := fixture.State
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = []string{fixture.AgentRef}
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.PendingAgentRefs = []string{fixture.AgentRef}
			step.Reason = "wait-subagents-running"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, []orquestacoreworkflow.OrchestrationEventV0{fixture.Events[0]}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	reviewSource := &acceptedServiceReviewGateSourceV0{Fixture: fixture}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T13:00:00Z",
		CorrelationID:              "corr-service-wait-review-open-revision-001",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  6,
		MaxStepsPerBurst:           6,
		MaxDispatchesPerWait:       2,
		MaxCommands:                20,
		MaxOutboxPerCycle:          8,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                eventSink,
		OutboxLedger:               outboxLedger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)),
		ReviewGateSource:           reviewSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !reviewSource.Called ||
		result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		!serviceStringInSetV0(result.Run.Reviews, fixture.ReviewRequestID) ||
		!strings.Contains(strings.Join(result.Run.ReviewResults, " "), fixture.ReviewResultRef) ||
		!serviceStringInSetV0(result.Run.AcceptedReviews, fixture.AcceptedReviewRef) {
		t.Fatalf("review no aplicada: called=%v run=%+v", reviewSource.Called, result.Run)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-run-required-tests" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}

func TestContinueAppDirectorV0ReviewChangesRequestedSplitTaskDurableV0(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-continue-review-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-continue-review-split-b",
		fixture.TaskRef,
	)
	parentTask := serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)
	parentTask.WaveRef = firstFollowup.WaveRef
	parentTask.CohortRef = firstFollowup.CohortRef
	parentTask.DelegationDepth = 0
	parentTask.MaxChildAgents = 2

	run := fixture.Run
	run.Tasks = []string{fixture.TaskRef}
	run.Agents = []string{fixture.AgentRef}
	run.StartedAgents = []string{fixture.AgentRef}
	run.DeliveredAgents = []string{fixture.AgentRef}
	run.DeliveredTasks = []string{fixture.TaskRef}
	run.Deliveries = []string{fixture.DeliveryRef}
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	}
	run.Reviews = nil
	run.ReviewResults = nil
	run.AcceptedReviews = nil
	run.ReworkRequests = nil
	run.ReplanDecisions = nil
	run.FunctionContracts = []string{"contract:function:rework-split:v0"}
	run.LastEventID = fixture.Events[0].EventID
	run.LastSequence = 1

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, []orquestacoreworkflow.OrchestrationEventV0{fixture.Events[0]}); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parentTask)
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	reviewSource := &changesRequestedServiceReviewGateSourceV0{Fixture: fixture}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T10:10:00Z",
		CorrelationID:              "corr-app-director-continue-review-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  12,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       4,
		MaxCommands:                40,
		MaxOutboxPerCycle:          12,
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		EventReader:       eventSink,
		OutboxLedger:      outboxLedger,
		DirectorTaskStore: taskStore,
		WaitStateWriter:   waitStore,
		WaitStateStore:    waitStore,
		ReviewGateSource:  reviewSource,
		ReviewReworkReplanSource: serviceOperationalDirectorSplitReviewReworkReplanSourceV0{
			Fixture:    fixture,
			SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{firstFollowup, secondFollowup},
		},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}

	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	if !reviewSource.Called ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!strings.Contains(strings.Join(result.Run.ReworkRequests, " "), fixture.ReworkRequestRef) ||
		!strings.Contains(strings.Join(result.Run.ReplanDecisions, " "), fixture.ReplanDecisionRef) ||
		!serviceStringInSetV0(result.Run.Tasks, firstFollowup.TaskID) ||
		!serviceStringInSetV0(result.Run.Tasks, secondFollowup.TaskID) ||
		!serviceStringInSetV0(result.Run.StartedAgents, followupAgentRefs[0]) ||
		!serviceStringInSetV0(result.Run.StartedAgents, followupAgentRefs[1]) {
		t.Fatalf("split no aplicado: called=%v result=%+v followups=%+v", reviewSource.Called, result, followupAgentRefs)
	}
	loaded, err := taskStore.LoadWorkflowTasksV0(context.Background(), fixture.RunRef, []string{firstFollowup.TaskID, secondFollowup.TaskID})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 followups: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].ParentTaskRef != fixture.TaskRef ||
		loaded[1].ParentTaskRef != fixture.TaskRef {
		t.Fatalf("followups no persistidos como hijos: %+v", loaded)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("state=%+v waitStep=%+v reviewStep=%+v", state, waitStep, reviewStep)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("replan no idempotente en ciclo integrado: got=%d events=%+v", got, eventSink.EventsV0())
	}
}

func TestOperationalDirectorOlaCohorteAmpliaOfflineScopeReviewTestsClose(t *testing.T) {
	runRef := "run-app-director-operational-wide-wave-offline"
	planRef := "plan-ref-app-director-operational-wide-wave-offline"
	parentTaskRef := "parent-task-app-director-operational-wide-wave-offline"
	waveRef := "wave-service-operational-closure-plan-state"
	cohortRef := "cohort-service-operational-closure-plan-state"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-app-director-wide-wave-a",
		DeliveryRef:     "delivery-ref-app-director-wide-wave-a",
		ReviewRequestID: "review-request-ref-app-director-wide-wave-a",
		ReviewResultRef: "review-result-ref-app-director-wide-wave-a",
		AcceptedRef:     "accepted-review-ref-app-director-wide-wave-a",
		TestEvidenceRef: "test-evidence-ref-app-director-wide-wave-a",
		ValidationRef:   "validation-ref-app-director-wide-wave-a",
		ClosureRef:      "closure-ref-app-director-wide-wave-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-app-director-wide-wave-b",
		DeliveryRef:     "delivery-ref-app-director-wide-wave-b",
		ReviewRequestID: "review-request-ref-app-director-wide-wave-b",
		ReviewResultRef: "review-result-ref-app-director-wide-wave-b",
		AcceptedRef:     "accepted-review-ref-app-director-wide-wave-b",
		TestEvidenceRef: "test-evidence-ref-app-director-wide-wave-b",
		ValidationRef:   "validation-ref-app-director-wide-wave-b",
		ClosureRef:      "closure-ref-app-director-wide-wave-b",
	}
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef)
	outOfScopeAgent := "agent-ref-app-director-wide-wave-out-of-scope"
	waitAgentRefs := []string{agentA, agentB}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.Agents = []string{agentA, agentB, outOfScopeAgent}
	run.StartedAgents = []string{agentA, agentB, outOfScopeAgent}
	run.DeliveredAgents = []string{agentA}
	run.DeliveredTasks = []string{childA.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalDirectorWideWaveWaitStateForTestV0(
			runRef,
			planRef,
			parentTaskRef,
			waveRef,
			cohortRef,
			childA,
			childB,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T14:00:00Z",
		CorrelationID:              "corr-app-director-wide-wave-offline",
		OperationalDirectorPlanRef: planRef,
		WaitWaveRef:                waveRef,
		WaitCohortRef:              cohortRef,
		WaitParentTaskRef:          parentTaskRef,
	}
	loopRequest := orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:        runRef,
		WaitAgentRefs: waitAgentRefs,
	}
	partial, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStateStore},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    run,
		},
		loopRequest,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 partial: %v", err)
	}
	if partial.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("delivery parcial cerro scope: %+v", partial)
	}

	run.DeliveredAgents = []string{agentA, agentB}
	run.DeliveredTasks = []string{childA.TaskRef, childB.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef, childB.DeliveryRef}
	run.Reviews = []string{childA.ReviewRequestID, childB.ReviewRequestID}
	run.ReviewResults = []string{
		serviceOperationalClosureReviewResultProjectionForTestV0(childA),
		serviceOperationalClosureReviewResultProjectionForTestV0(childB),
	}
	run.AcceptedReviews = []string{childA.AcceptedRef, childB.AcceptedRef}
	if err := runStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 complete: %v", err)
	}
	eventReader := serviceOperationalClosureEventReaderForTestV0{
		Events: serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB),
	}
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childA),
		serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childB),
	)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
		serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childA),
		serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childB),
	)
	closureSource := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childA.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childA),
			childB.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childB),
		},
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader:                eventReader,
		DirectorTaskStore:          taskStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
		OperationalClosureSource:   closureSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	scoped, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    run,
		},
		loopRequest,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 complete: %v", err)
	}
	if scoped.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		serviceStringInSetV0(scoped.Run.DeliveredAgents, outOfScopeAgent) {
		t.Fatalf("scope no quedo acotado: %+v", scoped)
	}
	changed, err := applyOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, scoped)
	if err != nil || !changed {
		t.Fatalf("applyOperationalDirectorPlanStateAfterLoopV0 changed=%v err=%v", changed, err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 after apply: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, childA.DeliveryRef) ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, childB.DeliveryRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, childA.TestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, childB.TestEvidenceRef) {
		t.Fatalf("state=%+v review=%+v tests=%+v replan=%+v", state, reviewStep, testsStep, replanStep)
	}

	first, issues, err := maybeCloseOperationalDirectorV0(context.Background(), request, ports, scoped, loopRequest)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	if first.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		!serviceStringInSetV0(first.Run.ClosedTasks, childA.TaskRef) ||
		serviceStringInSetV0(first.Run.ClosedTasks, childB.TaskRef) {
		t.Fatalf("primer cierre no fue parcial por task: %+v", first.Run)
	}
	request.OccurredAt = "2026-05-22T14:00:01Z"
	second, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		loopRequest,
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 second err=%v issues=%+v", err, issues)
	}
	if second.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childA.TaskRef) ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childB.TaskRef) ||
		serviceStringInSetV0(second.Run.DeliveredAgents, outOfScopeAgent) {
		t.Fatalf("cierre amplio no completo scope acotado: %+v", second.Run)
	}
	if len(closureSource.LastInput.WaitAgentRefs) != 2 ||
		!serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, agentA) ||
		!serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, agentB) ||
		serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, outOfScopeAgent) {
		t.Fatalf("closure source amplio scope=%+v", closureSource.LastInput.WaitAgentRefs)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewATestsRequeridos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-run-required-tests" {
		t.Fatalf("active_step_id=%s state=%+v", state.ActiveStepID, state)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
		!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-pending") ||
		!serviceStringInSetV0(testsStep.TaskRefs, fixture.TaskRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewAceptadaReplayNoDuplicaPlanState(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}
	for _, occurredAt := range []string{"2026-05-17T14:20:00Z", "2026-05-17T14:20:01Z"} {
		if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 occurredAt,
			OperationalDirectorPlanRef: fixture.PlanRef,
		}, ports, loop); err != nil {
			t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 %s: %v", occurredAt, err)
		}
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.ActiveStepID != "step-run-required-tests" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v reviewStep=%+v testsStep=%+v", state, reviewStep, testsStep)
	}
	for label, values := range map[string][]string{
		"delivery":        reviewStep.DeliveryRefs,
		"review_result":   reviewStep.ReviewResultRefs,
		"accepted_review": reviewStep.AcceptedReviewRefs,
	} {
		want := map[string]string{
			"delivery":        fixture.DeliveryRef,
			"review_result":   fixture.ReviewResultRef,
			"accepted_review": fixture.AcceptedReviewRef,
		}[label]
		if serviceCountStringV0(values, want) != 1 {
			t.Fatalf("%s refs=%+v want una vez %s", label, values, want)
		}
	}
	if serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-accepted-v0") != 1 ||
		serviceCountStringV0(testsStep.BlockerRefs, "required-tests-pending") != 1 ||
		serviceCountStringV0(testsStep.TaskRefs, fixture.TaskRef) != 1 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewNegativaRegistraReworkReplan(t *testing.T) {
	for _, status := range []orquestacoreworkflow.ReviewResultStatusV0{
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		orquestacoreworkflow.ReviewResultStatusRejectedV0,
	} {
		t.Run(string(status), func(t *testing.T) {
			fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(t, status)
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

			if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
				RunRef:                     fixture.RunRef,
				OccurredAt:                 "2026-05-17T14:20:10Z",
				OperationalDirectorPlanRef: fixture.PlanRef,
			}, StartAppDirectorPortsV0{
				EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
				OperationalPlanStateStore:  planStateStore,
				OperationalPlanStateWriter: planStateStore,
			}, orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
				Run:    fixture.Run,
			}); err != nil {
				t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
			}
			state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
			if err != nil {
				t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
			}
			reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
			testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
			replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
			if state.ActiveStepID != "step-review-deliveries" ||
				state.ReplanAttempts != 1 ||
				reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
				!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
				!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
				!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
				!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) ||
				len(reviewStep.AcceptedReviewRefs) != 0 ||
				testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 ||
				replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
				t.Fatalf("state=%+v reviewStep=%+v testsStep=%+v replanStep=%+v", state, reviewStep, testsStep, replanStep)
			}
		})
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewChangesRequestedIdempotenteYNoReentraWaitAntiguo(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:10Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-17T14:20:11Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		len(state.PendingAgentRefs) != 0 ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
		len(reviewStep.DeliveryRefs) != 1 ||
		reviewStep.DeliveryRefs[0] != fixture.DeliveryRef ||
		len(reviewStep.ReviewResultRefs) != 1 ||
		reviewStep.ReviewResultRefs[0] != fixture.ReviewResultRef ||
		len(reviewStep.ReworkRequestRefs) != 1 ||
		reviewStep.ReworkRequestRefs[0] != fixture.ReworkRequestRef ||
		len(reviewStep.ReplanDecisionRefs) != 1 ||
		reviewStep.ReplanDecisionRefs[0] != fixture.ReplanDecisionRef ||
		len(reviewStep.AcceptedReviewRefs) != 0 ||
		len(reviewStep.BlockerRefs) != 1 ||
		reviewStep.BlockerRefs[0] != "review-rework-replan-recorded" {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	for _, ref := range []string{
		"evidence-ref-delivery-review-accepted",
		"evidence-ref-review-requested",
		"evidence-ref-review-result-changes_requested",
		"evidence-ref-rework-requested-changes_requested",
		"evidence-ref-replan-decision-changes_requested",
	} {
		if !serviceStringInSetV0(reviewStep.EvidenceRefs, ref) {
			t.Fatalf("reviewStep evidence_refs=%+v falta %s", reviewStep.EvidenceRefs, ref)
		}
	}

	got, applied, err := continueRequestWithLoadedOperationalDirectorPlanStateV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		state,
		StartAppDirectorPortsV0{},
	)
	if err != nil {
		t.Fatalf("continueRequestWithLoadedOperationalDirectorPlanStateV0: %v", err)
	}
	if applied ||
		len(got.WaitAgentRefs) != 0 ||
		got.WaitWaveRef != "" ||
		got.WaitCohortRef != "" ||
		got.WaitParentTaskRef != "" {
		t.Fatalf("reentrada inesperada applied=%v request=%+v", applied, got)
	}
	_, err = continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewChangesRequestedAbreWaitDeSplitFollowups(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-operational-plan-state-review-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-operational-plan-state-review-split-b",
		fixture.TaskRef,
	)
	followupRefs := []string{firstFollowup.TaskID, secondFollowup.TaskID}
	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	fixture.Run.Tasks = append(fixture.Run.Tasks, followupRefs...)
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) +
			"#followups:" + firstFollowup.TaskID + "+" + secondFollowup.TaskID,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
		FollowupRefs:   followupRefs,
		Summary:        "Replan split por review negativa.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-split-followups"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(firstFollowup, secondFollowup)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T14:30:00Z",
		CorrelationID:              "corr-app-director-review-split-followups",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		DirectorTaskStore:          taskStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-21T14:30:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != firstFollowup.WaveRef ||
		state.ActiveCohortRef != firstFollowup.CohortRef ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followups") ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[1]) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != firstFollowup.WaveRef ||
		reentered.WaitCohortRef != firstFollowup.CohortRef ||
		reentered.WaitParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followupAgentRefs=%+v old=%s", reentered, followupAgentRefs, fixture.AgentRef)
	}

	deliveredFollowupsRun := fixture.Run
	deliveredFollowupsRun.DeliveredAgents = append(deliveredFollowupsRun.DeliveredAgents, followupAgentRefs...)
	request.OccurredAt = "2026-05-21T14:30:02Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredFollowupsRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 delivered followups: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 delivered followups: %v", err)
	}
	reviewStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(reviewStep.TaskRefs, secondFollowup.TaskID) ||
		serviceStringInSetV0(reviewStep.TaskRefs, fixture.TaskRef) ||
		len(reviewStep.ReviewResultRefs) != 0 ||
		len(reviewStep.ReworkRequestRefs) != 0 ||
		len(reviewStep.ReplanDecisionRefs) != 0 {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}

func TestReviewReworkSplitTaskDurableReabreWaitSoloFollowupsNuevos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-review-rework-durable-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-review-rework-durable-split-b",
		fixture.TaskRef,
	)
	parentTask := serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)
	parentTask.WaveRef = firstFollowup.WaveRef
	parentTask.CohortRef = firstFollowup.CohortRef
	parentTask.DelegationDepth = 0
	parentTask.MaxChildAgents = 2
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parentTask)
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{
			DirectorTaskStore: taskStore,
			ReviewReworkReplanSource: serviceOperationalDirectorSplitReviewReworkReplanSourceV0{
				Fixture:    fixture,
				SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{firstFollowup, secondFollowup},
			},
		},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           fixture.Run,
		OccurredAt:    "2026-05-23T09:10:00Z",
		CorrelationID: "corr-app-director-review-rework-durable-split",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan followups=%d candidates=%+v", len(candidates.ReplanFollowupCandidates), candidates)
	}
	input := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput
	if input.DecisionPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
		len(input.MicrotaskCandidates) != 2 {
		t.Fatalf("input=%+v", input)
	}
	loaded, err := taskStore.LoadWorkflowTasksV0(context.Background(), fixture.RunRef, input.DecisionPayload.FollowupRefs)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 followups: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].ParentTaskRef != fixture.TaskRef ||
		loaded[1].ParentTaskRef != fixture.TaskRef {
		t.Fatalf("loaded followups=%+v", loaded)
	}

	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	run := fixture.Run
	run.Tasks = append(run.Tasks, input.DecisionPayload.FollowupRefs...)
	run.Agents = append(run.Agents, "agent-ref-app-director-review-rework-unrelated-live")
	run.ReplanDecisions = []string{
		input.DecisionPayload.ReplanRef + "#source:" + input.DecisionPayload.SourceRef +
			"#task:" + input.DecisionPayload.TaskRef +
			"#action:" + string(input.DecisionPayload.AcceptedAction) +
			"#followups:" + strings.Join(input.DecisionPayload.FollowupRefs, "+"),
	}
	events := append([]orquestacoreworkflow.OrchestrationEventV0(nil), fixture.Events[:4]...)
	events = append(events, serviceOperationalDirectorPlanStateEventForTestV0(
		t,
		fixture.RunRef,
		5,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.ReplanDecisionRecordedPayloadV0(input.DecisionPayload),
	))
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T09:10:01Z",
		CorrelationID:              "corr-app-director-review-rework-durable-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		DirectorTaskStore:          taskStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitParentTaskRef != fixture.TaskRef ||
		reentered.WaitWaveRef != firstFollowup.WaveRef ||
		reentered.WaitCohortRef != firstFollowup.CohortRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, "agent-ref-app-director-review-rework-unrelated-live") {
		t.Fatalf("reentered=%+v followups=%+v old=%s", reentered, followupAgentRefs, fixture.AgentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewChangesRequestedAbreWaitDeRetryAgent(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupAgentRef := "agent-ref-app-director-operational-plan-state-review-retry-followup"
	capacityRef := "capacity-ref-app-director-operational-plan-state-review-retry-followup"
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + capacityRef + "+" + followupAgentRef,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{capacityRef, followupAgentRef},
		Summary:        "Replan retry por review negativa.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-retry-followup"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T15:10:00Z",
		CorrelationID:              "corr-app-director-review-retry-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-21T15:10:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != "" ||
		state.ActiveCohortRef != "" ||
		state.ActiveParentTaskRef != "" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followup-agents") ||
		!serviceStringInSetV0(waitStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	deliveredRetryRun := fixture.Run
	deliveredRetryRun.DeliveredAgents = append(deliveredRetryRun.DeliveredAgents, followupAgentRef)
	request.OccurredAt = "2026-05-21T15:10:02Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRetryRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 delivered retry: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 delivered retry: %v", err)
	}
	reviewStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reviewStep.AgentRefs, fixture.AgentRef) ||
		len(reviewStep.ReviewResultRefs) != 0 ||
		len(reviewStep.ReworkRequestRefs) != 0 ||
		len(reviewStep.ReplanDecisionRefs) != 0 {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeTestsAReplanConEvidenciaPassed(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:30Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0EjecutaRunnerDeTestsRequeridos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[2] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: fixture.ReviewResultRef,
		ReviewRequestID: fixture.ReviewRequestID,
		DeliveryRef:     fixture.DeliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:         "Review aceptada sin evidencia de test precocinada.",
		EvidenceRefs:    []string{"evidence-ref-review-result-accepted"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-001"},
			},
		},
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T11:20:00Z",
		CorrelationID:              "correlation-service-required-tests-001",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
		RequiredTestRunner: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor:       executor,
			EvidenceWriter: testEvidenceStore,
		},
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("commands=%+v", executor.commands)
	}
	if state.ActiveStepID != "step-replan-or-close" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}

	evidence, err := testEvidenceStore.LoadRequiredTestEvidenceV0(context.Background(), fixture.RunRef, testsStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].TaskRef != fixture.TaskRef ||
		evidence[0].DeliveryRef != fixture.DeliveryRef ||
		evidence[0].ReviewResultRef != fixture.ReviewResultRef ||
		evidence[0].AcceptedReviewRef != fixture.AcceptedReviewRef ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		!serviceStringInSetV0(evidence[0].EvidenceRefs, "artifact-ref-service-required-test-output-001") {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0BloqueaTestsSinEvidenciaNiRunnerYReentra(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T19:10:00Z",
		CorrelationID:              "corr-service-required-tests-missing-evidence",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		!serviceStringInSetV0(state.BlockerRefs, "required-tests-evidence-missing") ||
		state.ReplanAttempts != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-evidence-missing" ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-evidence-missing") ||
		len(testsStep.RequiredTestEvidenceRefs) != 0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}

	if err := testEvidenceStore.SaveRequiredTestEvidenceV0(
		context.Background(),
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		),
	); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T19:10:01Z",
		CorrelationID:              "corr-service-required-tests-missing-evidence-reenter",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != fixture.WaveRef ||
		reentered.WaitCohortRef != fixture.CohortRef ||
		reentered.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v fixture=%+v", reentered, fixture)
	}

	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	testsStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-replan-or-close" ||
		len(state.BlockerRefs) != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		len(testsStep.BlockerRefs) != 0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0RequiredTestsEvidenceMissingEmiteQualityGateIdempotente(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	fixture.Run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	}
	fixture.Run.LastSequence = 4
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T20:30:00Z",
		CorrelationID:              "corr-app-director-required-tests-missing-gate",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(),
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-22T20:30:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		state.ReplanAttempts != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-evidence-missing" ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-evidence-missing") ||
		len(testsStep.ReplanDecisionRefs) != 0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}

	events := eventSink.EventsV0()
	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("quality gate payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
		}
	}
	if gateEvents != 1 || replanEvents != 0 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, events)
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != fixture.TaskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "required-tests-evidence-missing") ||
		!serviceStringInSetV0(gatePayload.EvidenceRefs, fixture.DeliveryRef) ||
		!serviceStringInSetV0(gatePayload.EvidenceRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	run, err := runStore.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != 1 ||
		len(run.ReplanDecisions) != 0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("run=%+v gate=%+v", run, gatePayload)
	}
}

func TestContinueAppDirectorV0CierraCicloReviewRunnerYReplanOrClose(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[2] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: fixture.ReviewResultRef,
		ReviewRequestID: fixture.ReviewRequestID,
		DeliveryRef:     fixture.DeliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:         "Review aceptada; el runner debe generar la evidencia durable.",
		EvidenceRefs:    []string{"evidence-ref-review-result-accepted"},
	})
	run := serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.ProjectRef = fixture.Run.ProjectRef
	run.AppSpecRef = fixture.Run.AppSpecRef
	run.Tasks = append([]string(nil), fixture.Run.Tasks...)
	run.Agents = append([]string(nil), fixture.Run.Agents...)
	run.StartedAgents = append([]string(nil), fixture.Run.StartedAgents...)
	run.DeliveredAgents = append([]string(nil), fixture.Run.DeliveredAgents...)
	run.DeliveredTasks = append([]string(nil), fixture.Run.DeliveredTasks...)
	run.Deliveries = append([]string(nil), fixture.Run.Deliveries...)
	run.Reviews = append([]string(nil), fixture.Run.Reviews...)
	run.ReviewResults = append([]string(nil), fixture.Run.ReviewResults...)
	run.AcceptedReviews = append([]string(nil), fixture.Run.AcceptedReviews...)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-continue-001"},
			},
		},
	}
	closureSource := &serviceOperationalDirectorClosureSourceFromRequestForTestV0{
		Fixture: fixture,
	}

	result, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-22T12:00:00Z",
			CorrelationID:              "correlation-service-review-runner-close-001",
			OperationalDirectorPlanRef: fixture.PlanRef,
			MaxBursts:                  1,
			MaxStepsPerBurst:           1,
			MaxDispatchesPerWait:       1,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
			OutboxLedger:               outboxLedger,
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)),
			RequiredTestEvidenceStore:  testEvidenceStore,
			RequiredTestRunner:         orquestacionnucleoapp.RequiredTestRunnerV0{Executor: executor, EvidenceWriter: testEvidenceStore},
			OperationalClosureSource:   closureSource,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
			},
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("result no cerrado: %+v", result)
	}
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("runner no ejecutado una vez: commands=%+v", executor.commands)
	}
	if !closureSource.Called || len(closureSource.LastRequest.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("closure source sin evidencia generada: called=%v request=%+v", closureSource.Called, closureSource.LastRequest)
	}
	evidence, err := testEvidenceStore.LoadRequiredTestEvidenceV0(
		context.Background(),
		fixture.RunRef,
		closureSource.LastRequest.RequiredTestEvidenceRefs,
	)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TaskRef != fixture.TaskRef ||
		evidence[0].DeliveryRef != fixture.DeliveryRef ||
		evidence[0].AcceptedReviewRef != fixture.AcceptedReviewRef {
		t.Fatalf("evidence generada invalida: %+v", evidence)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(replanStep.RequiredTestEvidenceRefs, closureSource.LastRequest.RequiredTestEvidenceRefs[0]) != 1 {
		t.Fatalf("plan state no cerrado por ciclo integrado: state=%+v replanStep=%+v", state, replanStep)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 1 {
		t.Fatalf("RunClosed esperado una vez: got=%d events=%+v", got, eventSink.EventsV0())
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0BloqueaTestsConEvidenciaFailed(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:40Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-failed") ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedEmiteQualityGateYReplanAutomatico(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	fixture.Run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	}
	fixture.Run.LastSequence = 4
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 initial: %v", err)
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T09:10:00Z",
		CorrelationID:              "corr-app-director-required-tests-auto-replan",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-22T09:10:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed" ||
		serviceCountStringV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) != 1 ||
		serviceCountStringV0(testsStep.BlockerRefs, "required-tests-failed") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-failed-v0") != 1 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}

	events := eventSink.EventsV0()
	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("quality gate payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
			if err := json.Unmarshal(event.Payload, &replanPayload); err != nil {
				t.Fatalf("replan payload: %v", err)
			}
		}
	}
	if gateEvents != 1 || replanEvents != 1 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, events)
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != fixture.TaskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != fixture.TaskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		len(replanPayload.FollowupRefs) != 2 ||
		!strings.HasPrefix(replanPayload.FollowupRefs[0], "capacity-ref-app-director-required-tests-retry-") ||
		!strings.HasPrefix(replanPayload.FollowupRefs[1], "agent-ref-app-director-required-tests-retry-") {
		t.Fatalf("replanPayload=%+v gate=%+v", replanPayload, gatePayload)
	}
	run, err := runStore.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != 1 ||
		len(run.ReplanDecisions) != 1 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		serviceStringInSetV0(run.CapacityRequests, replanPayload.FollowupRefs[0]) ||
		serviceStringInSetV0(run.Agents, replanPayload.FollowupRefs[1]) {
		t.Fatalf("run=%+v replanPayload=%+v", run, replanPayload)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
		CorrelationID:              "corr-app-director-required-tests-auto-replan-reenter",
	}, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 auto replan: %v", err)
	}
	if continueRequestHasWaitScopeV0(reentered) {
		t.Fatalf("la reentrada debe dejar avanzar al scheduler sin wait scope prematuro: %+v", reentered)
	}
}

func TestContinueAppDirectorV0TestsFailedReplanFollowupCierraSinDuplicarEventos(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	run := serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.ProjectRef = fixture.Run.ProjectRef
	run.AppSpecRef = fixture.Run.AppSpecRef
	run.Tasks = append([]string(nil), fixture.Run.Tasks...)
	run.Agents = append([]string(nil), fixture.Run.Agents...)
	run.StartedAgents = append([]string(nil), fixture.Run.StartedAgents...)
	run.DeliveredAgents = append([]string(nil), fixture.Run.DeliveredAgents...)
	run.DeliveredTasks = append([]string(nil), fixture.Run.DeliveredTasks...)
	run.Deliveries = append([]string(nil), fixture.Run.Deliveries...)
	run.Reviews = append([]string(nil), fixture.Run.Reviews...)
	run.ReviewResults = append([]string(nil), fixture.Run.ReviewResults...)
	run.AcceptedReviews = append([]string(nil), fixture.Run.AcceptedReviews...)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-followup-001"},
			},
		},
	}
	closureSource := &serviceOperationalDirectorClosureSourceRefsForTestV0{
		TaskRef:           fixture.TaskRef,
		ValidationRef:     "validation-ref-service-required-tests-replan-followup-001",
		ClosureRef:        "closure-ref-service-required-tests-replan-followup-001",
		DeliveryRef:       "delivery-ref-app-director-required-tests-replan-followup",
		AcceptedReviewRef: "accepted-review-ref-app-director-required-tests-replan-followup",
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                eventSink,
		OutboxLedger:               outboxLedger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)),
		RequiredTestEvidenceStore:  testEvidenceStore,
		RequiredTestRunner:         orquestacionnucleoapp.RequiredTestRunnerV0{Executor: executor, EvidenceWriter: testEvidenceStore},
		OperationalClosureSource:   closureSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T23:40:00Z",
		CorrelationID:              "corr-service-required-tests-replan-followup-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  4,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       2,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}

	first, err := ContinueAppDirectorV0(ctx, request, ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if first.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("first no debe cerrar tras tests fallidos: %+v", first.Run)
	}
	if closureSource.Called {
		t.Fatalf("closure source no debe llamarse antes del replan: request=%+v", closureSource.LastRequest)
	}
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	var gateEvents, replanEvents int
	for _, event := range eventSink.EventsV0() {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("QualityGateRecorded payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
			if err := json.Unmarshal(event.Payload, &replanPayload); err != nil {
				t.Fatalf("ReplanDecisionRecorded payload: %v", err)
			}
		}
	}
	if gateEvents != 1 || replanEvents != 1 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, eventSink.EventsV0())
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != fixture.TaskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("quality gate no causal del test fallido: %+v failed=%s", gatePayload, fixture.RequiredTestEvidenceRef)
	}
	if replanPayload.ReplanRef == "" || len(replanPayload.FollowupRefs) != 2 {
		t.Fatalf("replanPayload=%+v events=%+v", replanPayload, eventSink.EventsV0())
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != fixture.TaskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("replan no cuelga del quality gate fallido: replan=%+v gate=%+v", replanPayload, gatePayload)
	}
	followupAgentRef := ""
	for _, ref := range replanPayload.FollowupRefs {
		if strings.HasPrefix(ref, "agent-ref-") {
			followupAgentRef = ref
		}
	}
	if followupAgentRef == "" {
		t.Fatalf("replan sin agente followup: %+v", replanPayload)
	}

	materializedRun, err := runStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 materialized: %v", err)
	}
	if len(materializedRun.QualityGates) != 1 ||
		len(materializedRun.ReplanDecisions) != 1 ||
		serviceStringInSetV0(materializedRun.Agents, followupAgentRef) ||
		materializedRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("run tras replan invalido: run=%+v followup=%s", materializedRun, followupAgentRef)
	}
	materializedRun.Agents = compactServiceRefsV0(append(materializedRun.Agents, followupAgentRef))
	materializedRun.StartedAgents = compactServiceRefsV0(append(materializedRun.StartedAgents, followupAgentRef))
	if err := runStore.SaveRunV0(ctx, materializedRun); err != nil {
		t.Fatalf("SaveRunV0 materialized: %v", err)
	}
	request.OccurredAt = "2026-05-22T23:40:01Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-wait"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 followup: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("wait scope no causal: request=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	deliveredRun, err := runStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 delivered: %v", err)
	}
	followupDeliveryRef := closureSource.DeliveryRef
	followupReviewRequestRef := "review-request-ref-app-director-required-tests-replan-followup"
	followupReviewResultRef := "review-result-ref-app-director-required-tests-replan-followup"
	followupAcceptedReviewRef := closureSource.AcceptedReviewRef
	deliveredRun.DeliveredAgents = compactServiceRefsV0(append(deliveredRun.DeliveredAgents, followupAgentRef))
	deliveredRun.Deliveries = compactServiceRefsV0(append(deliveredRun.Deliveries, followupDeliveryRef))
	deliveredRun.DeliveredTasks = compactServiceRefsV0(append(deliveredRun.DeliveredTasks, fixture.TaskRef))
	deliveredRun.Reviews = compactServiceRefsV0(append(deliveredRun.Reviews, followupReviewRequestRef))
	deliveredRun.ReviewResults = compactServiceRefsV0(append(deliveredRun.ReviewResults,
		followupReviewResultRef+"#review_result:accepted#review_request:"+followupReviewRequestRef+"#delivery:"+followupDeliveryRef,
	))
	deliveredRun.AcceptedReviews = compactServiceRefsV0(append(deliveredRun.AcceptedReviews, followupAcceptedReviewRef))
	deliveredRun.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	for index := range deliveredRun.Phases {
		switch deliveredRun.Phases[index].ID {
		case orquestacoreworkflow.OrchestrationPhaseProgramacionV0:
			deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusClosedV0
		case orquestacoreworkflow.OrchestrationPhaseRevisionV0:
			deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
		default:
			if deliveredRun.Phases[index].Status == orquestacoreworkflow.OrchestrationPhaseStatusActiveV0 {
				deliveredRun.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
			}
		}
	}
	if err := runStore.SaveRunV0(ctx, deliveredRun); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	followupEvents := []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  followupDeliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       fixture.TaskRef,
			AgentRef:     followupAgentRef,
			Summary:      "Entrega followup causal tras replan por tests fallidos.",
			EvidenceRefs: []string{"evidence-ref-delivery-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: followupReviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     followupDeliveryRef,
			Summary:         "Review followup causal.",
			EvidenceRefs:    []string{"evidence-ref-review-requested-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 7, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: followupReviewResultRef,
			ReviewRequestID: followupReviewRequestRef,
			DeliveryRef:     followupDeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review followup aceptada.",
			EvidenceRefs:    []string{"evidence-ref-review-result-required-tests-replan-followup"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 8, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: followupAcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   followupReviewRequestRef,
			DeliveryRef:       followupDeliveryRef,
			Summary:           "Review followup aceptada con cadena causal.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted-required-tests-replan-followup"},
		}),
	}
	if err := eventSink.AppendRunEventsV0(ctx, fixture.RunRef, followupEvents); err != nil {
		t.Fatalf("AppendRunEventsV0 followup: %v", err)
	}

	request.OccurredAt = "2026-05-22T23:40:02Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-close"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusBlockedV0,
		Run:    deliveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 followup: %v", err)
	}
	stateAfterTests, err := planStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 after tests: %v", err)
	}
	testsStepAfterFollowup := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterTests, "step-run-required-tests")
	replanStepAfterFollowup := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterTests, "step-replan-or-close")
	if stateAfterTests.ActiveStepID != "step-replan-or-close" ||
		testsStepAfterFollowup.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStepAfterFollowup.Reason != "required-tests-passed" ||
		replanStepAfterFollowup.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(replanStepAfterFollowup.RequiredTestEvidenceRefs) != 1 ||
		serviceStringInSetV0(replanStepAfterFollowup.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state tras followup no avanza con evidencia passed nueva: state=%+v tests=%+v replan=%+v failed=%s", stateAfterTests, testsStepAfterFollowup, replanStepAfterFollowup, fixture.RequiredTestEvidenceRef)
	}
	postTestsRun, err := runStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 post tests: %v", err)
	}
	closed, issues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    postTestsRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef:           fixture.RunRef,
			WaitAgentRefs:    []string{followupAgentRef},
			WaitScopeApplied: true,
		},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 close: err=%v issues=%+v", err, issues)
	}
	if closed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		state, _ := planStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
		t.Fatalf("run no cerrado: loop=%s run=%+v state=%+v", closed.Status, closed.Run, state)
	}
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("runner commands=%+v", executor.commands)
	}
	if !closureSource.Called || len(closureSource.LastRequest.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("closure source request=%+v called=%v", closureSource.LastRequest, closureSource.Called)
	}
	if !closureSource.LastRequest.WaitScopeApplied ||
		!serviceStringInSetV0(closureSource.LastRequest.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(closureSource.LastRequest.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(closureSource.LastRequest.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("closure source recibio scope/evidencia no causal: request=%+v old_agent=%s failed_evidence=%s", closureSource.LastRequest, fixture.AgentRef, fixture.RequiredTestEvidenceRef)
	}
	evidence, err := testEvidenceStore.LoadRequiredTestEvidenceV0(ctx, fixture.RunRef, closureSource.LastRequest.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TaskRef != fixture.TaskRef ||
		evidence[0].TestCommand != fixture.RequiredTest ||
		evidence[0].DeliveryRef != followupDeliveryRef ||
		evidence[0].ReviewRequestID != followupReviewRequestRef ||
		evidence[0].ReviewResultRef != followupReviewResultRef ||
		evidence[0].AcceptedReviewRef != followupAcceptedReviewRef {
		t.Fatalf("evidence no causal del followup: %+v", evidence)
	}
	eventsAfterClose := eventSink.EventsV0()
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 2 {
		t.Fatalf("QualityGateRecorded esperado fallo+aceptado: got=%d events=%+v", got, eventsAfterClose)
	}
	postCloseRun, err := runStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 post close: %v", err)
	}
	if blockers := orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(postCloseRun, fixture.TaskRef); len(blockers) != 0 {
		t.Fatalf("quality gate requerido no resuelto: blockers=%v gates=%v", blockers, postCloseRun.QualityGates)
	}
	stateAfterClose, err := planStateStore.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 post close: %v", err)
	}
	closedReplanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterClose, "step-replan-or-close")
	if stateAfterClose.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		stateAfterClose.ClosureReason != "operational-closure-succeeded" ||
		closedReplanStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(closedReplanStep.RequiredTestEvidenceRefs, evidence[0].EvidenceRef) != 1 ||
		serviceStringInSetV0(closedReplanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("plan state cerrado sin evidencia passed causal: state=%+v step=%+v passed=%s failed=%s", stateAfterClose, closedReplanStep, evidence[0].EvidenceRef, fixture.RequiredTestEvidenceRef)
	}
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded duplicado: got=%d events=%+v", got, eventsAfterClose)
	}
	if got := serviceCountEventsByTypeV0(eventsAfterClose, orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 1 {
		t.Fatalf("RunClosed esperado una vez: got=%d events=%+v", got, eventsAfterClose)
	}

	replayClosureSource := &serviceOperationalDirectorClosureSourceRefsForTestV0{
		TaskRef:           fixture.TaskRef,
		ValidationRef:     closureSource.ValidationRef,
		ClosureRef:        closureSource.ClosureRef,
		DeliveryRef:       followupDeliveryRef,
		AcceptedReviewRef: followupAcceptedReviewRef,
	}
	replayPorts := ports
	replayPorts.OperationalClosureSource = replayClosureSource
	request.OccurredAt = "2026-05-22T23:40:03Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-replay"
	replay, err := ContinueAppDirectorV0(ctx, request, replayPorts)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 replay: %v", err)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		replayClosureSource.Called ||
		len(eventSink.EventsV0()) != len(eventsAfterClose) {
		t.Fatalf("replay duplico o llamo cierre: run=%+v called=%v before=%d after=%d", replay.Run, replayClosureSource.Called, len(eventsAfterClose), len(eventSink.EventsV0()))
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0RequiredTestsFailedSinReplanCausalPermaneceBloqueado(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:10:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 first: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if testsStep.Reason != "required-tests-failed" ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-failed") ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}

	request.OccurredAt = "2026-05-21T16:10:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 second: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 second: %v", err)
	}
	testsStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 ||
		testsStep.RequiredTestEvidenceRefs[0] != fixture.RequiredTestEvidenceRef {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}

	_, err = continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedConReplanSinFollowupMaterializadoBloqueaHastaReentrada(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed-pending-followup"
	replanRef := "replan-ref-app-director-required-tests-failed-pending-followup"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-pending-followup"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:     fixture.RunRef,
			GateRef:    gateRef,
			PhaseID:    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef: fixture.TaskRef,
			Decision:   orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:  []string{fixture.RequiredTestEvidenceRef},
			Summary:    "Tests requeridos fallidos.",
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan pendiente de materializar followup.",
		}),
	)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:20:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		len(state.PendingAgentRefs) != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		len(testsStep.ReplanDecisionRefs) != 0 ||
		waitStep.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v testsStep=%+v waitStep=%+v", state, testsStep, waitStep)
	}

	lateRun := fixture.Run
	lateRun.Agents = append(lateRun.Agents, followupAgentRef)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:20:01Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(lateRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 late followup: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 late followup: %v", err)
	}
	testsStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("state=%+v testsStep=%+v waitStep=%+v", state, testsStep, waitStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedBloqueadoReabreWaitConReplanPosterior(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T17:00:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 initial block: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 initial block: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed" ||
		len(testsStep.ReplanDecisionRefs) != 0 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}

	gateRef := "quality-gate-ref-app-director-required-tests-failed-late-replan"
	replanRef := "replan-ref-app-director-required-tests-failed-late-replan"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-late-replan"
	replannedRun := fixture.Run
	replannedRun.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	replannedRun.Agents = append(replannedRun.Agents, followupAgentRef)
	replannedRun.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	replannedEvents := append(append([]orquestacoreworkflow.OrchestrationEventV0(nil), fixture.Events...),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:       fixture.RunRef,
			GateRef:      gateRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef:   fixture.TaskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:    []string{fixture.RequiredTestEvidenceRef},
			Summary:      "Tests requeridos fallidos.",
			EvidenceRefs: []string{"evidence-ref-quality-gate-required-tests-failed-late-replan"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan posterior por tests requeridos fallidos.",
			EvidenceRefs:   []string{"evidence-ref-replan-required-tests-failed-late-replan"},
		}),
	)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T17:00:01Z",
		CorrelationID:              "corr-app-director-required-tests-failed-late-replan",
		OperationalDirectorPlanRef: fixture.PlanRef,
		WaitAgentRefs:              []string{fixture.AgentRef},
		WaitWaveRef:                fixture.WaveRef,
		WaitCohortRef:              fixture.CohortRef,
		WaitParentTaskRef:          fixture.ParentTaskRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(replannedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: replannedEvents},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 late replan: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) ||
		reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 late replan: %v", err)
	}
	testsStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("state=%+v", state)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		!serviceStringInSetV0(testsStep.BlockerRefs, gateRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "required-tests-failed-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedReabreWaitSplitTaskPosterior(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T10:00:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 initial block: %v", err)
	}

	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-required-tests-failed-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-required-tests-failed-split-b",
		fixture.TaskRef,
	)
	followupRefs := []string{firstFollowup.TaskID, secondFollowup.TaskID}
	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	gateRef := "quality-gate-ref-app-director-required-tests-failed-late-split"
	replanRef := "replan-ref-app-director-required-tests-failed-late-split"
	replannedRun := fixture.Run
	replannedRun.Tasks = append(replannedRun.Tasks, followupRefs...)
	replannedRun.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	replannedRun.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) +
			"#followups:" + firstFollowup.TaskID + "+" + secondFollowup.TaskID,
	}
	replannedEvents := append(append([]orquestacoreworkflow.OrchestrationEventV0(nil), fixture.Events...),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:       fixture.RunRef,
			GateRef:      gateRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef:   fixture.TaskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:    []string{fixture.RequiredTestEvidenceRef},
			Summary:      "Tests requeridos fallidos.",
			EvidenceRefs: []string{"evidence-ref-quality-gate-required-tests-failed-late-split"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
			FollowupRefs:   followupRefs,
			Summary:        "Split posterior por tests requeridos fallidos.",
			EvidenceRefs:   []string{"evidence-ref-replan-required-tests-failed-late-split"},
		}),
	)

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T10:00:01Z",
		CorrelationID:              "corr-app-director-required-tests-failed-late-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
		WaitAgentRefs:              []string{fixture.AgentRef},
		WaitWaveRef:                fixture.WaveRef,
		WaitCohortRef:              fixture.CohortRef,
		WaitParentTaskRef:          fixture.ParentTaskRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(replannedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: replannedEvents},
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(firstFollowup, secondFollowup),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 late split: %v", err)
	}
	if reentered.WaitWaveRef != firstFollowup.WaveRef ||
		reentered.WaitCohortRef != firstFollowup.CohortRef ||
		reentered.WaitParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followupAgentRefs=%+v old=%s", reentered, followupAgentRefs, fixture.AgentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 late split: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != firstFollowup.WaveRef ||
		state.ActiveCohortRef != firstFollowup.CohortRef ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("state=%+v", state)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		!serviceStringInSetV0(testsStep.BlockerRefs, gateRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "required-tests-failed-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedReentraConStateFile(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed-state-file"
	replanRef := "replan-ref-app-director-required-tests-failed-state-file"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-state-file"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:     fixture.RunRef,
			GateRef:    gateRef,
			PhaseID:    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef: fixture.TaskRef,
			Decision:   orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:  []string{fixture.RequiredTestEvidenceRef},
			Summary:    "Tests requeridos fallidos.",
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan pendiente de materializar followup.",
		}),
	)
	rootDir := t.TempDir()
	store := mustServiceStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(context.Background(), fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(context.Background(), fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
	)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:30:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	recovered := mustServiceStateFileStoreForTestV0(t, rootDir)
	blockedState, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	blockedStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-run-required-tests")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		blockedStep.Reason != "required-tests-failed" ||
		!serviceStringInSetV0(blockedStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("blockedState=%+v blockedStep=%+v", blockedState, blockedStep)
	}

	lateRun := fixture.Run
	lateRun.Agents = append(lateRun.Agents, followupAgentRef)
	if err := recovered.SaveRunV0(context.Background(), lateRun); err != nil {
		t.Fatalf("SaveRunV0 late: %v", err)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:30:01Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   recovered,
		EventReader:                recovered,
		OperationalPlanStateStore:  recovered,
		OperationalPlanStateWriter: recovered,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	reloaded := mustServiceStateFileStoreForTestV0(t, rootDir)
	state, err := reloaded.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reloaded: %v", err)
	}
	evidence, err := reloaded.LoadRequiredTestEvidenceV0(context.Background(), fixture.RunRef, []string{fixture.RequiredTestEvidenceRef})
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0 reloaded: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
		t.Fatalf("state=%+v testsStep=%+v waitStep=%+v evidence=%+v", state, testsStep, waitStep, evidence)
	}
}

func TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0ConStateBloqueadoConservaPlanRef(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	state := fixture.State
	state.PlanRef = defaultOperationalDirectorDecisionPlanRefV0(fixture.RunRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-run-required-tests"
	for index, step := range state.Steps {
		if step.StepID == "step-run-required-tests" {
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].Reason = "required-tests-failed"
			state.Steps[index].RequiredTestEvidenceRefs = []string{fixture.RequiredTestEvidenceRef}
		}
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	got, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef: fixture.RunRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	if got.OperationalDirectorPlanRef != state.PlanRef {
		t.Fatalf("plan_ref=%q, want %q", got.OperationalDirectorPlanRef, state.PlanRef)
	}
}

func TestOperationalDirectorPlanStateAfterRequiredTestsReplanV0ReabreSoloFollowupCausalEnScopeMultitarea(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed-multitask"
	replanRef := "replan-ref-app-director-required-tests-failed-multitask"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-multitask"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:     fixture.RunRef,
			GateRef:    gateRef,
			PhaseID:    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef: fixture.TaskRef,
			Decision:   orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:  []string{fixture.RequiredTestEvidenceRef},
			Summary:    "Tests requeridos fallidos en scope multitarea.",
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan parcial no aceptable para scope multitarea.",
		}),
	)
	state := fixture.State
	state.ActiveStepID = "step-run-required-tests"
	activeStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
	for index, step := range state.Steps {
		if step.StepID == "step-run-required-tests" {
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{fixture.TaskRef, "task-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].AgentRefs = []string{fixture.AgentRef, "agent-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].DeliveryRefs = []string{fixture.DeliveryRef, "delivery-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].ReviewResultRefs = []string{fixture.ReviewResultRef, "review-result-ref-app-director-required-tests-failed-multitask-002"}
			activeStep = state.Steps[index]
		}
	}

	next, changed, err := operationalDirectorPlanStateAfterRequiredTestsReplanV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-21T16:40:00Z",
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		StartAppDirectorPortsV0{EventReader: serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events}},
		state,
		activeStep,
		fixture.Run,
		[]operationalDirectorPlanAcceptedReviewMatchV0{
			{TaskRef: fixture.TaskRef},
			{TaskRef: "task-ref-app-director-required-tests-failed-multitask-002"},
		},
		[]string{fixture.RequiredTestEvidenceRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStateAfterRequiredTestsReplanV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-wait-subagents")
	if !changed ||
		next.ActiveStepID != "step-wait-subagents" ||
		next.ReplanAttempts != 1 ||
		!serviceStringInSetV0(next.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(next.PendingAgentRefs, "agent-ref-app-director-required-tests-failed-multitask-002") ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(waitStep.TaskRefs) != 1 ||
		waitStep.TaskRefs[0] != fixture.TaskRef ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) {
		t.Fatalf("next=%+v changed=%v", next, changed)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedConQualityGateReplanRetryAbreWait(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed"
	replanRef := "replan-ref-app-director-required-tests-failed"
	capacityRef := "capacity-ref-app-director-required-tests-failed-retry"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-retry"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + capacityRef + "+" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:       fixture.RunRef,
			GateRef:      gateRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef:   fixture.TaskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:    []string{fixture.RequiredTestEvidenceRef},
			Summary:      "Tests requeridos fallidos.",
			EvidenceRefs: []string{"evidence-ref-quality-gate-required-tests-failed"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{capacityRef, followupAgentRef},
			Summary:        "Replan por tests requeridos fallidos.",
			EvidenceRefs:   []string{"evidence-ref-replan-required-tests-failed"},
		}),
	)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:00:00Z",
		CorrelationID:              "corr-app-director-required-tests-failed-replan",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-21T16:00:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-failed-replan-v0") {
		t.Fatalf("state=%+v", state)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		!serviceStringInSetV0(testsStep.BlockerRefs, gateRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "required-tests-failed-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	deliveredRetryRun := fixture.Run
	deliveredRetryRun.DeliveredAgents = append(deliveredRetryRun.DeliveredAgents, followupAgentRef)
	request.OccurredAt = "2026-05-21T16:00:02Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRetryRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 delivered retry: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 delivered retry: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reviewStep.AgentRefs, fixture.AgentRef) ||
		len(reviewStep.ReviewResultRefs) != 0 {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaTestsConEvidenciaDeOtraReview(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	evidence := serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
	)
	evidence.ReviewResultRef = "review-result-ref-otra-review"
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(evidence)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:50Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.ActiveStepID != "step-run-required-tests" ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		!serviceStringInSetV0(state.BlockerRefs, "required-tests-evidence-missing") ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-evidence-missing" ||
		len(testsStep.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewAReplanSinTests(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	fixture.State.Mode = orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:21:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v replanStep=%+v", state, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewFueraDeScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[0] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
		DeliveryRef:  fixture.DeliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ref-review-out-of-scope",
		AgentRef:     "agent-ref-review-out-of-scope",
		Summary:      "Entrega fuera del scope activo.",
		EvidenceRefs: []string{"evidence-ref-review-out-of-scope"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:22:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("state=%+v", state)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewConOutboxPendiente(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:23:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		PendingOutboxCount: 1,
		Run:                fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("state=%+v", state)
	}
}

func TestContinueAppDirectorV0PlanStateReentradaRequiereStore(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-missing-store"
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(serviceRunForWaitRefsTestV0(runRef))
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	_, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T13:57:00Z",
		CorrelationID:              "corr-app-director-operational-plan-reentry-missing-store",
		OperationalDirectorPlanRef: "plan-ref-missing-store",
	}, StartAppDirectorPortsV0{
		RunStore:     store,
		EventSink:    sink,
		OutboxLedger: ledger,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "ports.operational_plan_state_store" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraReviewConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if got.WaitWaveRef != fixture.WaveRef ||
		got.WaitCohortRef != fixture.CohortRef ||
		got.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(got.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("request=%+v fixture=%+v", got, fixture)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0NoReentraReviewChangesRequested(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	fixture.State.ReplanAttempts = 1
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		if step.StepID != "step-review-deliveries" {
			continue
		}
		step.Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
		step.DeliveryRefs = []string{fixture.DeliveryRef}
		step.ReviewResultRefs = []string{fixture.ReviewResultRef}
		step.ReworkRequestRefs = []string{fixture.ReworkRequestRef}
		step.ReplanDecisionRefs = []string{fixture.ReplanDecisionRef}
		step.BlockerRefs = []string{"review-rework-replan-recorded"}
		step.Reason = "review-rework-replan-recorded"
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreReviewChangesRequestedConFollowupTardio(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupAgentRef := "agent-ref-app-director-operational-plan-state-review-late-retry-followup"
	capacityRef := "capacity-ref-app-director-operational-plan-state-review-late-retry-followup"
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + capacityRef + "+" + followupAgentRef,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{capacityRef, followupAgentRef},
		Summary:        "Replan retry por review negativa con agente tardio.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-late-retry-followup"},
	})
	fixture.State.ReplanAttempts = 1
	fixture.State.EvidenceRefs = []string{"evidence-ref-app-director-operational-plan-state-review-rework-replan-v0"}
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		if step.StepID != "step-review-deliveries" {
			continue
		}
		step.Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
		step.DeliveryRefs = []string{fixture.DeliveryRef}
		step.ReviewResultRefs = []string{fixture.ReviewResultRef}
		step.ReworkRequestRefs = []string{fixture.ReworkRequestRef}
		step.ReplanDecisionRefs = []string{fixture.ReplanDecisionRef}
		step.BlockerRefs = []string{"review-rework-replan-recorded"}
		step.Reason = "review-rework-replan-recorded"
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:15:00Z",
		CorrelationID:              "corr-app-director-review-late-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0") ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followup-agents") ||
		!serviceStringInSetV0(waitStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	replayed, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:15:01Z",
		CorrelationID:              "corr-app-director-review-late-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if !serviceStringInSetV0(replayed.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(replayed.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("replayed=%+v followup=%s old=%s", replayed, followupAgentRef, fixture.AgentRef)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if state.ReplanAttempts != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0") != 1 {
		t.Fatalf("state replay=%+v", state)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraRunRequiredTestsConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		switch step.StepID {
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.AcceptedReviewRefs = []string{fixture.AcceptedReviewRef}
		case "step-run-required-tests":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.TaskRefs = []string{fixture.TaskRef}
			step.AgentRefs = []string{fixture.AgentRef}
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.BlockerRefs = []string{"required-tests-pending"}
		}
	}
	fixture.State.ActiveStepID = "step-run-required-tests"
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if got.WaitWaveRef != fixture.WaveRef ||
		got.WaitCohortRef != fixture.CohortRef ||
		got.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(got.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("request=%+v fixture=%+v", got, fixture)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RechazaActiveStepNoSoportado(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-unsupported"
	planRef := "plan-ref-reentry-unsupported"
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-ref-reentry-unsupported",
		PlanRef:         planRef,
		RequestRef:      "request-ref-reentry-unsupported",
		RunRef:          runRef,
		ProjectRef:      "orquesta",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-replan-or-close",
		ActiveWaveRef:   "wave-reentry-unsupported",
		ActiveCohortRef: "cohort-reentry-unsupported",
		ObservedAt:      "2026-05-17T13:58:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:    "step-replan-or-close",
			Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:   "wave-reentry-unsupported",
			CohortRef: "cohort-reentry-unsupported",
		}},
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RespetaWaitExplicito(t *testing.T) {
	request := ContinueAppDirectorRequestV0{
		RunRef:                     "run-app-director-operational-plan-explicit-wait",
		OperationalDirectorPlanRef: "plan-ref-explicit-wait",
		WaitAgentRefs:              []string{"agent-ref-explicit"},
	}
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if len(got.WaitAgentRefs) != 1 || got.WaitAgentRefs[0] != "agent-ref-explicit" {
		t.Fatalf("request=%+v", got)
	}
}

func TestContinueAppDirectorV0RecuperaWaitStatePersistidoSinAmpliarCohorte(t *testing.T) {
	runRef := "run-app-director-operational-plan-wait-recovery"
	planRef := "plan-ref-wait-recovery"
	waitRef := "wait-ref-old"
	taskA := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-wait-recovery-a", "wave-01", "cohort-a")
	taskB := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-wait-recovery-b", "wave-01", "cohort-a")
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskID)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskB.TaskID)
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-ref-wait-recovery",
		PlanRef:         planRef,
		RequestRef:      "request-ref-wait-recovery",
		RunRef:          runRef,
		ProjectRef:      "orquesta",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-wait-subagents",
		ActiveWaveRef:   "wave-01",
		ActiveCohortRef: "cohort-a",
		PendingAgentRefs: []string{
			agentA,
			agentB,
		},
		ObservedAt: "2026-05-17T13:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:           "step-wait-subagents",
			Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:          "wave-01",
			CohortRef:        "cohort-a",
			TaskRefs:         []string{taskA.TaskID, taskB.TaskID},
			WaitRefs:         []string{waitRef},
			AgentRefs:        []string{agentA, agentB},
			PendingAgentRefs: []string{agentA, agentB},
			BlockerRefs:      []string{"wait-subagents"},
		}},
	}
	waitState := orquestacionnucleoapp.WorkflowTaskWaitStateV0{
		SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          waitRef,
		RunRef:           runRef,
		ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
		CohortRef:        "cohort-a",
		WaveRef:          "wave-01",
		TaskRefs:         []string{taskA.TaskID},
		AgentRefs:        []string{agentA},
		PendingAgentRefs: []string{agentA},
		Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
		Attempt:          1,
		ObservedAt:       "2026-05-17T13:58:00Z",
	}
	run := serviceRunForWaitRefsTestV0(runRef, taskA.TaskID, taskB.TaskID)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	waitStateStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(waitState)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(taskA, taskB)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-17T14:00:00Z",
		CorrelationID:              "corr-wait-recovery",
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                  runStore,
		DirectorTaskStore:         taskStore,
		WaitStateStore:            waitStateStore,
		OperationalPlanStateStore: planStateStore,
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if len(reentered.WaitAgentRefs) != 1 ||
		reentered.WaitAgentRefs[0] != agentA ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentB) ||
		reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" {
		t.Fatalf("reentered=%+v agentA=%s agentB=%s", reentered, agentA, agentB)
	}
	loopRequest, err := existingDirectorLoopRequestV0(context.Background(), reentered, ports)
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	if len(loopRequest.WaitAgentRefs) != 1 ||
		loopRequest.WaitAgentRefs[0] != agentA ||
		serviceStringInSetV0(loopRequest.WaitAgentRefs, agentB) {
		t.Fatalf("loop wait refs=%+v agentA=%s agentB=%s", loopRequest.WaitAgentRefs, agentA, agentB)
	}
}

func TestMaterializeContinueOperationalDirectorPlanV0UsaFaseActualSiNoSeIndica(t *testing.T) {
	runRef := "run-app-director-operational-plan-phase-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.FunctionContracts = []string{contractRef}
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()

	materialized, err := materializeContinueOperationalDirectorPlanV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:47:00Z",
		CorrelationID:           "corr-app-director-operational-plan-phase-001",
		RequestedBy:             "orquesta-app-director-test",
		OperationalDirectorPlan: serviceOperationalDirectorPlanForContinueTestV0(t, runRef),
	}, StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:         orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("materializeContinueOperationalDirectorPlanV0: %T %#v", err, err)
	}
	if len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	if materialized.Tasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 {
		t.Fatalf("phase_id=%s", materialized.Tasks[0].PhaseID)
	}
}

func serviceOperationalDirectorWideWaveWaitStateForTestV0(
	runRef string,
	planRef string,
	parentTaskRef string,
	waveRef string,
	cohortRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	taskRefs := make([]string, 0, len(tasks))
	agentRefs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskRefs = append(taskRefs, task.TaskRef)
		agentRefs = append(agentRefs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef))
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-" + planRef,
		PlanRef:             planRef,
		RequestRef:          "request-ref-" + planRef,
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-wait-subagents",
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		PendingAgentRefs:    agentRefs,
		RequiredTestRefs:    []string{"go test ./..."},
		ObservedAt:          "2026-05-22T13:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:        "step-launch-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      taskRefs,
				AgentRefs:     agentRefs,
			},
			{
				StepID:           "step-wait-subagents",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          waveRef,
				CohortRef:        cohortRef,
				ParentTaskRef:    parentTaskRef,
				TaskRefs:         taskRefs,
				AgentRefs:        agentRefs,
				WaitRefs:         []string{"wait-ref-" + planRef},
				PendingAgentRefs: agentRefs,
				BlockerRefs:      []string{"wait-subagents"},
				Reason:           "wait-subagents-running",
			},
			{
				StepID:        "step-review-deliveries",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-run-required-tests",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-replan-or-close",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
		},
	}
}

type serviceOperationalDirectorPlanStateReviewFixtureV0 struct {
	RunRef                  string
	PlanRef                 string
	TaskRef                 string
	AgentRef                string
	WaveRef                 string
	CohortRef               string
	ParentTaskRef           string
	DeliveryRef             string
	ReviewRequestID         string
	ReviewResultRef         string
	AcceptedReviewRef       string
	ReworkRequestRef        string
	ReplanDecisionRef       string
	RequiredTest            string
	RequiredTestEvidenceRef string
	Run                     orquestacoreworkflow.OrchestrationRunV0
	State                   orquestacionnucleoapp.OperationalDirectorPlanStateV0
	Events                  []orquestacoreworkflow.OrchestrationEventV0
}

func serviceOperationalDirectorPlanStateReviewFixtureForTestV0(
	t *testing.T,
	withRequiredTests bool,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	runRef := "run-app-director-operational-plan-state-review-accepted"
	planRef := "plan-ref-app-director-operational-plan-state-review-accepted"
	taskRef := "task-ref-app-director-operational-plan-state-review-accepted"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	waveRef := "wave-app-director-operational-plan-state-review-accepted"
	cohortRef := "cohort-app-director-operational-plan-state-review-accepted"
	parentTaskRef := "parent-task-ref-app-director-operational-plan-state-review-accepted"
	deliveryRef := "delivery-ref-app-director-operational-plan-state-review-accepted"
	reviewRequestID := "review-request-ref-app-director-operational-plan-state-review-accepted"
	reviewResultRef := "review-result-ref-app-director-operational-plan-state-review-accepted"
	acceptedReviewRef := "accepted-review-ref-app-director-operational-plan-state-review-accepted"
	requiredTest := "go test -count=1 ./modulos/orquesta-app-director-service"
	requiredTestEvidenceRef := "test-evidence-ref-" + deliveryRef
	requiredTests := []string(nil)
	reviewResultEvidenceRefs := []string{"evidence-ref-review-result-accepted"}
	if withRequiredTests {
		requiredTests = []string{requiredTest}
		reviewResultEvidenceRefs = append(reviewResultEvidenceRefs, requiredTestEvidenceRef)
	}
	run := serviceRunForWaitRefsTestV0(runRef, taskRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{reviewRequestID}
	run.ReviewResults = []string{reviewResultRef + "#review_result:accepted#review_request:" + reviewRequestID + "#delivery:" + deliveryRef}
	run.AcceptedReviews = []string{acceptedReviewRef}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-app-director-operational-plan-state-review-accepted",
		PlanRef:             planRef,
		RequestRef:          "request-ref-app-director-operational-plan-state-review-accepted",
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-review-deliveries",
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		RequiredTestRefs:    requiredTests,
		ObservedAt:          "2026-05-17T14:19:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:        "step-launch-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-wait-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
				WaitRefs:      []string{"wait-ref-app-director-operational-plan-state-review-accepted"},
			},
			{
				StepID:        "step-review-deliveries",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-run-required-tests",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-replan-or-close",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
		},
	}
	events := []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  deliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       taskRef,
			AgentRef:     agentRef,
			Summary:      "Entrega del scope activo.",
			EvidenceRefs: []string{"evidence-ref-delivery-review-accepted"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestID,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Review del scope activo.",
			EvidenceRefs:    []string{"evidence-ref-review-requested"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestID,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review aceptada.",
			EvidenceRefs:    reviewResultEvidenceRefs,
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestID,
			DeliveryRef:       deliveryRef,
			Summary:           "Review aceptada.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted"},
		}),
	}
	return serviceOperationalDirectorPlanStateReviewFixtureV0{
		RunRef:                  runRef,
		PlanRef:                 planRef,
		TaskRef:                 taskRef,
		AgentRef:                agentRef,
		WaveRef:                 waveRef,
		CohortRef:               cohortRef,
		ParentTaskRef:           parentTaskRef,
		DeliveryRef:             deliveryRef,
		ReviewRequestID:         reviewRequestID,
		ReviewResultRef:         reviewResultRef,
		AcceptedReviewRef:       acceptedReviewRef,
		RequiredTest:            requiredTest,
		RequiredTestEvidenceRef: requiredTestEvidenceRef,
		Run:                     run,
		State:                   state,
		Events:                  events,
	}
}

func serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
	t *testing.T,
	status orquestacoreworkflow.ReviewResultStatusV0,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.ReviewResultRef = "review-result-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.AcceptedReviewRef = ""
	fixture.ReworkRequestRef = "rework-request-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.ReplanDecisionRef = "replan-decision-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.Run.ReviewResults = []string{
		fixture.ReviewResultRef + "#review_result:" + string(status) +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.AcceptedReviews = nil
	fixture.Run.ReworkRequests = []string{
		fixture.ReworkRequestRef + "#review_result:" + fixture.ReviewResultRef +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + fixture.TaskRef,
	}
	fixture.Events = []orquestacoreworkflow.OrchestrationEventV0{
		fixture.Events[0],
		fixture.Events[1],
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: fixture.ReviewResultRef,
			ReviewRequestID: fixture.ReviewRequestID,
			DeliveryRef:     fixture.DeliveryRef,
			Status:          status,
			Summary:         "Review pide cambios.",
			EvidenceRefs:    []string{"evidence-ref-review-result-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 4, orquestacoreworkflow.OrchestrationEventReworkRequestedV0, orquestacoreworkflow.ReworkRequestedPayloadV0{
			ReworkRequestRef: fixture.ReworkRequestRef,
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  fixture.ReviewResultRef,
			ReviewRequestID:  fixture.ReviewRequestID,
			DeliveryRef:      fixture.DeliveryRef,
			Summary:          "Rework requerido por review negativa.",
			EvidenceRefs:     []string{"evidence-ref-rework-requested-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      fixture.ReplanDecisionRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      fixture.ReworkRequestRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{fixture.TaskRef},
			Summary:        "Replan por review negativa.",
			EvidenceRefs:   []string{"evidence-ref-replan-decision-" + string(status)},
		}),
	}
	return fixture
}

func serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       fixture.RequiredTestEvidenceRef,
		RunRef:            fixture.RunRef,
		TaskRef:           fixture.TaskRef,
		TestCommand:       fixture.RequiredTest,
		Status:            status,
		DeliveryRef:       fixture.DeliveryRef,
		ReviewRequestID:   fixture.ReviewRequestID,
		ReviewResultRef:   fixture.ReviewResultRef,
		AcceptedReviewRef: fixture.AcceptedReviewRef,
		OccurredAt:        "2026-05-17T14:20:00Z",
		EvidenceRefs:      []string{"test-output-ref-" + fixture.DeliveryRef},
	}
}

func serviceOperationalDirectorWorkflowTaskForFixtureV0(
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             fixture.TaskRef,
		RunID:              fixture.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Tarea del ciclo integrado del Director Operativo",
		Summary:            "Trabajo de programacion con review, runner requerido y cierre causal.",
		WriteSet:           []string{"modulos/orquesta-app-director-service"},
		AcceptanceCriteria: []string{"Review aceptada, test requerido evidenciado y cierre causal."},
		RequiredTests:      []string{fixture.RequiredTest},
		ParentTaskRef:      fixture.ParentTaskRef,
		CohortRef:          fixture.CohortRef,
		WaveRef:            fixture.WaveRef,
		DelegationDepth:    1,
		MaxChildAgents:     0,
	}
}

type serviceOperationalDirectorClosureSourceFromRequestForTestV0 struct {
	Fixture     serviceOperationalDirectorPlanStateReviewFixtureV0
	LastRequest AppDirectorOperationalClosureRequestV0
	Called      bool
}

func (source *serviceOperationalDirectorClosureSourceFromRequestForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   source.Fixture.TaskRef,
		DeliveryRef:              source.Fixture.DeliveryRef,
		AcceptedReviewRef:        source.Fixture.AcceptedReviewRef,
		ValidationRef:            "validation-ref-service-review-runner-close-001",
		ClosureRef:               "closure-ref-service-review-runner-close-001",
		RequiredTestEvidenceRefs: append([]string(nil), request.RequiredTestEvidenceRefs...),
		EvidenceRefs:             compactServiceRefsV0(append(request.EvidenceRefs, "evidence-ref-service-review-runner-close-001")),
	}, true, nil
}

type serviceOperationalDirectorClosureSourceRefsForTestV0 struct {
	TaskRef           string
	DeliveryRef       string
	AcceptedReviewRef string
	ValidationRef     string
	ClosureRef        string
	LastRequest       AppDirectorOperationalClosureRequestV0
	Called            bool
}

func (source *serviceOperationalDirectorClosureSourceRefsForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   source.TaskRef,
		DeliveryRef:              source.DeliveryRef,
		AcceptedReviewRef:        source.AcceptedReviewRef,
		ValidationRef:            source.ValidationRef,
		ClosureRef:               source.ClosureRef,
		RequiredTestEvidenceRefs: append([]string(nil), request.RequiredTestEvidenceRefs...),
		EvidenceRefs:             compactServiceRefsV0(append(request.EvidenceRefs, "evidence-ref-service-closure-source-refs-test-v0")),
	}, true, nil
}

type serviceOperationalDirectorSplitReviewReworkReplanSourceV0 struct {
	Fixture    serviceOperationalDirectorPlanStateReviewFixtureV0
	SplitTasks []orquestacoreworkflow.WorkflowTaskV0
}

func (source serviceOperationalDirectorSplitReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	if !serviceRunProjectionHasRefPrefixV0(request.Run.ReworkRequests, source.Fixture.ReworkRequestRef, "#review_result:") ||
		serviceRunHasAnyRefV0(request.Run.Agents, serviceWorkflowTaskAgentRefsV0(source.SplitTasks)...) {
		return nil, nil
	}
	return []orquestacionnucleoapp.ReviewReworkReplanPlanV0{{
		CandidateRef:     "candidate-ref-app-director-review-rework-durable-split",
		ReplanRef:        source.Fixture.ReplanDecisionRef,
		SignalRef:        "signal-ref-app-director-review-rework-durable-split",
		ReworkRequestRef: source.Fixture.ReworkRequestRef,
		TaskRef:          source.Fixture.TaskRef,
		ReasonRef:        "review-rework-durable-split-required",
		RequestedAction:  orquestacorereplanner.ReplanActionSplitTaskV0,
		Summary:          "Dividir retrabajo de review negativa en microtareas durables.",
		EvidenceRefs:     []string{"evidence-ref-app-director-review-rework-durable-split"},
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: source.Fixture.ReviewResultRef,
			ReviewRequestID: source.Fixture.ReviewRequestID,
			DeliveryRef:     source.Fixture.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
			Summary:         "Review negativa requiere split.",
			EvidenceRefs:    []string{"evidence-ref-review-result-durable-split"},
		},
		SplitTasks: append([]orquestacoreworkflow.WorkflowTaskV0(nil), source.SplitTasks...),
	}}, nil
}

type changesRequestedServiceReviewGateSourceV0 struct {
	Fixture serviceOperationalDirectorPlanStateReviewFixtureV0
	Called  bool
}

func (source *changesRequestedServiceReviewGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	_ orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	source.Called = true
	return []orquestacionnucleoapp.ReviewGateObservationV0{{
		ReviewRequestID:  source.Fixture.ReviewRequestID,
		ReviewResultRef:  source.Fixture.ReviewResultRef,
		ReworkRequestRef: source.Fixture.ReworkRequestRef,
		DeliveryRef:      source.Fixture.DeliveryRef,
		PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:           orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		Summary:          "Review pide split duradero de followups.",
		EvidenceRefs: []string{
			"evidence-ref-service-review-gate-changes-requested",
		},
	}}, nil
}

type acceptedServiceReviewGateSourceV0 struct {
	Fixture serviceOperationalDirectorPlanStateReviewFixtureV0
	Called  bool
}

func (source *acceptedServiceReviewGateSourceV0) BuildReviewGateObservationsV0(
	_ context.Context,
	_ orquestacionnucleoapp.ReviewGateObservationRequestV0,
) ([]orquestacionnucleoapp.ReviewGateObservationV0, error) {
	source.Called = true
	return []orquestacionnucleoapp.ReviewGateObservationV0{{
		ReviewRequestID:   source.Fixture.ReviewRequestID,
		ReviewResultRef:   source.Fixture.ReviewResultRef,
		AcceptedReviewRef: source.Fixture.AcceptedReviewRef,
		DeliveryRef:       source.Fixture.DeliveryRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
		Status:            orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:           "Review aceptada por fuente inyectada.",
		EvidenceRefs: []string{
			"evidence-ref-service-review-gate-accepted",
			source.Fixture.RequiredTestEvidenceRef,
		},
	}}, nil
}

func serviceWorkflowTaskAgentRefsV0(tasks []orquestacoreworkflow.WorkflowTaskV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskID))
	}
	return compactServiceRefsV0(refs)
}

func serviceRunHasAnyRefV0(values []string, refs ...string) bool {
	for _, ref := range refs {
		if serviceStringInSetV0(values, ref) {
			return true
		}
	}
	return false
}

func serviceRunProjectionHasRefPrefixV0(values []string, ref string, marker string) bool {
	ref = strings.TrimSpace(ref)
	marker = strings.TrimSpace(marker)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == ref || strings.HasPrefix(value, ref+marker) {
			return true
		}
	}
	return false
}

type serviceContinueWaiterForTestV0 struct {
	Continue bool
	Calls    int
}

func (waiter *serviceContinueWaiterForTestV0) WaitExternalProgressV0(
	_ context.Context,
	_ orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	waiter.Calls++
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     waiter.Continue,
		EvidenceRefs: []string{"evidence-ref-service-continue-waiter-v0"},
	}, nil
}

func serviceOperationalDirectorReplanSplitTaskForTestV0(
	runRef string,
	taskRef string,
	parentTaskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Tarea split de replan",
		Summary:       "Corregir una parte acotada tras review negativa.",
		WriteSet:      []string{"app/replan_split_" + taskRef + ".go"},
		AcceptanceCriteria: []string{
			"Entrega corregida y trazable para nueva review.",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-director-service"},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:rework-split:v0", FunctionName: "NewWorkflowTaskV0"},
		},
		ParentTaskRef:   parentTaskRef,
		CohortRef:       "cohort-ref-app-director-replan-split-followups",
		WaveRef:         "wave-ref-app-director-replan-split-followups",
		DelegationDepth: 1,
		MaxChildAgents:  0,
	}
}

type fakeServiceRequiredTestCommandExecutorV0 struct {
	results  map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0
	commands []string
}

func (executor *fakeServiceRequiredTestCommandExecutorV0) RunRequiredTestCommandV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	executor.commands = append(executor.commands, request.TestCommand)
	return executor.results[request.TestCommand], nil
}

func serviceOperationalDirectorPlanStateEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	eventType string,
	payload any,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestacoreworkflow.OrchestrationEventV0{
		EventID:        eventType + "-plan-state-review-" + string(rune('0'+sequence)),
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		OccurredAt:     "2026-05-17T14:19:00Z",
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}

func serviceOperationalDirectorPlanStateStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no encontrado en %+v", stepID, state.Steps)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func serviceOperationalDirectorPlanForContinueTestV0(
	t *testing.T,
	runRef string,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	return serviceOperationalDirectorPlanForContinueWithParallelTestV0(t, runRef, 1)
}

func serviceOperationalDirectorPlanWithParallelLaunchesForTestV0(
	t *testing.T,
	runRef string,
	maxParallelAgents int,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	return serviceOperationalDirectorPlanForContinueWithParallelTestV0(t, runRef, maxParallelAgents)
}

func serviceOperationalDirectorPlanForContinueWithParallelTestV0(
	t *testing.T,
	runRef string,
	maxParallelAgents int,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:        "req-app-director-operational-plan-001",
		RunRef:            runRef,
		ProjectRef:        "orquesta",
		Objective:         "Cerrar tramo launch wait del Director Operativo.",
		Mode:              orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		WorktreeRef:       "worktree-ref-app-director-operational-plan",
		WorktreeIsolated:  true,
		BranchRef:         "branch-ref-app-director-operational-plan",
		MaxParallelAgents: maxParallelAgents,
		WriteSet:          serviceOperationalDirectorPlanWriteSetForTestV0(maxParallelAgents),
		RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-app-director-service"},
	})
	if !result.Accepted || !result.ReadyToLaunch {
		t.Fatalf("plan no listo: %+v", result)
	}
	return result.Plan
}

func serviceOperationalDirectorPlanWriteSetForTestV0(maxParallelAgents int) []string {
	if maxParallelAgents <= 1 {
		return []string{"docs/director_operational_plan_test.md"}
	}
	return []string{
		"docs/director_operational_plan_test.md",
		"modulos/orquesta-app-director-service/operational_director_v0.go",
		"modulos/orquesta-app-director-service/operational_director_v0_test.go",
	}
}
