package orquestacorereplanner

import (
	"encoding/json"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func ValidateAgentReworkSignalV0(signal AgentReworkSignalV0) error {
	if err := validateAgentReworkSignalRequiredFieldsV0(signal); err != nil {
		return err
	}
	if err := ValidateAgentReworkVerdictV0(signal.AssessmentStatus); err != nil {
		return err
	}
	if err := ValidateAgentReworkActionV0(signal.AssessmentStatus, signal.AssessmentAction, signal.RequestedAction); err != nil {
		return err
	}
	if err := validateAgentReworkSignalActionFieldsV0(signal); err != nil {
		return err
	}
	if err := validateAgentReworkSignalCollectionsV0(signal); err != nil {
		return err
	}
	if replanProposalHasLongStringV0(agentReworkSignalTextFieldsV0(signal)) {
		return agentReworkSignalErrorV0(ErrAgentReworkSignalPayloadInvalidoV0, "payload")
	}
	if replanProposalHasForbiddenDetailsV0(agentReworkSignalTextFieldsV0(signal)) {
		return agentReworkSignalErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAgentReworkSignalCompactPayloadV0(signal)
}

func ValidateAgentReworkVerdictV0(verdict string) error {
	switch strings.TrimSpace(verdict) {
	case orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
		orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0,
		orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0,
		orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0:
		return nil
	default:
		return agentReworkSignalErrorV0(ErrAgentReworkVerdictNoSoportadoV0, "assessment_status")
	}
}

func ValidateAgentReworkActionV0(verdict string, assessmentAction string, requestedAction ReplanRecommendedActionV0) error {
	verdict = strings.TrimSpace(verdict)
	assessmentAction = strings.TrimSpace(assessmentAction)
	requestedAction = ReplanRecommendedActionV0(strings.TrimSpace(string(requestedAction)))

	if !agentReworkAssessmentActionMatchesVerdictV0(verdict, assessmentAction) {
		return agentReworkSignalErrorV0(ErrAgentReworkActionNoSoportadaV0, "assessment_action")
	}
	if !agentReworkRequestedActionMatchesAssessmentActionV0(assessmentAction, requestedAction) {
		return agentReworkSignalErrorV0(ErrAgentReworkActionNoSoportadaV0, "requested_action")
	}
	return nil
}

func agentReworkAssessmentActionMatchesVerdictV0(verdict string, assessmentAction string) bool {
	if assessmentAction == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 {
		return true
	}
	if verdict == orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0 {
		return assessmentAction == orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0
	}
	if verdict == orquestacoreworkflow.AgentAssessmentVerdictGarbageV0 ||
		verdict == orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0 ||
		verdict == orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0 ||
		verdict == orquestacoreworkflow.AgentAssessmentVerdictTimeoutV0 {
		return assessmentAction == orquestacoreworkflow.AgentAssessmentActionStopAgentV0
	}
	return false
}

func agentReworkRequestedActionMatchesAssessmentActionV0(assessmentAction string, requestedAction ReplanRecommendedActionV0) bool {
	if assessmentAction == orquestacoreworkflow.AgentAssessmentActionAskDirectorV0 {
		return requestedAction == ReplanActionAskDirectorV0 ||
			requestedAction == ReplanActionRetryTaskV0 ||
			requestedAction == ReplanActionReplaceAgentV0
	}
	if assessmentAction == orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0 {
		return requestedAction == ReplanActionRetryTaskV0 ||
			requestedAction == ReplanActionAskDirectorV0
	}
	if assessmentAction == orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
		return requestedAction == ReplanActionReplaceAgentV0 ||
			requestedAction == ReplanActionAskDirectorV0 ||
			requestedAction == ReplanActionAbortTaskV0
	}
	return false
}

func validateAgentReworkSignalRequiredFieldsV0(signal AgentReworkSignalV0) error {
	fields := map[string]string{
		"signal_ref":        signal.SignalRef,
		"run_ref":           signal.RunRef,
		"task_ref":          signal.TaskRef,
		"agent_request_id":  signal.AgentRequestID,
		"source_ref":        signal.SourceRef,
		"assessment_status": signal.AssessmentStatus,
		"assessment_action": signal.AssessmentAction,
		"requested_action":  string(signal.RequestedAction),
		"summary":           signal.Summary,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return agentReworkSignalErrorV0(ErrAgentReworkSignalInvalidoV0, field)
		}
	}
	return nil
}

func validateAgentReworkSignalActionFieldsV0(signal AgentReworkSignalV0) error {
	if signal.RequestedAction == ReplanActionReplaceAgentV0 && strings.TrimSpace(signal.ReplacementRole) == "" {
		return agentReworkSignalErrorV0(ErrAgentReworkSignalInvalidoV0, "replacement_role")
	}
	return nil
}

func validateAgentReworkSignalCollectionsV0(signal AgentReworkSignalV0) error {
	if len(signal.EvidenceRefs) > maxReplanProposalEvidenceRefsV0 {
		return agentReworkSignalErrorV0(ErrAgentReworkSignalInvalidoV0, "evidence_refs")
	}
	for _, ref := range signal.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return agentReworkSignalErrorV0(ErrAgentReworkSignalInvalidoV0, "evidence_refs")
		}
	}
	return nil
}

func validateAgentReworkSignalCompactPayloadV0(signal AgentReworkSignalV0) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return agentReworkSignalErrorV0(ErrAgentReworkSignalPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanProposalPayloadBytesV0 {
		return agentReworkSignalErrorV0(ErrAgentReworkSignalPayloadInvalidoV0, "payload")
	}
	return nil
}

func agentReworkSignalTextFieldsV0(signal AgentReworkSignalV0) []string {
	values := []string{
		signal.SignalRef,
		signal.RunRef,
		signal.TaskRef,
		signal.AgentRequestID,
		signal.SourceRef,
		signal.AssessmentStatus,
		signal.AssessmentAction,
		string(signal.RequestedAction),
		signal.ReplacementRole,
		signal.ReasonRef,
		signal.Summary,
	}
	return append(values, signal.EvidenceRefs...)
}

func agentReworkSignalErrorV0(code string, field string) AgentReworkSignalErrorV0 {
	return AgentReworkSignalErrorV0{Code: code, Field: field}
}
