package orquestaappdirectorservice

import (
	"context"
	"strconv"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaorchestrationbudget "orquesta/modulos/orquesta-orchestration-budget"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestLoadOperationalRunEventsV0UsaPaginaCanonicaV0(t *testing.T) {
	ctx := context.Background()
	total := orquestaorchestrationbudget.RunEventReadPageLimitV0 + 5
	reader := &operationalRunEventPagedReaderBudgetTestV0{total: total}

	got, err := loadOperationalRunEventsV0(ctx, reader, "run-ref-operational-budget-001")
	if err != nil {
		t.Fatalf("load operational events: %v", err)
	}
	if len(got) != total {
		t.Fatalf("events len=%d, want %d", len(got), total)
	}
	if len(reader.limits) == 0 || reader.limits[0] != orquestaorchestrationbudget.RunEventReadPageLimitV0 {
		t.Fatalf("page limit inicial=%v, want %d", reader.limits, orquestaorchestrationbudget.RunEventReadPageLimitV0)
	}
}

func TestLoadOperationalRunEventsV0CortaPorMaximoCanonicoV0(t *testing.T) {
	ctx := context.Background()
	reader := &operationalRunEventPagedReaderBudgetTestV0{total: orquestaorchestrationbudget.RunEventReadMaxV0 + 1}

	_, err := loadOperationalRunEventsV0(ctx, reader, "run-ref-operational-budget-002")
	if err == nil || !strings.Contains(err.Error(), "events_full_history_budget_exceeded") {
		t.Fatalf("err=%v, want events_full_history_budget_exceeded", err)
	}
	for _, limit := range reader.limits {
		if limit != orquestaorchestrationbudget.RunEventReadPageLimitV0 {
			t.Fatalf("page limits=%v, want only %d", reader.limits, orquestaorchestrationbudget.RunEventReadPageLimitV0)
		}
	}
}

type operationalRunEventPagedReaderBudgetTestV0 struct {
	total  int
	limits []int
}

func (reader *operationalRunEventPagedReaderBudgetTestV0) LoadRunEventsV0(
	context.Context,
	string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	return nil, nil
}

func (reader *operationalRunEventPagedReaderBudgetTestV0) LoadRunEventsPageV0(
	_ context.Context,
	request orquestacionnucleoapp.RunEventPageRequestV0,
) (orquestacionnucleoapp.RunEventPageResultV0, error) {
	reader.limits = append(reader.limits, request.Limit)
	offset := 0
	if strings.TrimSpace(request.Cursor) != "" {
		parsed, err := strconv.Atoi(request.Cursor)
		if err != nil {
			return orquestacionnucleoapp.RunEventPageResultV0{}, err
		}
		offset = parsed
	}
	if offset > reader.total {
		offset = reader.total
	}
	end := offset + request.Limit
	if end > reader.total {
		end = reader.total
	}
	result := orquestacionnucleoapp.RunEventPageResultV0{
		Events: make([]orquestacoreworkflow.OrchestrationEventV0, end-offset),
		Total:  reader.total,
	}
	if end < reader.total {
		result.HasMore = true
		result.NextCursor = strconv.Itoa(end)
	}
	return result, nil
}
