package orquestaoutboxdispatch

type fakePendingReaderV0 struct {
	pending []OutboxPendingEntryV0
	calls   []PendingOutboxFilterV0
}

func (f *fakePendingReaderV0) ListPendingOutboxV0(
	filter PendingOutboxFilterV0,
) ([]OutboxPendingEntryV0, []DispatchIssueV0) {
	f.calls = append(f.calls, filter)
	return cloneEntriesV0(f.pending), nil
}

type fakeClaimerPortV0 struct {
	claimed bool
	calls   []OutboxDispatchClaimV0
}

func (f *fakeClaimerPortV0) ClaimOutboxDispatchV0(
	claim OutboxDispatchClaimV0,
) (OutboxDispatchClaimResultV0, []DispatchIssueV0) {
	f.calls = append(f.calls, claim)
	return OutboxDispatchClaimResultV0{
		Claimed:        f.claimed,
		AlreadyClaimed: !f.claimed,
		MessageID:      claim.MessageID,
		TargetPort:     claim.TargetPort,
	}, nil
}

type fakeExecutorPortV0 struct {
	result OutboxDispatchExecutionResultV0
	err    error
	calls  []DispatchIntentV0
}

func (f *fakeExecutorPortV0) ExecuteOutboxDispatchV0(
	intent DispatchIntentV0,
) (OutboxDispatchExecutionResultV0, error) {
	f.calls = append(f.calls, intent)
	return f.result, f.err
}

type fakeAckPortV0 struct {
	calls []OutboxDispatchAckV0
}

func (f *fakeAckPortV0) AckOutboxDispatchV0(ack OutboxDispatchAckV0) []DispatchIssueV0 {
	f.calls = append(f.calls, ack)
	return nil
}

func runOncePortsV0(
	pending []OutboxPendingEntryV0,
	claimed bool,
	execution OutboxDispatchExecutionResultV0,
	err error,
) (*fakePendingReaderV0, *fakeClaimerPortV0, *fakeExecutorPortV0, *fakeAckPortV0) {
	return &fakePendingReaderV0{pending: pending},
		&fakeClaimerPortV0{claimed: claimed},
		&fakeExecutorPortV0{result: execution, err: err},
		&fakeAckPortV0{}
}
