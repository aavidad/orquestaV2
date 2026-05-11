package orquestaoutboxdispatch

import "testing"

func TestChooseDispatchBatchV0SeleccionaVariosPendientesElegibles(t *testing.T) {
	decision := ChooseDispatchBatchV0(DispatchBatchSelectionV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
		MaxReady:   2,
		Pending: []OutboxPendingEntryV0{
			validPendingEntryV0("outbox-201"),
			validPendingEntryV0("outbox-202"),
			validPendingEntryV0("outbox-203"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if len(decision.Intents) != 2 {
		t.Fatalf("intents=%d, want 2", len(decision.Intents))
	}
	if decision.Intents[0].MessageID != "outbox-201" ||
		decision.Intents[1].MessageID != "outbox-202" {
		t.Fatalf("intents=%+v", decision.Intents)
	}
}

func TestChooseDispatchBatchV0NoDuplicaReclamadosYConservaOrden(t *testing.T) {
	decision := ChooseDispatchBatchV0(DispatchBatchSelectionV0{
		RunID:             "run-001",
		TargetPort:        "agent_launcher",
		MaxReady:          3,
		ClaimedMessageIDs: []string{"outbox-211"},
		Pending: []OutboxPendingEntryV0{
			validPendingEntryV0("outbox-211"),
			validPendingEntryForPortV0("outbox-director-001", "director"),
			validPendingEntryV0("outbox-212"),
			validPendingEntryV0("outbox-213"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if len(decision.Intents) != 2 {
		t.Fatalf("intents=%d, want 2", len(decision.Intents))
	}
	if decision.Intents[0].MessageID != "outbox-212" ||
		decision.Intents[1].MessageID != "outbox-213" {
		t.Fatalf("intents=%+v", decision.Intents)
	}
	if len(decision.Issues) != 1 || decision.Issues[0].Code != issueDuplicateV0 {
		t.Fatalf("issues=%+v", decision.Issues)
	}
}

func TestChooseDispatchBatchV0DefaultMaxReadyEsUno(t *testing.T) {
	decision := ChooseDispatchBatchV0(DispatchBatchSelectionV0{
		RunID:      "run-001",
		TargetPort: "agent_launcher",
		Pending: []OutboxPendingEntryV0{
			validPendingEntryV0("outbox-221"),
			validPendingEntryV0("outbox-222"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if len(decision.Intents) != 1 || decision.Intents[0].MessageID != "outbox-221" {
		t.Fatalf("intents=%+v", decision.Intents)
	}
}

func TestChooseDispatchBatchV0FiltraPorTipoDeMensaje(t *testing.T) {
	decision := ChooseDispatchBatchV0(DispatchBatchSelectionV0{
		RunID:       "run-001",
		TargetPort:  "agent_launcher",
		MessageType: "launch_runtime_agent",
		MaxReady:    3,
		Pending: []OutboxPendingEntryV0{
			validPendingEntryForTypeV0("outbox-stop-001", "agent_launcher", "stop_runtime_agent"),
			validPendingEntryForTypeV0("outbox-launch-001", "agent_launcher", "launch_runtime_agent"),
			validPendingEntryForTypeV0("outbox-launch-002", "agent_launcher", "launch_runtime_agent"),
		},
	})

	if decision.Kind != DispatchDecisionReadyV0 {
		t.Fatalf("kind=%s, want %s", decision.Kind, DispatchDecisionReadyV0)
	}
	if len(decision.Intents) != 2 {
		t.Fatalf("intents=%d, want 2", len(decision.Intents))
	}
	if decision.Intents[0].MessageID != "outbox-launch-001" ||
		decision.Intents[1].MessageID != "outbox-launch-002" {
		t.Fatalf("intents=%+v", decision.Intents)
	}
}

func TestChooseDispatchBatchV0InvalidoSinTargetPort(t *testing.T) {
	decision := ChooseDispatchBatchV0(DispatchBatchSelectionV0{
		RunID:    "run-001",
		MaxReady: 2,
		Pending:  []OutboxPendingEntryV0{validPendingEntryV0("outbox-231")},
	})

	if decision.Kind != DispatchDecisionInvalidRequestV0 {
		t.Fatalf("kind=%s, want %s", decision.Kind, DispatchDecisionInvalidRequestV0)
	}
	if len(decision.Intents) != 0 {
		t.Fatalf("intents=%+v, want none", decision.Intents)
	}
}
