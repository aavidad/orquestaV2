package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
	"time"
)

const agentLeaseObservedAtLayoutV0 = "2006-01-02T15:04:05Z"

var forbiddenAgentLeaseExpiredFragmentsV0 = forbiddenReviewReworkFragmentsV0

func validateRegisterAgentLeaseExpiredPayloadDataV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) error {
	if err := validateRegisterAgentLeaseExpiredRequiredV0(payload); err != nil {
		return err
	}
	if err := validateAgentLeaseExpiredCommonV0(payload, true); err != nil {
		return err
	}
	return validateRegisterAgentLeaseExpiredPayloadSizeV0(payload)
}

func validateAgentLeaseExpiredPayloadDataV0(payload AgentLeaseExpiredPayloadV0) error {
	commandPayload := RegisterAgentLeaseExpiredCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"run_ref":            commandPayload.RunRef,
		"agent_request_id":   commandPayload.AgentRequestID,
		"lease_ref":          commandPayload.LeaseRef,
		"reason_code":        commandPayload.ReasonCode,
		"observed_at":        commandPayload.ObservedAt,
		"recommended_action": string(commandPayload.RecommendedAction),
	}); err != nil {
		return err
	}
	if err := validateAgentLeaseExpiredCommonV0(commandPayload, false); err != nil {
		return err
	}
	return validateAgentLeaseExpiredPayloadSizeV0(payload)
}

func validateRegisterAgentLeaseExpiredRequiredV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) error {
	return requireCommandPayloadFieldsV0(map[string]string{
		"run_ref":            payload.RunRef,
		"agent_request_id":   payload.AgentRequestID,
		"lease_ref":          payload.LeaseRef,
		"reason_code":        payload.ReasonCode,
		"observed_at":        payload.ObservedAt,
		"recommended_action": string(payload.RecommendedAction),
	})
}

func validateAgentLeaseExpiredCommonV0(payload RegisterAgentLeaseExpiredCommandPayloadV0, command bool) error {
	if !isSupportedAgentLeaseRecommendedActionV0(payload.RecommendedAction) {
		return agentLeaseExpiredPayloadErrorV0(command, "payload.recommended_action")
	}
	if !validAgentLeaseObservedAtV0(payload.ObservedAt) {
		return agentLeaseExpiredPayloadErrorV0(command, "payload.observed_at")
	}
	if agentLeaseExpiredProjectionFieldUnsafeV0(payload) {
		return agentLeaseExpiredPayloadErrorV0(command, "payload")
	}
	if agentLeaseExpiredStringsInvalidV0(payload.EvidenceRefs) {
		return agentLeaseExpiredPayloadErrorV0(command, "payload.evidence_refs")
	}
	if agentLeaseExpiredHasLongStringV0(agentLeaseExpiredTextFieldsV0(payload)) {
		return agentLeaseExpiredPayloadErrorV0(command, "payload")
	}
	if agentLeaseExpiredHasForbiddenDetailsV0(agentLeaseExpiredTextFieldsV0(payload)) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func agentLeaseExpiredPayloadErrorV0(command bool, field string) error {
	if command {
		return commandErrorV0(ErrPayloadInvalidoV0, field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, field)
}

func isSupportedAgentLeaseRecommendedActionV0(action AgentLeaseRecommendedActionV0) bool {
	switch AgentLeaseRecommendedActionV0(strings.TrimSpace(string(action))) {
	case AgentLeaseActionRetryV0,
		AgentLeaseActionAskDirectorV0,
		AgentLeaseActionStopAgentV0,
		AgentLeaseActionMarkFailedV0,
		AgentLeaseActionMarkStoppedV0,
		AgentLeaseActionReplanTaskV0,
		AgentLeaseActionAlertOnlyV0:
		return true
	default:
		return false
	}
}

func validAgentLeaseObservedAtV0(value string) bool {
	_, err := time.Parse(agentLeaseObservedAtLayoutV0, strings.TrimSpace(value))
	return err == nil
}

func agentLeaseExpiredStringsInvalidV0(values []string) bool {
	if len(values) > maxAgentLeaseExpiredEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxAgentLeaseExpiredStringV0 {
			return true
		}
	}
	return false
}

func agentLeaseExpiredHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxAgentLeaseExpiredStringV0 {
			return true
		}
	}
	return false
}

func agentLeaseExpiredHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenAgentLeaseExpiredFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func validateRegisterAgentLeaseExpiredPayloadSizeV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxAgentLeaseExpiredPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateAgentLeaseExpiredPayloadSizeV0(payload AgentLeaseExpiredPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxAgentLeaseExpiredPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func agentLeaseExpiredTextFieldsV0(payload RegisterAgentLeaseExpiredCommandPayloadV0) []string {
	values := []string{
		payload.RunRef,
		payload.AgentRequestID,
		payload.LeaseRef,
		payload.ReasonCode,
		payload.ObservedAt,
		string(payload.RecommendedAction),
	}
	return append(values, payload.EvidenceRefs...)
}
