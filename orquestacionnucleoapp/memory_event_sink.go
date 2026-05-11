package orquestacionnucleoapp

import (
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
		sink.events = append(sink.events, event)
	}
	return nil
}

func (sink *InMemoryEventSinkV0) EventsV0() []orquestacoreworkflow.OrchestrationEventV0 {
	return cloneEventsV0(sink.events)
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
