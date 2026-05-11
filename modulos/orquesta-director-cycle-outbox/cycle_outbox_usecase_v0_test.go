package orquestadirectorcycleoutbox

import (
	"context"
	"errors"
	"reflect"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestRecordDirectorCycleOutboxV0SaveAndListPendingRefs(t *testing.T) {
	ledger := newCycleOutboxPersistenceLedgerAdapterV0()
	message := mustCycleOutboxCapacityMessageV0(t, cycleOutboxRunRefV0, "outbox-ref-cycle-001")

	result, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger:   ledger,
		RunRef:   cycleOutboxRunRefV0,
		Messages: []orquestacoreworkflow.OutboxMessageV0{message},
	})
	if err != nil {
		t.Fatalf("record outbox: %v", err)
	}
	if result.SavedCount != 1 || result.PendingCount != 1 {
		t.Fatalf("unexpected counters: %+v", result)
	}
	if !reflect.DeepEqual(result.PendingOutboxRefs, []string{"outbox-ref-cycle-001"}) {
		t.Fatalf("pending refs=%v", result.PendingOutboxRefs)
	}
}

func TestRecordDirectorCycleOutboxV0ListsExistingPendingWithoutMessages(t *testing.T) {
	ledger := newCycleOutboxPersistenceLedgerAdapterV0()
	message := mustCycleOutboxCapacityMessageV0(t, cycleOutboxRunRefV0, "outbox-ref-cycle-existing")
	_, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger:   ledger,
		RunRef:   cycleOutboxRunRefV0,
		Messages: []orquestacoreworkflow.OutboxMessageV0{message},
	})
	if err != nil {
		t.Fatalf("seed outbox: %v", err)
	}

	result, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger: ledger,
		RunRef: cycleOutboxRunRefV0,
	})
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	if result.SavedCount != 0 || !reflect.DeepEqual(result.PendingOutboxRefs, []string{"outbox-ref-cycle-existing"}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRecordDirectorCycleOutboxV0RejectsForeignRun(t *testing.T) {
	message := mustCycleOutboxCapacityMessageV0(t, "run-foreign-001", "outbox-ref-cycle-foreign")
	_, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger:   newCycleOutboxPersistenceLedgerAdapterV0(),
		RunRef:   cycleOutboxRunRefV0,
		Messages: []orquestacoreworkflow.OutboxMessageV0{message},
	})
	assertCycleOutboxErrorV0(t, err, ErrDirectorCycleOutboxInvalidoV0, "messages.run_id")
}

func TestRecordDirectorCycleOutboxV0PropagaIssuesLedger(t *testing.T) {
	message := mustCycleOutboxCapacityMessageV0(t, cycleOutboxRunRefV0, "outbox-ref-cycle-ledger")
	_, err := RecordDirectorCycleOutboxV0(context.Background(), DirectorCycleOutboxRecordInputV0{
		Ledger:   failingCycleOutboxLedgerV0{},
		RunRef:   cycleOutboxRunRefV0,
		Messages: []orquestacoreworkflow.OutboxMessageV0{message},
	})
	assertCycleOutboxErrorV0(t, err, ErrDirectorCycleOutboxLedgerV0, "save_pending")
}

func assertCycleOutboxErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr DirectorCycleOutboxErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}
