package orquestacionnucleoapp

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type InMemoryEventSinkV0 struct {
	events []orquestacoreworkflow.OrchestrationEventV0
}

func NewInMemoryEventSinkV0() *InMemoryEventSinkV0 {
	return &InMemoryEventSinkV0{}
}

func (sink *InMemoryEventSinkV0) AppendRunEventsV0(
	ctx context.Context,
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, event := range events {
		if strings.TrimSpace(event.RunID) != strings.TrimSpace(runRef) {
			return fmt.Errorf("evento de otro run: %s", event.RunID)
		}
		if duplicate, err := sink.eventAlreadyAppendedV0(event); err != nil {
			return err
		} else if duplicate {
			continue
		}
		sink.events = append(sink.events, event)
	}
	return nil
}

func (sink *InMemoryEventSinkV0) eventAlreadyAppendedV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (bool, error) {
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" {
		return false, nil
	}
	runRef := strings.TrimSpace(event.RunID)
	for _, existing := range sink.events {
		if strings.TrimSpace(existing.RunID) != runRef || strings.TrimSpace(existing.EventID) != eventID {
			continue
		}
		if bytes.Equal(existing.Payload, event.Payload) {
			return true, nil
		}
		return false, fmt.Errorf("evento conflictivo: %s", eventID)
	}
	return false, nil
}

func (sink *InMemoryEventSinkV0) EventsV0() []orquestacoreworkflow.OrchestrationEventV0 {
	return cloneEventsV0(sink.events)
}

func (sink *InMemoryEventSinkV0) LoadRunEventsV0(
	ctx context.Context,
	runRef string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = strings.TrimSpace(runRef)
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(sink.events))
	for _, event := range sink.events {
		if strings.TrimSpace(event.RunID) == runRef {
			events = append(events, event)
		}
	}
	return cloneEventsV0(events), nil
}

func cloneEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(events))
	for _, event := range events {
		event.Payload = append(event.Payload[:0:0], event.Payload...)
		cloned = append(cloned, event)
	}
	return cloned
}
