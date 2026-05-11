package orquestaoutboxdispatch

import "testing"

func TestChooseNextDispatchV0NoDuplicaEntradaReclamada(t *testing.T) {
	decision := ChooseNextDispatchV0(DispatchSelectionV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
		Pending: []OutboxPendingEntryV0{
			validPendingEntryV0("outbox-001"),
		},
		ClaimedMessageIDs: []string{"outbox-001"},
	})

	if decision.Kind != DispatchDecisionNoPendingV0 {
		t.Fatalf("decision kind=%s, want %s", decision.Kind, DispatchDecisionNoPendingV0)
	}
	if decision.Intent.MessageID != "" {
		t.Fatalf("intent inesperado para duplicado: %+v", decision.Intent)
	}
	if len(decision.Issues) != 1 || decision.Issues[0].Code != issueDuplicateV0 {
		t.Fatalf("issues duplicado=%+v", decision.Issues)
	}
}

func TestChooseNextDispatchV0NoDespachaSiNoHayPendiente(t *testing.T) {
	decision := ChooseNextDispatchV0(DispatchSelectionV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
	})

	if decision.Kind != DispatchDecisionNoPendingV0 {
		t.Fatalf("decision kind=%s, want %s", decision.Kind, DispatchDecisionNoPendingV0)
	}
	if decision.Intent.MessageID != "" {
		t.Fatalf("intent inesperado sin pendiente: %+v", decision.Intent)
	}
}

func TestChooseNextDispatchV0SeleccionaPrimeraPendienteElegible(t *testing.T) {
	decision := ChooseNextDispatchV0(DispatchSelectionV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
		Pending: []OutboxPendingEntryV0{
			validPendingEntryForPortV0("outbox-director-001", "director"),
			validPendingEntryV0("outbox-002"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("decision kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if decision.Intent.MessageID != "outbox-002" {
		t.Fatalf("intent message_id=%s", decision.Intent.MessageID)
	}
	if decision.Intent.CorrelationID != "corr-outbox-002" {
		t.Fatalf("intent correlation_id=%s", decision.Intent.CorrelationID)
	}
}

func TestChooseNextDispatchV0FiltraPorTipoDeMensaje(t *testing.T) {
	decision := ChooseNextDispatchV0(DispatchSelectionV0{
		RunID:       "run-001",
		TargetPort:  "agent_launcher",
		MessageType: "stop_runtime_agent",
		Pending: []OutboxPendingEntryV0{
			validPendingEntryForTypeV0("outbox-launch-001", "agent_launcher", "launch_runtime_agent"),
			validPendingEntryForTypeV0("outbox-stop-001", "agent_launcher", "stop_runtime_agent"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("decision kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if decision.Intent.MessageID != "outbox-stop-001" {
		t.Fatalf("intent message_id=%s", decision.Intent.MessageID)
	}
	if decision.Intent.MessageType != "stop_runtime_agent" {
		t.Fatalf("intent message_type=%s", decision.Intent.MessageType)
	}
}

func validPendingEntryV0(messageID string) OutboxPendingEntryV0 {
	return validPendingEntryForPortV0(messageID, "agent_launcher")
}

func validPendingEntryForPortV0(messageID string, targetPort string) OutboxPendingEntryV0 {
	return validPendingEntryForTypeV0(messageID, targetPort, "launch_runtime_agent")
}

func validPendingEntryForTypeV0(messageID string, targetPort string, messageType string) OutboxPendingEntryV0 {
	return OutboxPendingEntryV0{
		MessageID:      messageID,
		RunID:          "run-001",
		TargetPort:     targetPort,
		MessageType:    messageType,
		IdempotencyKey: "idem-" + messageID,
		CorrelationID:  "corr-" + messageID,
		PayloadVersion: "v0",
		Payload:        []byte(`{"agent_request_id":"agent-001"}`),
	}
}
