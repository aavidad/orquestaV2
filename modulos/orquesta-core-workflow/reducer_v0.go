package orquestacoreworkflow

func ApplyEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	if err := ValidateOrchestrationEventV0(event); err != nil {
		return current, err
	}
	return applyValidatedEventV0(current, event)
}

func applyValidatedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	applier, ok := lookupEventApplierV0(event.EventType)
	if !ok {
		return current, eventErrorV0(ErrEventoNoSoportadoV0, "event_type")
	}
	return applier(current, event)
}

func ReplayEventsV0(events []OrchestrationEventV0) (OrchestrationRunV0, error) {
	return ReplayDurableEventsV0(events)
}
