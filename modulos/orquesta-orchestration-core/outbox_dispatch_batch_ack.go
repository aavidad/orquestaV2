package orquestacionnucleoapp

import (
	"context"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

const (
	OutboxDispatchBatchAckClosedV0  = "closed"
	OutboxDispatchBatchAckPendingV0 = "pending"
	OutboxDispatchBatchAckFailedV0  = "ack_failed"
	OutboxDispatchBatchAckInvalidV0 = "invalid_request"
)

type OutboxDispatchBatchAckRequestV0 struct {
	Intents []orquestaoutboxdispatch.DispatchIntentV0
	Acks    []orquestaoutboxdispatch.OutboxDispatchAckObservationV0
	Acker   orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type OutboxDispatchBatchAckResultV0 struct {
	Status        string
	AckedCount    int
	PendingCount  int
	FailedCount   int
	Issues        int
	AckedMessages []string
}

func RunOutboxDispatchBatchAckClosureV0(
	ctx context.Context,
	request OutboxDispatchBatchAckRequestV0,
) (OutboxDispatchBatchAckResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OutboxDispatchBatchAckResultV0{}, err
	}
	if request.Acker == nil {
		return OutboxDispatchBatchAckResultV0{
			Status: OutboxDispatchBatchAckInvalidV0,
			Issues: 1,
		}, nil
	}
	closure := orquestaoutboxdispatch.PlanOutboxDispatchAckClosureV0(
		orquestaoutboxdispatch.OutboxDispatchAckClosureInputV0{
			Intents: request.Intents,
			Acks:    request.Acks,
		},
	)
	result := OutboxDispatchBatchAckResultV0{
		AckedCount:   len(closure.Acked),
		PendingCount: len(closure.Pending),
		FailedCount:  len(closure.Failed),
		Issues:       len(closure.Issues),
	}
	result.Issues = ackBatchClosureFailuresV0(
		request.Acker,
		closure.Failed,
		result.Issues,
	)
	result.AckedMessages, result.Issues = ackBatchClosureSuccessesV0(
		request.Acker,
		closure.Acked,
		result.Issues,
	)
	result.Status = batchAckClosureStatusV0(result)
	return result, nil
}

func ackBatchClosureSuccessesV0(
	acker orquestaoutboxdispatch.OutboxDispatchAckPortV0,
	items []orquestaoutboxdispatch.OutboxDispatchAckClosureItemV0,
	issueCount int,
) ([]string, int) {
	ackedMessages := make([]string, 0, len(items))
	for _, item := range items {
		issues := acker.AckOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchAckV0{
			MessageID:    item.MessageID,
			RunID:        item.RunID,
			TargetPort:   item.TargetPort,
			DispatchRef:  item.DispatchRef,
			EvidenceRefs: item.EvidenceRefs,
		})
		issueCount += len(issues)
		if len(issues) == 0 {
			ackedMessages = append(ackedMessages, item.MessageID)
		}
	}
	return compactStringsV0(ackedMessages), issueCount
}

func ackBatchClosureFailuresV0(
	acker orquestaoutboxdispatch.OutboxDispatchAckPortV0,
	items []orquestaoutboxdispatch.OutboxDispatchAckClosureItemV0,
	issueCount int,
) int {
	observer, ok := acker.(orquestaoutboxdispatch.OutboxDispatchAckObservationPortV0)
	if !ok {
		return issueCount
	}
	for _, item := range items {
		issues := observer.AckOutboxDispatchObservationV0(orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			MessageID:    item.MessageID,
			RunID:        item.RunID,
			TargetPort:   item.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0,
			DispatchRef:  item.DispatchRef,
			EvidenceRefs: item.EvidenceRefs,
			Issues:       item.Issues,
		})
		issueCount += len(issues)
	}
	return issueCount
}

func batchAckClosureStatusV0(
	result OutboxDispatchBatchAckResultV0,
) string {
	if result.Issues > 0 || result.FailedCount > 0 {
		return OutboxDispatchBatchAckFailedV0
	}
	if result.PendingCount > 0 {
		return OutboxDispatchBatchAckPendingV0
	}
	return OutboxDispatchBatchAckClosedV0
}
