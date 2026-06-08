package orquestaruntime

import (
	"fmt"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	AgentLauncherTargetPortV0  = "agent_launcher"
	AgentLauncherMessageTypeV0 = "LaunchRuntimeAgent"
)

type AgentLauncherInboundErrorCodeV0 string

const (
	AgentLauncherPayloadInvalidoV0         AgentLauncherInboundErrorCodeV0 = "agent_launcher_payload_invalido"
	AgentLauncherTargetInvalidoV0          AgentLauncherInboundErrorCodeV0 = "agent_launcher_target_invalido"
	AgentLauncherOperacionInvalidaV0       AgentLauncherInboundErrorCodeV0 = "agent_launcher_operacion_invalida"
	AgentLauncherEnrichmentRequeridaV0     AgentLauncherInboundErrorCodeV0 = "agent_launcher_enrichment_requerida"
	AgentLauncherFunctionRefRequeridaV0    AgentLauncherInboundErrorCodeV0 = "function_contract_ref_requerida"
	AgentLauncherFunctionNoResueltaV0      AgentLauncherInboundErrorCodeV0 = "function_contract_no_resuelta"
	AgentLauncherCapacityRefRequeridaV0    AgentLauncherInboundErrorCodeV0 = "capacity_decision_ref_requerida"
	AgentLauncherCapacityNoResueltaV0      AgentLauncherInboundErrorCodeV0 = "capacity_decision_no_resuelta"
	AgentLauncherRuntimeBindingRefsV0      AgentLauncherInboundErrorCodeV0 = "runtime_binding_refs_requeridas"
	AgentLauncherEvidenceRefsV0            AgentLauncherInboundErrorCodeV0 = "evidence_refs_requeridas"
	AgentLauncherContextBundleNoResueltoV0 AgentLauncherInboundErrorCodeV0 = "context_bundle_no_resuelto"
	AgentLauncherReferenciaNoOpacaV0       AgentLauncherInboundErrorCodeV0 = "referencia_no_opaca"
	AgentLauncherSecretoDetectadoV0        AgentLauncherInboundErrorCodeV0 = "secreto_detectado"
	AgentLauncherRuntimeLaunchInvalidaV0   AgentLauncherInboundErrorCodeV0 = "runtime_launch_request_invalida"
)

type AgentLauncherInboundV0 struct {
	TargetPort     string                       `json:"target_port"`
	MessageType    string                       `json:"message_type"`
	CorrelationID  string                       `json:"correlation_id"`
	IdempotencyKey string                       `json:"idempotency_key"`
	Payload        *LaunchRuntimeAgentRequestV0 `json:"payload"`
}

