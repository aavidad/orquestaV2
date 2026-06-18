package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// Reproduce el bug de frontera observado con Codex real (Bolsa, 2026-06-18):
// run en programacion con 2 tareas; la base (sin deps) ya entregada; la dependiente
// (DependsOn base) tiene su dependencia satisfecha pero NUNCA se lanza: el director
// se queda sin pedir agente para ella. El arreglo debe hacer que ContinueAppDirectorV0
// re-evalue la frontera y genere el lanzamiento de la dependiente.
func TestContinueAppDirectorV0RelanzaFronteraTrasEntregaV0(t *testing.T) {
	runRef := "run-frontier-redispatch-001"
	baseRef := "task-base"
	depRef := "task-dependiente"
	baseAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(baseRef)

	base := frontierTaskForTestV0(runRef, baseRef, []string{"go.mod"}, nil)
	dep := frontierTaskForTestV0(runRef, depRef, []string{"internal/api.go"}, []string{baseRef})

	run := mustActiveProgrammingRunForFrontierTestV0(t, runRef)
	run.Tasks = []string{baseRef, depRef}
	// La base ya se lanzo y entrego; la dependiente sigue abierta.
	run.Agents = []string{baseAgent}
	run.StartedAgents = []string{baseAgent}
	run.DeliveredAgents = []string{baseAgent}
	run.DeliveredTasks = []string{baseRef}

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(base, dep)
	outbox := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-06-18T10:00:00Z",
		CorrelationID:        "corr-frontier-redispatch",
		RequestedBy:          "test-frontier",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 6,
		MaxCommands:          32,
		MaxOutboxPerCycle:    16,
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      outbox,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outbox),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outbox),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}

	// La dependiente debe haberse pedido (capacidad o agente solicitado).
	finalRun := result.Run
	requestedDep := frontierRunRequestedTaskV0(finalRun, depRef)
	if !requestedDep {
		t.Fatalf("la tarea dependiente desbloqueada NO se relanzo tras la entrega de la base: agents_requested=%v capacity_requests=%v open=%v",
			finalRun.Agents, finalRun.CapacityRequests, finalRun.Tasks)
	}
}

// Captura la causa raiz del bug real: cuando el loop corre ACOTADO por WaitAgentRefs
// al scope de la ola inicial (lo que hace el coordinador del stack Codex via
// queuedOperationalDirectorWaitAgentRefsV0), la frontera NO se re-evalua fuera de
// ese scope y la dependiente desbloqueada se queda sin lanzar. El arreglo debe hacer
// que la frontera de tareas schedulables se re-evalue aunque el wait este acotado.
func TestContinueAppDirectorV0RelanzaFronteraAunConWaitScopeAcotadoV0(t *testing.T) {
	runRef := "run-frontier-scoped-001"
	baseRef := "task-base"
	depRef := "task-dependiente"
	baseAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(baseRef)

	base := frontierTaskForTestV0(runRef, baseRef, []string{"go.mod"}, nil)
	dep := frontierTaskForTestV0(runRef, depRef, []string{"internal/api.go"}, []string{baseRef})

	run := mustActiveProgrammingRunForFrontierTestV0(t, runRef)
	run.Tasks = []string{baseRef, depRef}
	run.Agents = []string{baseAgent}
	run.StartedAgents = []string{baseAgent}
	run.DeliveredAgents = []string{baseAgent}
	run.DeliveredTasks = []string{baseRef}

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(base, dep)
	outbox := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-06-18T10:05:00Z",
		CorrelationID:        "corr-frontier-scoped",
		RequestedBy:          "test-frontier-scoped",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 6,
		MaxCommands:          32,
		MaxOutboxPerCycle:    16,
		// Scope acotado al agente de la ola inicial: lo que hace el coordinador.
		WaitAgentRefs: []string{baseAgent},
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      outbox,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outbox),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outbox),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !frontierRunRequestedTaskV0(result.Run, depRef) {
		t.Fatalf("con wait scope acotado la dependiente NO se relanzo (bug de frontera): agents=%v capacity=%v",
			result.Run.Agents, result.Run.CapacityRequests)
	}
}

