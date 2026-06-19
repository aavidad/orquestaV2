package orquestapersistence

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestInMemoryOutboxLedgerV0GuardaYListaPendientesPorRunYTarget(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	messages := validOutboxLedgerMessagesV0(t)

	accepted, issues := ledger.GuardarPendientes(context.Background(), messages)
	if len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}
	if len(accepted) != len(messages) {
		t.Fatalf("accepted=%d want %d", len(accepted), len(messages))
	}

	agentPending, issues := ledger.ListarPendientes(context.Background(), OutboxPendingFilterV0{
		RunID:      "run-ack-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) != 0 {
		t.Fatalf("list agent issues=%+v", issues)
	}
	if got := outboxLedgerMessageIDsV0(agentPending); !reflect.DeepEqual(got, []string{"outbox-launch-001", "outbox-stop-001"}) {
		t.Fatalf("agent pending=%v", got)
	}

	directorPending, issues := ledger.ListPending(context.Background(), OutboxPendingFilterV0{
		RunID:      "run-ack-001",
		TargetPort: orquestacoreworkflow.OutboxTargetDirectorV0,
	})
	if len(issues) != 0 {
		t.Fatalf("list director issues=%+v", issues)
	}
	if got := outboxLedgerMessageIDsV0(directorPending); !reflect.DeepEqual(got, []string{"outbox-question-001"}) {
		t.Fatalf("director pending=%v", got)
	}
}

func TestInMemoryOutboxLedgerV0AckDispatchedRetiraSoloEsePendiente(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	launch := validLaunchOutboxLedgerMessageV0(t)
	stop := validStopOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{launch, stop}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	snapshot, issues := ledger.MarkDispatched(context.Background(), validOutboxLedgerAckV0(launch))
	if len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}
	if snapshot.Status != OutboxDispatchStatusDispatchedV0 || snapshot.MessageID != launch.MessageID {
		t.Fatalf("snapshot invalido=%+v", snapshot)
	}

	pending, issues := ledger.ListPending(context.Background(), OutboxPendingFilterV0{
		RunID:      "run-ack-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}
	if got := outboxLedgerMessageIDsV0(pending); !reflect.DeepEqual(got, []string{"outbox-stop-001"}) {
		t.Fatalf("pending=%v", got)
	}
}

func TestInMemoryOutboxLedgerV0SaveYAckSonIdempotentes(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)

	first, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message})
	if len(issues) != 0 {
		t.Fatalf("first save issues=%+v", issues)
	}
	second, issues := ledger.GuardarPendientes(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message})
	if len(issues) != 0 {
		t.Fatalf("second save issues=%+v", issues)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("save snapshots differ: first=%+v second=%+v", first, second)
	}

	ack := validOutboxLedgerAckV0(message)
	firstAck, issues := ledger.RegistrarAck(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("first ack issues=%+v", issues)
	}
	secondAck, issues := ledger.MarkDispatched(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("second ack issues=%+v", issues)
	}
	if !reflect.DeepEqual(firstAck, secondAck) {
		t.Fatalf("ack snapshots differ: first=%+v second=%+v", firstAck, secondAck)
	}

	pending, issues := ledger.ListPending(context.Background(), OutboxPendingFilterV0{RunID: "run-ack-001"})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}
	if got := outboxLedgerMessageIDsV0(pending); len(got) != 0 {
		t.Fatalf("pending after ack=%v", got)
	}
}

func TestInMemoryOutboxLedgerV0IdempotenciaIgnoraCorrelationOperativa(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	retry := message
	retry.CorrelationID = "corr-ack-retry-002"
	accepted, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{retry})
	if len(issues) != 0 {
		t.Fatalf("retry con distinta correlacion no debe romper idempotencia: %+v", issues)
	}
	if len(accepted) != 1 || accepted[0].CorrelationID != message.CorrelationID {
		t.Fatalf("accepted=%+v, want stored original correlation %q", accepted, message.CorrelationID)
	}
}

func TestInMemoryOutboxLedgerV0RechazaPayloadYAckIncompatibles(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	conflicting := message
	conflicting.Payload = mustOutboxLedgerPayloadV0(t, orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-001",
		RunID:              "run-ack-001",
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-001",
		CapacityRequestRef: "capacity-ref-001",
		Role:               "builder",
		Summary:            "Implementar otra microtarea compacta.",
	})
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{conflicting})
	if !HasOutboxLedgerIssueV0(issues, ErrConflictoIdempotenciaV0, "message_id") {
		t.Fatalf("expected save conflict, got %+v", issues)
	}

	ack := validOutboxLedgerAckV0(message)
	if _, issues := ledger.MarkDispatched(context.Background(), ack); len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}
	ack.DispatchRef = "dispatch-ref-002"
	_, issues = ledger.RegistrarAck(context.Background(), ack)
	if !HasOutboxLedgerIssueV0(issues, ErrConflictoIdempotenciaV0, "ack") {
		t.Fatalf("expected ack conflict, got %+v", issues)
	}
}

func TestInMemoryOutboxLedgerV0MantieneConflictoSiCambiaCausation(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	retry := message
	retry.CausationEventID = "event-agent-requested-retry-002"
	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{retry})
	if !HasOutboxLedgerIssueV0(issues, ErrConflictoIdempotenciaV0, "message_id") {
		t.Fatalf("expected causation conflict, got %+v", issues)
	}
}

func TestInMemoryOutboxLedgerV0RechazaOutboxInvalido(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)
	message.PayloadVersion = "outbox_payload.v1"

	_, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message})
	if !HasOutboxLedgerIssueV0(issues, orquestacoreworkflow.ErrOutboxPayloadInvalidoV0, "messages.0.payload_version") {
		t.Fatalf("expected payload_version issue, got %+v", issues)
	}
}

func TestInMemoryOutboxLedgerV0JSONSnapshotsNoExponenDetallesProhibidos(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	messages := validOutboxLedgerMessagesV0(t)
	if _, issues := ledger.SavePending(context.Background(), messages); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}
	snapshot, issues := ledger.MarkDispatched(context.Background(), validOutboxLedgerAckV0(messages[0]))
	if len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}
	pending, issues := ledger.ListPending(context.Background(), OutboxPendingFilterV0{RunID: "run-ack-001"})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}

	assertOutboxLedgerJSONSafeV0(t, pending)
	assertOutboxLedgerJSONSafeV0(t, snapshot)
}

func assertOutboxLedgerJSONSafeV0(t *testing.T, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	lower := strings.ToLower(string(data))
	for _, fragment := range []string{"sqlite", "postgres", "mysql", "mongo", "dsn", "sql", "provider", "home", "oauth", "transcript", "prompt"} {
		if strings.Contains(lower, fragment) {
			t.Fatalf("snapshot contains forbidden fragment %q: %s", fragment, string(data))
		}
	}
}
