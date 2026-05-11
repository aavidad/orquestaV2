package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

const (
	OutboxDispatchBatchPlannedV0        = "planned"
	OutboxDispatchBatchNoPendingV0      = "no_pending"
	OutboxDispatchBatchAlreadyClaimedV0 = "already_claimed"
	OutboxDispatchBatchInvalidV0        = "invalid_request"
)

type OutboxDispatchBatchPlanRequestV0 struct {
	RunRef      string
	TargetPort  string
	MessageType string
	MaxReady    int
	Reader      orquestaoutboxdispatch.PendingOutboxReaderPortV0
	Claimer     orquestaoutboxdispatch.OutboxDispatchClaimerPortV0
}

type OutboxDispatchBatchPlanResultV0 struct {
	Status            string
	RunRef            string
	TargetPort        string
	Intents           []orquestaoutboxdispatch.DispatchIntentV0
	ClaimedMessageIDs []string
	Issues            int
}

func RunOutboxDispatchBatchPlanV0(
	ctx context.Context,
	request OutboxDispatchBatchPlanRequestV0,
) (OutboxDispatchBatchPlanResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return OutboxDispatchBatchPlanResultV0{}, err
	}
	request = normalizeOutboxDispatchBatchPlanRequestV0(request)
	result := OutboxDispatchBatchPlanResultV0{
		Status:     OutboxDispatchBatchInvalidV0,
		RunRef:     request.RunRef,
		TargetPort: request.TargetPort,
	}
	if invalidIssues := countInvalidBatchPlanPortsV0(request); invalidIssues > 0 {
		result.Issues = invalidIssues
		return result, nil
	}

	pending, issues := request.Reader.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       request.RunRef,
		TargetPort:  request.TargetPort,
		MessageType: request.MessageType,
	})
	result.Issues += len(issues)
	decision := orquestaoutboxdispatch.ChooseDispatchBatchV0(orquestaoutboxdispatch.DispatchBatchSelectionV0{
		RunID:       request.RunRef,
		TargetPort:  request.TargetPort,
		MessageType: request.MessageType,
		Pending:     pending,
		MaxReady:    candidateBatchLimitV0(request.MaxReady, len(pending)),
	})
	result.Issues += len(decision.Issues)
	if decision.Kind != orquestaoutboxdispatch.DispatchDecisionReadyV0 {
		result.Status = batchPlanStatusFromDecisionV0(decision.Kind)
		return result, nil
	}

	result.Intents, result.ClaimedMessageIDs, result.Issues = claimBatchPlanIntentsV0(
		request.Claimer,
		decision.Intents,
		effectiveBatchReadyLimitV0(request.MaxReady),
		result.Issues,
	)
	if len(result.Intents) == 0 {
		result.Status = OutboxDispatchBatchAlreadyClaimedV0
		return result, nil
	}
	result.Status = OutboxDispatchBatchPlannedV0
	return result, nil
}

func normalizeOutboxDispatchBatchPlanRequestV0(
	request OutboxDispatchBatchPlanRequestV0,
) OutboxDispatchBatchPlanRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.TargetPort = strings.TrimSpace(request.TargetPort)
	request.MessageType = strings.TrimSpace(request.MessageType)
	return request
}

func countInvalidBatchPlanPortsV0(request OutboxDispatchBatchPlanRequestV0) int {
	issues := 0
	if request.TargetPort == "" {
		issues++
	}
	if request.Reader == nil {
		issues++
	}
	if request.Claimer == nil {
		issues++
	}
	return issues
}

func claimBatchPlanIntentsV0(
	claimer orquestaoutboxdispatch.OutboxDispatchClaimerPortV0,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
	maxReady int,
	issueCount int,
) ([]orquestaoutboxdispatch.DispatchIntentV0, []string, int) {
	claimedIntents := make([]orquestaoutboxdispatch.DispatchIntentV0, 0, len(intents))
	claimedIDs := make([]string, 0, len(intents))
	for _, intent := range intents {
		if len(claimedIntents) >= maxReady {
			break
		}
		claim, issues := claimer.ClaimOutboxDispatchV0(batchPlanClaimFromIntentV0(intent))
		issueCount += len(issues)
		if !claim.Claimed {
			continue
		}
		claimedIntents = append(claimedIntents, intent)
		claimedIDs = append(claimedIDs, intent.MessageID)
	}
	return claimedIntents, compactStringsV0(claimedIDs), issueCount
}

func candidateBatchLimitV0(requested int, pendingCount int) int {
	if pendingCount <= 0 {
		return effectiveBatchReadyLimitV0(requested)
	}
	return pendingCount
}

func effectiveBatchReadyLimitV0(requested int) int {
	if requested <= 0 {
		return 1
	}
	return requested
}

func batchPlanClaimFromIntentV0(
	intent orquestaoutboxdispatch.DispatchIntentV0,
) orquestaoutboxdispatch.OutboxDispatchClaimV0 {
	return orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      intent.MessageID,
		RunID:          intent.RunID,
		TargetPort:     intent.TargetPort,
		IdempotencyKey: intent.IdempotencyKey,
	}
}

func batchPlanStatusFromDecisionV0(
	kind orquestaoutboxdispatch.DispatchDecisionKindV0,
) string {
	if kind == orquestaoutboxdispatch.DispatchDecisionNoPendingV0 {
		return OutboxDispatchBatchNoPendingV0
	}
	return OutboxDispatchBatchInvalidV0
}
