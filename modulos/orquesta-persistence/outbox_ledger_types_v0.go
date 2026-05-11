package orquestapersistence

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const (
	OutboxDispatchStatusDispatchedV0 = "dispatched"
	OutboxDispatchStatusFailedV0     = "failed"
)

type OutboxPendingFilterV0 struct {
	RunID      string `json:"run_id,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}

type OutboxDispatchAckV0 struct {
	MessageID    string   `json:"message_id"`
	RunID        string   `json:"run_id"`
	TargetPort   string   `json:"target_port"`
	Status       string   `json:"status"`
	DispatchRef  string   `json:"dispatch_ref"`
	DispatchedAt string   `json:"dispatched_at"`
	ErrorCode    string   `json:"error_code,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OutboxDispatchSnapshotV0 struct {
	MessageID    string   `json:"message_id"`
	RunID        string   `json:"run_id"`
	TargetPort   string   `json:"target_port"`
	Status       string   `json:"status"`
	DispatchRef  string   `json:"dispatch_ref"`
	DispatchedAt string   `json:"dispatched_at"`
	ErrorCode    string   `json:"error_code,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OutboxLedgerIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type outboxLedgerRecordV0 struct {
	message            orquestacoreworkflow.OutboxMessageV0
	messageFingerprint []byte
	ack                *OutboxDispatchAckV0
	ackFingerprint     []byte
}
