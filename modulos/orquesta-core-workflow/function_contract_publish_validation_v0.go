package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenFunctionContractPublishFragmentsV0 = []string{
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

func validatePublishFunctionContractCommandPayloadDataV0(payload PublishFunctionContractCommandPayloadV0) error {
	if err := validatePublishFunctionContractRequiredV0(payload); err != nil {
		return err
	}
	if functionContractPublishStringsInvalidV0(payload.FunctionNames, payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if functionContractPublishHasLongStringV0(publishFunctionContractTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if functionContractPublishHasForbiddenDetailsV0(publishFunctionContractTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validatePublishFunctionContractPayloadSizeV0(payload)
}

func validateFunctionContractPublishedPayloadDataV0(payload FunctionContractPublishedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"contract_ref": payload.ContractRef,
		"phase_id":     payload.PhaseID,
		"decision_ref": payload.DecisionRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateFunctionContractPublishedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if functionContractPublishStringsInvalidV0(payload.FunctionNames, payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if functionContractPublishHasLongStringV0(functionContractPublishedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if functionContractPublishHasForbiddenDetailsV0(functionContractPublishedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateFunctionContractPublishedPayloadSizeV0(payload)
}

func validatePublishFunctionContractRequiredV0(payload PublishFunctionContractCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"contract_ref": payload.ContractRef,
		"phase_id":     payload.PhaseID,
		"decision_ref": payload.DecisionRef,
		"summary":      payload.Summary,
	}); err != nil {
		return err
	}
	return validatePublishFunctionContractCommandPhaseIDV0(payload.PhaseID)
}

func validatePublishFunctionContractCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhasePlanificacionMicrotareasV0 &&
		phase != OrchestrationPhaseProgramacionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateFunctionContractPublishedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhasePlanificacionMicrotareasV0 &&
		phase != OrchestrationPhaseProgramacionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validatePublishFunctionContractPayloadSizeV0(payload PublishFunctionContractCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxFunctionContractPublishPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateFunctionContractPublishedPayloadSizeV0(payload FunctionContractPublishedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxFunctionContractPublishPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func functionContractPublishStringsInvalidV0(functionNames []string, evidenceRefs []string) bool {
	return compactStringsInvalidV0(functionNames) || compactStringsInvalidV0(evidenceRefs)
}

func compactStringsInvalidV0(values []string) bool {
	if len(values) > maxFunctionContractPublishRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxFunctionContractPublishStringV0 {
			return true
		}
	}
	return false
}

func functionContractPublishHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxFunctionContractPublishStringV0 {
			return true
		}
	}
	return false
}

func functionContractPublishHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenFunctionContractPublishFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func publishFunctionContractTextFieldsV0(payload PublishFunctionContractCommandPayloadV0) []string {
	values := []string{payload.ContractRef, payload.PhaseID, payload.DecisionRef, payload.Summary}
	values = append(values, payload.FunctionNames...)
	return append(values, payload.EvidenceRefs...)
}

func functionContractPublishedTextFieldsV0(payload FunctionContractPublishedPayloadV0) []string {
	values := []string{payload.ContractRef, payload.PhaseID, payload.DecisionRef, payload.Summary}
	values = append(values, payload.FunctionNames...)
	return append(values, payload.EvidenceRefs...)
}
