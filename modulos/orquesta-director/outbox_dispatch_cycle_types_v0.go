package orquestadirector

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	OutboxDispatchStatusDispatchedV0 = "dispatched"
	OutboxDispatchStatusFailedV0     = "failed"

	ErrDirectorOutboxDispatchCycleInvalidoV0       = "director_outbox_dispatch_cycle_invalido"
	ErrDirectorOutboxDispatchCycleLedgerV0         = "director_outbox_dispatch_cycle_ledger"
	ErrDirectorOutboxDispatchCycleDispatchFailedV0 = "director_outbox_dispatch_cycle_dispatch_failed"
	ErrOutboxDispatchRefRequeridoV0                = "dispatch_ref_requerido"
	ErrOutboxDispatchFailedV0                      = "dispatch_failed"
)

type OutboxLedgerPortV0 interface {
	SavePending(
		ctx context.Context,
		messages []orquestacoreworkflow.OutboxMessageV0,
	) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0)
	ListPending(
		ctx context.Context,
		filter OutboxPendingFilterV0,
	) ([]orquestacoreworkflow.OutboxMessageV0, []OutboxDispatchCycleIssueV0)
	MarkDispatched(
		ctx context.Context,
		ack OutboxDispatchAckV0,
	) (OutboxDispatchSnapshotV0, []OutboxDispatchCycleIssueV0)
}

type OutboxDispatcherPortV0 interface {
	DispatchOutboxMessageV0(
		ctx context.Context,
		message orquestacoreworkflow.OutboxMessageV0,
	) (OutboxDispatchReceiptV0, error)
}

type OutboxDispatchCodedErrorV0 interface {
	OutboxDispatchErrorCodeV0() string
}

type OutboxDispatchCycleInputV0 struct {
	Ledger        OutboxLedgerPortV0                     `json:"-"`
	Dispatcher    OutboxDispatcherPortV0                 `json:"-"`
	RunID         string                                 `json:"run_id"`
	TargetPort    string                                 `json:"target_port"`
	DispatchedAt  string                                 `json:"dispatched_at"`
	Messages      []orquestacoreworkflow.OutboxMessageV0 `json:"messages,omitempty"`
	CorrelationID string                                 `json:"correlation_id,omitempty"`
}

type OutboxDispatchCycleResultV0 struct {
	RunID              string                       `json:"run_id"`
	TargetPort         string                       `json:"target_port"`
	SavedCount         int                          `json:"saved_count"`
	PendingBeforeCount int                          `json:"pending_before_count"`
	DispatchedCount    int                          `json:"dispatched_count"`
	FailedCount        int                          `json:"failed_count"`
	PendingAfterCount  int                          `json:"pending_after_count"`
	Snapshots          []OutboxDispatchSnapshotV0   `json:"snapshots,omitempty"`
	Issues             []OutboxDispatchCycleIssueV0 `json:"issues,omitempty"`
}

type OutboxPendingFilterV0 struct {
	RunID      string `json:"run_id,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}

type OutboxDispatchReceiptV0 struct {
	DispatchRef  string   `json:"dispatch_ref"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
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

type OutboxDispatchCycleIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type OutboxDispatchCycleErrorV0 struct {
	Code          string                       `json:"code"`
	Message       string                       `json:"message"`
	Field         string                       `json:"field,omitempty"`
	Retryable     bool                         `json:"retryable"`
	Issues        []OutboxDispatchCycleIssueV0 `json:"issues,omitempty"`
	CorrelationID string                       `json:"correlation_id,omitempty"`
}

func (err OutboxDispatchCycleErrorV0) Error() string {
	return err.Code
}
