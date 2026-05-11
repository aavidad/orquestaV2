package orquestaoutboxdispatch

type RunOutboxDispatchOnceStatusV0 string

const (
	RunOutboxDispatchOnceDispatchedV0     RunOutboxDispatchOnceStatusV0 = "dispatched"
	RunOutboxDispatchOnceNoPendingV0      RunOutboxDispatchOnceStatusV0 = "no_pending"
	RunOutboxDispatchOnceAlreadyClaimedV0 RunOutboxDispatchOnceStatusV0 = "already_claimed"
	RunOutboxDispatchOnceDispatchFailedV0 RunOutboxDispatchOnceStatusV0 = "dispatch_failed"
	RunOutboxDispatchOnceAckFailedV0      RunOutboxDispatchOnceStatusV0 = "ack_failed"
	RunOutboxDispatchOnceInvalidV0        RunOutboxDispatchOnceStatusV0 = "invalid_request"
)

type RunOutboxDispatchOnceInputV0 struct {
	RunID       string
	TargetPort  string
	MessageType string
	Reader      PendingOutboxReaderPortV0
	Claimer     OutboxDispatchClaimerPortV0
	Executor    OutboxDispatchExecutorPortV0
	Acker       OutboxDispatchAckPortV0
}

type RunOutboxDispatchOnceResultV0 struct {
	Status     RunOutboxDispatchOnceStatusV0
	RunID      string
	TargetPort string
	Decision   DispatchDecisionV0
	Intent     DispatchIntentV0
	Execution  OutboxDispatchExecutionResultV0
	Ack        OutboxDispatchAckV0
	Issues     []DispatchIssueV0
}

type OutboxDispatchClaimV0 struct {
	MessageID      string
	RunID          string
	TargetPort     string
	IdempotencyKey string
}

type OutboxDispatchClaimResultV0 struct {
	Claimed        bool
	AlreadyClaimed bool
	MessageID      string
	TargetPort     string
}

type OutboxDispatchExecutionResultV0 struct {
	DispatchRef  string
	EvidenceRefs []string
}

type OutboxDispatchAckV0 struct {
	MessageID    string
	RunID        string
	TargetPort   string
	DispatchRef  string
	EvidenceRefs []string
}
