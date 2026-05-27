package orquestadirector

import (
	"context"
	"strconv"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestOutboxDispatchCycleRailPolicyV0PreservaCodigosOperativos(t *testing.T) {
	for index, code := range []string{
		"provider_timeout",
		"db_adapter_unavailable",
		"runtime_backpressure",
	} {
		t.Run(code, func(t *testing.T) {
			ledger := &cycleAcceptingLedgerV0{}
			message := validCycleOutboxMessageV0(t, "operational-code-"+strconv.Itoa(index))
			dispatcher := &cycleFakeOutboxDispatcherV0{
				failMessageID: message.MessageID,
				errorCode:     code,
			}

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
			if len(result.Snapshots) != 1 || result.Snapshots[0].ErrorCode != code {
				t.Fatalf("error_code=%v, want %s", result.Snapshots, code)
			}
		})
	}
}

func TestOutboxDispatchCycleRailPolicyV0RedactaCodigoSensible(t *testing.T) {
	ledger := &cycleAcceptingLedgerV0{}
	message := validCycleOutboxMessageV0(t, "sensitive")
	dispatcher := &cycleFakeOutboxDispatcherV0{
		failMessageID: message.MessageID,
		errorCode:     "api_key=valor-real",
	}

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
	if len(result.Snapshots) != 1 || result.Snapshots[0].ErrorCode != ErrOutboxDispatchFailedV0 {
		t.Fatalf("error_code=%v, want %s", result.Snapshots, ErrOutboxDispatchFailedV0)
	}
}

type cycleAcceptingLedgerV0 struct {
	pending []orquestacoreworkflow.OutboxMessageV0
}

func (l *cycleAcceptingLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0) {
	l.pending = append(l.pending, messages...)
	return messages, nil
}

func (l *cycleAcceptingLedgerV0) ListPending(
	ctx context.Context,
	filter OutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0) {
	return append([]orquestacoreworkflow.OutboxMessageV0(nil), l.pending...), nil
}

func (l *cycleAcceptingLedgerV0) MarkDispatched(
	ctx context.Context,
	ack OutboxDispatchAckV0,
) (OutboxDispatchSnapshotV0, []OutboxDispatchCycleIssueV0) {
	l.pending = nil
	return OutboxDispatchSnapshotV0{
		MessageID:    ack.MessageID,
		RunID:        ack.RunID,
		TargetPort:   ack.TargetPort,
		Status:       ack.Status,
		DispatchRef:  ack.DispatchRef,
		DispatchedAt: ack.DispatchedAt,
		ErrorCode:    ack.ErrorCode,
		EvidenceRefs: ack.EvidenceRefs,
	}, nil
}
