package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
)

type memoryRunStoreV0 struct {
	runs map[string]orquestacoreworkflow.OrchestrationRunV0
}

func newMemoryRunStoreV0(runs ...orquestacoreworkflow.OrchestrationRunV0) *memoryRunStoreV0 {
	store := &memoryRunStoreV0{runs: map[string]orquestacoreworkflow.OrchestrationRunV0{}}
	for _, run := range runs {
		store.runs[strings.TrimSpace(run.RunID)] = run
	}
	return store
}

func (store *memoryRunStoreV0) LoadRunV0(
	ctx context.Context,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	run, ok := store.runs[strings.TrimSpace(runRef)]
	if !ok {
		return orquestacoreworkflow.OrchestrationRunV0{}, RunNotFoundErrorV0{RunRef: runRef}
	}
	return run, nil
}

func (store *memoryRunStoreV0) SaveRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(run.RunID) == "" {
		return fmt.Errorf("run_id requerido")
	}
	store.runs[strings.TrimSpace(run.RunID)] = run
	return nil
}

type recordingEventSinkV0 struct {
	events []orquestacoreworkflow.OrchestrationEventV0
}

func (sink *recordingEventSinkV0) AppendRunEventsV0(
	ctx context.Context,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, event := range events {
		if strings.TrimSpace(event.RunID) != strings.TrimSpace(runRef) {
			return fmt.Errorf("evento de otro run: %s", event.RunID)
		}
		sink.events = append(sink.events, event)
	}
	return nil
}

type memoryOutboxLedgerV0 struct {
	messages []orquestacoreworkflow.OutboxMessageV0
}

func (ledger *memoryOutboxLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{Code: "context_cancelado", Field: "context", Message: err.Error()}}
	}
	ledger.messages = append(ledger.messages, cloneMessagesV0(messages)...)
	return cloneMessagesV0(messages), nil
}

func (ledger *memoryOutboxLedgerV0) ListPending(
	ctx context.Context,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{{Code: "context_cancelado", Field: "context", Message: err.Error()}}
	}
	var out []orquestacoreworkflow.OutboxMessageV0
	for _, message := range ledger.messages {
		if filter.RunRef != "" && strings.TrimSpace(message.RunID) != strings.TrimSpace(filter.RunRef) {
			continue
		}
		if filter.TargetPort != "" && strings.TrimSpace(message.TargetPort) != strings.TrimSpace(filter.TargetPort) {
			continue
		}
		out = append(out, message)
	}
	return cloneMessagesV0(out), nil
}

func cloneMessagesV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) []orquestacoreworkflow.OutboxMessageV0 {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(messages))
	for _, message := range messages {
		message.Payload = append(message.Payload[:0:0], message.Payload...)
		cloned = append(cloned, message)
	}
	return cloned
}
