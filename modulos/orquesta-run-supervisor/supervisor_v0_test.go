package orquestarunsupervisor

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func TestSuperviseRunsRespetaMaxTicksV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{
		results: []orquestaruncoordinator.RunCoordinatorTickResultV0{
			tickResultV0("run-a"),
			tickResultV0("run-b"),
			tickResultV0("run-c"),
		},
	}
	result, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, supervisorCommandV0(2))
	if err != nil {
		t.Fatalf("SuperviseRunsV0: %v", err)
	}
	if result.StopReason != RunSupervisorStopMaxTicksV0 ||
		len(result.Ticks) != 2 ||
		result.TotalExecutions != 2 {
		t.Fatalf("result=%+v", result)
	}
}

func TestSuperviseRunsRespetaMaxExecutionsV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{
		results: []orquestaruncoordinator.RunCoordinatorTickResultV0{
			tickResultV0("run-a", "run-b"),
			tickResultV0("run-c"),
		},
	}
	command := supervisorCommandV0(5)
	command.MaxRunsPerTick = 2
	command.MaxExecutions = 2

	result, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, command)
	if err != nil {
		t.Fatalf("SuperviseRunsV0: %v", err)
	}
	if result.StopReason != RunSupervisorStopMaxExecutionsV0 ||
		len(result.Ticks) != 1 ||
		result.TotalExecutions != 2 {
		t.Fatalf("result=%+v", result)
	}
}

func TestSuperviseRunsParaSinEjecucionesV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{
		results: []orquestaruncoordinator.RunCoordinatorTickResultV0{{}},
	}
	command := supervisorCommandV0(5)
	command.StopOnNoExecution = true

	result, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, command)
	if err != nil {
		t.Fatalf("SuperviseRunsV0: %v", err)
	}
	if result.StopReason != RunSupervisorStopNoExecutionV0 ||
		len(result.Ticks) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestSuperviseRunsPropagaCommandYExcluyeRunsEjecutadasV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{
		results: []orquestaruncoordinator.RunCoordinatorTickResultV0{
			tickResultV0("run-a"),
			tickResultV0("run-b"),
		},
	}
	command := supervisorCommandV0(2)
	command.QueueRef = "global"
	command.AppRefs = []string{" app-1 ", "app-1", "app-2"}
	command.QueueLimit = 10
	command.MaxRunsPerTick = 1
	command.CorrelationID = " corr-1 "

	result, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, command)
	if err != nil {
		t.Fatalf("SuperviseRunsV0: %v", err)
	}
	if result.TotalExecutions != 2 {
		t.Fatalf("result=%+v", result)
	}
	if ticker.commands[0].QueueRef != "global" ||
		!reflect.DeepEqual(ticker.commands[0].AppRefs, []string{"app-1", "app-2"}) ||
		ticker.commands[0].QueueLimit != 10 ||
		ticker.commands[0].CorrelationID != "corr-1" {
		t.Fatalf("first command=%+v", ticker.commands[0])
	}
	if !reflect.DeepEqual(ticker.commands[1].ExcludeRunRefs, []string{"run-a"}) {
		t.Fatalf("second excludes=%+v", ticker.commands[1].ExcludeRunRefs)
	}
}

func TestSuperviseRunsNoMutaInputV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{results: []orquestaruncoordinator.RunCoordinatorTickResultV0{tickResultV0("run-a")}}
	command := supervisorCommandV0(1)
	command.AppRefs = []string{" app-1 "}
	want := append([]string(nil), command.AppRefs...)

	if _, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, command); err != nil {
		t.Fatalf("SuperviseRunsV0: %v", err)
	}
	if !reflect.DeepEqual(command.AppRefs, want) {
		t.Fatalf("command mutated got %#v want %#v", command.AppRefs, want)
	}
}

func TestSuperviseRunsPropagaErrorYContextV0(t *testing.T) {
	ticker := &fakeSupervisorTickerV0{err: errors.New("tick failed")}
	result, err := SuperviseRunsV0(context.Background(), RunSupervisorDepsV0{Ticker: ticker}, supervisorCommandV0(1))
	if err == nil || result.StopReason != RunSupervisorStopTickErrorV0 {
		t.Fatalf("err=%v result=%+v", err, result)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = SuperviseRunsV0(ctx, RunSupervisorDepsV0{Ticker: ticker}, supervisorCommandV0(1))
	if err == nil || result.StopReason != RunSupervisorStopContextDoneV0 {
		t.Fatalf("ctx err=%v result=%+v", err, result)
	}
}

type fakeSupervisorTickerV0 struct {
	commands []orquestaruncoordinator.RunCoordinatorTickCommandV0
	results  []orquestaruncoordinator.RunCoordinatorTickResultV0
	err      error
}

func (fake *fakeSupervisorTickerV0) RunGlobalTickV0(
	_ context.Context,
	command orquestaruncoordinator.RunCoordinatorTickCommandV0,
) (orquestaruncoordinator.RunCoordinatorTickResultV0, error) {
	fake.commands = append(fake.commands, command)
	if fake.err != nil {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, fake.err
	}
	if len(fake.results) == 0 {
		return orquestaruncoordinator.RunCoordinatorTickResultV0{}, nil
	}
	result := fake.results[0]
	fake.results = fake.results[1:]
	return result, nil
}

func supervisorCommandV0(maxTicks int) RunSupervisorCommandV0 {
	return RunSupervisorCommandV0{
		MaxTicks:          maxTicks,
		MaxRunsPerTick:    1,
		StopOnNoExecution: true,
		OccurredAt:        time.Date(2026, 5, 11, 14, 0, 0, 0, time.UTC),
	}
}

func tickResultV0(runRefs ...string) orquestaruncoordinator.RunCoordinatorTickResultV0 {
	result := orquestaruncoordinator.RunCoordinatorTickResultV0{}
	for _, runRef := range runRefs {
		result.Executions = append(result.Executions, orquestaruncoordinator.RunExecutionSummaryV0{
			RunRef:  runRef,
			Outcome: "drained",
		})
	}
	return result
}
