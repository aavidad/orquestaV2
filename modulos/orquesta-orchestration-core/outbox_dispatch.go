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
		MessageID:    result.Intent.MessageID,
		DispatchRef:  result.Execution.DispatchRef,
		EvidenceRefs: append([]string(nil), result.Execution.EvidenceRefs...),
		Issues:       len(result.Issues),
	}
}
