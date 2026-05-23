package orquestaoperatormcp

import "testing"

func TestOperatorMCPSimulatedConnectorV0CubreEstadoBurstOutboxYConsulta(t *testing.T) {
	connector := NewOperatorMCPSimulatedConnectorV0(OperatorMCPSimulatedConnectorConfigV0{
		OutboxItems: []OperatorMCPOutboxItemV0{{
			MessageRef: "message-ref-1",
			Kind:       "launch_agent",
			TargetRef:  "agent-ref-1",
		}},
	})
	status, err := connector.QueryOperatorStatusV0(OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		StatusConnectorRef: "state-connector-ref-simulated",
		IncludeSections:    []string{"estado", "bloqueos"},
	})
	if err != nil || status.Status != "simulated" || len(status.Sections) != 2 {
		t.Fatalf("status simulado inesperado: status=%+v err=%v", status, err)
	}
	burst, err := connector.RequestOperatorSupervisedBurstV0(OperatorSupervisedBurstRequestV0{
		RequestRef:        "req-1",
		RunRef:            "run-1",
		BurstConnectorRef: "burst-connector-ref-simulated",
		SupervisionRef:    "supervision-1",
		MaxSteps:          4,
	})
	if err != nil || burst.ExecutedSteps != 2 || burst.BurstRef == "" {
		t.Fatalf("burst simulado inesperado: burst=%+v err=%v", burst, err)
	}
	outbox, err := connector.ListOperatorPendingOutboxV0(OperatorPendingOutboxQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		OutboxConnectorRef: "outbox-connector-ref-simulated",
		Limit:              1,
		IncludeKinds:       []string{"launch_runtime_agent"},
	})
	if err != nil || outbox.PendingCount != 1 || outbox.Items[0].Kind != "launch_runtime_agent" {
		t.Fatalf("outbox simulado inesperado: outbox=%+v err=%v", outbox, err)
	}
	query, err := connector.RaiseOperatorDirectedQueryV0(OperatorDirectedQueryV0{
		QueryRef:          "query-1",
		TargetRef:         "director-1",
		QueryConnectorRef: "consulta-connector-ref-simulated",
		Question:          "Que falta?",
	})
	if err != nil || !query.Accepted || query.AnswerRef == "" {
		t.Fatalf("consulta simulada inesperada: query=%+v err=%v", query, err)
	}
}

func TestOperatorMCPSimulatedConnectorV0RechazaConectorNoDeclarado(t *testing.T) {
	connector := NewOperatorMCPSimulatedConnectorV0(OperatorMCPSimulatedConnectorConfigV0{})
	_, err := connector.QueryOperatorStatusV0(OperatorStatusQueryV0{
		RequestRef:         "req-1",
		SubjectRef:         "run-1",
		StatusConnectorRef: "status-connector-ref-externo",
	})
	if err == nil {
		t.Fatalf("esperaba error por conector no declarado")
	}
}
