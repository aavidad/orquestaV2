package orquestaappcodexstack

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func codexStackRealSmokeAssertProgrammingParallelWaveV0(
	t *testing.T,
	events []orquestacoreworkflow.OrchestrationEventV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	minStartedBeforeDelivery int,
) {
	t.Helper()
	got := codexStackRealSmokeMaxProgrammingStartsBeforeDeliveryV0(events, descriptors)
	if got < minStartedBeforeDelivery {
		t.Fatalf(
			"programacion sin ola paralela real: max_started_before_delivery=%d want>=%d programming=%v",
			got,
			minStartedBeforeDelivery,
			codexStackRealSmokeProgrammingDescriptorsV0(descriptors),
		)
	}
}

func codexStackRealSmokeMaxProgrammingStartsBeforeDeliveryV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) int {
	programmingAgents := map[string]bool{}
	programmingDeliveries := map[string]bool{}
	for _, descriptor := range codexStackRealSmokeProgrammingReceiptDescriptorsV0(descriptors) {
		programmingAgents[descriptor.AgentRef] = true
		programmingDeliveries[descriptor.Spec.AgentPacket.DeliveryRefs.AckRef] = true
	}
	current := 0
	maxStarted := 0
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventAgentStartedV0:
			var payload orquestacoreworkflow.AgentStartedPayloadV0
			if json.Unmarshal(event.Payload, &payload) != nil || !programmingAgents[payload.AgentRequestID] {
				continue
			}
			current++
			if current > maxStarted {
				maxStarted = current
			}
		case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
			var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
			if json.Unmarshal(event.Payload, &payload) == nil && programmingDeliveries[payload.DeliveryRef] {
				current = 0
			}
		}
	}
	return maxStarted
}
