package orquestaappdirectorservice

import (
	"context"
	"errors"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

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

func mustServiceStateFileStoreForTestV0(
	t *testing.T,
	rootDir string,
) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func serviceCapacityDispatcherWithPortsForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
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
			OccurredAt:      "2026-05-21T13:10:01Z",
			CorrelationID:   "corr-app-director-service-file-capacity-001",
			RequestedBy:     "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

func serviceAgentLauncherDispatcherWithPortsForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
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
			OccurredAt:    "2026-05-21T13:10:02Z",
			CorrelationID: "corr-app-director-service-file-agent-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
		Acker: ledger,
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

func serviceDirectorDecisionCommandMetaForTestV0(
	runRef string,
	action string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "command-ref-director-decision-" + action,
		RunID:          runRef,
		IdempotencyKey: "idem-director-decision-" + action,
		CorrelationID:  "corr-director-decision-" + action,
		RequestedBy:    "orquesta-app-director-service-test",
		OccurredAt:     "2026-05-22T12:59:00Z",
	}
}

func serviceMustDirectorDecisionCommandForTestV0(
	t *testing.T,
	build func() (orquestacoreworkflow.OrchestrationCommandV0, error),
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := build()
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	return command
}

func serviceApplyDirectorDecisionCommandForTestV0(
	t *testing.T,
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(context.Background(), store, sink, command); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0(%s): %v", command.CommandType, err)
	}
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
