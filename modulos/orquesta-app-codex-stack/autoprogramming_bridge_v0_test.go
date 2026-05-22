package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestPrepareAutoprogrammingRunV0PersisteWorkflowTasksYRunContinuable(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      ledger,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:       autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:    "2026-05-22T10:00:00Z",
		CorrelationID: "corr-autoprogramming-bridge-001",
		RequestedBy:   "orquesta-test",
		MaxBursts:     4,
		MaxCommands:   8,
	}, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !bridged.Accepted || len(bridged.Tasks) != 1 || len(bridged.WaitAgentRefs) != 1 {
		t.Fatalf("bridged=%+v", bridged)
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), bridged.Run.RunID, bridged.Run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 ||
		storedTasks[0].TaskID != bridged.Tasks[0].TaskID ||
		storedTasks[0].RunID != bridged.Run.RunID ||
		storedTasks[0].RequiredTests[0] != "go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0" {
		t.Fatalf("stored_tasks=%+v bridged=%+v", storedTasks, bridged.Tasks)
	}
	run, err := runStore.LoadRunV0(context.Background(), bridged.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(run.Tasks) != 1 ||
		run.Tasks[0] != bridged.Tasks[0].TaskID {
		t.Fatalf("run no continuable: %+v", run)
	}
	agentTask, err := (CodexLaunchSpecResolverV0{TaskStore: taskStore}).agentTaskV0(
		context.Background(),
		orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:   bridged.Run.RunID,
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef: bridged.Tasks[0].TaskID,
		},
		"programacion",
	)
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if agentTask.TaskRef != bridged.Tasks[0].TaskID ||
		len(agentTask.WriteSet) != 1 ||
		agentTask.WriteSet[0] != "modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go" {
		t.Fatalf("agent_task=%+v", agentTask)
	}
	if bridged.Continue.RunRef != bridged.Run.RunID ||
		len(bridged.Continue.WaitAgentRefs) != 1 ||
		bridged.Continue.WaitAgentRefs[0] != bridged.WaitAgentRefs[0] {
		t.Fatalf("continue=%+v wait=%v", bridged.Continue, bridged.WaitAgentRefs)
	}

	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		bridged.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, bridged.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, bridged.WaitAgentRefs[0])
	}
}

func TestPrepareAutoprogrammingRunV0NoPersisteRequestInvalida(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequiredTests = nil

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted || len(bridged.Issues) == 0 {
		t.Fatalf("bridged=%+v", bridged)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe persistirse, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0EsIdempotenteYNoSobrescribeRunViva(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      ledger,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}
	request := AutoprogrammingBridgeRequestV0{
		Request:       autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:    "2026-05-22T10:10:00Z",
		CorrelationID: "corr-autoprogramming-idempotent-001",
		RequestedBy:   "orquesta-test",
		MaxBursts:     4,
		MaxCommands:   8,
	}

	first, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 first: %v", err)
	}
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		first.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, first.WaitAgentRefs[0])
	}

	second, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 second: %v", err)
	}
	run, err := runStore.LoadRunV0(context.Background(), first.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("prepare-run repetido sobrescribio started_agents: run=%+v", run)
	}
	if second.Run.RunID != first.Run.RunID ||
		second.Continue.RunRef != first.Run.RunID ||
		second.WaitAgentRefs[0] != first.WaitAgentRefs[0] {
		t.Fatalf("second=%+v first=%+v", second, first)
	}
}

func autoprogrammingBridgeRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	return orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:       "run-autoprogramming-bridge-001",
		ProjectRef:       "project-ref-autoprogramming-bridge-001",
		WorktreeRef:      "worktree-ref-autoprogramming-bridge-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-bridge-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "source-task-ref-autoprogramming-bridge-001",
			Area:    "app-codex-stack",
		}},
		WriteSet: []string{
			"modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go",
		},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0",
		},
	}
}

func autoprogrammingBridgeCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			OccurredAt: "2026-05-22T10:01:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeAgentDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-22T10:02:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
