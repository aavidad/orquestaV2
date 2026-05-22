package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestStartAppDirectorV0StartsDirectorThroughInjectedPorts(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		t.Fatalf("phase=%q", result.Run.CurrentPhase)
	}
	if !serviceStringInSetV0(result.StartedAgents, result.DirectorTask.AgentRequestID) {
		t.Fatalf("started=%v task=%+v", result.StartedAgents, result.DirectorTask)
	}
	if result.DirectorTask.TaskRef == "" || len(result.Run.Tasks) != 0 {
		t.Fatalf("director_task no debe materializar microtarea inicial: task=%+v run_tasks=%v", result.DirectorTask, result.Run.Tasks)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0) {
		t.Fatalf("eventos sin BrainstormRequested: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	batchExecutor := &serviceBatchLauncherForTestV0{
		executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:34:00Z",
			CorrelationID: "corr-app-director-service-agent-team-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validHighAutonomyStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
			},
			BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
				TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
				MaxReady:   4,
				Reader:     ledger,
				Claimer:    ledger,
				Executor:   batchExecutor,
				Acker:      ledger,
			}},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.DirectorTasks) != 4 {
		t.Fatalf("director_tasks=%d %+v", len(result.DirectorTasks), result.DirectorTasks)
	}
	for _, task := range result.DirectorTasks {
		if !serviceStringInSetV0(result.StartedAgents, task.AgentRequestID) {
			t.Fatalf("started=%v missing=%s", result.StartedAgents, task.AgentRequestID)
		}
	}
	if batchExecutor.totalBatchSize != len(result.DirectorTasks) {
		t.Fatalf("batch_total=%d tasks=%d", batchExecutor.totalBatchSize, len(result.DirectorTasks))
	}
	if batchExecutor.maxBatchSize > len(result.DirectorTasks) {
		t.Fatalf("batch_max=%d", batchExecutor.maxBatchSize)
	}
}

func TestStartAppDirectorV0ReturnsFactoryValidationIssues(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.AppSpecRequest.Locale = "locale invalido"

	result, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err != nil {
		t.Fatalf("request invalida no debe ser error de transporte: %v", err)
	}
	if result.Status != StartAppDirectorStatusInvalidV0 || len(result.ValidationIssues) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Run.RunID != "" || len(result.StartedAgents) != 0 {
		t.Fatalf("request invalida no debe crear run: %+v", result)
	}
}

func TestStartAppDirectorV0RejectsOperationalDirectorPlanInicialSinContratoFuncional(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.OperationalDirectorPlan = serviceOperationalDirectorPlanForContinueTestV0(t, request.RunRef)

	_, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err == nil || err.Error() != "app_director_service_invalido: operational_director_function_contract_refs" {
		t.Fatalf("err=%v", err)
	}
}

func TestStartAppDirectorV0RequiresInjectedPorts(t *testing.T) {
	_, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{},
	)
	if err == nil {
		t.Fatalf("esperaba error por puertos ausentes")
	}
}

func validStartAppDirectorRequestForTestV0() StartAppDirectorRequestV0 {
	observability := true
	return StartAppDirectorRequestV0{
		RunRef:        "run-app-director-service-001",
		ProjectRef:    "project-app-director-service-001",
		OccurredAt:    "2026-05-09T22:30:00Z",
		CorrelationID: "corr-app-director-service-001",
		RequestedBy:   "orquesta-app-director-service-test",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-ref-app-director-service-001",
			Source:        "orquesta-web",
			Locale:        "es-ES",
			Nombre:        "Agenda",
			Objetivo:      "Gestionar contactos y citas desde una API y una web.",
			TipoApp:       "mixed",
			PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
				Lenguaje:     "go",
				Arquitectura: "hexagonal",
			},
			Calidad: orquestafactory.CalidadRequestV0{
				Pruebas:        "media",
				Accesibilidad:  "basica",
				Observabilidad: &observability,
			},
		},
	}
}

func validHighAutonomyStartAppDirectorRequestForTestV0() StartAppDirectorRequestV0 {
	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = "run-app-director-service-team-001"
	request.ProjectRef = "project-app-director-service-team-001"
	request.CorrelationID = "corr-app-director-service-team-001"
	request.AppSpecRequest.RequestID = "request-ref-app-director-service-team-001"
	request.AppSpecRequest.Datos = orquestafactory.DatosRequestV0{
		DBRequired:         true,
		NecesidadFuncional: "Guardar contactos y citas mediante un puerto de persistencia.",
	}
	request.AppSpecRequest.Calidad.Pruebas = "alta"
	request.AppSpecRequest.Agentes = orquestafactory.AgentesRequestV0{Autonomia: "alta"}
	return request
}

func serviceCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-09T22:31:00Z",
			CorrelationID:   "corr-app-director-service-capacity-001",
			RequestedBy:     "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

func serviceAgentLauncherDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:32:00Z",
			CorrelationID: "corr-app-director-service-agent-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

type serviceBatchLauncherForTestV0 struct {
	executor       orquestacionnucleoapp.AgentLauncherExecutorV0
	totalBatchSize int
	maxBatchSize   int
}

func (executor *serviceBatchLauncherForTestV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.totalBatchSize += len(intents)
	if len(intents) > executor.maxBatchSize {
		executor.maxBatchSize = len(intents)
	}
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(intents))
	for _, intent := range intents {
		execution, err := executor.executor.ExecuteOutboxDispatchV0(intent)
		if err != nil {
			return nil, err
		}
		acks = append(acks, orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			MessageID:    intent.MessageID,
			RunID:        intent.RunID,
			TargetPort:   intent.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
			DispatchRef:  execution.DispatchRef,
			EvidenceRefs: execution.EvidenceRefs,
		})
	}
	return acks, nil
}

func serviceStringInSetV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func serviceHasEventTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
