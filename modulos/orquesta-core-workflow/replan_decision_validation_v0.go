package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenReplanDecisionFragmentsV0 = forbiddenReviewReworkFragmentsV0

func validateRecordReplanDecisionPayloadDataV0(payload RecordReplanDecisionCommandPayloadV0) error {
	if err := validateRecordReplanDecisionRequiredV0(payload); err != nil {
		return err
	}
	if err := validateReplanDecisionCommonV0(payload, true); err != nil {
		return err
	}
	return validateRecordReplanDecisionPayloadSizeV0(payload)
}

func validateReplanDecisionRecordedPayloadDataV0(payload ReplanDecisionRecordedPayloadV0) error {
	commandPayload := RecordReplanDecisionCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"replan_ref":      commandPayload.ReplanRef,
		"run_ref":         commandPayload.RunRef,
		"task_ref":        commandPayload.TaskRef,
		"source_ref":      commandPayload.SourceRef,
		"accepted_action": string(commandPayload.AcceptedAction),
		"summary":         commandPayload.Summary,
	}); err != nil {
		return err
	}
	if err := validateReplanDecisionCommonV0(commandPayload, false); err != nil {
		return err
	}
	return validateReplanDecisionRecordedPayloadSizeV0(payload)
}

func validateRecordReplanDecisionRequiredV0(payload RecordReplanDecisionCommandPayloadV0) error {
	return requireCommandPayloadFieldsV0(map[string]string{
		"replan_ref":      payload.ReplanRef,
		"run_ref":         payload.RunRef,
		"task_ref":        payload.TaskRef,
		"source_ref":      payload.SourceRef,
		"accepted_action": string(payload.AcceptedAction),
		"summary":         payload.Summary,
	})
}

func validateReplanDecisionCommonV0(payload RecordReplanDecisionCommandPayloadV0, command bool) error {
	if !isSupportedReplanDecisionActionV0(payload.AcceptedAction) {
		return replanDecisionPayloadErrorV0(command, "payload.accepted_action")
	}
	if len(payload.FollowupRefs) == 0 || len(payload.FollowupRefs) > maxReplanDecisionRefsV0 {
		return replanDecisionPayloadErrorV0(command, "payload.followup_refs")
	}
	if replanDecisionProjectionFieldUnsafeV0(payload) {
		return replanDecisionPayloadErrorV0(command, "payload")
	}
	if replanDecisionStringsInvalidV0(payload.FollowupRefs) {
		return replanDecisionPayloadErrorV0(command, "payload.followup_refs")
	}
	if replanDecisionStringsInvalidV0(payload.EvidenceRefs) {
		return replanDecisionPayloadErrorV0(command, "payload.evidence_refs")
	}
	if replanDecisionHasLongStringV0(replanDecisionTextFieldsV0(payload)) {
		return replanDecisionPayloadErrorV0(command, "payload")
	}
	if replanDecisionHasForbiddenDetailsV0(replanDecisionTextFieldsV0(payload)) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func replanDecisionPayloadErrorV0(command bool, field string) error {
	if command {
		return commandErrorV0(ErrPayloadInvalidoV0, field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, field)
}

func isSupportedReplanDecisionActionV0(action ReplanDecisionActionV0) bool {
	switch ReplanDecisionActionV0(strings.TrimSpace(string(action))) {
	case ReplanDecisionActionSplitTaskV0,
		ReplanDecisionActionRetryTaskV0,
		ReplanDecisionActionReplaceAgentV0,
		ReplanDecisionActionEscalateCapacityV0,
		ReplanDecisionActionAskDirectorV0,
		ReplanDecisionActionAbortTaskV0:
		return true
	default:
		return false
	}
}

func replanDecisionStringsInvalidV0(values []string) bool {
	if len(values) > maxReplanDecisionRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxReplanDecisionStringV0 {
			return true
		}
	}
	return false
}

func replanDecisionHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReplanDecisionStringV0 {
			return true
		}
	}
	return false
}

func replanDecisionHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenReplanDecisionFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func validateRecordReplanDecisionPayloadSizeV0(payload RecordReplanDecisionCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanDecisionPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateReplanDecisionRecordedPayloadSizeV0(payload ReplanDecisionRecordedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReplanDecisionPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func replanDecisionTextFieldsV0(payload RecordReplanDecisionCommandPayloadV0) []string {
	values := []string{
		payload.ReplanRef,
		payload.RunRef,
		payload.TaskRef,
		payload.SourceRef,
		string(payload.AcceptedAction),
		payload.Summary,
	}
	values = append(values, payload.FollowupRefs...)
	return append(values, payload.EvidenceRefs...)
}
