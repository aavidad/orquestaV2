package orquestaapprunner

import (
	"context"
	"testing"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestRunPreparedAppOrchestrationV0ArrancaBootstrapGrande(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
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
	if result.Status != AppOrchestrationRunStatusRunningV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if !runnerStringInSetV0(result.StartedAgents, "agent-agenda-bootstrap") {
		t.Fatalf("started=%v", result.StartedAgents)
	}
	if result.Progress.DeliveredUnits != 0 || result.Progress.TotalUnits != 11 {
		t.Fatalf("progress=%+v", result.Progress)
	}
}

func TestRunPreparedAppOrchestrationV0RegistraEntregaYAvanzaOla(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &runnerGatedDeliverySourceForTestV0{
		Observation: runnerDeliveryObservationForUnitTestV0(prepared.Plan.Units[0]),
	}
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.DeliverySource = source
	ports.ExternalWaiter = &runnerEnableDeliveryWaiterForTestV0{Source: source}
	request := validRunPreparedRequestForTestV0(prepared)
	request.MaxExternalWaits = 1

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0 managed: %v", err)
	}
	if result.Attempts != 2 || result.ExternalWaits != 1 {
		t.Fatalf("attempts=%d waits=%d result=%+v", result.Attempts, result.ExternalWaits, result)
	}
	if !runnerStringInSetV0(result.Run.Deliveries, prepared.Plan.Units[0].DeliveryRef) {
		t.Fatalf("deliveries=%v", result.Run.Deliveries)
	}
	if !runnerStringInSetV0(result.StartedAgents, "agent-agenda-architecture") {
		t.Fatalf("started=%v", result.StartedAgents)
	}
	if result.Progress.DeliveredUnits != 1 {
		t.Fatalf("progress=%+v", result.Progress)
	}
}

func TestRunPreparedAppOrchestrationV0CompletaPlanGrandeConEntregas(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.DeliverySource = runnerStartedAgentDeliverySourceForTestV0{Plan: prepared.Plan}
	request := validRunPreparedRequestForTestV0(prepared)
	request.MaxBursts = 120
	request.MaxStepsPerBurst = 8
	request.MaxDispatchesPerWait = 8
	request.MaxCommands = 24
	request.MaxOutboxPerCycle = 24

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		stored, _ := store.LoadRunV0(context.Background(), prepared.Run.RunID)
		t.Fatalf("RunPreparedAppOrchestrationV0 complete: %T %#v result=%+v stored=%+v", err, err, result, stored)
	}
	if result.Status != AppOrchestrationRunStatusCompleteV0 || !result.Progress.Complete {
		t.Fatalf("result=%+v", result)
	}
	if result.Progress.DeliveredUnits != len(prepared.Plan.Units) ||
		len(result.StartedAgents) != len(prepared.Plan.Units) {
		t.Fatalf("progress=%+v started=%v", result.Progress, result.StartedAgents)
	}
}

func TestRunPreparedAppOrchestrationV0EsperaExternaPorDefectoHastaCompletar(t *testing.T) {
	prepared := validPreparedLargeAppForRunTestV0(t)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := newRunnerStepwiseDeliverySourceForTestV0(prepared.Plan)
	ports := validRunPreparedPortsForTestV0(store, sink, ledger)
	ports.DeliverySource = source
	ports.ExternalWaiter = source
	request := validRunPreparedRequestForTestV0(prepared)
	request.MaxExternalWaits = 0

	result, err := RunPreparedAppOrchestrationV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("RunPreparedAppOrchestrationV0 wait default: %v", err)
	}
	if result.Status != AppOrchestrationRunStatusCompleteV0 || !result.Progress.Complete {
		t.Fatalf("result=%+v", result)
	}
	if result.ExternalWaits == 0 {
		t.Fatalf("external waits no usados: %+v", result)
	}
}

func validPreparedLargeAppForRunTestV0(t *testing.T) AppOrchestrationPreparedV0 {
	t.Helper()
	spec := validFactoryAppSpecForRunnerTestV0(t)
	spec.Data.PersistenceRequired = true
	prepared, err := PrepareAppOrchestrationV0(PrepareAppOrchestrationRequestV0{
		RunRef:     "run-app-runner-execute-001",
		ProjectRef: "project-app-runner-execute-001",
		OccurredAt: "2026-05-09T23:50:00Z",
		AppSpec:    spec,
	})
	if err != nil {
		t.Fatalf("PrepareAppOrchestrationV0: %v", err)
	}
	return prepared
}

