package orquestacoreworkflow

import "strings"

func runStartedPayloadFromCommandV0(command OrchestrationCommandV0, payload StartRunCommandPayloadV0) RunStartedPayloadV0 {
	return RunStartedPayloadV0{
		ProjectRef:  payload.ProjectRef,
		AppSpecRef:  payload.AppSpecRef,
		RequestedBy: strings.TrimSpace(command.RequestedBy),
	}
}

func phaseOpenedPayloadFromCommandV0(command OrchestrationCommandV0, payload OpenPhaseCommandPayloadV0) PhaseOpenedPayloadV0 {
	return PhaseOpenedPayloadV0{
		PhaseID:  payload.PhaseID,
		OpenedBy: strings.TrimSpace(command.RequestedBy),
		Reason:   payload.Reason,
	}
}

func runBlockedPayloadFromCommandV0(payload BlockRunCommandPayloadV0) RunBlockedPayloadV0 {
	return RunBlockedPayloadV0{
		BlockerID:    payload.BlockerID,
		ReasonCode:   payload.ReasonCode,
		Summary:      payload.Summary,
		SourceGroup:  payload.SourceGroup,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func runBlockerResolvedPayloadFromCommandV0(
	payload ResolveRunBlockerCommandPayloadV0,
) RunBlockerResolvedPayloadV0 {
	return RunBlockerResolvedPayloadV0{
		BlockerID:    payload.BlockerID,
		ReasonCode:   payload.ReasonCode,
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeResolveRunBlockerPayloadV0(
	payload ResolveRunBlockerCommandPayloadV0,
) ResolveRunBlockerCommandPayloadV0 {
	payload.BlockerID = strings.TrimSpace(payload.BlockerID)
	payload.ReasonCode = strings.TrimSpace(payload.ReasonCode)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	return payload
}

func normalizeRunBlockerResolvedPayloadV0(
	payload RunBlockerResolvedPayloadV0,
) RunBlockerResolvedPayloadV0 {
	payload.BlockerID = strings.TrimSpace(payload.BlockerID)
	payload.ReasonCode = strings.TrimSpace(payload.ReasonCode)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactStringsV0(payload.EvidenceRefs)
	return payload
}

func ensureRunBlockerResolvedEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload ResolveRunBlockerCommandPayloadV0,
) error {
	return ensureCommandEffectMatchesV0(
		run,
		command,
		OrchestrationEventRunBlockerResolvedV0,
		payload.BlockerID,
		runBlockerResolvedPayloadFromCommandV0(payload),
	)
}

func runBlockerResolvedEffectKnownV0(
	run OrchestrationRunV0,
	blockerID string,
) (bool, error) {
	_, ok, err := commandEffectForSubjectV0(
		run,
		OrchestrationEventRunBlockerResolvedV0,
		blockerID,
	)
	if err != nil {
		return false, commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	return ok, nil
}

func ensurePhaseOpenedEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload OpenPhaseCommandPayloadV0,
) error {
	return ensureCommandEffectMatchesV0(
		run,
		command,
		OrchestrationEventPhaseOpenedV0,
		phaseOpenedEffectSubjectFromCommandV0(command),
		phaseOpenedPayloadFromCommandV0(command, payload),
	)
}

func ensurePhaseClosedEffectMatchesV0(
	run OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload ClosePhaseCommandPayloadV0,
) error {
	meta := phaseClosedEventMetaV0(run, command, payload)
	record, err := commandEffectRecordFromExpectedEventV0(
		OrchestrationEventPhaseClosedV0,
		payload.ClosureRef,
		meta.IdempotencyKey,
		command.CommandID,
		meta.EventID,
		phaseClosedPayloadFromCommandV0(payload),
	)
	if err != nil {
		return err
	}
	return ensureCommandEffectRecordMatchesV0(
		run,
		OrchestrationEventPhaseClosedV0,
		payload.ClosureRef,
		record,
	)
}

func phaseOpenedEffectKnownV0(run OrchestrationRunV0, command OrchestrationCommandV0) (bool, error) {
	_, ok, err := commandEffectForSubjectV0(run, OrchestrationEventPhaseOpenedV0, phaseOpenedEffectSubjectFromCommandV0(command))
	if err != nil {
		return false, commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	return ok, nil
}

func phaseOpenedEffectSubjectFromCommandV0(command OrchestrationCommandV0) string {
	return commandEventIDV0(command, OrchestrationEventPhaseOpenedV0)
}

func phaseClosureEffectKnownV0(run OrchestrationRunV0, closureRef string) (bool, error) {
	_, ok, err := commandEffectForSubjectV0(run, OrchestrationEventPhaseClosedV0, closureRef)
	if err != nil {
		return false, commandErrorV0(ErrTransicionInvalidaV0, "command_effects")
	}
	return ok, nil
}
