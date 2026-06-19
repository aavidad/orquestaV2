package orquestadirectorcycleoutbox

import "context"

func RecordDirectorCycleOutboxV0(
	ctx context.Context,
	input DirectorCycleOutboxRecordInputV0,
) (DirectorCycleOutboxRecordResultV0, error) {
	input = normalizeDirectorCycleOutboxInputV0(input)
	result := newDirectorCycleOutboxResultV0(input)
	if err := validateDirectorCycleOutboxInputV0(ctx, input); err != nil {
		return resultWithCycleOutboxErrorV0(result, err)
	}
	if err := ctx.Err(); err != nil {
		return resultWithCycleOutboxIssueV0(result, input, ErrDirectorCycleOutboxInvalidoV0, "context", "context cancelado", true)
	}
	if len(input.Messages) > 0 {
		saved, issues := input.Ledger.SavePending(ctx, input.Messages)
		result.SavedCount = len(saved)
		result.SavedOutboxRefs = cycleOutboxMessageRefsV0(saved)
		if len(issues) > 0 {
			return resultWithCycleOutboxLedgerIssuesV0(result, input, "save_pending", issues)
		}
	}
	pending, issues := input.Ledger.ListPending(ctx, DirectorCycleOutboxPendingFilterV0{
		RunRef:     input.RunRef,
		TargetPort: input.TargetPort,
	})
	result.PendingCount = len(pending)
	result.PendingOutboxRefs = cycleOutboxMessageRefsV0(pending)
	if len(issues) > 0 {
		return resultWithCycleOutboxLedgerIssuesV0(result, input, "list_pending", issues)
	}
	return result, nil
}
