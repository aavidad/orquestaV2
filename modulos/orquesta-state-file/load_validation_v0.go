package orquestastatefile

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validateLoadedRunProjectionV0(run orquestacoreworkflow.OrchestrationRunV0) error {
	issues := orquestacoreworkflow.ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		return nil
	}
	issue := issues[0]
	field := strings.TrimSpace(issue.Field)
	if field == "" {
		field = "projection"
	}
	return storeErrorV0("run."+field, "run_projection_invalid:"+string(issue.Code))
}

func validateStateFileWorkflowEventV0(event orquestacoreworkflow.OrchestrationEventV0) error {
	if err := orquestacoreworkflow.ValidateOrchestrationEventV0(event); err != nil {
		if validateLegacyStateFileEventWithoutOccurredAtV0(event, err) == nil {
			return nil
		}
		return storeErrorV0("events.projection", "event_projection_invalid")
	}
	return nil
}

func validateLegacyStateFileEventWithoutOccurredAtV0(
	event orquestacoreworkflow.OrchestrationEventV0,
	err error,
) error {
	eventErr, ok := err.(orquestacoreworkflow.OrchestrationEventErrorV0)
	if !ok || eventErr.Code != orquestacoreworkflow.ErrEventoInvalidoV0 || eventErr.Field != "occurred_at" {
		return err
	}
	event.OccurredAt = "state-file-legacy-event-time"
	return orquestacoreworkflow.ValidateOrchestrationEventV0(event)
}

func validateLoadedRunEventsV0(
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) error {
	seen := make(map[string]string, len(events))
	for _, event := range events {
		if normalizeRefV0(event.RunID) != runRef {
			return storeErrorV0("events.ref", "event_ref_inconsistent")
		}
		if err := validateStateFileWorkflowEventV0(event); err != nil {
			return err
		}
		eventID := normalizeRefV0(event.EventID)
		if eventID == "" {
			return storeErrorV0("events.event_id", "event_projection_invalid")
		}
		if previous, ok := seen[eventID]; ok {
			return storeErrorV0("events.duplicate", fmt.Sprintf("event_duplicate:%s:%s", previous, eventID))
		}
		seen[eventID] = eventRecordRefV0(event, eventPayloadHashV0(compactRawMessageV0(event.Payload)))
	}
	if err := validateStrictWorkflowHistoryV0(events); err != nil {
		return storeErrorV0("events.sequence", "event_history_invalid")
	}
	return nil
}

func validateStrictWorkflowHistoryV0(events []orquestacoreworkflow.OrchestrationEventV0) error {
	if len(events) == 0 || events[0].EventType != orquestacoreworkflow.OrchestrationEventRunStartedV0 {
		return nil
	}
	return orquestacoreworkflow.ValidateStrictEventSequenceV0(strictValidationEventsV0(events))
}

func strictValidationEventsV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	validated := append([]orquestacoreworkflow.OrchestrationEventV0(nil), events...)
	for index := range validated {
		if strings.TrimSpace(validated[index].OccurredAt) == "" {
			validated[index].OccurredAt = "state-file-legacy-event-time"
		}
	}
	return validated
}
