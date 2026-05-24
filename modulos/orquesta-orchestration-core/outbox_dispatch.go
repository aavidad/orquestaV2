package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

type OutboxDispatchOnceRequestV0 struct {
	RunRef      string
	TargetPort  string
	MessageType string
	Reader      orquestaoutboxdispatch.PendingOutboxReaderPortV0
	Claimer     orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
	Executor    orquestaoutboxdispatch.OutboxDispatchExecutorPortV0
	Acker       orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type OutboxDispatchOnceResultV0 struct {
	Status       string
	RunRef       string
	TargetPort   string
	MessageType  string
	MessageID    string
	DispatchRef  string
	EvidenceRefs []string
	Issues       int
}

func RunOutboxDispatchOnceV0(
	ctx context.Context,
	request OutboxDispatchOnceRequestV0,
) (OutboxDispatchOnceResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OutboxDispatchOnceResultV0{}, err
	}
	request = normalizeOutboxDispatchRequestV0(request)
	result, err := orquestaoutboxdispatch.RunOutboxDispatchOnceV0(
		orquestaoutboxdispatch.RunOutboxDispatchOnceInputV0{
			RunID:       request.RunRef,
			TargetPort:  request.TargetPort,
			MessageType: request.MessageType,
			Reader:      request.Reader,
			Claimer:     request.Claimer,
			Executor:    request.Executor,
			Acker:       request.Acker,
		},
	)
	if result.Status == orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchFailedV0 ||
		result.Status == orquestaoutboxdispatch.RunOutboxDispatchOnceAckFailedV0 {
		_ = releaseOnceClaimV0(request.Claimer, result.Intent)
	}
	return compactOutboxDispatchResultV0(result), err
}

func normalizeOutboxDispatchRequestV0(
	request OutboxDispatchOnceRequestV0,
) OutboxDispatchOnceRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TargetPort = strings.TrimSpace(request.TargetPort)
	request.MessageType = strings.TrimSpace(request.MessageType)
	return request
}

func compactOutboxDispatchResultV0(
	result orquestaoutboxdispatch.RunOutboxDispatchOnceResultV0,
) OutboxDispatchOnceResultV0 {
	return OutboxDispatchOnceResultV0{
		Status:       string(result.Status),
		RunRef:       result.RunID,
		TargetPort:   result.TargetPort,
		MessageType:  result.Intent.MessageType,
		MessageID:    result.Intent.MessageID,
		DispatchRef:  result.Execution.DispatchRef,
		EvidenceRefs: append([]string(nil), result.Execution.EvidenceRefs...),
		Issues:       len(result.Issues),
	}
}

func releaseOnceClaimV0(
	claimer orquestaoutboxdispatch.OutboxDispatchClaimerPortV0,
	intent orquestaoutboxdispatch.DispatchIntentV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if intent.MessageID == "" {
		return nil
	}
	releaser, ok := claimer.(outboxDispatchClaimReleaserPortV0)
	if !ok {
		return nil
	}
	return releaser.ReleaseOutboxDispatchClaimV0(batchPlanClaimFromIntentV0(intent))
}
