package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func TestCodexStackAutoprogrammingExecutorV0UsaPuertosDelStackYDevuelveContinue(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackAutoprogrammingExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:       autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:    "2026-05-22T11:00:00Z",
		CorrelationID: "corr-autoprogramming-stack-executor-001",
		RequestedBy:   "orquesta-stack-executor-test",
		MaxBursts:     3,
		MaxCommands:   5,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Accepted || len(result.Tasks) != 1 || len(result.WaitAgentRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Continue.RunRef != result.Run.RunID ||
		result.Continue.CorrelationID != "corr-autoprogramming-stack-executor-001" ||
		result.Continue.WaitAgentRefs[0] != result.WaitAgentRefs[0] {
		t.Fatalf("continue=%+v wait=%v", result.Continue, result.WaitAgentRefs)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), result.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		run.Tasks[0] != result.Tasks[0].TaskID {
		t.Fatalf("run=%+v tasks=%+v", run, result.Tasks)
	}
	storedTasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		context.Background(),
		result.Run.RunID,
		result.Run.Tasks,
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 || storedTasks[0].TaskID != result.Tasks[0].TaskID {
		t.Fatalf("stored_tasks=%+v result_tasks=%+v", storedTasks, result.Tasks)
	}
}

func TestCodexStackAutoprogrammingExecutorV0RequiereStack(t *testing.T) {
	result, err := NewCodexStackAutoprogrammingExecutorV0(nil).Execute(
		context.Background(),
		AutoprogrammingBridgeRequestV0{Request: autoprogrammingBridgeRequestForTestV0()},
	)
	if err == nil {
		t.Fatalf("err nil result=%+v", result)
	}
}

func TestPrepareAutoprogrammingRunFromStackV0PropagaValidacionDePuertos(t *testing.T) {
	_, err := PrepareAutoprogrammingRunFromStackV0(
		context.Background(),
		StackV0{Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		}},
		AutoprogrammingBridgeRequestV0{Request: autoprogrammingBridgeRequestForTestV0()},
	)
	if err == nil {
		t.Fatalf("err nil")
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0PreparaRunYSupervisorArranca(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepareInput := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-api-001",
		CorrelationID:          "corr-autoprogramming-api-001",
		OccurredAt:             "2026-05-22T11:15:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	if !prepared.Accepted ||
		prepared.RunRef == "" ||
		len(prepared.WorkflowTaskRefs) != 1 ||
		len(prepared.WaitAgentRefs) != 1 ||
		prepared.Continue == nil ||
		prepared.Continue.RunRef != prepared.RunRef {
		t.Fatalf("prepared=%+v", prepared)
	}

	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-autoprogramming-supervisor-001",
		CorrelationID:        "corr-autoprogramming-api-001",
		RunRef:               prepared.RunRef,
		MaxTicks:             1,
		MaxRunsPerTick:       1,
		MaxExecutions:        1,
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 3,
		MaxCommands:          5,
		MaxOutboxPerCycle:    5,
		AllowRepeatedRuns:    true,
	})
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("started_agents=%v wait=%v", run.StartedAgents, prepared.WaitAgentRefs)
	}

	repeated := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	run, err = stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 repeated: %v", err)
	}
	if repeated.RunRef != prepared.RunRef ||
		repeated.WaitAgentRefs[0] != prepared.WaitAgentRefs[0] ||
		runtime.launchCountV0() != 1 ||
		!autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("repeated=%+v run=%+v launches=%d", repeated, run, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0EncolaYSupervisorGlobalArranca(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepareInput := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-queue-api-001",
		CorrelationID:          "corr-autoprogramming-queue-api-001",
		OccurredAt:             "2026-05-22T11:25:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, prepareInput)
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].AppRef != prepared.ProjectRef ||
		ranking.Ranked[0].PriorityScore != DefaultRunQueuePriorityScoreV0 {
		t.Fatalf("ranking=%+v prepared=%+v", ranking, prepared)
	}

	supervisor := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-autoprogramming-global-supervisor-001",
		CorrelationID:        "corr-autoprogramming-queue-api-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             1,
		MaxRunsPerTick:       1,
		MaxExecutions:        1,
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 3,
		MaxCommands:          5,
		MaxOutboxPerCycle:    5,
		AllowRepeatedRuns:    true,
	})
	if supervisor.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		supervisor.RunRef != prepared.RunRef ||
		supervisor.Last.SessionRef != prepared.RunRef ||
		runtime.launchCountV0() != 1 {
		t.Fatalf("supervisor=%+v launches=%d", supervisor, runtime.launchCountV0())
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) {
		t.Fatalf("started_agents=%v wait=%v", run.StartedAgents, prepared.WaitAgentRefs)
	}
}

