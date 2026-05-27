package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
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