func validRunPreparedRequestForTestV0(
	prepared AppOrchestrationPreparedV0,
) RunPreparedAppOrchestrationRequestV0 {
	return RunPreparedAppOrchestrationRequestV0{
		Prepared:             prepared,
		OccurredAt:           "2026-05-09T23:51:00Z",
		CorrelationID:        "corr-app-runner-execute-001",
		RequestedBy:          "orquesta-app-runner-test",
		MaxBursts:            10,
		MaxStepsPerBurst:     6,
		MaxDispatchesPerWait: 4,
		MaxCommands:          12,
		MaxOutboxPerCycle:    4,
	}
}

func validRunPreparedPortsForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) RunPreparedAppOrchestrationPortsV0 {
	return RunPreparedAppOrchestrationPortsV0{
		RunStore:     store,
		EventSink:    sink,
		OutboxLedger: ledger,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			runnerCapacityDispatcherForTestV0(store, sink, ledger),
			runnerAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	}
}

func runnerDeliveryObservationForUnitTestV0(unit orquestaappplanner.AppWorkUnitV0) orquestacionnucleoapp.AgentDeliveryObservationV0 {
	return orquestacionnucleoapp.AgentDeliveryObservationV0{
		CandidateRef: "delivery-candidate-ref-" + unit.DeliveryRef,
		DeliveryRef:  unit.DeliveryRef,
		PhaseID:      string(unit.PhaseID),
		TaskID:       unit.TaskRef,
		AgentRef:     unit.AgentRequestID,
		Summary:      "Entrega compacta de prueba.",
		EvidenceRefs: []string{"evidence-ref-app-runner-delivery-001"},
	}
}

type runnerStartedAgentDeliverySourceForTestV0 struct {
	Plan orquestaappplanner.AppMicrotaskPlanV0
}

func (source runnerStartedAgentDeliverySourceForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0, len(source.Plan.Units))
	for _, unit := range source.Plan.Units {
		if !runnerStringInSetV0(request.Run.StartedAgents, unit.AgentRequestID) ||
			runnerStringInSetV0(request.Run.Deliveries, unit.DeliveryRef) {
			continue
		}
		observations = append(observations, runnerDeliveryObservationForUnitTestV0(unit))
	}
	return observations, nil
}

type runnerStepwiseDeliverySourceForTestV0 struct {
	Plan        orquestaappplanner.AppMicrotaskPlanV0
	ReadyAgents map[string]bool
}

func newRunnerStepwiseDeliverySourceForTestV0(
	plan orquestaappplanner.AppMicrotaskPlanV0,
) *runnerStepwiseDeliverySourceForTestV0 {
	return &runnerStepwiseDeliverySourceForTestV0{
		Plan:        plan,
		ReadyAgents: map[string]bool{},
	}
}

func (source *runnerStepwiseDeliverySourceForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0, len(source.Plan.Units))
	for _, unit := range source.Plan.Units {
		if !source.ReadyAgents[unit.AgentRequestID] ||
			!runnerStringInSetV0(request.Run.StartedAgents, unit.AgentRequestID) ||
			runnerStringInSetV0(request.Run.Deliveries, unit.DeliveryRef) {
			continue
		}
		observations = append(observations, runnerDeliveryObservationForUnitTestV0(unit))
	}
	return observations, nil
}

func (source *runnerStepwiseDeliverySourceForTestV0) WaitExternalProgressV0(
	_ context.Context,
	request orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	for _, agentRef := range request.LastResult.Run.StartedAgents {
		source.ReadyAgents[agentRef] = true
	}
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-app-runner-stepwise-wait-001"},
	}, nil
}

type runnerGatedDeliverySourceForTestV0 struct {
	Ready       bool
	Observation orquestacionnucleoapp.AgentDeliveryObservationV0
}

func (source *runnerGatedDeliverySourceForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	if source == nil || !source.Ready ||
		runnerStringInSetV0(request.Run.Deliveries, source.Observation.DeliveryRef) {
		return nil, nil
	}
	return []orquestacionnucleoapp.AgentDeliveryObservationV0{source.Observation}, nil
}

type runnerEnableDeliveryWaiterForTestV0 struct {
	Source *runnerGatedDeliverySourceForTestV0
}

func (waiter *runnerEnableDeliveryWaiterForTestV0) WaitExternalProgressV0(
	context.Context,
	orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	waiter.Source.Ready = true
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-app-runner-wait-001"},
	}, nil
}
