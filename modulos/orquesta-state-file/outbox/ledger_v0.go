package orquestastatefileoutbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
)

// FileOutboxLedgerV0 persists outbox state in an explicit JSON file adapter.
func NewFileOutboxLedgerV0(dir string) (*FileOutboxLedgerV0, error) {
	path, err := ledgerPathV0(dir)
	if err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: dir_invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("file_outbox_ledger: dir_unavailable")
	}
	state, err := loadOutboxLedgerStateV0(path)
	if err != nil {
		return nil, err
	}
	return &FileOutboxLedgerV0{path: path, state: state}, nil
}

func (ledger *FileOutboxLedgerV0) SavePending(
	ctx context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if ledger == nil {
		return nil, directorPersistenceIssueV0()
	}
	if issues := contextIssueV0(ctx); len(issues) > 0 {
		return nil, issues
	}
	candidates, issues := buildCandidatesV0(messages)
	if len(issues) > 0 {
		return nil, issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	next := ledger.state.cloneV0()
	accepted, changed, issues := saveCandidatesV0(&next, candidates)
	if len(issues) > 0 {
		return nil, issues
	}
	if changed {
		if err := persistOutboxLedgerStateV0(ledger.path, next); err != nil {
			return nil, directorPersistenceIssueV0()
		}
		ledger.state = next
	}
	return accepted, nil
}

func (ledger *FileOutboxLedgerV0) ListPending(
	ctx context.Context,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	if ledger == nil {
		return nil, directorPersistenceIssueV0()
	}
	if issues := contextIssueV0(ctx); len(issues) > 0 {
		return nil, issues
	}
	filter, issues := normalizePendingFilterV0(filter)
	if len(issues) > 0 {
		return nil, issues
	}

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	pending := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(ledger.state.order))
	for _, messageID := range ledger.state.order {
		record := ledger.state.recordsByMessageID[messageID]
		if recordMatchesPendingFilterV0(record, filter) {
			pending = append(pending, cloneOutboxMessageV0(record.Message))
		}
	}
	return pending, nil
}

func saveCandidatesV0(
	state *outboxLedgerStateV0,
	candidates []outboxLedgerCandidateV0,
) ([]orquestacoreworkflow.OutboxMessageV0, bool, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	accepted := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(candidates))
	changed := false
	for _, candidate := range candidates {
		record, issues := state.compatibleRecordV0(candidate)
		if len(issues) > 0 {
			return nil, false, directorIssuesFromLedgerV0(issues)
		}
		if record != nil {
			accepted = append(accepted, cloneOutboxMessageV0(record.Message))
			continue
		}
		record = &outboxLedgerRecordV0{
			Message:            cloneOutboxMessageV0(candidate.message),
			messageFingerprint: append([]byte(nil), candidate.fingerprint...),
		}
		state.recordsByMessageID[candidate.message.MessageID] = record
		state.messageIDByIdemKey[candidate.message.IdempotencyKey] = candidate.message.MessageID
		state.order = append(state.order, candidate.message.MessageID)
		accepted = append(accepted, cloneOutboxMessageV0(record.Message))
		changed = true
	}
	return accepted, changed, nil
}
