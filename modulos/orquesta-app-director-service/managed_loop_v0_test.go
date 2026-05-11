package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestStartAppDirectorV0WaitsExternalBeforeConsumingDirectorDecisions(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &serviceDelayedDecisionSourceForTestV0{}
	deliverySource := &serviceGatedDirectorDeliverySourceForTestV0{}
	waiter := &serviceDecisionReadyWaiterForTestV0{
		Source:         source,
		DeliverySource: deliverySource,
	}

	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = "run-app-director-service-managed-wait-001"
	request.ProjectRef = "project-app-director-service-managed-wait-001"
	request.CorrelationID = "corr-app-director-service-managed-wait-001"
	request.AppSpecRequest.RequestID = "request-ref-app-director-service-managed-wait-001"
	request.MaxBursts = 8
	request.MaxDecisionCycles = 2

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DeliverySource:         deliverySource,
			DirectorDecisionSource: source,
			DirectorTaskStore:      taskStore,
			ExternalWaiter:         waiter,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if waiter.calls == 0 || !source.ready || source.calls == 0 {
		t.Fatalf("waiter_calls=%d source_ready=%v source_calls=%d", waiter.calls, source.ready, source.calls)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	taskRef := "task-ref-agenda-autonomy-001"
	if !serviceStringInSetV0(result.Run.Tasks, taskRef) {
		t.Fatalf("tasks=%v missing=%s", result.Run.Tasks, taskRef)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
		pendingRefs := servicePendingOutboxRefsForTestV0(t, ledger, result.Run.RunID)
		t.Fatalf("started_agents=%v missing=%s pending=%v", result.Run.StartedAgents, agentRef, pendingRefs)
	}
}

type serviceDelayedDecisionSourceForTestV0 struct {
	ready   bool
	calls   int
	emitted bool
}

func (source *serviceDelayedDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	if !source.ready || source.emitted {
		return nil, nil
	}
	source.emitted = true
	return serviceDirectorPlanDecisionsForTestV0(request.Run), nil
}

type serviceDecisionReadyWaiterForTestV0 struct {
	Source         *serviceDelayedDecisionSourceForTestV0
	DeliverySource *serviceGatedDirectorDeliverySourceForTestV0
	calls          int
}

func (waiter *serviceDecisionReadyWaiterForTestV0) WaitExternalProgressV0(
	ctx context.Context,
	_ orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.ExternalProgressWaitResultV0{}, err
	}
	waiter.calls++
	waiter.Source.ready = true
	waiter.DeliverySource.ready = true
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-app-director-managed-wait-001"},
	}, nil
}

type serviceGatedDirectorDeliverySourceForTestV0 struct {
	AgentRef string
	ready    bool
}

func (source *serviceGatedDirectorDeliverySourceForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	agentRef := serviceDirectorDeliveryAgentRefForTestV0(request.Run.StartedAgents, source.AgentRef)
	if source == nil ||
		!source.ready ||
		agentRef == "" ||
		serviceProjectionContainsV0(request.Run.PhaseArtifacts, serviceDirectorArtifactRefForTestV0) {
		return nil, nil
	}
	return []orquestacionnucleoapp.AgentDeliveryObservationV0{{
		ArtifactRef:  serviceDirectorArtifactRefForTestV0,
		DeliveryRef:  "receipt-ref-app-director-service-managed-wait-001",
		PhaseID:      string(request.Run.CurrentPhase),
		AgentRef:     agentRef,
		Summary:      "Arquitectura inicial disponible tras espera externa.",
		EvidenceRefs: []string{"evidence-ref-app-director-managed-delivery-001"},
	}}, nil
}
