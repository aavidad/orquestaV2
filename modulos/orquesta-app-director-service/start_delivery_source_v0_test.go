package orquestaappdirectorservice

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

const serviceDirectorArtifactRefForTestV0 = "artifact-ref-app-director-service-001"

func TestStartAppDirectorV0ConsumesDirectorDeliverySource(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	request := validStartAppDirectorRequestForTestV0()
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
			DeliverySource: serviceDirectorArtifactSourceForTestV0{},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("result=%+v", result)
	}
	if !serviceProjectionContainsV0(result.Run.PhaseArtifacts, serviceDirectorArtifactRefForTestV0) {
		t.Fatalf("phase_artifacts=%v", result.Run.PhaseArtifacts)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0) {
		t.Fatalf("sink sin PhaseArtifactRegistered: %+v", sink.EventsV0())
	}
}

type serviceDirectorArtifactSourceForTestV0 struct {
	AgentRef string
}

func (source serviceDirectorArtifactSourceForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	agentRef := serviceDirectorDeliveryAgentRefForTestV0(request.Run.StartedAgents, source.AgentRef)
	if agentRef == "" ||
		serviceProjectionContainsV0(request.Run.PhaseArtifacts, serviceDirectorArtifactRefForTestV0) {
		return nil, nil
	}
	return []orquestacionnucleoapp.AgentDeliveryObservationV0{{
		ArtifactRef:  serviceDirectorArtifactRefForTestV0,
		DeliveryRef:  "receipt-ref-app-director-service-001",
		PhaseID:      string(request.Run.CurrentPhase),
		AgentRef:     agentRef,
		Summary:      "Arquitectura inicial y plan de microtareas entregados por director.",
		EvidenceRefs: []string{"evidence-ref-app-director-delivery-source-001"},
	}}, nil
}

func serviceDirectorDeliveryAgentRefForTestV0(started []string, configured string) string {
	if strings.TrimSpace(configured) != "" {
		if serviceStringInSetV0(started, configured) {
			return configured
		}
		return ""
	}
	if len(started) == 0 {
		return ""
	}
	return started[0]
}

func serviceProjectionContainsV0(values []string, expected string) bool {
	for _, value := range values {
		if strings.Contains(value, expected) {
			return true
		}
	}
	return false
}
