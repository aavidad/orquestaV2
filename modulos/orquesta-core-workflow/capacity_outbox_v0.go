package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

type CapacityDecisionRequestV0 struct {
	CapacityRequestID          string                                `json:"capacity_request_id"`
	RunID                      string                                `json:"run_id"`
	PhaseID                    string                                `json:"phase_id"`
	TaskRef                    string                                `json:"task_ref,omitempty"`
	ReasonCode                 string                                `json:"reason_code"`
	Summary                    string                                `json:"summary"`
	MinimumRecommendedCapacity OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                              `json:"evidence_refs,omitempty"`
}

func newRequestCapacityDecisionOutboxV0(command OrchestrationCommandV0, event OrchestrationEventV0, payload RequestCapacityCommandPayloadV0) (OutboxMessageV0, error) {
	outboxPayload := CapacityDecisionRequestV0{
		CapacityRequestID:          payload.CapacityRequestID,
		RunID:                      strings.TrimSpace(command.RunID),
		PhaseID:                    payload.PhaseID,
		TaskRef:                    payload.TaskRef,
		ReasonCode:                 payload.ReasonCode,
		Summary:                    payload.Summary,
		MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
		EvidenceRefs:               cloneStringsV0(payload.EvidenceRefs),
	}
	payloadJSON, err := json.Marshal(outboxPayload)
	if err != nil {
		return OutboxMessageV0{}, outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	message := OutboxMessageV0{
		MessageID:        "outbox-requestcapacitydecision-" + strings.TrimSpace(command.IdempotencyKey),
		MessageType:      OutboxMessageRequestCapacityDecisionV0,
		RunID:            strings.TrimSpace(command.RunID),
		IdempotencyKey:   strings.TrimSpace(command.IdempotencyKey),
		CorrelationID:    strings.TrimSpace(command.CorrelationID),
		CausationEventID: strings.TrimSpace(event.EventID),
		TargetPort:       OutboxTargetCapacityV0,
		PayloadVersion:   OutboxPayloadVersionV0,
		Payload:          payloadJSON,
	}
	if err := ValidateOutboxMessageV0(message); err != nil {
		return OutboxMessageV0{}, err
	}
	return message, nil
}

func validateCapacityDecisionRequestPayloadV0(runID string, raw json.RawMessage) error {
	var payload CapacityDecisionRequestV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	payload = normalizeCapacityDecisionRequestPayloadV0(payload)
	if payload.RunID != strings.TrimSpace(runID) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	return validateCapacityDecisionRequestPayloadDataV0(payload)
}

func normalizeCapacityDecisionRequestPayloadV0(payload CapacityDecisionRequestV0) CapacityDecisionRequestV0 {
	return CapacityDecisionRequestV0{
		CapacityRequestID:          strings.TrimSpace(payload.CapacityRequestID),
		RunID:                      strings.TrimSpace(payload.RunID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TaskRef:                    strings.TrimSpace(payload.TaskRef),
		ReasonCode:                 strings.TrimSpace(payload.ReasonCode),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.MinimumRecommendedCapacity))),
		EvidenceRefs:               compactStringsV0(payload.EvidenceRefs),
	}
}

func validateCapacityDecisionRequestPayloadDataV0(payload CapacityDecisionRequestV0) error {
	if strings.TrimSpace(payload.CapacityRequestID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.capacity_request_id")
	}
	if strings.TrimSpace(payload.RunID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	if strings.TrimSpace(payload.PhaseID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.phase_id")
	}
	if strings.TrimSpace(payload.ReasonCode) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.reason_code")
	}
	if strings.TrimSpace(payload.Summary) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.summary")
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.phase_id")
	}
	if strings.TrimSpace(string(payload.MinimumRecommendedCapacity)) != "" && !isSupportedCapacityRecommendationV0(payload.MinimumRecommendedCapacity) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	if len(payload.EvidenceRefs) > maxCapacityRequestEvidenceRefsV0 || capacityRequestStringsInvalidV0(payload.EvidenceRefs) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.evidence_refs")
	}
	if capacityRequestHasForbiddenDetailsV0(capacityDecisionRequestTextFieldsV0(payload)) {
		return outboxErrorV0(ErrOutboxDetalleProhibidoV0, "payload")
	}
	return validateCapacityDecisionRequestPayloadSizeV0(payload)
}

func validateCapacityDecisionRequestPayloadSizeV0(payload CapacityDecisionRequestV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCapacityRequestPayloadBytesV0 {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	return nil
}

func capacityDecisionRequestTextFieldsV0(payload CapacityDecisionRequestV0) []string {
	values := []string{payload.CapacityRequestID, payload.RunID, payload.PhaseID, payload.TaskRef, payload.ReasonCode, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}
