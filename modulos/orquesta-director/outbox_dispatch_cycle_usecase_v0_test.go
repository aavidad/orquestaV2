package orquestadirector

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	cycleRunIDV0        = "run-cycle-001"
	cycleTargetPortV0   = orquestacoreworkflow.OutboxTargetAgentLauncherV0
	cycleDispatchedAtV0 = "2026-05-05T12:00:00Z"
)

func TestRunOutboxDispatchCycleV0SaveListDispatchAckRetiraPendientes(t *testing.T) {
	ctx := context.Background()
	ledger := newCyclePersistenceLedgerAdapterV0()
	dispatcher := &cycleFakeOutboxDispatcherV0{}
	messages := []orquestacoreworkflow.OutboxMessageV0{
		validCycleOutboxMessageV0(t, "001"),
		validCycleOutboxMessageV0(t, "002"),
	}

	result, err := RunOutboxDispatchCycleV0(ctx, OutboxDispatchCycleInputV0{
		Ledger:       ledger,
		Dispatcher:   dispatcher,
		RunID:        cycleRunIDV0,
		TargetPort:   cycleTargetPortV0,
		DispatchedAt: cycleDispatchedAtV0,
		Messages:     messages,
	})
	if err != nil {
		t.Fatalf("RunOutboxDispatchCycleV0: %v", err)
	}
	if result.SavedCount != 2 || result.PendingBeforeCount != 2 ||
		result.DispatchedCount != 2 || result.FailedCount != 0 || result.PendingAfterCount != 0 {
		t.Fatalf("result counters inesperados: %+v", result)
	}
	if got := cycleSnapshotStatusesV0(result.Snapshots); !reflect.DeepEqual(got, []string{
		OutboxDispatchStatusDispatchedV0,
		OutboxDispatchStatusDispatchedV0,
	}) {
		t.Fatalf("snapshot statuses=%v", got)
	}
	if got := dispatcher.calls; !reflect.DeepEqual(got, []string{messages[0].MessageID, messages[1].MessageID}) {
		t.Fatalf("dispatcher calls=%v", got)
	}
	pending := cycleListPendingV0(t, ctx, ledger)
	if len(pending) != 0 {
		t.Fatalf("pending tras ciclo=%v", cycleMessageIDsV0(pending))
	}
}

func TestRunOutboxDispatchCycleV0AckFailedRetiraMensajeFallido(t *testing.T) {
	ctx := context.Background()
	ledger := newCyclePersistenceLedgerAdapterV0()
	messages := []orquestacoreworkflow.OutboxMessageV0{
		validCycleOutboxMessageV0(t, "fail"),
		validCycleOutboxMessageV0(t, "next"),
	}
	dispatcher := &cycleFakeOutboxDispatcherV0{failMessageID: messages[0].MessageID}

	result, err := RunOutboxDispatchCycleV0(ctx, OutboxDispatchCycleInputV0{
		Ledger:       ledger,
		Dispatcher:   dispatcher,
		RunID:        cycleRunIDV0,
		TargetPort:   cycleTargetPortV0,
		DispatchedAt: cycleDispatchedAtV0,
		Messages:     messages,
	})
	if err == nil {
		t.Fatalf("RunOutboxDispatchCycleV0 failure: nil error")
	}
	var publicErr OutboxDispatchCycleErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("error type=%T, want OutboxDispatchCycleErrorV0", err)
	}
	if publicErr.Code != ErrDirectorOutboxDispatchCycleDispatchFailedV0 {
		t.Fatalf("error code=%s", publicErr.Code)
	}
	if result.FailedCount != 1 || result.DispatchedCount != 0 || result.PendingAfterCount != 1 {
		t.Fatalf("failure counters inesperados: %+v", result)
	}
	if len(result.Snapshots) != 1 ||
		result.Snapshots[0].Status != OutboxDispatchStatusFailedV0 ||
		result.Snapshots[0].ErrorCode != "fake_dispatch_failed" {
		t.Fatalf("failed snapshot inesperado: %+v", result.Snapshots)
	}
	pending := cycleListPendingV0(t, ctx, ledger)
	if got := cycleMessageIDsV0(pending); !reflect.DeepEqual(got, []string{messages[1].MessageID}) {
		t.Fatalf("pending tras fallo=%v", got)
	}
}

