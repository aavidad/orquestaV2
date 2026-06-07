package orquestaappcodexstack

import (
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	externalContextTruncatedRequiredTestV0 = "validar contexto externo no truncado o justificar materializacion externa"
	externalContextSanitizedRequiredTestV0 = "validar evidencia de saneamiento de contexto o pedir revision al director"
	externalContextRefOnlyRequiredTestV0   = "validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"
	externalContextTruncatedDoneCriteriaV0 = "Si agent_packet.context tiene entradas required=true y truncated=true, usa refs/materializacion externa cuando sea posible y deja nota contexto_truncado_resuelto o contexto_truncado_pendiente."
	externalContextSanitizedDoneCriteriaV0 = "Si agent_packet.context.sanitization_evidence tiene review_required=true, conserva la senal como nota de revision; no bloquees trabajo recuperable por ella."
	externalContextRefOnlyDoneCriteriaV0   = "Si agent_packet.context tiene entradas required=true y mode=ref_only, usa required_ref_action cuando ayude y deja nota contexto_ref_only_resuelto o contexto_ref_only_pendiente."
)

func taskWithContextGuardV0(
	task orquestaruntime.AgentStartTaskV0,
	bundle orquestacontext.ContextMaterializedBundleV0,
) orquestaruntime.AgentStartTaskV0 {
	if !contextBundleHasRequiredTruncatedEntryV0(bundle) &&
		!contextBundleHasRequiredRefOnlyEntryV0(bundle) &&
		!contextBundleHasSanitizationReviewSignalV0(bundle) {
		return task
	}
	if contextBundleHasRequiredTruncatedEntryV0(bundle) {
		task.DoneCriteria = compactStringsV0(append(task.DoneCriteria, externalContextTruncatedDoneCriteriaV0))
	}
	if contextBundleHasRequiredRefOnlyEntryV0(bundle) {
		task.DoneCriteria = compactStringsV0(append(task.DoneCriteria, externalContextRefOnlyDoneCriteriaV0))
	}
	if contextBundleHasSanitizationReviewSignalV0(bundle) {
		task.DoneCriteria = compactStringsV0(append(task.DoneCriteria, externalContextSanitizedDoneCriteriaV0))
	}
	return task
}

func packetPoliciesWithContextGuardV0(
	policies []string,
	bundle orquestacontext.ContextMaterializedBundleV0,
) []string {
	if len(bundle.SanitizationEvidence) > 0 {
		policies = compactStringsV0(append(policies, "context_sanitization_evidence_present"))
	}
	if contextBundleHasSanitizationReviewSignalV0(bundle) {
		policies = compactStringsV0(append(policies, "ask_director_on_sanitization_review"))
	}
	if contextBundleHasRequiredRefOnlyEntryV0(bundle) {
		policies = compactStringsV0(append(policies, "required_ref_only_context_guard"))
	}
	return policies
}

func contextBundleHasRequiredTruncatedEntryV0(
	bundle orquestacontext.ContextMaterializedBundleV0,
) bool {
	for _, entry := range bundle.Entries {
		if entry.Required && entry.Truncated {
			return true
		}
	}
	return false
}

func contextBundleHasRequiredRefOnlyEntryV0(
	bundle orquestacontext.ContextMaterializedBundleV0,
) bool {
	return orquestacontext.ContextBundleHasRequiredRefOnlyV0(bundle)
}

func contextBundleHasSanitizationReviewSignalV0(
	bundle orquestacontext.ContextMaterializedBundleV0,
) bool {
	if orquestacontext.ContextBundleRequiresSanitizationReviewV0(bundle) {
		return true
	}
	return strings.Contains(bundle.DirectorQuestionHint, "context_sanitization_review_required")
}
