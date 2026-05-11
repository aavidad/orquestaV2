package orquestacoreworkflow

import "testing"

func mustDeliveryReplayEventsV0(t *testing.T) []OrchestrationEventV0 {
	t.Helper()
	return []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-delivery", 1, "idem-start-delivery"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-vote-delivery", 2, "idem-open-vote-delivery", OrchestrationPhaseVotacionYDecisionV0),
		mustVoteRequestedEventWithKeyV0(t, "evt-durable-vote-delivery", 3, "idem-vote-delivery", "vote-request-001"),
		mustArchitectureDecisionAcceptedEventWithKeyV0(t, "evt-durable-decision-delivery", 4, "idem-decision-delivery", "decision-001"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-plan-delivery", 5, "idem-open-plan-delivery", OrchestrationPhasePlanificacionMicrotareasV0),
		mustFunctionContractPublishedEventWithKeyV0(t, "evt-durable-contract-delivery", 6, "idem-contract-delivery", "contract:function:workflow-task:v0"),
		mustMicrotaskCreatedEventWithKeyV0(t, "evt-durable-task-delivery", 7, "idem-task-delivery", "task-ncw-009"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-programacion-delivery", 8, "idem-open-programacion-delivery", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-delivery", 9, "idem-capacity-delivery", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-delivery", 10, "idem-capacity-decision-delivery", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent-delivery", 11, "idem-agent-delivery", "agent-request-001"),
		mustAgentStartedEventWithKeyV0(t, "evt-durable-agent-started-delivery", 12, "idem-agent-started-delivery", "agent-request-001"),
		mustDeliveryRegisteredEventWithKeyV0(t, "evt-durable-delivery", 13, "idem-delivery", "delivery-001"),
	}
}

func mustAgentStartedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, agentRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentStartedEventV0(meta, AgentStartedPayloadV0{
		AgentRequestID: agentRef,
		LaunchRef:      "launch-ref-" + agentRef,
		AckRef:         "ack-ref-" + agentRef,
		ReadinessRef:   "readiness-ref-" + agentRef,
		EvidenceRefs:   []string{"evidence-ref-agent-started-001"},
	})
	return mustReducerEventV0(t, event, err)
}
