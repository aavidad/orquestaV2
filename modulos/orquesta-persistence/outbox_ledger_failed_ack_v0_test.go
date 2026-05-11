package orquestapersistence

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestInMemoryOutboxLedgerV0AckFailedRetiraPendienteYNormalizaSnapshot(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)

	accepted, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message})
	if len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	ack := validFailedOutboxLedgerAckV0(message)
	ack.ErrorCode = "  entrega_fallida  "
	ack.EvidenceRefs = []string{" evidence-ref-failed-001 ", "", "evidence-ref-failed-002"}
	snapshot, issues := ledger.RegistrarAck(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("failed ack issues=%+v", issues)
	}
	if snapshot.Status != OutboxDispatchStatusFailedV0 || snapshot.MessageID != message.MessageID {
		t.Fatalf("snapshot invalido=%+v", snapshot)
	}
	if snapshot.ErrorCode != "entrega_fallida" {
		t.Fatalf("error_code=%q", snapshot.ErrorCode)
	}
	if !reflect.DeepEqual(snapshot.EvidenceRefs, []string{"evidence-ref-failed-001", "evidence-ref-failed-002"}) {
		t.Fatalf("evidence_refs=%+v", snapshot.EvidenceRefs)
	}

	pending, issues := ledger.ListPending(context.Background(), OutboxPendingFilterV0{
		RunID:      message.RunID,
		TargetPort: message.TargetPort,
	})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}
	if got := outboxLedgerMessageIDsV0(pending); len(got) != 0 {
		t.Fatalf("pending after failed ack=%v", got)
	}

	assertOutboxLedgerJSONNoForbiddenAckDetailsV0(t, accepted)
	assertOutboxLedgerJSONNoForbiddenAckDetailsV0(t, snapshot)
}

func TestInMemoryOutboxLedgerV0AckFailedReplayIdempotenteYConflicto(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validLaunchOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	ack := validFailedOutboxLedgerAckV0(message)
	first, issues := ledger.MarkDispatched(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("first failed ack issues=%+v", issues)
	}
	second, issues := ledger.RegistrarAck(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("second failed ack issues=%+v", issues)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("failed ack snapshots differ: first=%+v second=%+v", first, second)
	}

	conflictingError := ack
	conflictingError.ErrorCode = "entrega_rechazada"
	_, issues = ledger.MarkDispatched(context.Background(), conflictingError)
	if !HasOutboxLedgerIssueV0(issues, ErrConflictoIdempotenciaV0, "ack") {
		t.Fatalf("expected error_code conflict, got %+v", issues)
	}

	conflictingDispatch := ack
	conflictingDispatch.DispatchRef = "dispatch-ref-failed-002"
	_, issues = ledger.RegistrarAck(context.Background(), conflictingDispatch)
	if !HasOutboxLedgerIssueV0(issues, ErrConflictoIdempotenciaV0, "ack") {
		t.Fatalf("expected dispatch_ref conflict, got %+v", issues)
	}
}

func TestInMemoryOutboxLedgerV0AckFailedSinErrorCodeEsValidoPorContratoActual(t *testing.T) {
	ledger := NewInMemoryOutboxLedgerV0()
	message := validStopOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	ack := validFailedOutboxLedgerAckV0(message)
	ack.ErrorCode = ""
	snapshot, issues := ledger.MarkDispatched(context.Background(), ack)
	if len(issues) != 0 {
		t.Fatalf("failed ack without error_code issues=%+v", issues)
	}
	if snapshot.ErrorCode != "" || snapshot.Status != OutboxDispatchStatusFailedV0 {
		t.Fatalf("snapshot invalido=%+v", snapshot)
	}
}

func validFailedOutboxLedgerAckV0(message orquestacoreworkflow.OutboxMessageV0) OutboxDispatchAckV0 {
	return OutboxDispatchAckV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		Status:       OutboxDispatchStatusFailedV0,
		DispatchRef:  "dispatch-ref-failed-001",
		DispatchedAt: "2026-05-05T10:02:00Z",
		ErrorCode:    "entrega_fallida",
		EvidenceRefs: []string{"evidence-ref-failed-001"},
	}
}

func assertOutboxLedgerJSONNoForbiddenAckDetailsV0(t *testing.T, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	lower := strings.ToLower(string(data))
	for _, fragment := range []string{"db", "dsn", "sql", "provider", "home", "oauth", "transcript", "prompt"} {
		if strings.Contains(lower, fragment) {
			t.Fatalf("snapshot contains forbidden fragment %q: %s", fragment, string(data))
		}
	}
}
