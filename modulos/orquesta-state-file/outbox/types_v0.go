package orquestastatefileoutbox

import (
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	fileOutboxLedgerSchemaVersionV0 = "orquesta_state_file.outbox_ledger.v0"
	fileOutboxLedgerNameV0          = "outbox_ledger_v0.json"

	errPayloadInvalidV0         = "payload_invalido"
	errPersistenceUnavailableV0 = "persistencia_no_disponible"
	errIdempotencyConflictV0    = "conflicto_idempotencia"
)

type FileOutboxLedgerV0 struct {
	mu    sync.Mutex
	path  string
	state outboxLedgerStateV0
}

type outboxLedgerStateV0 struct {
	order              []string
	recordsByMessageID map[string]*outboxLedgerRecordV0
	messageIDByIdemKey map[string]string
}

type outboxLedgerSnapshotV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	Records       []outboxLedgerRecordV0 `json:"records"`
}

type outboxLedgerRecordV0 struct {
	Message            orquestacoreworkflow.OutboxMessageV0 `json:"message"`
	Claim              *outboxLedgerClaimV0                 `json:"claim,omitempty"`
	Ack                *outboxLedgerAckV0                   `json:"ack,omitempty"`
	messageFingerprint []byte
}

type outboxLedgerClaimV0 struct {
	MessageID      string `json:"message_id"`
	RunID          string `json:"run_id,omitempty"`
	TargetPort     string `json:"target_port,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type outboxLedgerAckV0 struct {
	MessageID    string   `json:"message_id"`
	RunID        string   `json:"run_id,omitempty"`
	TargetPort   string   `json:"target_port,omitempty"`
	DispatchRef  string   `json:"dispatch_ref,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type outboxLedgerCandidateV0 struct {
	message     orquestacoreworkflow.OutboxMessageV0
	fingerprint []byte
}
