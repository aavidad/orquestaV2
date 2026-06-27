package orquestacionnucleoapp

import (
	"context"
	"errors"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestComposedAgentLauncherDependenciesResolverV0ResuelveDependenciasPorPuertos(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-deps-001", "agent-request-deps-001")
	functions := &fakeFunctionContractResolverForDependenciesV0{
		contract: runtimeFunctionContractForExternalProcessTestV0(),
	}
	capacities := &fakeCapacityDecisionResolverForDependenciesV0{
		decision: runtimeCapacityDecisionForExternalProcessTestV0(),
	}
	bindings := &fakeRuntimeBindingResolverForDependenciesV0{
		binding: runtimeBindingForExternalProcessTestV0(),
	}
	evidence := &fakeLaunchEvidenceResolverForDependenciesV0{
		evidence: runtimeEvidenceRefsForExternalProcessTestV0(*inbound.Payload),
	}
	bundles := &fakeContextBundleResolverForDependenciesV0{
		bundle: contextBundleForExternalProcessTestV0(),
	}

	resolved, err := ComposedAgentLauncherDependenciesResolverV0{
		Ports: AgentLauncherDependencyPortsV0{
			FunctionContracts: functions,
			CapacityDecisions: capacities,
			RuntimeBindings:   bindings,
			EvidenceRefs:      evidence,
			ContextBundles:    bundles,
		},
	}.ResolveAgentLauncherDependenciesV0(context.TODO(), inbound)
	if err != nil {
		t.Fatalf("ResolveAgentLauncherDependenciesV0: %v", err)
	}
	if issues := orquestaruntime.ValidateAgentLauncherResolvedDependenciesV0(resolved); len(issues) > 0 {
		t.Fatalf("dependencias invalidas: %+v", issues)
	}
	if functions.gotTaskRef != inbound.Payload.TaskRef {
		t.Fatalf("task ref=%s", functions.gotTaskRef)
	}
	if capacities.gotCapacityRequestRef != inbound.Payload.CapacityRequestRef {
		t.Fatalf("capacity ref=%s", capacities.gotCapacityRequestRef)
	}
	if bindings.gotRequest.AgentRequestID != inbound.Payload.AgentRequestID ||
		bindings.gotDecision.DecisionRef != capacities.decision.DecisionRef {
		t.Fatalf("binding args inesperados: request=%+v decision=%+v", bindings.gotRequest, bindings.gotDecision)
	}
	if evidence.gotInbound.CorrelationID != inbound.CorrelationID ||
		bundles.gotInbound.IdempotencyKey != inbound.IdempotencyKey {
		t.Fatalf("inbound no propagado: evidence=%+v bundles=%+v", evidence.gotInbound, bundles.gotInbound)
	}
}

func TestComposedAgentLauncherDependenciesResolverV0BloqueaPuertoFaltante(t *testing.T) {
	_, err := ComposedAgentLauncherDependenciesResolverV0{}.
		ResolveAgentLauncherDependenciesV0(context.TODO(), externalProcessLauncherInboundV0("run-deps-port-001", "agent-request-deps-port-001"))

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "function_contract_resolver")
}

func TestComposedAgentLauncherDependenciesResolverV0BloqueaDependenciaNula(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-deps-null-001", "agent-request-deps-null-001")

	_, err := ComposedAgentLauncherDependenciesResolverV0{
		Ports: AgentLauncherDependencyPortsV0{
			FunctionContracts: &fakeFunctionContractResolverForDependenciesV0{
				contract: runtimeFunctionContractForExternalProcessTestV0(),
			},
			CapacityDecisions: &fakeCapacityDecisionResolverForDependenciesV0{},
			RuntimeBindings: &fakeRuntimeBindingResolverForDependenciesV0{
				binding: runtimeBindingForExternalProcessTestV0(),
			},
			EvidenceRefs: &fakeLaunchEvidenceResolverForDependenciesV0{
				evidence: runtimeEvidenceRefsForExternalProcessTestV0(*inbound.Payload),
			},
			ContextBundles: &fakeContextBundleResolverForDependenciesV0{
				bundle: contextBundleForExternalProcessTestV0(),
			},
		},
	}.ResolveAgentLauncherDependenciesV0(context.TODO(), inbound)

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "capacity_decision")
}

