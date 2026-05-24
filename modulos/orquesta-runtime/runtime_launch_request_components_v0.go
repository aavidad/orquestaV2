package orquestaruntime

import (
	"fmt"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestarails "orquesta/modulos/orquesta-rails"
)

func (v *runtimeLaunchRequestValidatorV0) validateFunctionContract(contract *RuntimeFunctionContractV0) {
	if contract == nil {
		v.add(FunctionContractRequeridoV0, "function_contract")
		return
	}
	v.requireConst("function_contract.source_contract", contract.SourceContract, FunctionContractSourceV0, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("function_contract.contract_ref", contract.ContractRef, RuntimeLaunchRequestInvalidaV0)
	v.requireConst("function_contract.contract_version", contract.ContractVersion, FunctionContractVersionV0, RuntimeLaunchRequestInvalidaV0)
	if contract.State != FunctionContractStateActiveV0 {
		v.add(FunctionContractNoActivaV0, "function_contract.state")
	}
	v.requireNonEmpty("function_contract.titulo", contract.Titulo)
	v.requireNonEmpty("function_contract.objetivo", contract.Objetivo)
	v.requireRelativePath("function_contract.archivo_objetivo", contract.ArchivoObjetivo, WriteSetInvalidoV0)
	v.requireNonEmpty("function_contract.simbolo_objetivo", contract.SimboloObjetivo)
	v.validateWriteSet(contract.WriteSet)
	v.requireNonEmptyList("function_contract.tests_obligatorios", contract.TestsObligatorios)
	v.requireNonEmptyList("function_contract.criterio_cierre", contract.CriterioCierre)
}

func (v *runtimeLaunchRequestValidatorV0) validateCapacityDecision(decision *RuntimeCapacityDecisionV0) {
	if decision == nil {
		v.add(CapacityDecisionRequeridaV0, "capacity_decision")
		return
	}
	v.requireOpaque("capacity_decision.decision_ref", decision.DecisionRef, CapacityDecisionNoSoportadaV0)
	v.requireConst("capacity_decision.contract_version", decision.ContractVersion, CapacityDecisionVersionV0, CapacityDecisionNoSoportadaV0)
	if !isOneOf(decision.NivelCapacidad, "low", "medium", "high", "xhigh") {
		v.add(CapacityDecisionNoSoportadaV0, "capacity_decision.nivel_capacidad")
	}
	if !isOneOf(decision.ReasoningEffort, "none", "low", "medium", "high", "xhigh") {
		v.add(CapacityDecisionNoSoportadaV0, "capacity_decision.reasoning_effort")
	}
	v.requireOpaque("capacity_decision.pool_ref", decision.PoolRef, PoolRefRequeridoV0)
	v.requireModelRef("capacity_decision.model_ref", decision.ModelRef, ModelRefRequeridoV0)
	v.requireOpaque("capacity_decision.quota_ref", decision.QuotaRef, CapacityDecisionNoSoportadaV0)
}

func (v *runtimeLaunchRequestValidatorV0) validateRuntimeBinding(binding *RuntimeBindingV0, decision *RuntimeCapacityDecisionV0) {
	if binding == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "runtime_binding")
		return
	}
	v.requireOpaque("runtime_binding.logical_agent_ref", binding.LogicalAgentRef, RuntimeLaunchRequestInvalidaV0)
	if !isOneOf(binding.RuntimeKind, "cli", "api", "local_runtime", "browser", "mobile_runtime", "remote_runtime", "other") {
		v.add(RuntimeLaunchRequestInvalidaV0, "runtime_binding.runtime_kind")
	}
	v.requireOpaque("runtime_binding.connector_ref", binding.ConnectorRef, RuntimeLaunchRequestInvalidaV0)
	v.requireProviderRef("runtime_binding.provider_ref", binding.ProviderRef)
	v.requireModelRef("runtime_binding.model_ref", binding.ModelRef, ModelRefRequeridoV0)
	v.requireHomeRef("runtime_binding.home_ref", binding.HomeRef)
	if !isOneOf(binding.CredentialKind, "oauth_ref", "api_key_ref", "local_profile_ref", "service_ref", "none") {
		v.add(RuntimeLaunchRequestInvalidaV0, "runtime_binding.credential_kind")
	}
	v.requireOpaque("runtime_binding.credential_ref", binding.CredentialRef, CredentialRefRequeridoV0)
	if decision != nil && binding.ModelRef != "" && decision.ModelRef != "" && binding.ModelRef != decision.ModelRef {
		v.add(CapacityDecisionNoSoportadaV0, "runtime_binding.model_ref")
	}
}

