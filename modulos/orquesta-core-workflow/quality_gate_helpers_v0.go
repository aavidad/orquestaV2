package orquestacoreworkflow

import "strings"

func qualityGateRecordedPayloadFromCommandV0(payload RecordQualityGateCommandPayloadV0) QualityGateRecordedPayloadV0 {
	return QualityGateRecordedPayloadV0{
		RunRef:       payload.RunRef,
		GateRef:      payload.GateRef,
		PhaseID:      payload.PhaseID,
		SubjectRef:   payload.SubjectRef,
		Decision:     payload.Decision,
		IssueRefs:    cloneStringsV0(payload.IssueRefs),
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRecordQualityGatePayloadV0(payload RecordQualityGateCommandPayloadV0) RecordQualityGateCommandPayloadV0 {
	return RecordQualityGateCommandPayloadV0{
		RunRef:       strings.TrimSpace(payload.RunRef),
		GateRef:      strings.TrimSpace(payload.GateRef),
		PhaseID:      strings.TrimSpace(payload.PhaseID),
		SubjectRef:   strings.TrimSpace(payload.SubjectRef),
		Decision:     QualityGateDecisionV0(strings.TrimSpace(string(payload.Decision))),
		IssueRefs:    compactUniqueStringsV0(payload.IssueRefs),
		Summary:      strings.TrimSpace(payload.Summary),
		EvidenceRefs: compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeQualityGateRecordedPayloadV0(payload QualityGateRecordedPayloadV0) QualityGateRecordedPayloadV0 {
	normalized := normalizeRecordQualityGatePayloadV0(RecordQualityGateCommandPayloadV0(payload))
	return qualityGateRecordedPayloadFromCommandV0(normalized)
}

func ensureRecordQualityGateCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RecordQualityGateCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if strings.TrimSpace(current.RunID) != payload.RunRef {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.run_ref")
	}
	if !phaseIsCurrentAndActiveV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureQualityGateRecordedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload QualityGateRecordedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if strings.TrimSpace(current.RunID) != payload.RunRef {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.run_ref")
	}
	if !phaseIsCurrentAndActiveV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}
