package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

type LaunchRuntimeAgentRequestV0 struct {
	AgentRequestID     string   `json:"agent_request_id"`
	RunID              string   `json:"run_id"`
	PhaseID            string   `json:"phase_id"`
	TaskRef            string   `json:"task_ref,omitempty"`
	CapacityRequestRef string   `json:"capacity_request_ref"`
	Role               string   `json:"role"`
	Summary            string   `json:"summary"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	SkillRefs          []string `json:"skill_refs,omitempty"`
}

func newLaunchRuntimeAgentOutboxV0(command OrchestrationCommandV0, event OrchestrationEventV0, payload RequestAgentCommandPayloadV0) (OutboxMessageV0, error) {
	outboxPayload := LaunchRuntimeAgentRequestV0{
		AgentRequestID:     payload.AgentRequestID,
		RunID:              strings.TrimSpace(command.RunID),
		PhaseID:            payload.PhaseID,
		TaskRef:            payload.TaskRef,
		CapacityRequestRef: payload.CapacityRequestRef,
		Role:               payload.Role,
		Summary:            payload.Summary,
		EvidenceRefs:       cloneStringsV0(payload.EvidenceRefs),
		SkillRefs:          cloneStringsV0(payload.SkillRefs),
	}
	payloadJSON, err := json.Marshal(outboxPayload)
	if err != nil {
		return OutboxMessageV0{}, outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	message := OutboxMessageV0{
		MessageID:        "outbox-launchruntimeagent-" + strings.TrimSpace(command.IdempotencyKey),
		MessageType:      OutboxMessageLaunchRuntimeAgentV0,
		RunID:            strings.TrimSpace(command.RunID),
		IdempotencyKey:   strings.TrimSpace(command.IdempotencyKey),
		CorrelationID:    strings.TrimSpace(command.CorrelationID),
		CausationEventID: strings.TrimSpace(event.EventID),
		TargetPort:       OutboxTargetAgentLauncherV0,
		PayloadVersion:   OutboxPayloadVersionV0,
		Payload:          payloadJSON,
	}
	if err := ValidateOutboxMessageV0(message); err != nil {
		return OutboxMessageV0{}, err
	}
	return message, nil
}

func validateLaunchRuntimeAgentPayloadV0(runID string, raw json.RawMessage) error {
	var payload LaunchRuntimeAgentRequestV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	payload = normalizeLaunchRuntimeAgentPayloadV0(payload)
	if payload.RunID != strings.TrimSpace(runID) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	return validateLaunchRuntimeAgentPayloadDataV0(payload)
}

func normalizeLaunchRuntimeAgentPayloadV0(payload LaunchRuntimeAgentRequestV0) LaunchRuntimeAgentRequestV0 {
	return LaunchRuntimeAgentRequestV0{
		AgentRequestID:     strings.TrimSpace(payload.AgentRequestID),
		RunID:              strings.TrimSpace(payload.RunID),
		PhaseID:            strings.TrimSpace(payload.PhaseID),
		TaskRef:            strings.TrimSpace(payload.TaskRef),
		CapacityRequestRef: strings.TrimSpace(payload.CapacityRequestRef),
		Role:               strings.TrimSpace(payload.Role),
		Summary:            strings.TrimSpace(payload.Summary),
		EvidenceRefs:       compactStringsV0(payload.EvidenceRefs),
		SkillRefs:          compactUniqueStringsV0(payload.SkillRefs),
	}
}

func validateLaunchRuntimeAgentPayloadDataV0(payload LaunchRuntimeAgentRequestV0) error {
	if strings.TrimSpace(payload.AgentRequestID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.agent_request_id")
	}
	if strings.TrimSpace(payload.RunID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	if strings.TrimSpace(payload.PhaseID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.phase_id")
	}
	if strings.TrimSpace(payload.CapacityRequestRef) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.capacity_request_ref")
	}
	if strings.TrimSpace(payload.Role) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.role")
	}
	if strings.TrimSpace(payload.Summary) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.summary")
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.phase_id")
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentSkillRefsInvalidV0(payload.SkillRefs) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.skill_refs")
	}
	if agentRequestHasForbiddenDetailsV0(launchRuntimeAgentTextFieldsV0(payload)) {
		return outboxErrorV0(ErrOutboxDetalleProhibidoV0, "payload")
	}
	return validateLaunchRuntimeAgentPayloadSizeV0(payload)
}

func validateLaunchRuntimeAgentPayloadSizeV0(payload LaunchRuntimeAgentRequestV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	return nil
}

func launchRuntimeAgentTextFieldsV0(payload LaunchRuntimeAgentRequestV0) []string {
	values := []string{payload.AgentRequestID, payload.RunID, payload.PhaseID, payload.TaskRef, payload.CapacityRequestRef, payload.Role, payload.Summary}
	values = append(values, payload.EvidenceRefs...)
	return append(values, payload.SkillRefs...)
}
