package orquestapersistence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestFileOutboxLedgerV0RehidrataClaimYAckDurableTrasReinicio(t *testing.T) {
	dir := t.TempDir()
	ledger := newFileOutboxLedgerForTestV0(t, dir)
	messages := validOutboxLedgerMessagesV0(t)
	if _, issues := ledger.SavePending(context.Background(), messages); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}
	claim := fileTestClaimFromMessageV0(messages[0])
	if result, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) != 0 || !result.Claimed {
		t.Fatalf("claim result=%+v issues=%+v", result, issues)
	}
	failed := fileTestAckObservationV0(messages[1], orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0)
	if issues := ledger.AckOutboxDispatchObservationV0(failed); len(issues) != 0 {
		t.Fatalf("failed ack issues=%+v", issues)
	}

	reopened := newFileOutboxLedgerForTestV0(t, dir)
	entries, issues := reopened.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       "run-ack-001",
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(issues) != 0 {
		t.Fatalf("dispatch list issues=%+v", issues)
	}
	if got := fileTestEntryIDsV0(entries); !reflect.DeepEqual(got, []string{"outbox-launch-001"}) {
		t.Fatalf("entries=%v", got)
	}
	reclaimed, issues := reopened.ClaimOutboxDispatchV0(claim)
	if len(issues) != 0 || !reclaimed.Claimed || reclaimed.AlreadyClaimed {
		t.Fatalf("reclaim=%+v issues=%+v", reclaimed, issues)
	}
	snapshots, ledgerIssues := reopened.ListOutboxDispatchSnapshotsV0(OutboxPendingFilterV0{
		RunID:      "run-ack-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(ledgerIssues) != 0 {
		t.Fatalf("snapshots issues=%+v", ledgerIssues)
	}
	if len(snapshots) != 1 || snapshots[0].MessageID != "outbox-stop-001" ||
		snapshots[0].Status != OutboxDispatchStatusFailedV0 {
		t.Fatalf("snapshots=%+v", snapshots)
	}
	if issues := reopened.AckOutboxDispatchObservationV0(failed); len(issues) != 0 {
		t.Fatalf("failed ack replay issues=%+v", issues)
	}
}

func TestFileOutboxLedgerV0NoListaAckSuccessTrasReinicio(t *testing.T) {
	dir := t.TempDir()
	ledger := newFileOutboxLedgerForTestV0(t, dir)
	message := validLaunchOutboxLedgerMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}
	if result, issues := ledger.ClaimOutboxDispatchV0(fileTestClaimFromMessageV0(message)); len(issues) != 0 || !result.Claimed {
		t.Fatalf("claim result=%+v issues=%+v", result, issues)
	}
	if issues := ledger.AckOutboxDispatchV0(fileTestSuccessAckV0(message)); len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}
	reopened := newFileOutboxLedgerForTestV0(t, dir)
	pending, issues := reopened.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     "run-ack-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) != 0 || len(pending) != 0 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestFileOutboxLedgerV0MigraSnapshotLegacySinBloquearArranque(t *testing.T) {
	dir := t.TempDir()
	message := validLaunchOutboxLedgerMessageV0(t)
	legacy := fileOutboxLedgerSnapshotV0{
		SchemaVersion: legacyFileOutboxLedgerSchemaVersionV0,
		Records: []fileOutboxLedgerRecordV0{{
			Message: message,
			Ack: &fileOutboxLedgerAckV0{
				MessageID:    message.MessageID,
				RunID:        message.RunID,
				TargetPort:   message.TargetPort,
				DispatchRef:  "dispatch-legacy-" + message.MessageID,
				EvidenceRefs: []string{"evidence-ref-legacy-outbox"},
			},
		}},
	}
	writeFileOutboxLedgerSnapshotForTestV0(t, dir, legacy)

	ledger := newFileOutboxLedgerForTestV0(t, dir)
	snapshots, issues := ledger.ListOutboxDispatchSnapshotsV0(OutboxPendingFilterV0{
		RunID:      message.RunID,
		TargetPort: message.TargetPort,
	})
	if len(issues) != 0 {
		t.Fatalf("snapshots issues=%+v", issues)
	}
	if len(snapshots) != 1 ||
		snapshots[0].Status != OutboxDispatchStatusDispatchedV0 ||
		snapshots[0].DispatchRef != "dispatch-legacy-"+message.MessageID {
		t.Fatalf("snapshots=%+v", snapshots)
	}
	var persisted fileOutboxLedgerSnapshotV0
	readFileOutboxLedgerSnapshotForTestV0(t, dir, &persisted)
	if persisted.SchemaVersion != FileOutboxLedgerSchemaVersionV0 {
		t.Fatalf("schema no migrado: %q", persisted.SchemaVersion)
	}
}

func newFileOutboxLedgerForTestV0(t *testing.T, dir string) *FileOutboxLedgerV0 {
	t.Helper()
	ledger, err := NewFileOutboxLedgerV0(dir)
	if err != nil {
		t.Fatalf("new file ledger: %v", err)
	}
	return ledger
}

func writeFileOutboxLedgerSnapshotForTestV0(
	t *testing.T,
	dir string,
	snapshot fileOutboxLedgerSnapshotV0,
) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir ledger dir: %v", err)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal ledger snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileOutboxLedgerNameV0), append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write ledger snapshot: %v", err)
	}
}

func readFileOutboxLedgerSnapshotForTestV0(
	t *testing.T,
	dir string,
	dst *fileOutboxLedgerSnapshotV0,
) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, fileOutboxLedgerNameV0))
	if err != nil {
		t.Fatalf("read ledger snapshot: %v", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("decode ledger snapshot: %v", err)
	}
}

func fileTestClaimFromMessageV0(
	message orquestacoreworkflow.OutboxMessageV0,
) orquestaoutboxdispatch.OutboxDispatchClaimV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      message.MessageID,
		RunID:          message.RunID,
		TargetPort:     message.TargetPort,
		IdempotencyKey: message.IdempotencyKey,
	}
}

func fileTestSuccessAckV0(message orquestacoreworkflow.OutboxMessageV0) orquestaoutboxdispatch.OutboxDispatchAckV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		DispatchRef:  "dispatch-" + message.MessageID,
		EvidenceRefs: []string{"evidence-" + message.MessageID},
	}
}

func fileTestAckObservationV0(
	message orquestacoreworkflow.OutboxMessageV0,
	status orquestaoutboxdispatch.OutboxDispatchAckObservationStatusV0,
) orquestaoutboxdispatch.OutboxDispatchAckObservationV0 {
	return orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
		MessageID:    message.MessageID,
		RunID:        message.RunID,
		TargetPort:   message.TargetPort,
		Status:       status,
		DispatchRef:  "dispatch-" + message.MessageID,
		EvidenceRefs: []string{"evidence-" + message.MessageID},
	}
}

func fileTestEntryIDsV0(entries []orquestaoutboxdispatch.OutboxPendingEntryV0) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.MessageID)
	}
	return ids
}
