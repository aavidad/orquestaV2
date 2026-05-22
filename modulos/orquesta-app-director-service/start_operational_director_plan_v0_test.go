package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStartAppDirectorV0OperationalDirectorPlanMaterializaYEsperaScope(t *testing.T) {
	runRef := "run-app-director-start-operational-plan-001"
	contractRef := "contract:function:operational-director-start:v0"
	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = ""
	request.ProjectRef = "project-app-director-start-operational-plan-001"
	request.CorrelationID = "corr-app-director-start-operational-plan-001"
	request.AppSpecRequest.RequestID = "request-ref-app-director-start-operational-plan-001"
	request.OperationalDirectorPlan = serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	request.OperationalDirectorFunctionContractRefs = []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
		ContractRef:  contractRef,
		FunctionName: "ExecuteOperationalDirectorStartV0",
	}}

	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()

	result, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                    store,
		EventSink:                   sink,
		OutboxLedger:                ledger,
		DirectorTaskStore:           taskStore,
		WaitStateWriter:             waitStore,
		WaitStateStore:              waitStore,
		OperationalPlanStateWriter:  planStateStore,
		OperationalPlanStateStore:   planStateStore,
		WorkflowTaskDefaultCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Run.RunID != runRef {
		t.Fatalf("run_id=%q", result.Run.RunID)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("run tasks=%v", result.Run.Tasks)
	}
	taskRef := result.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !serviceStringInSetV0(result.StartedAgents, agentRef) ||
		serviceStringInSetV0(result.StartedAgents, result.DirectorTask.AgentRequestID) {
		t.Fatalf("started=%v workflow_agent=%s director_agent=%s", result.StartedAgents, agentRef, result.DirectorTask.AgentRequestID)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	task := tasks[0]
	if task.WaveRef == "" || task.CohortRef == "" ||
		len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != contractRef {
		t.Fatalf("task=%+v", task)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, request.OperationalDirectorPlan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != "running" ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, agentRef) {
		t.Fatalf("state=%+v wait_step=%+v", state, waitStep)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(
		context.Background(),
		runRef,
		appDirectorWaitRefV0(runRef, appDirectorWaitFilterV0{
			WaveRef:   task.WaveRef,
			CohortRef: task.CohortRef,
		}, request.CorrelationID),
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(waitState.AgentRefs, agentRef) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, agentRef) {
		t.Fatalf("wait_state=%+v agent=%s", waitState, agentRef)
	}
	for _, eventType := range []string{
		orquestacoreworkflow.OrchestrationEventVoteRequestedV0,
		orquestacoreworkflow.OrchestrationEventArchitectureDecisionAcceptedV0,
		orquestacoreworkflow.OrchestrationEventFunctionContractPublishedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
	} {
		if !serviceHasEventTypeV0(sink.EventsV0(), eventType) {
			t.Fatalf("eventos sin %s: %+v", eventType, sink.EventsV0())
		}
	}
}
