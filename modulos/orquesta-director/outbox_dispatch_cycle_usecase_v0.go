package orquestadirector

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func RunOutboxDispatchCycleV0(
	ctx context.Context,
	input OutboxDispatchCycleInputV0,
) (OutboxDispatchCycleResultV0, error) {
	input = normalizeOutboxDispatchCycleInputV0(input)
	result := OutboxDispatchCycleResultV0{
		RunID:      input.RunID,
		TargetPort: input.TargetPort,
	}
	if err := validateOutboxDispatchCycleInputV0(ctx, input); err != nil {
		return resultWithCycleErrorV0(result, err)
	}
	if err := ctx.Err(); err != nil {
		return resultWithCycleIssueV0(result, input, ErrDirectorOutboxDispatchCycleInvalidoV0, "context", "context cancelado")
	}
	if err := saveOutboxDispatchCycleMessagesV0(ctx, input, &result); err != nil {
		return result, err
	}
	pending, err := listOutboxDispatchCyclePendingV0(ctx, input, &result)
	if err != nil {
		return result, err
	}
	for _, message := range pending {
		if err := dispatchOutboxCyclePendingV0(ctx, input, message, &result); err != nil {
			_ = updateOutboxDispatchCyclePendingAfterV0(ctx, input, &result)
			return result, err
		}
	}
	if err := updateOutboxDispatchCyclePendingAfterV0(ctx, input, &result); err != nil {
		return result, err
	}
	return result, nil
}

func saveOutboxDispatchCycleMessagesV0(
	ctx context.Context,
	input OutboxDispatchCycleInputV0,
	result *OutboxDispatchCycleResultV0,
) error {
	if len(input.Messages) == 0 {
		return nil
	}
	saved, issues := input.Ledger.SavePending(ctx, input.Messages)
	result.SavedCount = len(saved)
	if len(issues) != 0 {
		return appendCycleLedgerIssuesV0(input, result, "save_pending", issues)
	}
	return nil
}

func listOutboxDispatchCyclePendingV0(
	ctx context.Context,
	input OutboxDispatchCycleInputV0,
	result *OutboxDispatchCycleResultV0,
) ([]orquestacoreworkflow.OutboxMessageV0, error) {
	pending, issues := input.Ledger.ListPending(ctx, OutboxPendingFilterV0{
		RunID:      input.RunID,
		TargetPort: input.TargetPort,
	})
	result.PendingBeforeCount = len(pending)
	if len(issues) != 0 {
		return nil, appendCycleLedgerIssuesV0(input, result, "list_pending", issues)
	}
	return cloneOutboxDispatchCycleMessagesV0(pending), nil
}

func dispatchOutboxCyclePendingV0(
	ctx context.Context,
	input OutboxDispatchCycleInputV0,
	message orquestacoreworkflow.OutboxMessageV0,
	result *OutboxDispatchCycleResultV0,
) error {
	if err := orquestacoreworkflow.ValidateOutboxMessageV0(message); err != nil {
		issue := outboxDispatchCycleIssueV0(ErrDirectorOutboxDispatchCycleInvalidoV0, "outbox", "outbox invalido")
		result.Issues = append(result.Issues, issue)
		return outboxDispatchCycleErrorV0(input, ErrDirectorOutboxDispatchCycleInvalidoV0, "outbox invalido", "outbox", result.Issues)
	}
	receipt, dispatchErr := input.Dispatcher.DispatchOutboxMessageV0(ctx, message)
	ack, issue := outboxDispatchAckFromAttemptV0(input, message, receipt, dispatchErr)
	if issue != nil {
		result.Issues = append(result.Issues, *issue)
		return outboxDispatchCycleErrorV0(input, ErrDirectorOutboxDispatchCycleInvalidoV0, issue.Message, issue.Field, result.Issues)
	}
	snapshot, issues := input.Ledger.MarkDispatched(ctx, ack)
	if len(issues) != 0 {
		return appendCycleLedgerIssuesV0(input, result, "mark_dispatched", issues)
	}
	result.Snapshots = append(result.Snapshots, snapshot)
	if dispatchErr != nil {
		result.FailedCount++
		issue := outboxDispatchCycleIssueV0(ack.ErrorCode, "dispatcher", "dispatcher failed")
		result.Issues = append(result.Issues, issue)
		return outboxDispatchCycleErrorV0(input, ErrDirectorOutboxDispatchCycleDispatchFailedV0, "dispatcher failed", "dispatcher", result.Issues)
	}
	result.DispatchedCount++
	return nil
}

func updateOutboxDispatchCyclePendingAfterV0(
	ctx context.Context,
	input OutboxDispatchCycleInputV0,
	result *OutboxDispatchCycleResultV0,
) error {
	pending, issues := input.Ledger.ListPending(ctx, OutboxPendingFilterV0{
		RunID:      input.RunID,
		TargetPort: input.TargetPort,
	})
	result.PendingAfterCount = len(pending)
	if len(issues) != 0 {
		return appendCycleLedgerIssuesV0(input, result, "list_pending_after", issues)
	}
	return nil
}

func appendCycleLedgerIssuesV0(
	input OutboxDispatchCycleInputV0,
	result *OutboxDispatchCycleResultV0,
	field string,
	issues []OutboxDispatchCycleIssueV0,
) error {
	result.Issues = append(result.Issues, issues...)
	return outboxDispatchCycleErrorV0(
		input,
		ErrDirectorOutboxDispatchCycleLedgerV0,
		"ledger outbox devolvio issues",
		field,
		result.Issues,
	)
}

func resultWithCycleErrorV0(
	result OutboxDispatchCycleResultV0,
	err error,
) (OutboxDispatchCycleResultV0, error) {
	cycleErr, ok := err.(OutboxDispatchCycleErrorV0)
	if !ok {
		return result, err
	}
	result.Issues = append(result.Issues, cycleErr.Issues...)
	return result, cycleErr
}

func resultWithCycleIssueV0(
	result OutboxDispatchCycleResultV0,
	input OutboxDispatchCycleInputV0,
	code string,
	field string,
	message string,
) (OutboxDispatchCycleResultV0, error) {
	issue := outboxDispatchCycleIssueV0(code, field, message)
	result.Issues = append(result.Issues, issue)
	return result, outboxDispatchCycleErrorV0(input, code, message, field, result.Issues)
}
