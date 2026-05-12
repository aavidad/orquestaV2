package orquestastatefileoutbox

import (
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func newFileOutboxLedgerForTestV0(t *testing.T, dir string) *FileOutboxLedgerV0 {
	t.Helper()
	ledger, err := NewFileOutboxLedgerV0(dir)
	if err != nil {
		t.Fatalf("NewFileOutboxLedgerV0: %v", err)
	}
	return ledger
}

func validOutboxMessagesV0(t *testing.T) []orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	return []orquestacoreworkflow.OutboxMessageV0{
		validLaunchMessageV0(t),
		validStopMessageV0(t),
		validDirectorQuestionMessageV0(t),
	}
}

func validLaunchMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload := orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-file-001",
		RunID:              "run-outbox-file-001",
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-file-001",
		CapacityRequestRef: "capacity-ref-file-001",
		Role:               "builder",
		Summary:            "Implementar microtarea compacta.",
		EvidenceRefs:       []string{"evidence-ref-file-launch-001"},
	}
	return outboxMessageV0(
		"outbox-launch-file-001",
		orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		"idem-launch-file-001",
		mustPayloadV0(t, payload),
	)
}

func validStopMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	payload := orquestacoreworkflow.StopRuntimeAgentRequestV0{
		AgentRequestID: "agent-request-file-001",
		RunID:          "run-outbox-file-001",
		ReasonCode:     "completed",
		Summary:        "Parada logica solicitada.",
		EvidenceRefs:   []string{"evidence-ref-file-stop-001"},
	}
	return outboxMessageV0(
		"outbox-stop-file-001",
		orquestacoreworkflow.OutboxMessageStopRuntimeAgentV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		"idem-stop-file-001",
		mustPayloadV0(t, payload),
	)
}

func validDirectorQuestionMessageV0(t *testing.T) orquestacoreworkflow.OutboxMessageV0 {
	t.Helper()
	message, err := orquestacoreworkflow.NewSendDirectorQuestionOutboxV0(
		orquestacoreworkflow.OutboxMessageMetaV0{
			MessageID:        "outbox-question-file-001",
			RunID:            "run-outbox-file-001",
			IdempotencyKey:   "idem-question-file-001",
			CorrelationID:    "corr-outbox-file-001",
			CausationEventID: "event-question-file-001",
		},
		orquestacoreworkflow.DirectorQuestionV0{
			QuestionID:   "question-file-001",
			RunID:        "run-outbox-file-001",
			SourceGroup:  "state-file",
			TargetGroup:  "director",
			Summary:      "Confirmar criterio local de entrega del ledger.",
			Options:      []string{"Mantener local", "Elevar despues"},
			EvidenceRefs: []string{"evidence-ref-file-question-001"},
			Blocking:     true,
			RequestedAt:  "2026-05-12T10:00:00Z",
		},
	)
	if err != nil {
		t.Fatalf("director question outbox: %v", err)
	}
	return message
}

func outboxMessageV0(
	messageID string,
	messageType string,
	targetPort string,
	idempotencyKey string,
	payload json.RawMessage,
) orquestacoreworkflow.OutboxMessageV0 {
	return orquestacoreworkflow.OutboxMessageV0{
		MessageID:        messageID,
		MessageType:      messageType,
		RunID:            "run-outbox-file-001",
		IdempotencyKey:   idempotencyKey,
		CorrelationID:    "corr-outbox-file-001",
		CausationEventID: "event-outbox-file-001",
		TargetPort:       targetPort,
		PayloadVersion:   orquestacoreworkflow.OutboxPayloadVersionV0,
		Payload:          payload,
	}
}

func mustPayloadV0(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func claimFromMessageV0(
	message orquestacoreworkflow.OutboxMessageV0,
) orquestaoutboxdispatch.OutboxDispatchClaimV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
}

func ackFromMessageV0(
	message orquestacoreworkflow.OutboxMessageV0,
	dispatchRef string,
) orquestaoutboxdispatch.OutboxDispatchAckV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		DispatchRef:  dispatchRef,
		EvidenceRefs: []string{"evidence-ref-file-dispatch-001"},
	}
}

func messageIDsV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.MessageID)
	}
	return ids
}

func entryIDsV0(entries []orquestaoutboxdispatch.OutboxPendingEntryV0) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.MessageID)
	}
	return ids
}

func hasDirectorIssueV0(
	issues []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0,
	code string,
	field string,
) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Field == field {
			return true
		}
	}
	return false
}