type LaunchRuntimeAgentRequestV0 struct {
	AgentRequestID     string   `json:"agent_request_id"`
	RunID              string   `json:"run_id"`
	PhaseID            string   `json:"phase_id"`
	TaskRef            string   `json:"task_ref"`
	CapacityRequestRef string   `json:"capacity_request_ref"`
	Role               string   `json:"role"`
	Summary            string   `json:"summary"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	SkillRefs          []string `json:"skill_refs,omitempty"`
}

type AgentLauncherResolvedDependenciesV0 struct {
	FunctionContract *RuntimeFunctionContractV0       `json:"function_contract,omitempty"`
	CapacityDecision *RuntimeCapacityDecisionV0       `json:"capacity_decision,omitempty"`
	RuntimeBinding   *RuntimeBindingV0                `json:"runtime_binding,omitempty"`
	EvidenceRefs     *RuntimeEvidenceRefsV0           `json:"evidence_refs,omitempty"`
	ContextBundle    *orquestacontext.ContextBundleV0 `json:"context_bundle,omitempty"`
}

type FunctionContractResolverV0 interface {
	ResolveFunctionContractV0(taskRef string) (*RuntimeFunctionContractV0, error)
}

type CapacityDecisionResolverV0 interface {
	ResolveCapacityDecisionV0(capacityRequestRef string) (*RuntimeCapacityDecisionV0, error)
}

type RuntimeBindingResolverV0 interface {
	ResolveRuntimeBindingV0(request LaunchRuntimeAgentRequestV0, decision RuntimeCapacityDecisionV0) (*RuntimeBindingV0, error)
}

type LaunchEvidenceResolverV0 interface {
	ResolveLaunchEvidenceV0(inbound AgentLauncherInboundV0) (*RuntimeEvidenceRefsV0, error)
}

type ContextBundleResolverV0 interface {
	ResolveContextBundleV0(inbound AgentLauncherInboundV0) (*orquestacontext.ContextBundleV0, error)
}

type AgentLauncherInboundErrorV0 struct {
	Code          AgentLauncherInboundErrorCodeV0 `json:"code"`
	MessageKey    string                          `json:"message_key"`
	Field         string                          `json:"field,omitempty"`
	Retryable     bool                            `json:"retryable"`
	CorrelationID string                          `json:"correlation_id,omitempty"`
	Evidence      []string                        `json:"evidence,omitempty"`
}

func (e AgentLauncherInboundErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func ValidateAgentLauncherInboundV0(inbound AgentLauncherInboundV0) []AgentLauncherInboundErrorV0 {
	v := agentLauncherInboundValidatorV0{correlationID: inbound.CorrelationID}

	v.requireConst("target_port", inbound.TargetPort, AgentLauncherTargetPortV0, AgentLauncherTargetInvalidoV0)
	v.requireConst("message_type", inbound.MessageType, AgentLauncherMessageTypeV0, AgentLauncherOperacionInvalidaV0)
	v.requireOpaque("correlation_id", inbound.CorrelationID, AgentLauncherPayloadInvalidoV0)
	v.requireOpaque("idempotency_key", inbound.IdempotencyKey, AgentLauncherPayloadInvalidoV0)
	if inbound.Payload == nil {
		v.add(AgentLauncherPayloadInvalidoV0, "payload")
		return v.errors
	}
	v.validatePayload(*inbound.Payload)

	return v.errors
}

func ValidateLaunchRuntimeAgentRequestV0(req LaunchRuntimeAgentRequestV0) []AgentLauncherInboundErrorV0 {
	v := agentLauncherInboundValidatorV0{}
	v.validatePayload(req)
	return v.errors
}

func ValidateAgentLauncherResolvedDependenciesV0(resolved AgentLauncherResolvedDependenciesV0) []AgentLauncherInboundErrorV0 {
	v := agentLauncherInboundValidatorV0{}
	if resolved.FunctionContract == nil {
		v.add(AgentLauncherFunctionNoResueltaV0, "function_contract")
	}
	if resolved.CapacityDecision == nil {
		v.add(AgentLauncherCapacityNoResueltaV0, "capacity_decision")
	}
	if resolved.RuntimeBinding == nil {
		v.add(AgentLauncherRuntimeBindingRefsV0, "runtime_binding")
	}
	if resolved.EvidenceRefs == nil {
		v.add(AgentLauncherEvidenceRefsV0, "evidence_refs")
	}
	if resolved.ContextBundle == nil || !resolved.ContextBundle.Valid() {
		v.add(AgentLauncherContextBundleNoResueltoV0, "context_bundle")
	}
	return v.errors
}

func (inbound AgentLauncherInboundV0) Validate() []AgentLauncherInboundErrorV0 {
	return ValidateAgentLauncherInboundV0(inbound)
}

func (inbound AgentLauncherInboundV0) Valid() bool {
	return len(ValidateAgentLauncherInboundV0(inbound)) == 0
}

func (req LaunchRuntimeAgentRequestV0) Validate() []AgentLauncherInboundErrorV0 {
	return ValidateLaunchRuntimeAgentRequestV0(req)
}

func (req LaunchRuntimeAgentRequestV0) Valid() bool {
	return len(ValidateLaunchRuntimeAgentRequestV0(req)) == 0
}

type agentLauncherInboundValidatorV0 struct {
	correlationID string
	errors        []AgentLauncherInboundErrorV0
}

func (v *agentLauncherInboundValidatorV0) validatePayload(req LaunchRuntimeAgentRequestV0) {
	v.requireOpaque("payload.agent_request_id", req.AgentRequestID, AgentLauncherPayloadInvalidoV0)
	v.requireOpaque("payload.run_id", req.RunID, AgentLauncherPayloadInvalidoV0)
	v.requireOpaque("payload.phase_id", req.PhaseID, AgentLauncherPayloadInvalidoV0)
	v.requireOpaque("payload.task_ref", req.TaskRef, AgentLauncherFunctionRefRequeridaV0)
	v.requireOpaque("payload.capacity_request_ref", req.CapacityRequestRef, AgentLauncherCapacityRefRequeridaV0)
	v.requireRole("payload.role", req.Role)
	v.requireSummary("payload.summary", req.Summary)
	for i, ref := range req.EvidenceRefs {
		v.requireOpaque(fmt.Sprintf("payload.evidence_refs[%d]", i), ref, AgentLauncherReferenciaNoOpacaV0)
	}
	for i, ref := range req.SkillRefs {
		v.requireOpaque(fmt.Sprintf("payload.skill_refs[%d]", i), ref, AgentLauncherReferenciaNoOpacaV0)
	}
}

func (v *agentLauncherInboundValidatorV0) requireConst(field, got, want string, code AgentLauncherInboundErrorCodeV0) {
	if got != want {
		v.add(code, field)
	}
}

func (v *agentLauncherInboundValidatorV0) requireOpaque(field, value string, missingCode AgentLauncherInboundErrorCodeV0) {
	if value == "" {
		v.add(missingCode, field)
		return
	}
	if looksLikeSecret(value) {
		v.add(AgentLauncherSecretoDetectadoV0, field)
		return
	}
	if !opaqueRefPatternV0.MatchString(value) {
		v.add(AgentLauncherReferenciaNoOpacaV0, field)
	}
}

func (v *agentLauncherInboundValidatorV0) requireRole(field, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		v.add(AgentLauncherPayloadInvalidoV0, field)
		return
	}
	if looksLikeSecret(trimmed) || !isCompactAgentLauncherTokenV0(trimmed) {
		v.add(AgentLauncherPayloadInvalidoV0, field)
	}
}

func (v *agentLauncherInboundValidatorV0) requireSummary(field, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 1000 {
		v.add(AgentLauncherPayloadInvalidoV0, field)
		return
	}
	if looksLikeSecret(trimmed) {
		v.add(AgentLauncherSecretoDetectadoV0, field)
	}
}

func (v *agentLauncherInboundValidatorV0) add(code AgentLauncherInboundErrorCodeV0, field string) {
	v.errors = append(v.errors, AgentLauncherInboundErrorV0{
		Code:          code,
		MessageKey:    "orquesta.runtime.agent_launcher." + string(code),
		Field:         field,
		Retryable:     isRetryableAgentLauncherErrorV0(code),
		CorrelationID: v.correlationID,
	})
}

func isRetryableAgentLauncherErrorV0(code AgentLauncherInboundErrorCodeV0) bool {
	return code == AgentLauncherEnrichmentRequeridaV0 ||
		code == AgentLauncherFunctionNoResueltaV0 ||
		code == AgentLauncherCapacityNoResueltaV0 ||
		code == AgentLauncherRuntimeBindingRefsV0 ||
		code == AgentLauncherEvidenceRefsV0 ||
		code == AgentLauncherContextBundleNoResueltoV0
}

func isCompactAgentLauncherTokenV0(value string) bool {
	if len(value) < 2 || len(value) > 80 {
		return false
	}
	for i := 0; i < len(value); i++ {
		b := value[i]
		if isASCIIAlpha(b) || (b >= '0' && b <= '9') {
			continue
		}
		if b == '.' || b == '_' || b == ':' || b == '-' {
			continue
		}
		return false
	}
	return true
}
