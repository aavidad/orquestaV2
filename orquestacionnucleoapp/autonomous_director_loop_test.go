package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestRunAutonomousDirectorLoopV0DecideLimitesYDevuelveStats(t *testing.T) {
	runRef := "run-nucleo-autonomous-director-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{"task-ref-auto-001", "task-ref-auto-002", "task-ref-auto-003", "task-ref-auto-004"}
	store := NewInMemoryRunStoreV0(run)
	service := ServiceV0{
		RunStore:          store,
		CandidateProvider: StaticCandidateProviderV0{},
		OutboxLedger:      NewInMemoryOutboxLedgerV0(),
	}

	result, err := service.RunAutonomousDirectorLoopV0(context.Background(), AutonomousDirectorLoopRequestV0{
		Loop: ProgressiveLoopRequestV0{
			RunRef:     runRef,
			OccurredAt: "2026-05-10T10:00:00Z",
		},
		Limits: AutonomousDirectorLimitsV0{
			MaxTeamSize:       4,
			MaxParallelAgents: 2,
		},
	})
	if err != nil {
		t.Fatalf("autonomous loop: %v", err)
	}
	if result.Decision.TeamSize != 4 ||
		result.Decision.MaxParallelAgents != 2 ||
		result.Decision.RecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("decision=%+v", result.Decision)
	}
	if result.Stats.Run.Counts.TasksOpen != 4 ||
		result.Stats.Decision.TeamSize != result.Decision.TeamSize ||
		result.Stats.Loop.Status != string(result.Loop.Status) {
		t.Fatalf("stats=%+v loop=%+v", result.Stats, result.Loop)
	}
}

func TestApplyAutonomousDirectorDecisionV0AjustaBatchConcurrency(t *testing.T) {
	executor := autonomousTunableBatchExecutorForTestV0{}
	service := ServiceV0{}
	loop := ProgressiveLoopRequestV0{
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   1,
			Executor:   executor,
		}},
	}

	tunedService, tunedLoop := applyAutonomousDirectorDecisionV0(service, loop, AutonomousDirectorDecisionV0{
		MaxCommandsPerCycle:  7,
		MaxOutboxPerCycle:    3,
		MaxBursts:            5,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 3,
		MaxParallelAgents:    3,
	})
	if tunedService.MaxCommands != 7 ||
		tunedService.MaxOutboxPerCycle != 3 ||
		tunedLoop.MaxBursts != 5 ||
		tunedLoop.MaxStepsPerBurst != 4 ||
		tunedLoop.MaxDispatchesPerWait != 3 ||
		tunedLoop.BatchDispatchers[0].MaxReady != 3 {
		t.Fatalf("service=%+v loop=%+v", tunedService, tunedLoop)
	}
	tunedExecutor, ok := tunedLoop.BatchDispatchers[0].Executor.(autonomousTunableBatchExecutorForTestV0)
	if !ok || tunedExecutor.MaxConcurrency != 3 {
		t.Fatalf("executor=%T %+v", tunedLoop.BatchDispatchers[0].Executor, tunedLoop.BatchDispatchers[0].Executor)
	}
}

type autonomousTunableBatchExecutorForTestV0 struct {
	MaxConcurrency int
}

func (executor autonomousTunableBatchExecutorForTestV0) WithAutonomousBatchConcurrencyV0(
	maxConcurrency int,
) OutboxDispatchBatchExecutorPortV0 {
	executor.MaxConcurrency = maxConcurrency
	return executor
}

func (executor autonomousTunableBatchExecutorForTestV0) ExecuteOutboxDispatchBatchV0(
	context.Context,
	[]orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	return nil, nil
}
