package orquestaapprunner

import (
	"context"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestPrepareAppOrchestrationV0CreatesRunAndProvider(t *testing.T) {
	spec := validFactoryAppSpecForRunnerTestV0(t)

	prepared, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		AppSpec: spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppOrchestrationV0: %v", err)
	}
	if prepared.SchemaVersion != AppOrchestrationPreparedSchemaVersionV0 {
		t.Fatalf("schema=%q", prepared.SchemaVersion)
	}
	if prepared.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		prepared.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("run inicial=%+v", prepared.Run)
	}
	if prepared.Run.AppSpecRef != spec.SpecID || prepared.Plan.AppRef != spec.App.Slug {
		t.Fatalf("refs run=%+v plan=%+v spec=%+v", prepared.Run, prepared.Plan, spec.App)
	}
	if len(prepared.Run.Tasks) != len(prepared.Plan.Units) ||
		len(prepared.Run.FunctionContracts) == 0 ||
		len(prepared.Run.Decisions) == 0 {
		t.Fatalf("run no materializa plan: tasks=%v contracts=%v decisions=%v",
			prepared.Run.Tasks, prepared.Run.FunctionContracts, prepared.Run.Decisions)
	}
	if prepared.InitialProgress.Complete ||
		!runnerStringInSetV0(prepared.InitialProgress.ReadyTaskRefs, "task-agenda-bootstrap") {
		t.Fatalf("initial progress=%+v", prepared.InitialProgress)
	}

	candidates := mustBuildRunnerCandidatesForTestV0(t, prepared)
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	first := candidates.WorkCandidates[0]
	if first.AgentCandidate == nil ||
		first.AgentCandidate.Payload.TaskRef != "task-agenda-bootstrap" {
		t.Fatalf("first candidate=%+v", first)
	}
}

func TestPrepareAppOrchestrationV0PreparaPlanGrandeDesdeAppSpec(t *testing.T) {
	spec := validFactoryAppSpecForRunnerTestV0(t)
	spec.Data.PersistenceRequired = true

	prepared, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		RunRef:     "run-app-runner-large-001",
		OccurredAt: "2026-05-09T21:10:00Z",
		AppSpec:    spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppOrchestrationV0 large: %v", err)
	}
	if len(prepared.Plan.Units) != 11 {
		t.Fatalf("large units=%d", len(prepared.Plan.Units))
	}

	first := mustBuildRunnerCandidatesForTestV0(t, prepared)
	if len(first.WorkCandidates) != 1 ||
		first.WorkCandidates[0].AgentCandidate.Payload.TaskRef != "task-agenda-bootstrap" {
		t.Fatalf("first wave=%+v", first.WorkCandidates)
	}

	nextRun := prepared.Run
	nextRun.Deliveries = append(nextRun.Deliveries, "ack-agenda-bootstrap")
	prepared.Run = nextRun
	second := mustBuildRunnerCandidatesForTestV0(t, prepared)
	if len(second.WorkCandidates) != 1 ||
		second.WorkCandidates[0].AgentCandidate.Payload.TaskRef != "task-agenda-architecture" {
		t.Fatalf("second wave=%+v", second.WorkCandidates)
	}
}

func TestPrepareAppOrchestrationV0RejectsSpecNoValidada(t *testing.T) {
	spec := validFactoryAppSpecForRunnerTestV0(t)
	spec.Validation.Estado = "provisional"

	if _, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		RunRef:     "run-app-runner-invalid-001",
		OccurredAt: "2026-05-09T21:00:00Z",
		AppSpec:    spec,
	}); err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestPreparedAppOrchestrationV0ProgressiveLoopRequestsBootstrapAgent(t *testing.T) {
	prepared, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		RunRef:        "run-app-runner-loop-001",
		ProjectRef:    "project-app-runner-loop-001",
		OccurredAt:    "2026-05-09T21:00:00Z",
		CorrelationID: "corr-app-runner-loop-001",
		RequestedBy:   "orquesta-app-runner-test",
		AppSpec:       validFactoryAppSpecForRunnerTestV0(t),
	})
	if err != nil {
		t.Fatalf("PrepareAppOrchestrationV0: %v", err)
	}

	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(prepared.Run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:          store,
		EventSink:         sink,
		CandidateProvider: prepared.CandidateProvider,
		OutboxLedger:      ledger,
		MaxCommands:       8,
		MaxOutboxPerCycle: 2,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               prepared.Run.RunID,
		OccurredAt:           "2026-05-09T21:01:00Z",
		MaxBursts:            6,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-app-runner-loop-001",
		EvidenceRefs:         []string{"evidence-ref-app-runner-loop-001"},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			runnerCapacityDispatcherForTestV0(store, sink, ledger),
			runnerAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("RunProgressiveLoopV0: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("status=%s result=%+v", result.Status, result)
	}
	if !runnerStringInSetV0(result.Run.StartedAgents, "agent-agenda-bootstrap") {
		t.Fatalf("started_agents=%v", result.Run.StartedAgents)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
}

func validFactoryAppSpecForRunnerTestV0(t *testing.T) orquestafactory.AppSpecV0 {
	t.Helper()
	observability := true
	spec, issues := orquestafactory.SolicitarNuevaAppV0(orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "request-ref-app-runner-001",
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
	}, time.Date(2026, 5, 9, 21, 0, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("SolicitarNuevaAppV0 issues: %+v", issues)
	}
	return spec
}

func mustBuildRunnerCandidatesForTestV0(
	t *testing.T,
	prepared AppOrchestrationPreparedV0,
) orquestacionnucleoapp.SchedulerCandidateSetV0 {
	t.Helper()
	candidates, err := prepared.CandidateProvider.BuildSchedulerCandidatesV0(
		context.Background(),
		orquestacionnucleoapp.SchedulerCandidateRequestV0{
			Run:           prepared.Run,
			OccurredAt:    "2026-05-09T21:00:00Z",
			CorrelationID: "corr-app-runner-candidates-001",
			EvidenceRefs:  []string{"evidence-ref-app-runner-candidates-001"},
		},
	)
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	return candidates
}

func runnerCapacityDispatcherForTestV0(
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
			Tier:            orquestacoreworkflow.OrchestrationCapacityMediumV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
			OccurredAt:      "2026-05-09T21:02:00Z",
			CorrelationID:   "corr-app-runner-capacity-001",
			RequestedBy:     "orquesta-app-runner-test",
		},
		Acker: ledger,
	}
}

func runnerAgentLauncherDispatcherForTestV0(
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
			OccurredAt:    "2026-05-09T21:03:00Z",
			CorrelationID: "corr-app-runner-agent-001",
			RequestedBy:   "orquesta-app-runner-test",
		},
		Acker: ledger,
	}
}

func runnerStringInSetV0(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