func TestRunOutboxDispatchCycleV0RechazaInputIncompleto(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0
		field string
	}{
		{
			name: "sin ledger",
			edit: func(input OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0 {
				input.Ledger = nil
				return input
			},
			field: "ledger",
		},
		{
			name: "sin dispatcher",
			edit: func(input OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0 {
				input.Dispatcher = nil
				return input
			},
			field: "dispatcher",
		},
		{
			name: "sin run_id",
			edit: func(input OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0 {
				input.RunID = ""
				return input
			},
			field: "run_id",
		},
		{
			name: "sin target_port",
			edit: func(input OutboxDispatchCycleInputV0) OutboxDispatchCycleInputV0 {
				input.TargetPort = ""
				return input
			},
			field: "target_port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunOutboxDispatchCycleV0(context.Background(), tt.edit(validCycleInputV0()))
			if err == nil {
				t.Fatalf("RunOutboxDispatchCycleV0: nil error")
			}
			var publicErr OutboxDispatchCycleErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("error type=%T, want OutboxDispatchCycleErrorV0", err)
			}
			if publicErr.Code != ErrDirectorOutboxDispatchCycleInvalidoV0 || publicErr.Field != tt.field {
				t.Fatalf("error=%+v, want code=%s field=%s", publicErr, ErrDirectorOutboxDispatchCycleInvalidoV0, tt.field)
			}
		})
	}
}

func TestRunOutboxDispatchCycleV0JSONNoExponeDetallesProhibidos(t *testing.T) {
	ledger := newCyclePersistenceLedgerAdapterV0()
	message := validCycleOutboxMessageV0(t, "json")
	dispatcher := &cycleFakeOutboxDispatcherV0{failMessageID: message.MessageID}
	result, err := RunOutboxDispatchCycleV0(context.Background(), OutboxDispatchCycleInputV0{
		Ledger:       ledger,
		Dispatcher:   dispatcher,
		RunID:        cycleRunIDV0,
		TargetPort:   cycleTargetPortV0,
		DispatchedAt: cycleDispatchedAtV0,
		Messages:     []orquestacoreworkflow.OutboxMessageV0{message},
	})
	if err == nil {
		t.Fatalf("RunOutboxDispatchCycleV0 failure: nil error")
	}
	var publicErr OutboxDispatchCycleErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("error type=%T, want OutboxDispatchCycleErrorV0", err)
	}
	data, err := json.Marshal(struct {
		Result OutboxDispatchCycleResultV0 `json:"result"`
		Error  OutboxDispatchCycleErrorV0  `json:"error"`
	}{Result: result, Error: publicErr})
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	serialized := strings.ToLower(string(data))
	for _, forbidden := range []string{"sqlite", "postgres", "mysql", "mongo", "dsn", "sql", "provider", "home", "oauth", "transcript", "prompt"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized result contains forbidden fragment %q: %s", forbidden, string(data))
		}
	}
}

type cycleFakeOutboxDispatcherV0 struct {
	failMessageID string
	calls         []string
}

func (d *cycleFakeOutboxDispatcherV0) DispatchOutboxMessageV0(
	ctx context.Context,
	message orquestacoreworkflow.OutboxMessageV0,
) (OutboxDispatchReceiptV0, error) {
	d.calls = append(d.calls, message.MessageID)
	receipt := OutboxDispatchReceiptV0{
		DispatchRef:  "dispatch-ref-" + message.MessageID,
		EvidenceRefs: []string{"evidence-ref-cycle-dispatch-001"},
	}
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	if message.MessageID == d.failMessageID {
		return receipt, cycleFakeDispatchErrorV0{}
	}
	return receipt, nil
}

type cycleFakeDispatchErrorV0 struct{}

func (cycleFakeDispatchErrorV0) Error() string {
	return "provider oauth home details must not leak"
}

func (cycleFakeDispatchErrorV0) OutboxDispatchErrorCodeV0() string {
	return "fake_dispatch_failed"
}
