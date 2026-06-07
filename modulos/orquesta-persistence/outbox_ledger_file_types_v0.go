package orquestapersistence

import (
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	FileOutboxLedgerSchemaVersionV0       = "orquesta_persistence.file_outbox_ledger.v0"
	legacyFileOutboxLedgerSchemaVersionV0 = "orquesta_state_file.outbox_ledger.v0"
	fileOutboxLedgerNameV0                = "outbox_ledger_v0.json"
)

type FileOutboxLedgerV0 struct {
	mu    sync.Mutex
	path  string
	state fileOutboxLedgerStateV0
}

type fileOutboxLedgerStateV0 struct {
	order              []string
	recordsByMessageID map[string]*fileOutboxLedgerRecordV0
	messageIDByIdemKey map[string]string
}

type fileOutboxLedgerSnapshotV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	Records       []fileOutboxLedgerRecordV0 `json:"records"`
}

type fileOutboxLedgerRecordV0 struct {
	Message            orquestacoreworkflow.OutboxMessageV0 `json:"message"`
	Claim              *fileOutboxLedgerClaimV0             `json:"claim,omitempty"`
	Ack                *fileOutboxLedgerAckV0               `json:"ack,omitempty"`
	messageFingerprint []byte
}

type fileOutboxLedgerClaimV0 struct {
	MessageID      string `json:"message_id"`
	RunID          string `json:"run_id,omitempty"`
	TargetPort     string `json:"target_port,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	ClaimRef       string `json:"claim_ref"`
	LeaseRef       string `json:"lease_ref"`
	ClaimedAt      string `json:"claimed_at,omitempty"`
	Recovered      bool   `json:"recovered_after_restart,omitempty"`
}

type fileOutboxLedgerAckV0 struct {
	MessageID    string                `json:"message_id"`
	RunID        string                `json:"run_id,omitempty"`
	TargetPort   string                `json:"target_port,omitempty"`
	Status       string                `json:"status"`
	DispatchRef  string                `json:"dispatch_ref,omitempty"`
	EvidenceRefs []string              `json:"evidence_refs,omitempty"`
	IssueCodes   []string              `json:"issue_codes,omitempty"`
	Issues       []OutboxLedgerIssueV0 `json:"issues,omitempty"`
}
