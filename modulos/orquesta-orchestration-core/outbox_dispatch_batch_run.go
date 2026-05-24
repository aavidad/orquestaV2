package orquestacionnucleoapp

import (
	"context"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

const (
	OutboxDispatchBatchRunDispatchedV0      = "dispatched"
	OutboxDispatchBatchRunNoPendingV0       = "no_pending"
	OutboxDispatchBatchRunAlreadyClaimedV0  = "already_claimed"
	OutboxDispatchBatchRunExecutionFailedV0 = "execution_failed"
	OutboxDispatchBatchRunAckFailedV0       = "ack_failed"
	OutboxDispatchBatchRunPendingV0         = "pending"
	OutboxDispatchBatchRunInvalidV0         = "invalid_request"
)

type OutboxDispatchBatchExecutorPortV0 interface {
	ExecuteOutboxDispatchBatchV0(
		ctx context.Context,
		intents []orquestaoutboxdispatch.DispatchIntentV0,
	) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error)
}

type outboxDispatchClaimReleaserPortV0 interface {
	ReleaseOutboxDispatchClaimV0(
		claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
	) []orquestaoutboxdispatch.DispatchIssueV0
}

type OutboxDispatchBatchRunRequestV0 struct {
	RunRef      string
	TargetPort  string
	MessageType string
	MaxReady    int
	Reader      orquestaoutboxdispatch.PendingOutboxReaderPortV0
	Claimer     orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
	Executor    OutboxDispatchBatchExecutorPortV0
	Acker       orquestaoutboxdispatch.OutboxDispatchAckPortV0
}

type OutboxDispatchBatchRunResultV0 struct {
	Status            string
	RunRef            string
	TargetPort        string
	MessageType       string
	PlannedCount      int
	AckedCount        int
	PendingCount      int
	FailedCount       int
	Issues            int
	ClaimedMessageIDs []string
	AckedMessages     []string
}

func RunOutboxDispatchBatchV0(
	ctx context.Context,
	request OutboxDispatchBatchRunRequestV0,
) (OutboxDispatchBatchRunResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OutboxDispatchBatchRunResultV0{}, err
	}
	if request.Executor == nil {
		return invalidBatchRunResultV0(request, 1), nil
	}
	plan, err := RunOutboxDispatchBatchPlanV0(ctx, OutboxDispatchBatchPlanRequestV0{
		RunRef:      request.RunRef,
		TargetPort:  request.TargetPort,
		MessageType: request.MessageType,
		MaxReady:    request.MaxReady,
		Reader:      request.Reader,
		Claimer:     request.Claimer,
	})
	if err != nil {
		return OutboxDispatchBatchRunResultV0{}, err
	}
	result := batchRunResultFromPlanV0(plan)
	if plan.Status != OutboxDispatchBatchPlannedV0 {
		return result, nil
	}

	acks, err := request.Executor.ExecuteOutboxDispatchBatchV0(ctx, plan.Intents)
	if err != nil {
		result.Status = OutboxDispatchBatchRunExecutionFailedV0
		result.Issues += releaseBatchClaimsV0(request.Claimer, plan.Intents, nil)
		return result, err
	}
	closure, err := RunOutboxDispatchBatchAckClosureV0(ctx, OutboxDispatchBatchAckRequestV0{
		Intents: plan.Intents,
		Acks:    acks,
		Acker:   request.Acker,
	})
	if err != nil {
		return OutboxDispatchBatchRunResultV0{}, err
	}
	result.AckedCount = closure.AckedCount
	result.PendingCount = closure.PendingCount
	result.FailedCount = closure.FailedCount
	result.Issues += closure.Issues
	result.AckedMessages = append([]string(nil), closure.AckedMessages...)
	result.Issues += releaseBatchClaimsV0(request.Claimer, plan.Intents, result.AckedMessages)
	result.Status = batchRunStatusFromClosureV0(closure)
	return result, nil
}

func invalidBatchRunResultV0(
	request OutboxDispatchBatchRunRequestV0,
	issues int,
) OutboxDispatchBatchRunResultV0 {
	return OutboxDispatchBatchRunResultV0{
		Status:      OutboxDispatchBatchRunInvalidV0,
		RunRef:      request.RunRef,
		TargetPort:  request.TargetPort,
		MessageType: request.MessageType,
		Issues:      issues,
	}
}

func batchRunResultFromPlanV0(
	plan OutboxDispatchBatchPlanResultV0,
) OutboxDispatchBatchRunResultV0 {
	return OutboxDispatchBatchRunResultV0{
		Status:            batchRunStatusFromPlanV0(plan.Status),
		RunRef:            plan.RunRef,
		TargetPort:        plan.TargetPort,
		MessageType:       plan.MessageType,
		PlannedCount:      len(plan.Intents),
		Issues:            plan.Issues,
		ClaimedMessageIDs: append([]string(nil), plan.ClaimedMessageIDs...),
	}
}

func batchRunStatusFromPlanV0(planStatus string) string {
	switch planStatus {
	case OutboxDispatchBatchPlannedV0:
		return OutboxDispatchBatchRunPendingV0
	case OutboxDispatchBatchNoPendingV0:
		return OutboxDispatchBatchRunNoPendingV0
	case OutboxDispatchBatchAlreadyClaimedV0:
		return OutboxDispatchBatchRunAlreadyClaimedV0
	default:
		return OutboxDispatchBatchRunInvalidV0
	}
}

func batchRunStatusFromClosureV0(
	closure OutboxDispatchBatchAckResultV0,
) string {
	switch closure.Status {
	case OutboxDispatchBatchAckClosedV0:
		return OutboxDispatchBatchRunDispatchedV0
	case OutboxDispatchBatchAckPendingV0:
		return OutboxDispatchBatchRunPendingV0
	case OutboxDispatchBatchAckFailedV0:
		return OutboxDispatchBatchRunAckFailedV0
	default:
		return OutboxDispatchBatchRunInvalidV0
	}
}

func releaseBatchClaimsV0(
	claimer orquestaoutboxdispatch.OutboxDispatchClaimerPortV0,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
	ackedMessages []string,
) int {
	releaser, ok := claimer.(outboxDispatchClaimReleaserPortV0)
	if !ok || len(intents) == 0 {
		return 0
	}
	acked := map[string]bool{}
	for _, messageID := range compactStringsV0(ackedMessages) {
		acked[messageID] = true
	}
	issues := 0
	for _, intent := range intents {
		if acked[intent.MessageID] {
			continue
		}
		issues += len(releaser.ReleaseOutboxDispatchClaimV0(batchPlanClaimFromIntentV0(intent)))
	}
	return issues
}
