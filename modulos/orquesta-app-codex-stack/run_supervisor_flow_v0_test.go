package orquestaappcodexstack

import (
	"context"
	"reflect"
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestCodexStackV0RunGlobalSupervisorAvanzaVariasAppsPorPrioridadV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	low := postDirectorAPIWithNameV0(t, stack, "app-baja-dos", "Agenda Baja Dos")
	high := postDirectorAPIWithNameV0(t, stack, "app-alta-dos", "Agenda Alta Dos")

	setStackRunPriorityForTestV0(t, stack, low.RunRef, low.AppSpec.Slug, 10)
	setStackRunPriorityForTestV0(t, stack, high.RunRef, high.AppSpec.Slug, 90)

	result, err := stack.RunGlobalSupervisorV0(context.Background(), supervisorCommandForStackTestV0(2))
	if err != nil {
		t.Fatalf("RunGlobalSupervisorV0: %v", err)
	}
	got := supervisorExecutionRefsForStackTestV0(result)
	want := []string{high.RunRef, low.RunRef}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, want, result)
	}
	if result.StopReason != orquestarunsupervisor.RunSupervisorStopMaxExecutionsV0 {
		t.Fatalf("stop_reason=%q result=%+v", result.StopReason, result)
	}
}

func supervisorCommandForStackTestV0(maxExecutions int) orquestarunsupervisor.RunSupervisorCommandV0 {
	return orquestarunsupervisor.RunSupervisorCommandV0{
		QueueRef:          DefaultRunQueueRefV0,
		MaxTicks:          4,
		MaxRunsPerTick:    1,
		MaxExecutions:     maxExecutions,
		StopOnNoExecution: true,
		OccurredAt:        time.Date(2026, 5, 11, 12, 10, 0, 0, time.UTC),
	}
}

func supervisorExecutionRefsForStackTestV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) []string {
	refs := []string{}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.RunRef)
		}
	}
	return refs
}
