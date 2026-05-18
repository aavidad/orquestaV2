package orquestastatefileoutbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

var _ orquestaappcodexstack.OutboxLedgerPortV0 = (*FileOutboxLedgerV0)(nil)

func TestFileOutboxLedgerV0ReabreClaimSinAckTrasRecrearInstancia(t *testing.T) {
	dir := t.TempDir()
	ledger := newFileOutboxLedgerForTestV0(t, dir)
	messages := validOutboxMessagesV0(t)
	if _, issues := ledger.SavePending(context.Background(), messages); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	claim := claimFromMessageV0(messages[0])
	if result, issues := ledger.ClaimOutboxDispatchV0(claim); len(issues) != 0 || !result.Claimed {
		t.Fatalf("claim result=%+v issues=%+v", result, issues)
	}
	ack := ackFromMessageV0(messages[1], "dispatch-stop-001")
	if issues := ledger.AckOutboxDispatchV0(ack); len(issues) != 0 {
		t.Fatalf("ack issues=%+v", issues)
	}

	reopened := newFileOutboxLedgerForTestV0(t, dir)
	agentPending, issues := reopened.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     "run-outbox-file-001",
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}
	if got := messageIDsV0(agentPending); !reflect.DeepEqual(got, []string{"outbox-launch-file-001"}) {
		t.Fatalf("agent pending=%v", got)
	}

	entries, dispatchIssues := reopened.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       "run-outbox-file-001",
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(dispatchIssues) != 0 {
		t.Fatalf("dispatch list issues=%+v", dispatchIssues)
	}
	if got := entryIDsV0(entries); !reflect.DeepEqual(got, []string{"outbox-launch-file-001"}) {
		t.Fatalf("entries=%v", got)
	}

	reclaimed, dispatchIssues := reopened.ClaimOutboxDispatchV0(claim)
	if len(dispatchIssues) != 0 || !reclaimed.Claimed || reclaimed.AlreadyClaimed {
		t.Fatalf("reclaim result=%+v issues=%+v", reclaimed, dispatchIssues)
	}
	ackedClaim, dispatchIssues := reopened.ClaimOutboxDispatchV0(claimFromMessageV0(messages[1]))
	if len(dispatchIssues) != 0 || !ackedClaim.AlreadyClaimed {
		t.Fatalf("acked claim result=%+v issues=%+v", ackedClaim, dispatchIssues)
	}
	if issues := reopened.AckOutboxDispatchV0(ack); len(issues) != 0 {
		t.Fatalf("ack replay issues=%+v", issues)
	}
}

func TestFileOutboxLedgerV0MantieneIdempotenciaYJSONEstructurado(t *testing.T) {
	dir := t.TempDir()
	ledger := newFileOutboxLedgerForTestV0(t, dir)
	message := validLaunchMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("idempotent save issues=%+v", issues)
	}

	reopened := newFileOutboxLedgerForTestV0(t, dir)
	pending, issues := reopened.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: "run-outbox-file-001",
	})
	if len(issues) != 0 {
		t.Fatalf("list issues=%+v", issues)
	}
	if got := messageIDsV0(pending); !reflect.DeepEqual(got, []string{"outbox-launch-file-001"}) {
		t.Fatalf("pending=%v", got)
	}

	conflicting := message
	conflicting.Payload = mustPayloadV0(t, orquestacoreworkflow.LaunchRuntimeAgentRequestV0{
		AgentRequestID:     "agent-request-file-001",
		RunID:              "run-outbox-file-001",
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef:            "task-ref-file-001",
		CapacityRequestRef: "capacity-ref-file-001",
		Role:               "builder",
		Summary:            "Implementar microtarea distinta.",
	})
	_, issues = reopened.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{conflicting})
	if !hasDirectorIssueV0(issues, errIdempotencyConflictV0, "message_id") {
		t.Fatalf("expected conflict, got %+v", issues)
	}

	data, err := os.ReadFile(filepath.Join(dir, fileOutboxLedgerNameV0))
	if err != nil {
		t.Fatalf("read ledger json: %v", err)
	}
	var snapshot outboxLedgerSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("json snapshot: %v", err)
	}
	if snapshot.SchemaVersion != fileOutboxLedgerSchemaVersionV0 || len(snapshot.Records) != 1 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	if text := string(data); !strings.Contains(text, `"message_id"`) || strings.Contains(text, "MessageID") {
		t.Fatalf("json no estructurado: %s", text)
	}
}

func TestFileOutboxLedgerV0ClaimsConcurrentesNoDuplican(t *testing.T) {
	ledger := newFileOutboxLedgerForTestV0(t, t.TempDir())
	message := validLaunchMessageV0(t)
	if _, issues := ledger.SavePending(context.Background(), []orquestacoreworkflow.OutboxMessageV0{message}); len(issues) != 0 {
		t.Fatalf("save issues=%+v", issues)
	}

	var wg sync.WaitGroup
	claimed := make(chan bool, 12)
	for index := 0; index < 12; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, issues := ledger.ClaimOutboxDispatchV0(claimFromMessageV0(message))
			if len(issues) > 0 {
				t.Errorf("claim issues=%+v", issues)
				return
			}
			claimed <- result.Claimed
		}()
	}
	wg.Wait()
	close(claimed)

	total := 0
	for value := range claimed {
		if value {
			total++
		}
	}
	if total != 1 {
		t.Fatalf("claimed=%d want=1", total)
	}
}
