package orquestadirector

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizePostLeaseActionInputV0(input PostLeaseActionInputV0) PostLeaseActionInputV0 {
	input.CommandMeta.CommandID = strings.TrimSpace(input.CommandMeta.CommandID)
	input.CommandMeta.RunID = strings.TrimSpace(input.CommandMeta.RunID)
	input.CommandMeta.IdempotencyKey = strings.TrimSpace(input.CommandMeta.IdempotencyKey)
	input.CommandMeta.CorrelationID = strings.TrimSpace(input.CommandMeta.CorrelationID)
	input.CommandMeta.RequestedBy = strings.TrimSpace(input.CommandMeta.RequestedBy)
	input.CommandMeta.OccurredAt = strings.TrimSpace(input.CommandMeta.OccurredAt)
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.AgentRequestID = strings.TrimSpace(input.AgentRequestID)
	input.LeaseRef = strings.TrimSpace(input.LeaseRef)
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	input.ObservedAt = strings.TrimSpace(input.ObservedAt)
	input.RecommendedAction = orquestacoreworkflow.AgentLeaseRecommendedActionV0(
		strings.TrimSpace(string(input.RecommendedAction)),
	)
	input.EvidenceRefs = compactPostLeaseEvidenceRefsV0(input.EvidenceRefs)
	input.QuestionID = strings.TrimSpace(input.QuestionID)
	return input
}

func compactPostLeaseEvidenceRefsV0(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		ref := strings.TrimSpace(value)
		if ref == "" || supervisorHasForbiddenDetailV0(ref) {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		result = append(result, ref)
	}
	return result
}

func validatePostLeaseActionInputV0(input PostLeaseActionInputV0) error {
	required := map[string]string{
		"command_meta.command_id":      input.CommandMeta.CommandID,
		"command_meta.run_id":          input.CommandMeta.RunID,
		"command_meta.idempotency_key": input.CommandMeta.IdempotencyKey,
		"command_meta.occurred_at":     input.CommandMeta.OccurredAt,
		"run_ref":                      input.RunRef,
		"agent_request_id":             input.AgentRequestID,
		"lease_ref":                    input.LeaseRef,
		"reason_code":                  input.ReasonCode,
		"observed_at":                  input.ObservedAt,
		"recommended_action":           string(input.RecommendedAction),
	}
	for field, value := range required {
		if value == "" {
			return postLeaseActionErrorV0(input, "campo requerido", field, nil)
		}
	}
	if input.CommandMeta.RunID != input.RunRef {
		return postLeaseActionErrorV0(input, "run_ref no coincide", "run_ref", nil)
	}
	if input.RecommendedAction == orquestacoreworkflow.AgentLeaseActionAskDirectorV0 &&
		input.QuestionID == "" {
		return postLeaseActionErrorV0(input, "question_id requerido", "question_id", nil)
	}
	return validatePostLeaseActionSafeFieldsV0(input)
}

func validatePostLeaseActionSafeFieldsV0(input PostLeaseActionInputV0) error {
	values := map[string]string{
		"command_meta.command_id":      input.CommandMeta.CommandID,
		"command_meta.run_id":          input.CommandMeta.RunID,
		"command_meta.idempotency_key": input.CommandMeta.IdempotencyKey,
		"command_meta.correlation_id":  input.CommandMeta.CorrelationID,
		"command_meta.requested_by":    input.CommandMeta.RequestedBy,
		"run_ref":                      input.RunRef,
		"agent_request_id":             input.AgentRequestID,
		"lease_ref":                    input.LeaseRef,
		"reason_code":                  input.ReasonCode,
		"question_id":                  input.QuestionID,
	}
	for field, value := range values {
		if value != "" && supervisorHasForbiddenDetailV0(value) {
			return postLeaseActionErrorV0(input, "detalle prohibido", field, nil)
		}
	}
	return nil
}

func postLeaseActionErrorV0(
	input PostLeaseActionInputV0,
	message string,
	field string,
	evidence []string,
) PostLeaseActionErrorV0 {
	return PostLeaseActionErrorV0{
		Code:          ErrDirectorPostLeaseActionInvalidaV0,
		Message:       message,
		Field:         field,
		Retryable:     false,
		Evidence:      compactSupervisorStringsV0(evidence),
		CorrelationID: strings.TrimSpace(input.CommandMeta.CorrelationID),
	}
}
