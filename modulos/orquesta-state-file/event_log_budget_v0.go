package orquestastatefile

import (
	"encoding/json"

	orquestaorchestrationbudget "orquesta/modulos/orquesta-orchestration-budget"
)

const (
	defaultEventLogMaxAppendEventsV0       = orquestaorchestrationbudget.RunEventStoreDefaultAppendEventLimitV0
	defaultEventLogMaxRunEventsV0          = orquestaorchestrationbudget.RunEventStoreMaxEventsV0
	defaultEventLogMaxEventBytesV0         = orquestaorchestrationbudget.StateFileEventPayloadMaxBytesV0
	defaultEventLogMaxEventSnapshotBytesV0 = stateFileJSONMaxBytesV0
	defaultEventLogPageLimitV0             = orquestaorchestrationbudget.RunEventStoreDefaultPageLimitV0
	maxEventLogPageLimitV0                 = orquestaorchestrationbudget.RunEventStorePageLimitV0
)

type eventLogBudgetV0 struct {
	MaxAppendEvents       int
	MaxRunEvents          int
	MaxEventBytes         int
	MaxEventSnapshotBytes int64
}

func normalizeEventLogBudgetV0(config ConfigV0) eventLogBudgetV0 {
	budget := eventLogBudgetV0{
		MaxAppendEvents:       config.MaxAppendEvents,
		MaxRunEvents:          config.MaxRunEvents,
		MaxEventBytes:         config.MaxEventBytes,
		MaxEventSnapshotBytes: config.MaxEventSnapshotBytes,
	}
	if budget.MaxAppendEvents <= 0 {
		budget.MaxAppendEvents = defaultEventLogMaxAppendEventsV0
	}
	if budget.MaxRunEvents <= 0 {
		budget.MaxRunEvents = defaultEventLogMaxRunEventsV0
	}
	if budget.MaxEventBytes <= 0 {
		budget.MaxEventBytes = defaultEventLogMaxEventBytesV0
	}
	if budget.MaxEventSnapshotBytes <= 0 {
		budget.MaxEventSnapshotBytes = defaultEventLogMaxEventSnapshotBytesV0
	}
	return budget
}

func (budget eventLogBudgetV0) validateAppendV0(events []json.RawMessage) error {
	if len(events) > budget.MaxAppendEvents {
		return storeErrorV0("events.budget", "events_append_limit_exceeded")
	}
	for _, payload := range events {
		if len(payload) > budget.MaxEventBytes {
			return storeErrorV0("events.budget", "event_payload_limit_exceeded")
		}
	}
	return nil
}

func boundedEventPageLimitV0(limit int) int {
	if limit <= 0 {
		return defaultEventLogPageLimitV0
	}
	if limit > maxEventLogPageLimitV0 {
		return maxEventLogPageLimitV0
	}
	return limit
}
