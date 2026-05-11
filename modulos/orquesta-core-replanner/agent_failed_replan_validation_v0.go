package orquestacorereplanner

import (
	"encoding/json"
	"strings"
)

func ValidateAgentFailedReplanSignalV0(signal AgentFailedReplanSignalV0) error {
	if err := validateAgentFailedReplanSignalRequiredFieldsV0(signal); err != nil {
		return err
	}
	if err := ValidateAgentFailedReplanActionV0(signal.RequestedAction); err != nil {
		return err
	}
	if err := validateAgentFailedReplanSignalActionFieldsV0(signal); err != nil {
		return err
	}
	if err := validateAgentFailedReplanSignalCollectionsV0(signal); err != nil {
		return err
	}
	if replanProposalHasLongStringV0(agentFailedReplanSignalTextFieldsV0(signal)) {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalPayloadInvalidoV0, "payload")
	}
	if replanProposalHasForbiddenDetailsV0(agentFailedReplanSignalTextFieldsV0(signal)) {
		return agentFailedReplanSignalErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAgentFailedReplanSignalCompactPayloadV0(signal)
}

func ValidateAgentFailedReplanActionV0(action ReplanRecommendedActionV0) error {
	switch ReplanRecommendedActionV0(strings.TrimSpace(string(action))) {
	case ReplanActionReplaceAgentV0, ReplanActionAskDirectorV0:
		return nil
	default:
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanActionNoSoportadaV0, "requested_action")
	}
}

func validateAgentFailedReplanSignalRequiredFieldsV0(signal AgentFailedReplanSignalV0) error {
	fields := map[string]string{
		"signal_ref":          signal.SignalRef,
		"run_ref":             signal.RunRef,
		"task_ref":            signal.TaskRef,
		"agent_request_id":    signal.AgentRequestID,
		"source_ref":          signal.SourceRef,
		"failure_reason_code": signal.FailureReasonCode,
		"requested_action":    string(signal.RequestedAction),
		"summary":             signal.Summary,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, field)
		}
	}
	if strings.TrimSpace(signal.SourceRef) != strings.TrimSpace(signal.AgentRequestID) {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, "source_ref")
	}
	return nil
}

func validateAgentFailedReplanSignalActionFieldsV0(signal AgentFailedReplanSignalV0) error {
	if signal.RequestedAction != ReplanActionReplaceAgentV0 {
		return nil
	}
	replacement := strings.TrimSpace(signal.ReplacementRole)
	if replacement == "" {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, "replacement_role")
	}
	if replacement == strings.TrimSpace(signal.AgentRequestID) {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, "replacement_role")
	}
	return nil
}

func validateAgentFailedReplanSignalCollectionsV0(signal AgentFailedReplanSignalV0) error {
	if len(signal.EvidenceRefs) > maxReplanProposalEvidenceRefsV0 {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, "evidence_refs")
	}
	for _, ref := range signal.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalInvalidoV0, "evidence_refs")
		}
	}
	return nil
}

func validateAgentFailedReplanSignalCompactPayloadV0(signal AgentFailedReplanSignalV0) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanProposalPayloadBytesV0 {
		return agentFailedReplanSignalErrorV0(ErrAgentFailedReplanSignalPayloadInvalidoV0, "payload")
	}
	return nil
}

func agentFailedReplanSignalTextFieldsV0(signal AgentFailedReplanSignalV0) []string {
	values := []string{
		signal.SignalRef,
		signal.RunRef,
		signal.TaskRef,
		signal.AgentRequestID,
		signal.SourceRef,
		signal.FailureReasonCode,
		string(signal.RequestedAction),
		signal.ReplacementRole,
		signal.ReasonRef,
		signal.Summary,
	}
	return append(values, signal.EvidenceRefs...)
}

func agentFailedReplanSignalErrorV0(code string, field string) AgentFailedReplanSignalErrorV0 {
	return AgentFailedReplanSignalErrorV0{Code: code, Field: field}
}
