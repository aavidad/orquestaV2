package orquestaappdirectorintake

import (
	"context"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestPrepareAppDirectorIntakeV0AutonomiaAltaCreatesDirectorTeam(t *testing.T) {
	prepared := mustPrepareHighAutonomyDirectorIntakeForTestV0(t)

	if len(prepared.DirectorTasks) != 4 {
		t.Fatalf("director_tasks=%d %+v", len(prepared.DirectorTasks), prepared.DirectorTasks)
	}
	if len(prepared.InitialEvents) != 2+len(prepared.DirectorTasks) {
		t.Fatalf("initial_events=%d tasks=%d", len(prepared.InitialEvents), len(prepared.DirectorTasks))
	}
	assertDirectorIntakeRolesV0(t, prepared.DirectorTasks,
		"director",
		"director_web",
		"director_api",
		"director_persistencia",
	)
	for _, task := range prepared.DirectorTasks {
		if !directorIntakeStringInSetV0(prepared.Run.Brainstorms, task.BrainstormRef) {
			t.Fatalf("brainstorms=%v missing=%s", prepared.Run.Brainstorms, task.BrainstormRef)
		}
	}
}

func TestAppDirectorCandidateProviderV0EmitsDirectorTeam(t *testing.T) {
	prepared := mustPrepareHighAutonomyDirectorIntakeForTestV0(t)

	candidates := mustBuildDirectorIntakeCandidatesForTestV0(t, prepared)

	if len(candidates.WorkCandidates) != maxAppDirectorWorkCandidatesPerTickV0 {
		t.Fatalf("work candidates=%d max=%d", len(candidates.WorkCandidates), maxAppDirectorWorkCandidatesPerTickV0)
	}
	for index, candidate := range candidates.WorkCandidates {
		task := prepared.DirectorTasks[index]
		if candidate.AgentCandidate == nil ||
			candidate.AgentCandidate.Payload.AgentRequestID != task.AgentRequestID ||
			candidate.AgentCandidate.Payload.Role != task.Role {
			t.Fatalf("candidate[%d]=%+v task=%+v", index, candidate.AgentCandidate, task)
		}
		if len(candidate.Claims) != 1 || candidate.Claims[0].ClaimRef != task.ClaimRef {
			t.Fatalf("claims[%d]=%+v task=%+v", index, candidate.Claims, task)
		}
	}
}

func TestAppDirectorIntakeV0ProgressiveLoopStartsDirectorTeamInBatch(t *testing.T) {
	prepared := mustPrepareHighAutonomyDirectorIntakeForTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(prepared.Run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	batchExecutor := &directorIntakeBatchLauncherForTestV0{
		executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:13:00Z",
			CorrelationID: "corr-app-director-team-agent-001",
			RequestedBy:   "orquesta-app-director-intake-test",
		},
	}
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:          store,
		EventSink:         sink,
		CandidateProvider: prepared.CandidateProvider,
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               prepared.Run.RunID,
		OccurredAt:           "2026-05-09T22:11:00Z",
		MaxBursts:            8,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 8,
		CorrelationID:        "corr-app-director-team-loop-001",
		EvidenceRefs:         []string{"evidence-ref-app-director-team-loop-001"},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			directorIntakeCapacityDispatcherForTestV0(store, sink, ledger),
		},
		BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			MaxReady:   4,
			Reader:     ledger,
			Claimer:    ledger,
			Executor:   batchExecutor,
			Acker:      ledger,
		}},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v result=%+v", err, result)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	for _, task := range prepared.DirectorTasks {
		if !directorIntakeStringInSetV0(result.Run.StartedAgents, task.AgentRequestID) {
			t.Fatalf("started_agents=%v missing=%s", result.Run.StartedAgents, task.AgentRequestID)
		}
	}
	if batchExecutor.totalBatchSize != len(prepared.DirectorTasks) {
		t.Fatalf("total_batch_size=%d tasks=%d", batchExecutor.totalBatchSize, len(prepared.DirectorTasks))
	}
	if batchExecutor.maxBatchSize > maxAppDirectorWorkCandidatesPerTickV0 {
		t.Fatalf("max_batch_size=%d max=%d", batchExecutor.maxBatchSize, maxAppDirectorWorkCandidatesPerTickV0)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

type directorIntakeBatchLauncherForTestV0 struct {
	executor       orquestacionnucleoapp.AgentLauncherExecutorV0
	lastBatchSize  int
	totalBatchSize int
	maxBatchSize   int
}

func (executor *directorIntakeBatchLauncherForTestV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.lastBatchSize = len(intents)
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

func mustPrepareHighAutonomyDirectorIntakeForTestV0(t *testing.T) AppDirectorIntakePreparedV0 {
	t.Helper()
	prepared, err := PrepareAppDirectorIntakeV0(PrepareAppDirectorIntakeRequestV0{
		RunRef:        "run-app-director-team-001",
		ProjectRef:    "project-app-director-team-001",
		OccurredAt:    "2026-05-09T22:10:00Z",
		CorrelationID: "corr-app-director-team-001",
		RequestedBy:   "orquesta-app-director-intake-test",
		AppSpec:       validHighAutonomyAppSpecForDirectorIntakeTestV0(t),
	})
	if err != nil {
		t.Fatalf("PrepareAppDirectorIntakeV0: %v", err)
	}
	return prepared
}

func validHighAutonomyAppSpecForDirectorIntakeTestV0(t *testing.T) orquestafactory.AppSpecV0 {
	t.Helper()
	observability := true
	spec, issues := orquestafactory.SolicitarNuevaAppV0(orquestafactory.AppSpecRequestV0{
		SchemaVersion:    orquestafactory.AppSpecRequestSchemaV0,
		RequestID:        "request-ref-app-director-team-001",
		Source:           "orquesta-web",
		Locale:           "es-ES",
		Nombre:           "Agenda",
		Objetivo:         "Gestionar contactos y citas desde una API y una web.",
		TipoApp:          "mixed",
		UsuariosObjetivo: []string{"usuarios internos"},
		PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
		},
		Datos: orquestafactory.DatosRequestV0{
			DBRequired:         true,
			NecesidadFuncional: "Guardar contactos y citas mediante un puerto de persistencia.",
		},
		Calidad: orquestafactory.CalidadRequestV0{
			Pruebas:        "alta",
			Accesibilidad:  "basica",
			Observabilidad: &observability,
		},
		Agentes: orquestafactory.AgentesRequestV0{
			Autonomia: "alta",
		},
	}, time.Date(2026, 5, 9, 22, 10, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("SolicitarNuevaAppV0 issues: %+v", issues)
	}
	return spec
}

func assertDirectorIntakeRolesV0(t *testing.T, tasks []AppDirectorTaskV0, roles ...string) {
	t.Helper()
	for _, role := range roles {
		found := false
		for _, task := range tasks {
			if task.Role == role {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("role %s not found in %+v", role, tasks)
		}
	}
}
