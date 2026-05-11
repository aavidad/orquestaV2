package orquestaappdirectorservice

import (
	"context"
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
