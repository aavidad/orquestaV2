package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestWorkflowTaskCandidateProviderV0HandlesDirectorTaskSet(t *testing.T) {
	runRef := "run-nucleo-workflow-task-director-set-001"
	tasks := directorLikeWorkflowTasksForCandidateProviderTestV0(runRef)
	run := mustActiveProgrammingRunV0(t, runRef)
	for _, task := range tasks {
		run.Tasks = append(run.Tasks, task.TaskID)
	}
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  newMemoryRunStoreV0(run),
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			TaskStore:       NewInMemoryWorkflowTaskStoreV0(tasks...),
			RequestedBy:     "orquesta-nucleo-test",
			DefaultCapacity: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		},
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-11T09:00:00Z",
		MaxSteps:      6,
		CorrelationID: "corr-workflow-task-director-set-001",
	})
	if err != nil {
		t.Fatalf("RunSupervisedBurstV0: %v result=%+v", err, result)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s result=%+v", result.Burst.FinalAction, result)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) == 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestWorkflowTaskCandidateProviderV0LaunchesDirectorTaskSetProgressively(t *testing.T) {
	runRef := "run-nucleo-workflow-task-director-progressive-001"
	tasks := directorLikeWorkflowTasksForCandidateProviderTestV0(runRef)
	run := mustActiveProgrammingRunV0(t, runRef)
	for _, task := range tasks {
		run.Tasks = append(run.Tasks, task.TaskID)
	}
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	batchExecutor := &singleAgentLauncherBatchExecutorForTestV0{
		executor: AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-11T09:20:00Z",
		},
	}
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			TaskStore:       NewInMemoryWorkflowTaskStoreV0(tasks...),
			RequestedBy:     "orquesta-nucleo-test",
			DefaultCapacity: orquestacoreworkflow.OrchestrationCapacityXHighV0,
		},
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-11T09:15:00Z",
		MaxBursts:            8,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-workflow-task-director-progressive-001",
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   4,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   batchExecutor,
			Acker:      ledger,
		}},
	})
	if err != nil {
		debugBurst, debugErr := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-11T09:16:00Z",
			MaxSteps:      1,
			CorrelationID: "corr-workflow-task-director-progressive-debug-001",
		})
		t.Fatalf("RunProgressiveLoopV0: %v bursts=%+v debug_err=%v debug_steps=%+v", err, result.Bursts, debugErr, debugBurst.Burst.Steps)
	}
	if result.Status != ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if len(result.Run.StartedAgents) == 0 {
		t.Fatalf("started agents=%v result=%+v", result.Run.StartedAgents, result)
	}
}

func directorLikeWorkflowTasksForCandidateProviderTestV0(runRef string) []orquestacoreworkflow.WorkflowTaskV0 {
	return []orquestacoreworkflow.WorkflowTaskV0{
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-domain-001",
			title:    "Implementar dominio y casos de uso de agenda",
			summary:  "Crear reglas puras, entidades, puertos y casos de uso principales de agenda.",
			writeSet: []string{"internal/agenda/domain", "internal/agenda/application", "internal/agenda/ports"},
			criteria: []string{
				"Reglas de intervalo y campos requeridos validadas.",
				"Casos de uso conectados solo por puertos.",
				"Errores estables para validacion y no encontrado.",
			},
			functions: []string{"createAgendaItem", "listAgendaItems", "updateAgendaItem", "deleteAgendaItem", "validateAgendaInput"},
		}),
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-api-001",
			title:    "Implementar interfaz API de agenda",
			summary:  "Crear borde API con DTOs, validacion de entrada y errores localizables por codigo.",
			writeSet: []string{"internal/agenda/interfaces/http", "internal/agenda/contracts"},
			criteria: []string{
				"DTOs separados del dominio.",
				"Entradas invalidas producen codigos de error estables.",
				"Flujos crear, listar, cambiar y borrar quedan conectados a casos de uso.",
			},
			functions: []string{"createAgendaItem", "listAgendaItems", "updateAgendaItem", "deleteAgendaItem", "localizeAgendaText"},
		}),
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-web-001",
			title:    "Implementar UI web de agenda",
			summary:  "Crear lista, formulario y estados de UI con textos localizables.",
			writeSet: []string{"web/src/agenda", "web/src/i18n"},
			criteria: []string{
				"UI cubre lista, crear, cambiar y borrar.",
				"Estados vacio, carga y error visibles.",
				"Todo texto visible se resuelve por catalogo i18n.",
			},
			functions: []string{"renderAgendaView", "localizeAgendaText", "listAgendaItems"},
		}),
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-persistence-001",
			title:    "Implementar puerto de persistencia de agenda",
			summary:  "Crear contrato de persistencia y conector intercambiable sin fijar tecnologia concreta.",
			writeSet: []string{"internal/agenda/connectors", "internal/agenda/ports"},
			criteria: []string{
				"Dominio y aplicacion no dependen del conector.",
				"Puerto cubre guardar, recuperar, listar, cambiar y borrar.",
				"Contrato probado con doble de pruebas.",
			},
			functions: []string{"createAgendaItem", "listAgendaItems", "updateAgendaItem", "deleteAgendaItem"},
		}),
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-tests-001",
			title:    "Cubrir pruebas de agenda",
			summary:  "Agregar pruebas de dominio, aplicacion, API, web e i18n.",
			writeSet: []string{"tests/agenda", "internal/agenda", "web/src/agenda"},
			criteria: []string{
				"Happy path crear, listar, cambiar y borrar cubierto.",
				"Errores de validacion, permiso y no encontrado cubiertos.",
				"Smoke principal documentado para validacion final.",
			},
			functions: []string{"createAgendaItem", "listAgendaItems", "updateAgendaItem", "deleteAgendaItem", "localizeAgendaText"},
		}),
		directorLikeWorkflowTaskForCandidateProviderTestV0(runRef, directorLikeWorkflowTaskSpecForTestV0{
			taskRef:  "task-agenda-docs-security-deploy-001",
			title:    "Completar seguridad, documentacion y deploy",
			summary:  "Documentar arranque, contrato API, configuracion, seguridad y deploy si aplica.",
			writeSet: []string{"docs/agenda", "ops/agenda", "docs/revision"},
			criteria: []string{
				"Documentacion de uso y pruebas disponible.",
				"Checklist de seguridad cubre datos sensibles, permisos y validacion.",
				"Deploy si aplica queda descrito sin tecnologia obligatoria.",
			},
			functions: []string{"localizeAgendaText", "renderAgendaView"},
		}),
	}
}

type directorLikeWorkflowTaskSpecForTestV0 struct {
	taskRef   string
	title     string
	summary   string
	writeSet  []string
	criteria  []string
	functions []string
}

func directorLikeWorkflowTaskForCandidateProviderTestV0(
	runRef string,
	spec directorLikeWorkflowTaskSpecForTestV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	refs := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(spec.functions))
	for _, functionName := range spec.functions {
		refs = append(refs, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  "contract-agenda-core-001",
			FunctionName: functionName,
		})
	}
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        spec.taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         spec.title,
		Summary:       spec.summary,
		WriteSet:      spec.writeSet,
		AcceptanceCriteria: append(
			[]string(nil),
			spec.criteria...,
		),
		FunctionContractRefs: refs,
	}
}
