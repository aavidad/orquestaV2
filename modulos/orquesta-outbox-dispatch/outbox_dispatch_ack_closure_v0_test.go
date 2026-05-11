package orquestaoutboxdispatch

import "testing"

func TestPlanOutboxDispatchAckClosureV0AckMissingQuedaPendiente(t *testing.T) {
	result := PlanOutboxDispatchAckClosureV0(OutboxDispatchAckClosureInputV0{
		Intents: []DispatchIntentV0{
			validDispatchIntentV0("outbox-301"),
		},
	})

	if len(result.Pending) != 1 || result.Pending[0].MessageID != "outbox-301" {
		t.Fatalf("pending=%+v, want outbox-301", result.Pending)
	}
	if result.Items[0].Status != OutboxDispatchAckClosurePendingV0 {
		t.Fatalf("status=%s, want %s", result.Items[0].Status, OutboxDispatchAckClosurePendingV0)
	}
	if len(result.Acked) != 0 || len(result.Failed) != 0 {
		t.Fatalf("acked=%+v failed=%+v, want none", result.Acked, result.Failed)
	}
}

func TestPlanOutboxDispatchAckClosureV0AckFailedQuedaFallidoPublico(t *testing.T) {
	result := PlanOutboxDispatchAckClosureV0(OutboxDispatchAckClosureInputV0{
		Intents: []DispatchIntentV0{validDispatchIntentV0("outbox-302")},
		Acks: []OutboxDispatchAckObservationV0{
			validAckObservationV0("outbox-302", OutboxDispatchAckObservationFailedV0),
		},
	})

	if len(result.Failed) != 1 || result.Failed[0].MessageID != "outbox-302" {
		t.Fatalf("failed=%+v, want outbox-302", result.Failed)
	}
	if result.Failed[0].Status != OutboxDispatchAckClosureFailedV0 {
		t.Fatalf("status=%s, want %s", result.Failed[0].Status, OutboxDispatchAckClosureFailedV0)
	}
	if len(result.Failed[0].Issues) != 1 || result.Failed[0].Issues[0].Code != ackClosureIssueFailedV0 {
		t.Fatalf("issues=%+v", result.Failed[0].Issues)
	}
	if len(result.Pending) != 0 || len(result.Acked) != 0 {
		t.Fatalf("pending=%+v acked=%+v, want none", result.Pending, result.Acked)
	}
}

func TestPlanOutboxDispatchAckClosureV0AckFailedConservaIssueSanitizado(t *testing.T) {
	ack := validAckObservationV0("outbox-302-detail", OutboxDispatchAckObservationFailedV0)
	ack.Issues = []DispatchIssueV0{{
		Code:    "external_agent_launch_blocked",
		Field:   "command_path",
		Message: "orquesta.runtime.external_agent_process.blocked",
	}}
	result := PlanOutboxDispatchAckClosureV0(OutboxDispatchAckClosureInputV0{
		Intents: []DispatchIntentV0{validDispatchIntentV0("outbox-302-detail")},
		Acks:    []OutboxDispatchAckObservationV0{ack},
	})

	if len(result.Failed) != 1 || len(result.Failed[0].Issues) != 1 {
		t.Fatalf("failed=%+v", result.Failed)
	}
	issue := result.Failed[0].Issues[0]
	if issue.Code != "external_agent_launch_blocked" ||
		issue.Field != "command_path" ||
		issue.Message != "orquesta.runtime.external_agent_process.blocked" {
		t.Fatalf("issue=%+v", issue)
	}
}

func TestPlanOutboxDispatchAckClosureV0AckSuccessIdempotenteRetiraSoloEseItem(t *testing.T) {
	result := PlanOutboxDispatchAckClosureV0(OutboxDispatchAckClosureInputV0{
		Intents: []DispatchIntentV0{
			validDispatchIntentV0("outbox-303"),
			validDispatchIntentV0("outbox-304"),
		},
		Acks: []OutboxDispatchAckObservationV0{
			validAckObservationV0("outbox-303", OutboxDispatchAckObservationSuccessV0),
			validAckObservationV0("outbox-303", OutboxDispatchAckObservationSuccessV0),
		},
	})

	if len(result.Acked) != 1 || result.Acked[0].MessageID != "outbox-303" {
		t.Fatalf("acked=%+v, want only outbox-303", result.Acked)
	}
	if len(result.Pending) != 1 || result.Pending[0].MessageID != "outbox-304" {
		t.Fatalf("pending=%+v, want only outbox-304", result.Pending)
	}
	if len(result.Failed) != 0 {
		t.Fatalf("failed=%+v, want none", result.Failed)
	}
}

func TestPlanOutboxDispatchAckClosureV0BatchParcialNoCierraOtrosItems(t *testing.T) {
	result := PlanOutboxDispatchAckClosureV0(OutboxDispatchAckClosureInputV0{
		Intents: []DispatchIntentV0{
			validDispatchIntentV0("outbox-305"),
			validDispatchIntentV0("outbox-306"),
			validDispatchIntentV0("outbox-307"),
		},
		Acks: []OutboxDispatchAckObservationV0{
			validAckObservationV0("outbox-306", OutboxDispatchAckObservationSuccessV0),
		},
	})

	if len(result.Acked) != 1 || result.Acked[0].MessageID != "outbox-306" {
		t.Fatalf("acked=%+v, want only outbox-306", result.Acked)
	}
	if len(result.Pending) != 2 ||
		result.Pending[0].MessageID != "outbox-305" ||
		result.Pending[1].MessageID != "outbox-307" {
		t.Fatalf("pending=%+v, want outbox-305 and outbox-307", result.Pending)
	}
	for _, item := range result.Items {
		if item.MessageID != "outbox-306" && item.Status != OutboxDispatchAckClosurePendingV0 {
			t.Fatalf("item=%+v, want pending", item)
		}
	}
}

func validDispatchIntentV0(messageID string) DispatchIntentV0 {
	return DispatchIntentV0{
		MessageID:      messageID,
		RunID:          "run-001",
		TargetPort:     "agent_launcher",
		MessageType:    "agent_launch_requested",
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-" + messageID,
		PayloadVersion: "v0",
		Payload:        []byte(`{}`),
	}
}

func validAckObservationV0(
	messageID string,
	status OutboxDispatchAckObservationStatusV0,
) OutboxDispatchAckObservationV0 {
	return OutboxDispatchAckObservationV0{
		MessageID:    messageID,
		RunID:        "run-001",
		TargetPort:   "agent_launcher",
		Status:       status,
		DispatchRef:  "dispatch-" + messageID,
		EvidenceRefs: []string{"evidence-" + messageID},
	}
}