// Reproduce el camino REAL del supervise: con ExternalWaiter (managed loop) y wait
// scope acotado a la ola inicial. Esta es la hipotesis del bug: el managed loop
// espera en vez de re-pedir la frontera desbloqueada.
func TestContinueAppDirectorV0RelanzaFronteraConManagedLoopV0(t *testing.T) {
	runRef := "run-frontier-managed-001"
	baseRef := "task-base"
	depRef := "task-dependiente"
	baseAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(baseRef)

	base := frontierTaskForTestV0(runRef, baseRef, []string{"go.mod"}, nil)
	dep := frontierTaskForTestV0(runRef, depRef, []string{"internal/api.go"}, []string{baseRef})

	run := mustActiveProgrammingRunForFrontierTestV0(t, runRef)
	run.Tasks = []string{baseRef, depRef}
	run.Agents = []string{baseAgent}
	run.StartedAgents = []string{baseAgent}
	run.DeliveredAgents = []string{baseAgent}
	run.DeliveredTasks = []string{baseRef}

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(base, dep)
	outbox := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	waiter := &serviceContinueWaiterForTestV0{Continue: true}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-06-18T10:10:00Z",
		CorrelationID:        "corr-frontier-managed",
		RequestedBy:          "test-frontier-managed",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 6,
		MaxCommands:          32,
		MaxOutboxPerCycle:    16,
		MaxExternalWaits:     2,
		WaitAgentRefs:        []string{baseAgent},
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      outbox,
		DirectorTaskStore: taskStore,
		ExternalWaiter:    waiter,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outbox),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outbox),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !frontierRunRequestedTaskV0(result.Run, depRef) {
		t.Fatalf("con managed loop + wait scope la dependiente NO se relanzo (BUG REAL reproducido): agents=%v capacity=%v",
			result.Run.Agents, result.Run.CapacityRequests)
	}
}

// Reproduce el camino REAL del autoprogramming/supervise: con DirectorDecisionSource
// presente (como AutoprogrammingDirectorDecisionSourceV0) que NO emite decisiones
// cuando no todas las tareas entregaron. La hipotesis: con decision-source que no
// progresa, runExistingDirectorAutonomyLoopV0 no relanza la frontera desbloqueada.
func TestContinueAppDirectorV0RelanzaFronteraConDecisionSourceV0(t *testing.T) {
	runRef := "run-frontier-decisionsource-001"
	baseRef := "task-base"
	depRef := "task-dependiente"
	baseAgent := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(baseRef)

	base := frontierTaskForTestV0(runRef, baseRef, []string{"go.mod"}, nil)
	dep := frontierTaskForTestV0(runRef, depRef, []string{"internal/api.go"}, []string{baseRef})

	run := mustActiveProgrammingRunForFrontierTestV0(t, runRef)
	run.Tasks = []string{baseRef, depRef}
	run.Agents = []string{baseAgent}
	run.StartedAgents = []string{baseAgent}
	run.DeliveredAgents = []string{baseAgent}
	run.DeliveredTasks = []string{baseRef}

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(base, dep)
	outbox := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-06-18T10:15:00Z",
		CorrelationID:        "corr-frontier-decisionsource",
		RequestedBy:          "test-frontier-decisionsource",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 6,
		MaxCommands:          32,
		MaxOutboxPerCycle:    16,
		MaxDecisionCycles:    4,
		WaitAgentRefs:        []string{baseAgent},
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      outbox,
		DirectorTaskStore: taskStore,
		// Decision source que no emite nada (como autoprogramming sin ola completa).
		DirectorDecisionSource: serviceDirectorDecisionSourceForTestV0{},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outbox),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outbox),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !frontierRunRequestedTaskV0(result.Run, depRef) {
		t.Fatalf("con DirectorDecisionSource que no progresa, la dependiente NO se relanzo (BUG REAL): agents=%v capacity=%v",
			result.Run.Agents, result.Run.CapacityRequests)
	}
}

func frontierRunRequestedTaskV0(run orquestacoreworkflow.OrchestrationRunV0, taskRef string) bool {
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	for _, a := range run.Agents {
		if a == agentRef {
			return true
		}
	}
	// Tambien cuenta si se pidio capacidad para la tarea (paso previo al agente).
	for _, c := range run.CapacityRequests {
		if c == "capacity-ref-"+taskRef {
			return true
		}
	}
	return false
}

func frontierTaskForTestV0(runRef, taskRef string, writeSet, dependsOn []string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea " + taskRef,
		Summary:       "Trabajo compacto de programacion.",
		WriteSet:      writeSet,
		DependsOn:     dependsOn,
		AcceptanceCriteria: []string{
			"Solo se modifica el write-set declarado.",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:workitem:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}

func mustActiveProgrammingRunForFrontierTestV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-frontier",
		AppSpecRef:    "app-spec-frontier",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{
			{
				ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
				RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			},
		},
	}
}