func TestCodexStackAutoprogrammingSupervisorGlobalReemplazaAskDirectorPerdido(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-global-stopped-001",
		CorrelationID:          "corr-autoprogramming-global-stopped-001",
		OccurredAt:             "2026-05-22T11:35:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}
	first := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-autoprogramming-global-stopped-supervisor-001",
		CorrelationID:        "corr-autoprogramming-global-stopped-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             1,
		MaxRunsPerTick:       1,
		MaxExecutions:        1,
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 3,
		MaxCommands:          5,
		MaxOutboxPerCycle:    5,
		AllowRepeatedRuns:    true,
	})
	if first.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 || runtime.launchCountV0() != 1 {
		t.Fatalf("first=%+v launches=%d", first, runtime.launchCountV0())
	}
	agentRef := prepared.WaitAgentRefs[0]
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, prepared.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)

	second := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-autoprogramming-global-stopped-supervisor-002",
		CorrelationID:        "corr-autoprogramming-global-stopped-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             1,
		MaxRunsPerTick:       1,
		MaxExecutions:        1,
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		AllowRepeatedRuns:    true,
	})
	if second.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		t.Fatalf("second=%+v", second)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.LostAgents, agentRef) ||
		autoprogrammingBridgeStringInSetForTestV0(run.StoppedAgents, agentRef) ||
		autoprogrammingBridgeStringInSetForTestV0(run.ConfirmedStoppedAgents, agentRef) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("supervisor global no activo revision recuperable: lost=%v stopped=%v confirmed=%v assessments=%v", run.LostAgents, run.StoppedAgents, run.ConfirmedStoppedAgents, run.AgentAssessments)
	}

	third := postRunSupervisorStackV0(t, stack, orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-autoprogramming-global-stopped-supervisor-003",
		CorrelationID:        "corr-autoprogramming-global-stopped-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             1,
		MaxRunsPerTick:       1,
		MaxExecutions:        1,
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		AllowRepeatedRuns:    true,
	})
	if third.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 {
		t.Fatalf("third=%+v", third)
	}
	run, err = stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 third: %v", err)
	}
	if len(run.StartedAgents) != 2 || runtime.launchCountV0() != 2 ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) ||
		!codexStackRefsContainPartV0(run.ReplanDecisions, "#action:"+string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)) {
		t.Fatalf("supervisor no reemplazo ask_director perdido: started=%v launches=%d assessments=%v replans=%v run=%+v",
			run.StartedAgents,
			runtime.launchCountV0(),
			run.AgentAssessments,
			run.ReplanDecisions,
			run,
		)
	}
}

