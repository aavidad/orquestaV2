package orquestacionnucleoapp

import (
	"context"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentLauncherDependencyPortsV0 struct {
	FunctionContracts orquestaruntime.FunctionContractResolverV0
	CapacityDecisions orquestaruntime.CapacityDecisionResolverV0
	RuntimeBindings   orquestaruntime.RuntimeBindingResolverV0
	EvidenceRefs      orquestaruntime.LaunchEvidenceResolverV0
	ContextBundles    orquestaruntime.ContextBundleResolverV0
}

type ComposedAgentLauncherDependenciesResolverV0 struct {
	Ports AgentLauncherDependencyPortsV0
}

var _ AgentLauncherDependenciesResolverPortV0 = ComposedAgentLauncherDependenciesResolverV0{}

func (resolver ComposedAgentLauncherDependenciesResolverV0) ResolveAgentLauncherDependenciesV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestaruntime.AgentLauncherResolvedDependenciesV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, err
	}
	if err := resolver.validatePortsV0(); err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, err
	}
	if issues := orquestaruntime.ValidateAgentLauncherInboundV0(inbound); len(issues) > 0 {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, errorFromAgentLauncherIssueV0(issues[0])
	}
	payload := *inbound.Payload

	functionContract, err := resolver.Ports.FunctionContracts.ResolveFunctionContractV0(payload.TaskRef)
	if err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, wrapSpecResolverErrorV0("function_contract", err)
	}
	capacityDecision, err := resolver.Ports.CapacityDecisions.ResolveCapacityDecisionV0(payload.CapacityRequestRef)
	if err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, wrapSpecResolverErrorV0("capacity_decision", err)
	}
	if capacityDecision == nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"capacity_decision",
			string(orquestaruntime.AgentLauncherCapacityNoResueltaV0),
		)
	}
	runtimeBinding, err := resolver.Ports.RuntimeBindings.ResolveRuntimeBindingV0(payload, *capacityDecision)
	if err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, wrapSpecResolverErrorV0("runtime_binding", err)
	}
	evidenceRefs, err := resolver.Ports.EvidenceRefs.ResolveLaunchEvidenceV0(inbound)
	if err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, wrapSpecResolverErrorV0("evidence_refs", err)
	}
	contextBundle, err := resolver.Ports.ContextBundles.ResolveContextBundleV0(inbound)
	if err != nil {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, wrapSpecResolverErrorV0("context_bundle", err)
	}
	resolved := orquestaruntime.AgentLauncherResolvedDependenciesV0{
		FunctionContract: functionContract,
		CapacityDecision: capacityDecision,
		RuntimeBinding:   runtimeBinding,
		EvidenceRefs:     evidenceRefs,
		ContextBundle:    contextBundle,
	}
	if issues := orquestaruntime.ValidateAgentLauncherResolvedDependenciesV0(resolved); len(issues) > 0 {
		return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, errorFromAgentLauncherIssueV0(issues[0])
	}
	return resolved, nil
}

func (resolver ComposedAgentLauncherDependenciesResolverV0) validatePortsV0() error {
	switch {
	case resolver.Ports.FunctionContracts == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "function_contract_resolver", "function_contract_resolver requerido")
	case resolver.Ports.CapacityDecisions == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "capacity_decision_resolver", "capacity_decision_resolver requerido")
	case resolver.Ports.RuntimeBindings == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "runtime_binding_resolver", "runtime_binding_resolver requerido")
	case resolver.Ports.EvidenceRefs == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "launch_evidence_resolver", "launch_evidence_resolver requerido")
	case resolver.Ports.ContextBundles == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "context_bundle_resolver", "context_bundle_resolver requerido")
	default:
		return nil
	}
}
