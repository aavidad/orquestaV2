package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStartAppDirectorV0ConsumesDirectorDecisionSource(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := serviceDirectorDecisionSourceForTestV0{
		Decisions: []orquestadirectoragent.DirectorAgentDecisionV0{
			serviceOpenVoteDirectorDecisionForTestV0("run-app-director-service-001"),
		},
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DirectorDecisionSource: source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseOpenedV0) {
		t.Fatalf("sink sin PhaseOpened: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0IgnoresPersistedDirectorDecisionsAlreadyApplied(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &servicePersistentDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			DeliverySource: serviceDirectorArtifactSourceForTestV0{
				AgentRef: "agent-agenda-director",
			},
			DirectorDecisionSource: source,
			DirectorTaskStore:      taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if source.calls < 2 {
		t.Fatalf("decision_source_calls=%d", source.calls)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if len(result.Run.Tasks) != 1 || result.Run.Tasks[0] != "task-ref-agenda-autonomy-001" {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
}

func TestStartAppDirectorV0NoSaltaDecisionPendienteDelDirector(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 3
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DeliverySource:         serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource: servicePendingDecisionSourceForTestV0{},
			DirectorTaskStore:      taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 {
		t.Fatalf("current_phase=%s, want votacion_y_decision", result.Run.CurrentPhase)
	}
	if len(result.Run.Tasks) != 0 {
		t.Fatalf("tasks=%v, want empty", result.Run.Tasks)
	}
}

func TestStartAppDirectorV0BloqueaDecisionInvalidaDelSource(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	decision := serviceMicrotaskDecisionForTestV0("run-app-director-service-001")
	decision.CreateMicrotask.Task.Summary = ""

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			DirectorDecisionSource: serviceDirectorDecisionSourceForTestV0{
				Decisions: []orquestadirectoragent.DirectorAgentDecisionV0{decision},
			},
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	requireServiceDecisionRecoveryBlockV0(
		t,
		result.Run,
		sink.EventsV0(),
		"director_decision.decision.create_microtask.task.summary",
	)
	got, loadErr := store.LoadRunV0(context.Background(), result.Run.RunID)
	if loadErr != nil {
		t.Fatalf("LoadRunV0: %v", loadErr)
	}
	if got.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("stored status=%s", got.Status)
	}
}

func TestStartAppDirectorV0BloqueaErrorDelDecisionSourceSinReintentar(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &serviceFailingDecisionSourceForTestV0{
		err: errors.New("decision source unavailable"),
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DirectorDecisionSource: source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if source.calls != 1 {
		t.Fatalf("decision_source_calls=%d, want 1", source.calls)
	}
	requireServiceDecisionRecoveryBlockV0(
		t,
		result.Run,
		sink.EventsV0(),
		"director_decision_source",
	)
}

type serviceDirectorDecisionSourceForTestV0 struct {
	Decisions []orquestadirectoragent.DirectorAgentDecisionV0
}

func (source serviceDirectorDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), source.Decisions...), nil
}

type servicePersistentDecisionSourceForTestV0 struct {
	calls int
}

func (source *servicePersistentDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	return serviceDirectorPlanDecisionsForTestV0(request.Run), nil
}

type servicePendingDecisionSourceForTestV0 struct{}

func (source servicePendingDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decisions := serviceDirectorPlanDecisionsForTestV0(request.Run)
	for i := range decisions {
		if decisions[i].AcceptDecision != nil {
			decisions[i].AcceptDecision.VoteRef = "vote-ref-pending-mismatch-001"
		}
	}
	return decisions, nil
}

type serviceFailingDecisionSourceForTestV0 struct {
	calls int
	err   error
}

func (source *serviceFailingDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	return nil, source.err
}

func serviceOpenVoteDirectorDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-service-open-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-service-open-vote-001",
		Summary:       "Abrir fase de decision.",
		EvidenceRefs:  []string{"evidence-ref-service-open-vote-001"},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Reason:  "Preparar votacion del director.",
		},
	}
}

func requireServiceDecisionRecoveryBlockV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
	field string,
) {
	t.Helper()
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%s, want bloqueada", run.Status)
	}
	if len(run.Blockers) != 1 || !strings.HasPrefix(run.Blockers[0], "app-director-decision-") {
		t.Fatalf("blockers=%v", run.Blockers)
	}
	for _, event := range events {
		if event.EventType != orquestacoreworkflow.OrchestrationEventRunBlockedV0 {
			continue
		}
		var payload orquestacoreworkflow.RunBlockedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("RunBlocked payload: %v", err)
		}
		if strings.Contains(payload.Summary, "field="+field) {
			return
		}
	}
	t.Fatalf("sin RunBlocked con field=%s: %+v", field, events)
}
