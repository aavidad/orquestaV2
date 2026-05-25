package orquestaruntime

import (
	"encoding/json"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestarails "orquesta/modulos/orquesta-rails"
)

func BuildAgentStartPacketV0(
	request RuntimeLaunchRequestV0,
	materialized orquestacontext.ContextMaterializedBundleV0,
) AgentStartPacketV0 {
	packet := AgentStartPacketV0{
		SchemaVersion: AgentStartPacketSchemaVersionV0,
		RequestID:     request.RequestID,
		CorrelationID: request.CorrelationID,
		Locale:        request.Locale,
	}
	if issues := ValidateRuntimeLaunchRequestV0(request); len(issues) > 0 {
		packet.Issues = append(packet.Issues, issues...)
		return packet
	}
	if issues := validateAgentStartMaterializedContextV0(request, materialized); len(issues) > 0 {
		packet.Issues = append(packet.Issues, issues...)
		return packet
	}
	packet.WorkOrderRef = request.ContextBundle.WorkOrderRef
	packet.TargetModule = request.ContextBundle.TargetModule
	packet.Phase = request.ContextBundle.Phase
	packet.CapacityLevel = request.ContextBundle.CapacityLevel
	packet.Task = agentStartTaskFromLaunchV0(request)
	packet.Context = materialized
	packet.DeliveryRefs = agentStartDeliveryRefsFromLaunchV0(request)
	packet.Policies = agentStartPoliciesFromLaunchV0(request, materialized)
	if agentStartPacketHasForbiddenOperationalDetailV0(packet) {
		packet.Issues = append(packet.Issues, runtimeLaunchIssueV0(AgentStartPacketInvalidoV0, "packet"))
	}
	if AgentStartPacketWriteSetClosedV0(packet) &&
		agentStartPacketHasWriteSetExpansionContradictionV0(packet) {
		packet.Issues = append(packet.Issues, runtimeLaunchIssueV0(AgentStartPacketInvalidoV0, "task"))
	}
	return packet
}

func validateAgentStartMaterializedContextV0(
	request RuntimeLaunchRequestV0,
	materialized orquestacontext.ContextMaterializedBundleV0,
) []RuntimeLaunchErrorV0 {
	if materialized.SchemaVersion == "" {
		return []RuntimeLaunchErrorV0{runtimeLaunchIssueV0(AgentStartPacketContextMaterializadoRequeridoV0, "context")}
	}
	if !materialized.Valid() {
		return []RuntimeLaunchErrorV0{runtimeLaunchIssueV0(AgentStartPacketContextMaterializadoInvalidoV0, "context")}
	}
	if request.ContextBundle == nil ||
		materialized.BundleRef != request.ContextBundle.BundleRef ||
		materialized.WorkOrderRef != request.ContextBundle.WorkOrderRef ||
		materialized.TargetModule != request.ContextBundle.TargetModule {
		return []RuntimeLaunchErrorV0{runtimeLaunchIssueV0(AgentStartPacketContextMaterializadoInvalidoV0, "context")}
	}
	return nil
}

func agentStartTaskFromLaunchV0(request RuntimeLaunchRequestV0) AgentStartTaskV0 {
	return AgentStartTaskV0{
		TaskRef:       request.Task.TaskRef,
		Priority:      request.Task.Priority,
		Title:         request.FunctionContract.Titulo,
		Objective:     request.FunctionContract.Objetivo,
		TargetSymbol:  request.FunctionContract.SimboloObjetivo,
		WriteSet:      append([]string(nil), request.FunctionContract.WriteSet...),
		RequiredTests: append([]string(nil), request.FunctionContract.TestsObligatorios...),
		DoneCriteria:  append([]string(nil), request.FunctionContract.CriterioCierre...),
	}
}

func agentStartDeliveryRefsFromLaunchV0(request RuntimeLaunchRequestV0) AgentStartDeliveryRefsV0 {
	return AgentStartDeliveryRefsV0{
		MailboxRef:    request.EvidenceRefs.MailboxRef,
		AckRef:        request.EvidenceRefs.AckRef,
		ReadinessRef:  request.EvidenceRefs.ReadinessRef,
		CheckpointRef: request.EvidenceRefs.CheckpointRef,
	}
}

func agentStartPoliciesFromLaunchV0(
	request RuntimeLaunchRequestV0,
	materialized orquestacontext.ContextMaterializedBundleV0,
) []string {
	policies := []string{
		"context_small_by_refs",
		"ask_director_on_missing_context",
		"refs_only_for_credentials",
		"write_set_closed",
		"ack_required",
		"capacity_" + request.CapacityDecision.NivelCapacidad,
	}
	if len(materialized.SanitizationEvidence) > 0 {
		policies = append(policies, "context_sanitization_evidence_present")
	}
	if orquestacontext.ContextBundleRequiresSanitizationReviewV0(materialized) {
		policies = append(policies, "ask_director_on_sanitization_review")
	}
	if orquestacontext.ContextBundleHasRequiredRefOnlyV0(materialized) {
		policies = append(policies, "required_ref_only_context_guard")
	}
	return policies
}

func AgentStartPacketHasPolicyV0(packet AgentStartPacketV0, policy string) bool {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return false
	}
	for _, candidate := range packet.Policies {
		if strings.TrimSpace(candidate) == policy {
			return true
		}
	}
	return false
}

func AgentStartPacketWriteSetClosedV0(packet AgentStartPacketV0) bool {
	return AgentStartPacketHasPolicyV0(packet, "write_set_closed")
}

func runtimeLaunchIssueV0(code RuntimeLaunchErrorCodeV0, field string) RuntimeLaunchErrorV0 {
	return RuntimeLaunchErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.launch." + string(code),
		Field:      field,
		Retryable:  false,
	}
}

func agentStartPacketHasForbiddenOperationalDetailV0(packet AgentStartPacketV0) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	data, err := json.Marshal(packet)
	if err != nil {
		return true
	}
	lower := strings.ToLower(string(data))
	for _, fragment := range []string{
		"provider-ref", "home-ref", "credential-ref", "oauth_ref",
		"model-ref", "bearer ", "access_token", "refresh_token",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	if agentStartPacketHasSecretPrefixV0(lower) {
		return true
	}
	return false
}

func agentStartPacketHasSecretPrefixV0(value string) bool {
	index := strings.Index(value, "sk-")
	for index >= 0 {
		if index == 0 || !agentStartPacketIsRefCharV0(value[index-1]) {
			return true
		}
		next := strings.Index(value[index+1:], "sk-")
		if next < 0 {
			return false
		}
		index += next + 1
	}
	return false
}

func agentStartPacketHasWriteSetExpansionContradictionV0(packet AgentStartPacketV0) bool {
	text := strings.ToLower(strings.Join(append(
		[]string{packet.Task.Objective},
		packet.Task.DoneCriteria...,
	), "\n"))
	for _, marker := range []string{
		"si debes tocar otros ficheros",
		"si hace falta ampliarlo",
		"justificadlo en el ack",
		"justificado en el ack",
		"justificando cualquier toque fuera del write-set",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func agentStartPacketIsRefCharV0(char byte) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') ||
		char == '_' ||
		char == '-'
}
