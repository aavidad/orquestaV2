package main

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestServerSupervisorWakeupDecoratorsV0DisparanSoloTrasMutacionesEjecutables(t *testing.T) {
	ctx := context.Background()
	var causes []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
	}

	runStore := serverWakeupRunStoreV0{
		inner:  orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		wakeup: relay,
	}
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID: "run-ref-wakeup-store-001",
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	fileStore, err := orquestarunfile.NewRunFileStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	queue := serverWakeupRunQueueV0{inner: fileStore, wakeup: relay}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-ready-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-closed-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusClosedV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 closed: %v", err)
	}

	control := serverWakeupRunControlV0{inner: fileStore, wakeup: relay}
	if _, err := control.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if _, err := control.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("ResumeRunV0: %v", err)
	}
	if _, err := control.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-ref-wakeup-ready-001",
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	want := []string{"run_store_saved", "run_queue_ready", "run_control_resume", "run_control_complete"}
	if !stringSlicesEqualV0(causes, want) {
		t.Fatalf("causes=%v want=%v", causes, want)
	}
}

func TestServerWakeupDirectorCycleOutboxLedgerV0DisparaAlGuardarPendientes(t *testing.T) {
	ctx := context.Background()
	var causes []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
	}
	ledger := serverWakeupDirectorCycleOutboxLedgerV0{
		inner:  &fakeServerDirectorCycleOutboxLedgerV0{},
		wakeup: relay,
	}

	saved, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{{
		MessageID:      "message-ref-wakeup-outbox-001",
		RunID:          "run-ref-wakeup-outbox-001",
		TargetPort:     "target-port-test",
		IdempotencyKey: "idem-wakeup-outbox-001",
	}})
	if len(issues) > 0 {
		t.Fatalf("SavePending issues=%+v", issues)
	}
	if len(saved) != 1 {
		t.Fatalf("saved=%+v", saved)
	}
	if !stringSlicesEqualV0(causes, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("causes=%v", causes)
	}

	if _, issues := ledger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: "run-ref-wakeup-outbox-001",
	}); len(issues) > 0 {
		t.Fatalf("ListPending issues=%+v", issues)
	}
	if !stringSlicesEqualV0(causes, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("ListPending no debe disparar wakeup: causes=%v", causes)
	}
}

type fakeServerDirectorCycleOutboxLedgerV0 struct {
	pending []orquestacoreworkflow.OutboxMessageV0
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) SavePending(
	_ context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	ledger.pending = append(ledger.pending, messages...)
	return append([]orquestacoreworkflow.OutboxMessageV0(nil), messages...), nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ListPending(
	_ context.Context,
	_ orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	return append([]orquestacoreworkflow.OutboxMessageV0(nil), ledger.pending...), nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ListPendingOutboxV0(
	_ orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	return nil, nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ClaimOutboxDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) (orquestaoutboxdispatch.OutboxDispatchClaimResultV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		Claimed: true, MessageID: claim.MessageID, TargetPort: claim.TargetPort,
	}, nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) AckOutboxDispatchV0(
	_ orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	return nil
}

func stringSlicesEqualV0(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
