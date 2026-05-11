package orquestaoutboxdispatch

type DispatchDecisionKindV0 string

const (
	DispatchDecisionReadyV0          DispatchDecisionKindV0 = "ready"
	DispatchDecisionNoPendingV0      DispatchDecisionKindV0 = "no_pending"
	DispatchDecisionInvalidRequestV0 DispatchDecisionKindV0 = "invalid_request"
)

type OutboxPendingEntryV0 struct {
	MessageID      string
	RunID          string
	TargetPort     string
	MessageType    string
	IdempotencyKey string
	CorrelationID  string
	PayloadVersion string
	Payload        []byte
}

type DispatchSelectionV0 struct {
	RunID             string
	TargetPort        string
	MessageType       string
	Pending           []OutboxPendingEntryV0
	ClaimedMessageIDs []string
}

type DispatchBatchSelectionV0 struct {
	RunID             string
	TargetPort        string
	MessageType       string
	Pending           []OutboxPendingEntryV0
	ClaimedMessageIDs []string
	MaxReady          int
}

type DispatchIntentV0 struct {
	MessageID      string
	RunID          string
	TargetPort     string
	MessageType    string
	IdempotencyKey string
	CorrelationID  string
	PayloadVersion string
	Payload        []byte
}

type DispatchDecisionV0 struct {
	Kind   DispatchDecisionKindV0
	Reason string
	Intent DispatchIntentV0
	Issues []DispatchIssueV0
}

type DispatchBatchDecisionV0 struct {
	Kind    DispatchDecisionKindV0
	Reason  string
	Intents []DispatchIntentV0
	Issues  []DispatchIssueV0
}

type DispatchIssueV0 struct {
	Code    string
	Field   string
	Message string
}
