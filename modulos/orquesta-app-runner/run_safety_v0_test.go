package orquestaapprunner

import (
	"context"
	"errors"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
	"testing"
)

var errRunnerTransientStoreForTestV0 = errors.New("store temporalmente no disponible")

func TestRunPreparedAppOrchestrationV0NoReiniciaRunSiLoadFalla(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := &runnerTransientLoadRunStoreV0{}
	_, err := RunPreparedAppOrchestrationV0(
		context.Background(),
		validRunPreparedRequestForTestV0(prepared),
		RunPreparedAppOrchestrationPortsV0{
			RunStore:     store,
			EventSink:    orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			OutboxLedger: orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{{
				TargetPort: "unused-test-dispatcher",
			}},
		},
	)

	if !errors.Is(err, errRunnerTransientStoreForTestV0) {
		t.Fatalf("err=%v", err)
	}
	if store.SaveCalled {
		t.Fatalf("SaveRunV0 no debe llamarse si LoadRunV0 falla por infraestructura")
	}
}

func TestRunPreparedAppOrchestrationV0ReconstruyeProviderDesdePlan(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	prepared.CandidateProvider = orquestaappplanner.AppPlanCandidateProviderV0{}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := RunPreparedAppOrchestrationV0(
		context.Background(),
		validRunPreparedRequestForTestV0(prepared),
		validRunPreparedPortsForTestV0(store, sink, ledger),
	)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0: %v", err)
	}
	if !runnerStringInSetV0(result.StartedAgents, "agent-agenda-bootstrap") {
		t.Fatalf("started=%v", result.StartedAgents)
	}
}

type runnerTransientLoadRunStoreV0 struct {
	SaveCalled bool
}

func (store *runnerTransientLoadRunStoreV0) LoadRunV0(
	context.Context,
	string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	return orquestacoreworkflow.OrchestrationRunV0{}, errRunnerTransientStoreForTestV0
}

func (store *runnerTransientLoadRunStoreV0) SaveRunV0(
	context.Context,
	orquestacoreworkflow.OrchestrationRunV0,
) error {
	store.SaveCalled = true
	return nil
}
