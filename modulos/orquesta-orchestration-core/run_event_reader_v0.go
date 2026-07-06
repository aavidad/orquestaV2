package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaorchestrationbudget "orquesta/modulos/orquesta-orchestration-budget"
)

const (
	runEventsFullReadPageLimitV0 = orquestaorchestrationbudget.RunEventReadPageLimitV0
	runEventsFullReadMaxV0       = orquestaorchestrationbudget.RunEventReadMaxV0
)

func loadRunEventsWithBudgetV0(
	ctx context.Context,
	reader RunEventReaderPortV0,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	if paged, ok := reader.(RunEventPagedReaderPortV0); ok {
		return loadRunEventsWithPagesV0(ctx, paged, runRef)
	}
	events, err := reader.LoadRunEventsV0(ctx, runRef)
	if err != nil {
		return nil, err
	}
	if len(events) > runEventsFullReadMaxV0 {
		return nil, errorV0(ErrNucleoOrquestacionStoreV0, "events.budget", "events_full_history_budget_exceeded")
	}
	return events, nil
}

func loadRunEventsWithPagesV0(
	ctx context.Context,
	reader RunEventPagedReaderPortV0,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	var out []orquestacoreworkflow.OrchestrationEventV0
	cursor := ""
	for {
		page, err := reader.LoadRunEventsPageV0(ctx, RunEventPageRequestV0{
			RunRef: runRef,
			Limit:  runEventsFullReadPageLimitV0,
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, page.Events...)
		if len(out) > runEventsFullReadMaxV0 {
			return nil, errorV0(ErrNucleoOrquestacionStoreV0, "events.budget", "events_full_history_budget_exceeded")
		}
		if !page.HasMore {
			return out, nil
		}
		cursor = page.NextCursor
	}
}
