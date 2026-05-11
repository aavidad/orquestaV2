package orquestapersistence

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validOutboxLedgerMessagesV0(t *testing.T) []orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	return []orquestacoreworkflow.OutboxMessageV0{
		validLaunchOutboxLedgerMessageV0(t),
		validStopOutboxLedgerMessageV0(t),
		validDirectorQuestionOutboxLedgerMessageV0(t),
	}
}

func validLaunchOutboxLedgerMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload := orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-001",
		RunID:              "run-ack-001",
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-001",
		CapacityRequestRef: "capacity-ref-001",
		Role:               "builder",
		Summary:            "Implementar microtarea compacta.",
		EvidenceRefs:       []string{"evidence-ref-launch-001"},
	}
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:        "outbox-launch-001",
		MessageType:      orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		RunID:            "run-ack-001",
		IdempotencyKey:   "idem-launch-001",
		CorrelationID:    "corr-ack-001",
		CausationEventID: "event-agent-requested-001",
		TargetPort:       orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:          mustOutboxLedgerPayloadV0(t, payload),
	}
}

func validStopOutboxLedgerMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload := orquestacoreworkflow.StopRuntimeAgentRequestV0{
		AgentRequestID: "agent-request-001",
		RunID:          "run-ack-001",
		ReasonCode:     "completed",
		Summary:        "Parada logica solicitada.",
		EvidenceRefs:   []string{"evidence-ref-stop-001"},
	}
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:        "outbox-stop-001",
		MessageType:      orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		RunID:            "run-ack-001",
		IdempotencyKey:   "idem-stop-001",
		CorrelationID:    "corr-ack-001",
		CausationEventID: "event-agent-stop-001",
		TargetPort:       orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:          mustOutboxLedgerPayloadV0(t, payload),
	}
}

func validDirectorQuestionOutboxLedgerMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	message, err := orquestacoreworkflow.NewSendDirectorQuestionOutboxV0(
		orquestacoreworkflow.OutboxMessageMetaV0{
			MessageID:        "outbox-question-001",
			RunID:            "run-ack-001",
			IdempotencyKey:   "idem-question-001",
			CorrelationID:    "corr-ack-001",
			CausationEventID: "event-question-001",
		},
		orquestacoreworkflow.DirectorQuestionV0{
			QuestionID:   "question-001",
			RunID:        "run-ack-001",
			SourceGroup:  "persistence",
			TargetGroup:  "director",
			Summary:      "Confirmar criterio local de entrega del ledger.",
			Options:      []string{"Mantener local", "Elevar despues"},
			EvidenceRefs: []string{"evidence-ref-question-001"},
			Blocking:     true,
			RequestedAt:  "2026-05-05T10:00:00Z",
		},
	)
	if err != nil {
		t.Fatalf("director question outbox: %v", err)
	}
	return message
}

func validOutboxLedgerAckV0(message orquestacoreworkflow.OutboxMessageV0) OutboxDispatchAckV0 {
	return OutboxDispatchAckV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		Status:       OutboxDispatchStatusDispatchedV0,
		DispatchRef:  "dispatch-ref-001",
		DispatchedAt: "2026-05-05T10:01:00Z",
		EvidenceRefs: []string{"evidence-ref-dispatch-001"},
	}
}

func mustOutboxLedgerPayloadV0(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func outboxLedgerMessageIDsV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.MessageID)
	}
	return ids
}
