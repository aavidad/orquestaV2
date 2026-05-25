package orquestaoutboxdispatch

type PendingOutboxReaderPortV0 interface {
	ListPendingOutboxV0(filter PendingOutboxFilterV0) ([]OutboxPendingEntryV0, []DispatchIssueV0)
}

type PendingOutboxFilterV0 struct {
	RunID       string
	TargetPort  string
	MessageType string
}

type OutboxDispatchClaimerPortV0 interface {
	ClaimOutboxDispatchV0(claim OutboxDispatchClaimV0) (OutboxDispatchClaimResultV0, []DispatchIssueV0)
}

type OutboxDispatchExecutorPortV0 interface {
	ExecuteOutboxDispatchV0(intent DispatchIntentV0) (OutboxDispatchExecutionResultV0, error)
}

type OutboxDispatchAckPortV0 interface {
	AckOutboxDispatchV0(ack OutboxDispatchAckV0) []DispatchIssueV0
}

type OutboxDispatchAckObservationPortV0 interface {
	AckOutboxDispatchObservationV0(ack OutboxDispatchAckObservationV0) []DispatchIssueV0
}
