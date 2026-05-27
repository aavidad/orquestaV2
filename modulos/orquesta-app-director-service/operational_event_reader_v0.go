package orquestaappdirectorservice

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	operationalRunEventsPageLimitV0 = 250
	operationalRunEventsMaxV0       = 10000
)

func loadOperationalRunEventsV0(
	ctx context.Context,
	reader orquestacionnucleoapp.RunEventReaderPortV0,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	if paged, ok := reader.(orquestacionnucleoapp.RunEventPagedReaderPortV0); ok {
		return loadOperationalRunEventsPagedV0(ctx, paged, runRef)
	}
	events, err := reader.LoadRunEventsV0(ctx, runRef)
	if err != nil {
		return nil, err
	}
	if len(events) > operationalRunEventsMaxV0 {
		return nil, orquestacionnucleoapp.ErrorV0{
			Code:    orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0,
			Field:   "events.budget",
			Message: "events_full_history_budget_exceeded",
		}
	}
	return events, nil
}

func loadOperationalRunEventsPagedV0(
	ctx context.Context,
	reader orquestacionnucleoapp.RunEventPagedReaderPortV0,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	var out []orquestacoreworkflow.OrchestrationEventV0
	cursor := ""
	for {
		page, err := reader.LoadRunEventsPageV0(ctx, orquestacionnucleoapp.RunEventPageRequestV0{
			RunRef: runRef,
			Limit:  operationalRunEventsPageLimitV0,
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, page.Events...)
		if len(out) > operationalRunEventsMaxV0 {
			return nil, orquestacionnucleoapp.ErrorV0{
				Code:    orquestacionnucleoapp.ErrNucleoOrquestacionStoreV0,
				Field:   "events.budget",
				Message: "events_full_history_budget_exceeded",
			}
		}
		if !page.HasMore {
			return out, nil
		}
		cursor = page.NextCursor
	}
}