func (v *runtimeLaunchRequestValidatorV0) validateEvidenceRefs(refs *RuntimeEvidenceRefsV0) {
	if refs == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "evidence_refs")
		return
	}
	v.requireOpaque("evidence_refs.mailbox_ref", refs.MailboxRef, MailboxRefRequeridoV0)
	v.requireOpaque("evidence_refs.ack_ref", refs.AckRef, AckRefRequeridoV0)
	v.requireOpaque("evidence_refs.readiness_ref", refs.ReadinessRef, ReadinessRefRequeridoV0)
	v.optionalOpaque("evidence_refs.checkpoint_ref", refs.CheckpointRef)
}

func (v *runtimeLaunchRequestValidatorV0) validateContextBundle(
	bundle *orquestacontext.ContextBundleV0,
	contract *RuntimeFunctionContractV0,
	decision *RuntimeCapacityDecisionV0,
) {
	if bundle == nil {
		v.add(ContextBundleRequeridoV0, "context_bundle")
		return
	}
	if !bundle.Valid() {
		v.add(ContextBundleInvalidoV0, "context_bundle")
		return
	}
	if decision != nil && bundle.CapacityLevel != decision.NivelCapacidad {
		v.add(ContextBundleInvalidoV0, "context_bundle.capacity_level")
	}
	if contract != nil && !contextBundleMatchesContractModuleV0(bundle.TargetModule, *contract) {
		v.add(ContextBundleInvalidoV0, "context_bundle.target_module")
	}
}

func (v *runtimeLaunchRequestValidatorV0) validateDelivery(delivery *RuntimeDeliveryV0) {
	if delivery == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "delivery")
		return
	}
	v.requireConst("delivery.mailbox_protocol", delivery.MailboxProtocol, DeliveryMailboxProtocolV0, RuntimeLaunchRequestInvalidaV0)
	if !delivery.AckRequired {
		v.add(RuntimeLaunchRequestInvalidaV0, "delivery.ack_required")
	}
	v.requirePositiveWindow("delivery.readiness_timeout_seconds", delivery.ReadinessTimeoutSeconds)
	v.requirePositiveWindow("delivery.max_startup_seconds", delivery.MaxStartupSeconds)
}

func (v *runtimeLaunchRequestValidatorV0) validateSafety(safety *RuntimeSafetyV0) {
	if safety == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "safety")
		return
	}
	v.requireConst("safety.secrets_policy", safety.SecretsPolicy, SafetyPolicyReferencesOnlyV0, RuntimeLaunchRequestInvalidaV0)
	v.requireConst("safety.home_paths_policy", safety.HomePathsPolicy, SafetyPolicyOpaqueRefsOnlyV0, RuntimeLaunchRequestInvalidaV0)
	v.requireConst("safety.provider_policy", safety.ProviderPolicy, SafetyPolicyOpaqueRefsOnlyV0, RuntimeLaunchRequestInvalidaV0)
	v.requireConst("safety.write_set_policy", safety.WriteSetPolicy, SafetyPolicyWriteSetClosedV0, RuntimeLaunchRequestInvalidaV0)
}

func (v *runtimeLaunchRequestValidatorV0) validateWriteSet(paths []string) {
	if len(paths) == 0 {
		v.add(WriteSetRequeridoV0, "function_contract.write_set")
		return
	}
	seen := map[string]bool{}
	for i, path := range paths {
		field := fmt.Sprintf("function_contract.write_set[%d]", i)
		if seen[path] || !orquestarails.WorkspaceRelativePathAllowedV0(path, true) {
			v.add(WriteSetInvalidoV0, field)
		}
		seen[path] = true
	}
}
