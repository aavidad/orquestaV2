package orquestadirectorcycleoutbox

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	ErrDirectorCycleOutboxInvalidoV0 = "director_cycle_outbox_invalido"
	ErrDirectorCycleOutboxLedgerV0   = "director_cycle_outbox_ledger"
)

type DirectorCycleOutboxLedgerPortV0 interface {
	SavePending(
		ctx context.Context,
		messages []orquestacoreworkflow.OutboxMessageV0,
	) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0)
	ListPending(
		ctx context.Context,
		filter DirectorCycleOutboxPendingFilterV0,
	) ([]orquestacoreworkflow.OutboxMessageV0, []DirectorCycleOutboxIssueV0)
}

type DirectorCycleOutboxRecordInputV0 struct {
	Ledger        DirectorCycleOutboxLedgerPortV0        `json:"-"`
	RunRef        string                                 `json:"run_ref"`
	TargetPort    string                                 `json:"target_port,omitempty"`
	Messages      []orquestacoreworkflow.OutboxMessageV0 `json:"messages,omitempty"`
	CorrelationID string                                 `json:"correlation_id,omitempty"`
}

type DirectorCycleOutboxRecordResultV0 struct {
	RunRef            string                       `json:"run_ref"`
	TargetPort        string                       `json:"target_port,omitempty"`
	SavedCount        int                          `json:"saved_count"`
	PendingCount      int                          `json:"pending_count"`
	SavedOutboxRefs   []string                     `json:"saved_outbox_refs,omitempty"`
	PendingOutboxRefs []string                     `json:"pending_outbox_refs,omitempty"`
	Issues            []DirectorCycleOutboxIssueV0 `json:"issues,omitempty"`
}

type DirectorCycleOutboxPendingFilterV0 struct {
	RunRef     string `json:"run_ref,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}

type DirectorCycleOutboxIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type DirectorCycleOutboxErrorV0 struct {
	Code          string                       `json:"code"`
	Message       string                       `json:"message"`
	Field         string                       `json:"field,omitempty"`
	Retryable     bool                         `json:"retryable"`
	Issues        []DirectorCycleOutboxIssueV0 `json:"issues,omitempty"`
	CorrelationID string                       `json:"correlation_id,omitempty"`
}

func (err DirectorCycleOutboxErrorV0) Error() string {
	return err.Code
}
