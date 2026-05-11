package orquestaruntime

import (
	"fmt"
	"strings"
)

const (
	AgentStopperTargetPortV0  = "agent_launcher"
	AgentStopperMessageTypeV0 = "StopRuntimeAgent"
)

type AgentStopperInboundErrorCodeV0 string

const (
	AgentStopperPayloadInvalidoV0   AgentStopperInboundErrorCodeV0 = "agent_stopper_payload_invalido"
	AgentStopperTargetInvalidoV0    AgentStopperInboundErrorCodeV0 = "agent_stopper_target_invalido"
	AgentStopperOperacionInvalidaV0 AgentStopperInboundErrorCodeV0 = "agent_stopper_operacion_invalida"
	AgentStopperReferenciaNoOpacaV0 AgentStopperInboundErrorCodeV0 = "referencia_no_opaca"
	AgentStopperSecretoDetectadoV0  AgentStopperInboundErrorCodeV0 = "secreto_detectado"
)

type AgentStopperInboundV0 struct {
	TargetPort     string                     `json:"target_port"`
	MessageType    string                     `json:"message_type"`
	CorrelationID  string                     `json:"correlation_id"`
	IdempotencyKey string                     `json:"idempotency_key"`
	Payload        *StopRuntimeAgentRequestV0 `json:"payload"`
}

type StopRuntimeAgentRequestV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	RunID          string   `json:"run_id"`
	ReasonCode     string   `json:"reason_code"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentStopperInboundErrorV0 struct {
	Code          AgentStopperInboundErrorCodeV0 `json:"code"`
	MessageKey    string                         `json:"message_key"`
	Field         string                         `json:"field,omitempty"`
	Retryable     bool                           `json:"retryable"`
	CorrelationID string                         `json:"correlation_id,omitempty"`
	Evidence      []string                       `json:"evidence,omitempty"`
}

func (e AgentStopperInboundErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func ValidateAgentStopperInboundV0(inbound AgentStopperInboundV0) []AgentStopperInboundErrorV0 {
	v := agentStopperInboundValidatorV0{correlationID: inbound.CorrelationID}

	v.requireConst("target_port", inbound.TargetPort, AgentStopperTargetPortV0, AgentStopperTargetInvalidoV0)
	v.requireConst("message_type", inbound.MessageType, AgentStopperMessageTypeV0, AgentStopperOperacionInvalidaV0)
	v.requireOpaque("correlation_id", inbound.CorrelationID, AgentStopperPayloadInvalidoV0)
	v.requireOpaque("idempotency_key", inbound.IdempotencyKey, AgentStopperPayloadInvalidoV0)
	if inbound.Payload == nil {
		v.add(AgentStopperPayloadInvalidoV0, "payload")
		return v.errors
	}
	v.validatePayload(*inbound.Payload)

	return v.errors
}

func ValidateStopRuntimeAgentRequestV0(req StopRuntimeAgentRequestV0) []AgentStopperInboundErrorV0 {
	v := agentStopperInboundValidatorV0{}
	v.validatePayload(req)
	return v.errors
}

func (inbound AgentStopperInboundV0) Validate() []AgentStopperInboundErrorV0 {
	return ValidateAgentStopperInboundV0(inbound)
}

func (inbound AgentStopperInboundV0) Valid() bool {
	return len(ValidateAgentStopperInboundV0(inbound)) == 0
}

func (req StopRuntimeAgentRequestV0) Validate() []AgentStopperInboundErrorV0 {
	return ValidateStopRuntimeAgentRequestV0(req)
}

func (req StopRuntimeAgentRequestV0) Valid() bool {
	return len(ValidateStopRuntimeAgentRequestV0(req)) == 0
}

type agentStopperInboundValidatorV0 struct {
	correlationID string
	errors        []AgentStopperInboundErrorV0
}

func (v *agentStopperInboundValidatorV0) validatePayload(req StopRuntimeAgentRequestV0) {
	v.requireOpaque("payload.agent_request_id", req.AgentRequestID, AgentStopperPayloadInvalidoV0)
	v.requireOpaque("payload.run_id", req.RunID, AgentStopperPayloadInvalidoV0)
	v.requireReasonCode("payload.reason_code", req.ReasonCode)
	v.requireSummary("payload.summary", req.Summary)
	for i, ref := range req.EvidenceRefs {
		v.requireOpaque(fmt.Sprintf("payload.evidence_refs[%d]", i), ref, AgentStopperReferenciaNoOpacaV0)
	}
}

func (v *agentStopperInboundValidatorV0) requireConst(field, got, want string, code AgentStopperInboundErrorCodeV0) {
	if got != want {
		v.add(code, field)
	}
}

func (v *agentStopperInboundValidatorV0) requireOpaque(field, value string, missingCode AgentStopperInboundErrorCodeV0) {
	if value == "" {
		v.add(missingCode, field)
		return
	}
	if looksLikeSecret(value) {
		v.add(AgentStopperSecretoDetectadoV0, field)
		return
	}
	if !opaqueRefPatternV0.MatchString(value) {
		v.add(AgentStopperReferenciaNoOpacaV0, field)
	}
}

func (v *agentStopperInboundValidatorV0) requireReasonCode(field, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		v.add(AgentStopperPayloadInvalidoV0, field)
		return
	}
	if looksLikeSecret(trimmed) {
		v.add(AgentStopperSecretoDetectadoV0, field)
		return
	}
	if !isCompactAgentLauncherTokenV0(trimmed) {
		v.add(AgentStopperPayloadInvalidoV0, field)
	}
}

func (v *agentStopperInboundValidatorV0) requireSummary(field, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 1000 {
		v.add(AgentStopperPayloadInvalidoV0, field)
		return
	}
	if looksLikeSecret(trimmed) {
		v.add(AgentStopperSecretoDetectadoV0, field)
	}
}

func (v *agentStopperInboundValidatorV0) add(code AgentStopperInboundErrorCodeV0, field string) {
	v.errors = append(v.errors, AgentStopperInboundErrorV0{
		Code:          code,
		MessageKey:    "orquesta.runtime.agent_stopper." + string(code),
		Field:         field,
		Retryable:     false,
		CorrelationID: v.correlationID,
	})
}
