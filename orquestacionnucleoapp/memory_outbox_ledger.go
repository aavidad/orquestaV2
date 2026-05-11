package orquestacionnucleoapp

import (
	"context"
	"strings"
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type InMemoryOutboxLedgerV0 struct {
	mu       sync.Mutex
	messages []orquestacoreworkflow.OutboxMessageV0
	claimed  map[string]bool
	acked    map[string]bool
}

func NewInMemoryOutboxLedgerV0() *InMemoryOutboxLedgerV0 {
	return &InMemoryOutboxLedgerV0{}
}

func (ledger *InMemoryOutboxLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
			{Code: "context_cancelado", Field: "context", Message: err.Error()},
		}
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	ledger.messages = append(ledger.messages, CloneOutboxMessagesV0(messages)...)
	return CloneOutboxMessagesV0(messages), nil
}

func (ledger *InMemoryOutboxLedgerV0) ListPending(
	ctx context.Context,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if err := ctx.Err(); err != nil {
		return nil, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
			{Code: "context_cancelado", Field: "context", Message: err.Error()},
		}
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	ledger.ensureStateV0()
	var out []orquestacoreworkflow.OutboxMessageV0
	for _, message := range ledger.messages {
		if ledger.acked[strings.TrimSpace(message.MessageID)] {
			continue
		}
		if filter.RunRef != "" && strings.TrimSpace(message.RunID) != strings.TrimSpace(filter.RunRef) {
			continue
		}
		if filter.TargetPort != "" && strings.TrimSpace(message.TargetPort) != strings.TrimSpace(filter.TargetPort) {
			continue
		}
		out = append(out, message)
	}
	return CloneOutboxMessagesV0(out), nil
}

func (ledger *InMemoryOutboxLedgerV0) ListPendingOutboxV0(
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     filter.RunID,
		TargetPort: filter.TargetPort,
	})
	if len(issues) > 0 {
		return nil, []orquestaoutboxdispatch.DispatchIssueV0{
			{Code: issues[0].Code, Field: issues[0].Field, Message: issues[0].Message},
		}
	}
	entries := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(pending))
	for _, message := range pending {
		if filter.MessageType != "" && strings.TrimSpace(message.MessageType) != strings.TrimSpace(filter.MessageType) {
			continue
		}
		entries = append(entries, orquestaoutboxdispatch.OutboxPendingEntryV0{
			MessageID:      strings.TrimSpace(message.MessageID),
			RunID:          strings.TrimSpace(message.RunID),
			TargetPort:     strings.TrimSpace(message.TargetPort),
			MessageType:    strings.TrimSpace(message.MessageType),
			IdempotencyKey: strings.TrimSpace(message.IdempotencyKey),
			CorrelationID:  strings.TrimSpace(message.CorrelationID),
			PayloadVersion: strings.TrimSpace(message.PayloadVersion),
			Payload:        append(message.Payload[:0:0], message.Payload...),
		})
	}
	return entries, nil
}

func (ledger *InMemoryOutboxLedgerV0) ClaimOutboxDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) (orquestaoutboxdispatch.OutboxDispatchClaimResultV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	ledger.ensureStateV0()
	messageID := strings.TrimSpace(claim.MessageID)
	if messageID == "" {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{}, []orquestaoutboxdispatch.DispatchIssueV0{
			{Code: "invalid_claim", Field: "message_id", Message: "message_id requerido"},
		}
	}
	if ledger.claimed[messageID] || ledger.acked[messageID] {
		return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
			AlreadyClaimed: true,
			MessageID:      messageID,
			TargetPort:     strings.TrimSpace(claim.TargetPort),
		}, nil
	}
	ledger.claimed[messageID] = true
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		Claimed:    true,
		MessageID:  messageID,
		TargetPort: strings.TrimSpace(claim.TargetPort),
	}, nil
}

func (ledger *InMemoryOutboxLedgerV0) AckOutboxDispatchV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	ledger.ensureStateV0()
	messageID := strings.TrimSpace(ack.MessageID)
	if messageID == "" {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			{Code: "invalid_ack", Field: "message_id", Message: "message_id requerido"},
		}
	}
	ledger.acked[messageID] = true
	delete(ledger.claimed, messageID)
	return nil
}

func (ledger *InMemoryOutboxLedgerV0) ensureStateV0() {
	if ledger.claimed == nil {
		ledger.claimed = map[string]bool{}
	}
	if ledger.acked == nil {
		ledger.acked = map[string]bool{}
	}
}

func CloneOutboxMessagesV0(
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
