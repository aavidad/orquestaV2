package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenDeliveryRegisterFragmentsV0 = []string{
	"db",
	"database",
	"sql",
	"dsn",
	"runtime",
	"provider",
	"proveedor",
	"model",
	"modelo",
	"home",
	"oauth",
	"codex",
	"claude",
	"ollama",
	"vllm",
	"adapter",
	"adaptador",
	"filesystem",
	"git",
	"docker",
	"tmux",
	"secret",
	"secreto",
	"token",
	"password",
	"credential",
	"credencial",
	"api_key",
}

func validateRegisterDeliveryCommandPayloadDataV0(payload RegisterDeliveryCommandPayloadV0) error {
	if err := validateRegisterDeliveryRequiredV0(payload); err != nil {
		return err
	}
	if deliveryRegisterStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if deliveryRegisterHasLongStringV0(registerDeliveryTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if deliveryRegisterHasForbiddenDetailsV0(registerDeliveryTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRegisterDeliveryPayloadSizeV0(payload)
}

func validateDeliveryRegisteredPayloadDataV0(payload DeliveryRegisteredPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"delivery_ref": payload.DeliveryRef,
		"phase_id":     payload.PhaseID,
		"task_id":      payload.TaskID,
		"agent_ref":    payload.AgentRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateDeliveryRegisteredPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if deliveryRegisterStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if deliveryRegisterHasLongStringV0(deliveryRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if deliveryRegisterHasForbiddenDetailsV0(deliveryRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateDeliveryRegisteredPayloadSizeV0(payload)
}

func validateRegisterDeliveryRequiredV0(payload RegisterDeliveryCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"delivery_ref": payload.DeliveryRef,
		"phase_id":     payload.PhaseID,
		"task_id":      payload.TaskID,
		"agent_ref":    payload.AgentRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	return validateRegisterDeliveryCommandPhaseIDV0(payload.PhaseID)
}

func validateRegisterDeliveryCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	return ValidateOrchestrationPhaseIDV0(phase)
}

func validateDeliveryRegisteredPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	return ValidateOrchestrationPhaseIDV0(phase)
}

func validateRegisterDeliveryPayloadSizeV0(payload RegisterDeliveryCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxDeliveryRegisterPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateDeliveryRegisteredPayloadSizeV0(payload DeliveryRegisteredPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxDeliveryRegisterPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func deliveryRegisterStringsInvalidV0(values []string) bool {
	if len(values) > maxDeliveryRegisterEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxDeliveryRegisterStringV0 {
			return true
		}
	}
	return false
}

func deliveryRegisterHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxDeliveryRegisterStringV0 {
			return true
		}
	}
	return false
}

func deliveryRegisterHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenDeliveryRegisterFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func registerDeliveryTextFieldsV0(payload RegisterDeliveryCommandPayloadV0) []string {
	values := []string{payload.DeliveryRef, payload.PhaseID, payload.TaskID, payload.AgentRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func deliveryRegisteredTextFieldsV0(payload DeliveryRegisteredPayloadV0) []string {
	values := []string{payload.DeliveryRef, payload.PhaseID, payload.TaskID, payload.AgentRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
