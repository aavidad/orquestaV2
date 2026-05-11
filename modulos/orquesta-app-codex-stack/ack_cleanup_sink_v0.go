package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const ackRuntimeCleanupTimeoutV0 = 2 * time.Second

type ackRuntimeCleanupEventSinkV0 struct {
	Inner   orquestacionnucleoapp.EventSinkPortV0
	Stopper orquestacionnucleoapp.AgentStopperPortV0
	Timeout time.Duration
}

func (sink ackRuntimeCleanupEventSinkV0) AppendRunEventsV0(
	ctx context.Context,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if sink.Inner != nil {
		if err := sink.Inner.AppendRunEventsV0(ctx, runRef, events); err != nil {
			return err
		}
	}
	sink.cleanupAcceptedACKEventsV0(events)
	return nil
}

func (sink ackRuntimeCleanupEventSinkV0) cleanupAcceptedACKEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) {
	for _, event := range events {
		agentRef, ok := ackRuntimeCleanupAgentRefV0(event)
		if !ok {
			continue
		}
		sink.cleanupAgentRuntimeV0(event, agentRef)
	}
}

func (sink ackRuntimeCleanupEventSinkV0) cleanupAgentRuntimeV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	agentRef string,
) {
	if sink.Stopper == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sink.timeoutV0())
	defer cancel()
	_, _ = sink.Stopper.StopAgentV0(ctx, ackRuntimeCleanupInboundV0(event, agentRef))
}

func (sink ackRuntimeCleanupEventSinkV0) timeoutV0() time.Duration {
	if sink.Timeout > 0 {
		return sink.Timeout
	}
	return ackRuntimeCleanupTimeoutV0
}

func ackRuntimeCleanupAgentRefV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (string, bool) {
	switch strings.TrimSpace(event.EventType) {
	case orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0:
		var payload orquestacoreworkflow.DeliveryRegisteredPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return "", false
		}
		return strings.TrimSpace(payload.AgentRef), strings.TrimSpace(payload.AgentRef) != ""
	case orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0:
		var payload orquestacoreworkflow.PhaseArtifactRegisteredPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return "", false
		}
		return strings.TrimSpace(payload.AgentRef), strings.TrimSpace(payload.AgentRef) != ""
	default:
		return "", false
	}
}

func ackRuntimeCleanupInboundV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	agentRef string,
) orquestaruntime.AgentStopperInboundV0 {
	eventRef := ackRuntimeCleanupEventRefV0(event)
	return orquestaruntime.AgentStopperInboundV0{
		TargetPort:     orquestaruntime.AgentStopperTargetPortV0,
		MessageType:    orquestaruntime.AgentStopperMessageTypeV0,
		CorrelationID:  ackRuntimeCleanupCorrelationIDV0(event),
		IdempotencyKey: "idem-app-stack-ack-cleanup-" + eventRef,
		Payload: &orquestaruntime.StopRuntimeAgentRequestV0{
			RunID:          strings.TrimSpace(event.RunID),
			AgentRequestID: strings.TrimSpace(agentRef),
			ReasonCode:     "ack_registered_cleanup",
			Summary:        "Limpieza terminal de runtime tras ACK registrado.",
			EvidenceRefs: []string{
				"evidence-ref-app-stack-ack-cleanup",
				eventRef,
			},
		},
	}
}

func ackRuntimeCleanupCorrelationIDV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) string {
	correlationID := strings.TrimSpace(event.CorrelationID)
	if correlationID != "" {
		return correlationID
	}
	return "corr-app-stack-ack-cleanup-" + ackRuntimeCleanupEventRefV0(event)
}

func ackRuntimeCleanupEventRefV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) string {
	if event.Sequence > 0 {
		return fmt.Sprintf("evtseq-%d", event.Sequence)
	}
	eventID := strings.TrimSpace(event.EventID)
	if len(eventID) <= 80 && eventID != "" {
		return eventID
	}
	return "event-ref"
}
