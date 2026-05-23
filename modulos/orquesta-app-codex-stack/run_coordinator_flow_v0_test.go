package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackV0ArrancarDirectorRegistraRunEnColaGlobalV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != director.RunRef ||
		ranking.Ranked[0].PriorityScore != DefaultRunQueuePriorityScoreV0 {
		t.Fatalf("ranking=%+v director=%+v", ranking, director)
	}
}

func TestCodexStackV0RunGlobalTickEjecutaRunConMasPrioridadV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	low := postDirectorAPIWithNameV0(t, stack, "app-baja", "Agenda Baja")
	high := postDirectorAPIWithNameV0(t, stack, "app-alta", "Agenda Alta")

	setStackRunPriorityForTestV0(t, stack, low.RunRef, low.AppSpec.Slug, 10)
	setStackRunPriorityForTestV0(t, stack, high.RunRef, high.AppSpec.Slug, 90)

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); !reflect.DeepEqual(got, []string{high.RunRef}) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{high.RunRef}, result)
	}
}

func TestCodexStackV0RunGlobalTickRespetaPausaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	paused := postDirectorAPIWithNameV0(t, stack, "app-pausada", "Agenda Pausada")
	next := postDirectorAPIWithNameV0(t, stack, "app-siguiente", "Agenda Siguiente")

	setStackRunPriorityForTestV0(t, stack, paused.RunRef, paused.AppSpec.Slug, 90)
	setStackRunPriorityForTestV0(t, stack, next.RunRef, next.AppSpec.Slug, 20)
	if _, err := stack.Stores.RunControl.PauseRunV0(context.Background(), orquestaruncontrol.PauseRunCommandV0{
		RunRef:      paused.RunRef,
		RequestedBy: "director",
		Reason:      "prueba prioridad pausada",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(result); !reflect.DeepEqual(got, []string{next.RunRef}) {
		t.Fatalf("executions got %#v want %#v result=%+v", got, []string{next.RunRef}, result)
	}
	if len(result.Skips) != 1 || result.Skips[0].RunRef != paused.RunRef {
		t.Fatalf("skips=%+v", result.Skips)
	}
}

func TestCodexStackV0RunGlobalTickMarcaRunCerradaComoTerminalEnColaV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusClosedV0
	if err := stack.Stores.RunStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v", err)
	}
	if len(result.Executions) != 1 ||
		result.Executions[0].RunRef != director.RunRef ||
		result.Executions[0].QueueStatus != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("result=%+v", result)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 0 {
		t.Fatalf("ranking=%+v", ranking)
	}
}

func TestCodexStackV0DrainQueueStatusMarcaRunEntregadaComoTerminalEnColaV0(t *testing.T) {
	taskRef := "task-ref-stack-delivered-queue-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-stack-delivered-queue-001"
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				CurrentPhase:    orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Tasks:           []string{taskRef},
				Agents:          []string{agentRef},
				StartedAgents:   []string{agentRef},
				DeliveredAgents: []string{agentRef},
				Deliveries:      []string{deliveryRef},
				DeliveredTasks:  []string{taskRef},
			},
		},
	}
	if got := stackDrainQueueStatusV0(result); got != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("queue_status=%q want %q", got, orquestarunqueue.RunStatusDeliveredV0)
	}
}

func postDirectorAPIWithNameV0(
	t *testing.T,
	stack StackV0,
	ref string,
	name string,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	spec := codexStackAppSpecRequestV0()
	spec.RequestID = "request-ref-" + ref
	spec.Nombre = name
	spec.Objetivo = "Gestionar " + name + " con API REST y web."
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:      "request-ref-http-" + ref,
		CorrelationID:  "corr-" + ref,
		AppSpecRequest: spec,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("director status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode director: %v", err)
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 {
		t.Fatalf("director result=%+v", result)
	}
	return result
}

func setStackRunPriorityForTestV0(
	t *testing.T,
	stack StackV0,
	runRef string,
	appRef string,
	priority int,
) {
	t.Helper()
	result := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:        "set_priority",
		QueueRef:      DefaultRunQueueRefV0,
		RunRef:        runRef,
		AppRef:        appRef,
		PriorityScore: priority,
		OccurredAt:    "2026-05-11T12:00:00Z",
		RequestedBy:   "stack-test",
	})
	if result.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		t.Fatalf("priority result=%+v", result)
	}
}

func globalTickCommandForTestV0() orquestaruncoordinator.RunCoordinatorTickCommandV0 {
	now := time.Date(2026, 5, 11, 12, 5, 0, 0, time.UTC)
	return orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef:   DefaultRunQueueRefV0,
		MaxRuns:    1,
		OccurredAt: now,
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            2,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          4,
			MaxOutboxPerCycle:    2,
			MaxDecisionCycles:    1,
			MaxExternalWaits:     1,
		},
	}
}

func executionRefsForStackCoordinatorTestV0(
	result orquestaruncoordinator.RunCoordinatorTickResultV0,
) []string {
	refs := make([]string, 0, len(result.Executions))
	for _, execution := range result.Executions {
		refs = append(refs, execution.RunRef)
	}
	return refs
}