func TestCodexStackAutoprogrammingSupervisorResidenteReemplazaAskDirectorPerdido(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-resident-lost-001",
		CorrelationID:          "corr-autoprogramming-resident-lost-001",
		OccurredAt:             "2026-05-22T11:45:00Z",
		RequestedBy:            "orquesta-stack-resident-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}
	command := orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          DefaultRunQueueRefV0,
		MaxTicks:          1,
		MaxRunsPerTick:    1,
		MaxExecutions:     1,
		AllowRepeatedRuns: true,
		CorrelationID:     "corr-autoprogramming-resident-lost-001",
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            8,
			MaxStepsPerBurst:     6,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxDecisionCycles:    1,
		},
	}
	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil || runtime.launchCountV0() != 1 {
		t.Fatalf("first result=%+v err=%v launches=%d", result, err, runtime.launchCountV0())
	}
	agentRef := prepared.WaitAgentRefs[0]
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, prepared.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)

	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil {
		t.Fatalf("second result=%+v err=%v", result, err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 second: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.LostAgents, agentRef) ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) {
		t.Fatalf("second no dejo assessment recuperable: lost=%v assessments=%v run=%+v", run.LostAgents, run.AgentAssessments, run)
	}

	if result, err := stack.RunGlobalSupervisorV0(ctx, command); err != nil {
		t.Fatalf("third result=%+v err=%v", result, err)
	}
	run, err = stack.Ports.RunStore.LoadRunV0(ctx, prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 third: %v", err)
	}
	if len(run.StartedAgents) != 2 || runtime.launchCountV0() != 2 ||
		!codexStackRefsContainPartV0(run.AgentAssessments, "#action:"+orquestacoreworkflow.AgentAssessmentActionAskDirectorV0) ||
		!codexStackRefsContainPartV0(run.ReplanDecisions, "#action:"+string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)) {
		t.Fatalf("supervisor residente no reemplazo ask_director perdido: started=%v launches=%d assessments=%v replans=%v run=%+v",
			run.StartedAgents,
			runtime.launchCountV0(),
			run.AgentAssessments,
			run.ReplanDecisions,
			run,
		)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0RespetaPrioridadSolicitada(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-low-priority-001",
		CorrelationID:          "corr-autoprogramming-low-priority-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		PriorityScore:          10,
	})
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].PriorityScore != 10 {
		t.Fatalf("ranking=%+v prepared=%+v", ranking, prepared)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0ReabreCandidatoStoppedComoReadyV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-reopen-stopped-001",
		CorrelationID:          "corr-autoprogramming-reopen-stopped-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	stopped := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		RequestID:     "request-autoprogramming-reopen-stopped-stop-001",
		CorrelationID: "corr-autoprogramming-reopen-stopped-001",
		Action:        orquestamcp.MCPRunQueuePriorityActionSetV0,
		QueueRef:      DefaultRunQueueRefV0,
		RunRef:        prepared.RunRef,
		AppRef:        prepared.ProjectRef,
		Status:        "stopped",
		PriorityScore: 0,
	})
	if stopped.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		t.Fatalf("stopped=%+v", stopped)
	}
	empty := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if len(empty.Ranked) != 0 {
		t.Fatalf("stopped candidate visible=%+v", empty)
	}

	reopened := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})
	if !reopened.Accepted ||
		reopened.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != prepared.RunRef ||
		ranking.Ranked[0].Status != "ready" {
		t.Fatalf("reopened=%+v ranking=%+v", reopened, ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoReencolaRunCerradaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-closed-idempotent-001",
		CorrelationID:          "corr-autoprogramming-closed-idempotent-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	if err := stack.Ports.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 closed: %v", err)
	}

	reprepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})

	if !reprepared.Accepted ||
		reprepared.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 0 {
		t.Fatalf("reprepared=%+v ranking=%+v", reprepared, ranking)
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoReencolaRunCanceladaPorControlV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	input := orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-cancel-control-001",
		CorrelationID:          "corr-autoprogramming-cancel-control-001",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
	}
	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	cancelled := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:      "cancel",
		RunRef:      prepared.RunRef,
		RequestedBy: "director",
		Reason:      "run apartada por humano",
	})
	if cancelled.Status != string(orquestaruncontrol.RunControlStatusCancelRequestedV0) {
		t.Fatalf("cancelled=%+v", cancelled)
	}
	reprepared := postAutoprogrammingPrepareRunStackV0(t, stack, input)
	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef: DefaultRunQueueRefV0,
		Limit:    1,
	})

	if !reprepared.Accepted ||
		reprepared.RunRef != prepared.RunRef ||
		len(ranking.Ranked) != 0 {
		t.Fatalf("reprepared=%+v ranking=%+v", reprepared, ranking)
	}
}

func TestCodexStackServerShutdownV0CierraRunPreparadaSinEntrarAlDirector(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-shutdown-001",
		CorrelationID:          "corr-autoprogramming-shutdown-001",
		OccurredAt:             "2026-05-23T12:00:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || prepared.RunRef == "" {
		t.Fatalf("prepared=%+v", prepared)
	}

	shutdown := postServerShutdownStackV0(t, stack, orquestamcp.MCPServerShutdownToolInputV0{
		RequestID:      "request-server-shutdown-prepared-001",
		CorrelationID:  "corr-autoprogramming-shutdown-001",
		Forced:         true,
		MaxTicks:       1,
		MaxRunsPerTick: 1,
		MaxExecutions:  1,
		RequestedBy:    "orquesta-director",
		Reason:         "shutdown de run preparada sin agentes vivos",
	})
	if shutdown.Estado != orquestamcp.MCPServerShutdownEstadoOKV0 ||
		!shutdown.ShutdownReady ||
		shutdown.Status != orquestaservershutdown.ServerShutdownStatusReadyV0 ||
		shutdown.RunsRequested != 1 ||
		shutdown.RunsStopped != 1 ||
		shutdown.AgentsInFlight != 0 ||
		runtime.launchCountV0() != 0 {
		t.Fatalf("shutdown=%+v launches=%d", shutdown, runtime.launchCountV0())
	}
}

func TestCodexStackAutoprogrammingPrepareRunAPIV0DevuelveErroresPublicos(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: "request-autoprogramming-api-invalid-001",
	})

	if result.Estado != orquestamcp.MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func postAutoprogrammingPrepareRunStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0,
) orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode prepare run: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/prepare-run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("prepare status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode prepare run: %v", err)
	}
	return result
}

func postServerShutdownStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPServerShutdownToolInputV0,
) orquestamcp.MCPServerShutdownToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode shutdown: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/server/shutdown", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("shutdown status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPServerShutdownToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode shutdown: %v", err)
	}
	return result
}