func TestComposedAgentLauncherDependenciesResolverV0PropagaErrorDePuerto(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-deps-port-error-001", "agent-request-deps-port-error-001")

	_, err := ComposedAgentLauncherDependenciesResolverV0{
		Ports: AgentLauncherDependencyPortsV0{
			FunctionContracts: &fakeFunctionContractResolverForDependenciesV0{
				err: errors.New("catalogo no disponible"),
			},
			CapacityDecisions: &fakeCapacityDecisionResolverForDependenciesV0{
				decision: runtimeCapacityDecisionForExternalProcessTestV0(),
			},
			RuntimeBindings: &fakeRuntimeBindingResolverForDependenciesV0{
				binding: runtimeBindingForExternalProcessTestV0(),
			},
			EvidenceRefs: &fakeLaunchEvidenceResolverForDependenciesV0{
				evidence: runtimeEvidenceRefsForExternalProcessTestV0(*inbound.Payload),
			},
			ContextBundles: &fakeContextBundleResolverForDependenciesV0{
				bundle: contextBundleForExternalProcessTestV0(),
			},
		},
	}.ResolveAgentLauncherDependenciesV0(context.TODO(), inbound)

	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "function_contract")
}

type fakeFunctionContractResolverForDependenciesV0 struct {
	gotTaskRef string
	contract   *orquestaruntime.RuntimeFunctionContractV0
	err        error
}

func (resolver *fakeFunctionContractResolverForDependenciesV0) ResolveFunctionContractV0(
	taskRef string,
) (*orquestaruntime.RuntimeFunctionContractV0, error) {
	resolver.gotTaskRef = taskRef
	return resolver.contract, resolver.err
}

type fakeCapacityDecisionResolverForDependenciesV0 struct {
	gotCapacityRequestRef string
	decision              *orquestaruntime.RuntimeCapacityDecisionV0
	err                   error
}

func (resolver *fakeCapacityDecisionResolverForDependenciesV0) ResolveCapacityDecisionV0(
	capacityRequestRef string,
) (*orquestaruntime.RuntimeCapacityDecisionV0, error) {
	resolver.gotCapacityRequestRef = capacityRequestRef
	return resolver.decision, resolver.err
}

type fakeRuntimeBindingResolverForDependenciesV0 struct {
	gotRequest  orquestaruntime.LaunchRuntimeAgentRequestV0
	gotDecision orquestaruntime.RuntimeCapacityDecisionV0
	binding     *orquestaruntime.RuntimeBindingV0
	err         error
}

func (resolver *fakeRuntimeBindingResolverForDependenciesV0) ResolveRuntimeBindingV0(
	request orquestaruntime.LaunchRuntimeAgentRequestV0,
	decision orquestaruntime.RuntimeCapacityDecisionV0,
) (*orquestaruntime.RuntimeBindingV0, error) {
	resolver.gotRequest = request
	resolver.gotDecision = decision
	return resolver.binding, resolver.err
}

type fakeLaunchEvidenceResolverForDependenciesV0 struct {
	gotInbound orquestaruntime.AgentLauncherInboundV0
	evidence   *orquestaruntime.RuntimeEvidenceRefsV0
	err        error
}

func (resolver *fakeLaunchEvidenceResolverForDependenciesV0) ResolveLaunchEvidenceV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (*orquestaruntime.RuntimeEvidenceRefsV0, error) {
	resolver.gotInbound = inbound
	return resolver.evidence, resolver.err
}

type fakeContextBundleResolverForDependenciesV0 struct {
	gotInbound orquestaruntime.AgentLauncherInboundV0
	bundle     *orquestacontext.ContextBundleV0
	err        error
}

func (resolver *fakeContextBundleResolverForDependenciesV0) ResolveContextBundleV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (*orquestacontext.ContextBundleV0, error) {
	resolver.gotInbound = inbound
	return resolver.bundle, resolver.err
}
