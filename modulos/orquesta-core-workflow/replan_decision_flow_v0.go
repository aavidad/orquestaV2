package orquestacoreworkflow

import "strings"

type replanDecisionSourceKindV0 string

const (
	replanDecisionSourceNoneV0        replanDecisionSourceKindV0 = ""
	replanDecisionSourceReworkV0      replanDecisionSourceKindV0 = "rework"
	replanDecisionSourceAgentV0       replanDecisionSourceKindV0 = "agent_failed"
	replanDecisionSourceAgentLostV0   replanDecisionSourceKindV0 = "agent_lost"
	replanDecisionSourceAssessmentV0  replanDecisionSourceKindV0 = "agent_assessment"
	replanDecisionSourceQualityGateV0 replanDecisionSourceKindV0 = "quality_gate_blocked"
)

func ensureRecordReplanDecisionCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RecordReplanDecisionCommandPayloadV0) error {
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return err
	}
	if !replanDecisionRunMatchesV0(current, payload.RunRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.run_ref")
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.task_ref")
	}
	sourceKind := replanDecisionSourceKindForRefV0(current, payload.SourceRef)
	if sourceKind == replanDecisionSourceNoneV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.source_ref")
	}
	if !replanDecisionPhaseAllowedForSourceV0(current, sourceKind) {
		return commandErrorV0(ErrTransicionInvalidaV0, "phase.status")
	}
	return nil
}

func ensureReplanDecisionRecordedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload ReplanDecisionRecordedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if !replanDecisionRunMatchesV0(current, payload.RunRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.run_ref")
	}
	if !microtaskAlreadyReflectedV0(current, payload.TaskRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.task_ref")
	}
	sourceKind := replanDecisionSourceKindForRefV0(current, payload.SourceRef)
	if sourceKind == replanDecisionSourceNoneV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.source_ref")
	}
	if !replanDecisionPhaseAllowedForSourceV0(current, sourceKind) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}
	return nil
}

func replanDecisionRunMatchesV0(current OrchestrationRunV0, runRef string) bool {
	return strings.TrimSpace(current.RunID) == strings.TrimSpace(runRef)
}

func replanDecisionSourceKindForRefV0(current OrchestrationRunV0, sourceRef string) replanDecisionSourceKindV0 {
	if reworkRequestAlreadyReflectedV0(current, sourceRef) {
		return replanDecisionSourceReworkV0
	}
	if agentFailedAlreadyReflectedV0(current, sourceRef) {
		return replanDecisionSourceAgentV0
	}
	if agentLostAlreadyReflectedV0(current, sourceRef) {
		return replanDecisionSourceAgentLostV0
	}
	if agentAssessmentAlreadyReflectedV0(current, sourceRef) {
		return replanDecisionSourceAssessmentV0
	}
	if qualityGateBlockedAlreadyReflectedV0(current, sourceRef) {
		return replanDecisionSourceQualityGateV0
	}
	return replanDecisionSourceNoneV0
}

func replanDecisionPhaseAllowedForSourceV0(current OrchestrationRunV0, sourceKind replanDecisionSourceKindV0) bool {
	switch sourceKind {
	case replanDecisionSourceReworkV0:
		return reviewPhaseCurrentV0(current, string(OrchestrationPhaseRevisionV0)) ||
			phaseIsCurrentAndActiveV0(current, OrchestrationPhaseProgramacionV0)
	case replanDecisionSourceAgentV0, replanDecisionSourceAgentLostV0,
		replanDecisionSourceAssessmentV0,
		replanDecisionSourceQualityGateV0:
		return phaseIsCurrentAndActiveV0(current, OrchestrationPhaseProgramacionV0)
	default:
		return false
	}
}
