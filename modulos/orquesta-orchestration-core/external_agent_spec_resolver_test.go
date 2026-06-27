package orquestacionnucleoapp

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestExternalAgentLaunchSpecResolverV0ConstruyeSpecDesdePuertos(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-spec-resolver-001", "agent-request-spec-001")
	processRequest := externalProcessRuntimeRequestForTestV0(t, "exit")

	resolution, err := externalAgentLaunchSpecResolverForTestV0(processRequest).
		ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !resolution.Spec.Valid() {
		t.Fatalf("spec invalida: %+v", resolution.Spec.Issues)
	}
	if resolution.CommandResolver == nil {
		t.Fatalf("command resolver requerido")
	}
	if resolution.Spec.RequestID != inbound.Payload.AgentRequestID ||
		resolution.Spec.CorrelationID != inbound.CorrelationID {
		t.Fatalf("trazabilidad inesperada: %+v", resolution.Spec)
	}
	if !resolution.Spec.AgentPacket.Valid() {
		t.Fatalf("packet invalido: %+v", resolution.Spec.AgentPacket.Issues)
	}
}

func TestExternalProcessAgentLauncherV0UsaSpecResolverPorPuertos(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-spec-launch-001", "agent-request-spec-launch-001")
	runtime := &recordingExternalProcessRuntimeV0{
		connector: orquestaruntime.NewProcessRuntimeConnectorV0(),
	}
	registry := NewInMemoryAgentProcessRegistryV0()
	launcher := ExternalProcessAgentLauncherV0{
		SpecResolver: externalAgentLaunchSpecResolverForTestV0(
			externalProcessRuntimeRequestForTestV0(t, "exit"),
		),
		Runtime:         runtime,
		ProcessStopper:  runtime,
		ProcessRegistry: registry,
	}

	result, err := launcher.LaunchAgentV0(context.Background(), inbound)
	if err != nil {
		t.Fatalf("LaunchAgentV0: %v", err)
	}
	if result.AgentRequestID != inbound.Payload.AgentRequestID ||
		result.LaunchRef == "" ||
		result.AckRef == "" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if !containsStringPrefixForTestV0(result.EvidenceRefs, "receipt-ref-process-runtime-launch-") {
		t.Fatalf("resultado sin receipt ref compacta: %+v", result.EvidenceRefs)
	}
	record, err := registry.ResolveAgentProcessV0(context.Background(), inbound.Payload.RunID, inbound.Payload.AgentRequestID)
	if err != nil {
		t.Fatalf("resolve registry: %v", err)
	}
	if !containsStringPrefixForTestV0(record.EvidenceRefs, "receipt-ref-process-runtime-launch-") {
		t.Fatalf("registry sin receipt ref compacta: %+v", record.EvidenceRefs)
	}
	waitExternalProcessRuntimeStatusV0(t, runtime.connector, runtime.lastSnapshot.ProcessRef, orquestaruntime.ProcessRuntimeStoppedV0)
}

func TestExternalAgentLaunchSpecResolverV0BloqueaDependenciasInvalidas(t *testing.T) {
	inbound := externalProcessLauncherInboundV0("run-spec-block-001", "agent-request-spec-block-001")
	resolver := ExternalAgentLaunchSpecResolverV0{
		Dependencies: failingAgentLauncherDependenciesResolverV0{
			err: errors.New("dependencias no disponibles"),
		},
		ContextMaterializer: staticRuntimeContextMaterializerV0{},
		Connector:           staticExternalAgentConnectorResolverV0{},
		Options:             externalAgentSpecResolverOptionsForTestV0(),
	}

	_, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), inbound)
	if err == nil {
		t.Fatalf("deberia bloquear dependencias invalidas")
	}
}

func TestContextBundleRuntimeMaterializerV0MaterializaContextoPorPuerto(t *testing.T) {
	request := runtimeLaunchRequestForExternalProcessTestV0(
		*externalProcessLauncherInboundV0("run-materializer-001", "agent-request-materializer-001").Payload,
		externalProcessLauncherInboundV0("run-materializer-001", "agent-request-materializer-001"),
	)

	materialized, err := ContextBundleRuntimeMaterializerV0{
		Reader: fakeContextRefReaderV0{},
	}.MaterializeRuntimeContextV0(context.Background(), request)
	if err != nil {
		t.Fatalf("MaterializeRuntimeContextV0: %v", err)
	}
	if !materialized.Valid() || materialized.BundleRef != request.ContextBundle.BundleRef {
		t.Fatalf("contexto materializado inesperado: %+v", materialized)
	}
}

func externalAgentLaunchSpecResolverForTestV0(
	processRequest orquestaruntime.ProcessRuntimeLaunchRequestV0,
) ExternalAgentLaunchSpecResolverV0 {
	return ExternalAgentLaunchSpecResolverV0{
		Dependencies:        externalProcessDependenciesResolverForTestV0{},
		ContextMaterializer: ContextBundleRuntimeMaterializerV0{Reader: fakeContextRefReaderV0{}},
		Connector: staticExternalAgentConnectorResolverV0{
			profile:         externalAgentConnectorProfileForProcessTestV0(),
			commandResolver: externalProcessCommandResolverForTestV0{processRequest: processRequest},
		},
		Options: externalAgentSpecResolverOptionsForTestV0(),
	}
}

func externalAgentSpecResolverOptionsForTestV0() orquestaruntime.AgentLauncherRuntimeLaunchOptionsV0 {
	return orquestaruntime.AgentLauncherRuntimeLaunchOptionsV0{
		Locale:                  "es-ES",
		AdapterRef:              "nucleo-external-agent-spec-resolver-v0",
		RequestedAt:             "2026-05-08T11:01:00Z",
		ReadinessTimeoutSeconds: 30,
		MaxStartupSeconds:       30,
	}
}

type externalProcessDependenciesResolverForTestV0 struct{}

func (externalProcessDependenciesResolverForTestV0) ResolveAgentLauncherDependenciesV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestaruntime.AgentLauncherResolvedDependenciesV0, error) {
	return ComposedAgentLauncherDependenciesResolverV0{
		Ports: AgentLauncherDependencyPortsV0{
			FunctionContracts: staticFunctionContractResolverForSpecTestV0{},
			CapacityDecisions: staticCapacityDecisionResolverForSpecTestV0{},
			RuntimeBindings:   staticRuntimeBindingResolverForSpecTestV0{},
			EvidenceRefs:      dynamicLaunchEvidenceResolverForSpecTestV0{},
			ContextBundles:    staticContextBundleResolverForSpecTestV0{},
		},
	}.ResolveAgentLauncherDependenciesV0(ctx, inbound)
}

type staticFunctionContractResolverForSpecTestV0 struct{}

func (staticFunctionContractResolverForSpecTestV0) ResolveFunctionContractV0(
	string,
) (*orquestaruntime.RuntimeFunctionContractV0, error) {
	return runtimeFunctionContractForExternalProcessTestV0(), nil
}

type staticCapacityDecisionResolverForSpecTestV0 struct{}

func (staticCapacityDecisionResolverForSpecTestV0) ResolveCapacityDecisionV0(
	string,
) (*orquestaruntime.RuntimeCapacityDecisionV0, error) {
	return runtimeCapacityDecisionForExternalProcessTestV0(), nil
}

type staticRuntimeBindingResolverForSpecTestV0 struct{}

func (staticRuntimeBindingResolverForSpecTestV0) ResolveRuntimeBindingV0(
	orquestaruntime.LaunchRuntimeAgentRequestV0,
	orquestaruntime.RuntimeCapacityDecisionV0,
) (*orquestaruntime.RuntimeBindingV0, error) {
	return runtimeBindingForExternalProcessTestV0(), nil
}

type dynamicLaunchEvidenceResolverForSpecTestV0 struct{}

func (dynamicLaunchEvidenceResolverForSpecTestV0) ResolveLaunchEvidenceV0(
	inbound orquestaruntime.AgentLauncherInboundV0,
) (*orquestaruntime.RuntimeEvidenceRefsV0, error) {
	if inbound.Payload == nil {
		return nil, errors.New("payload requerido")
	}
	return runtimeEvidenceRefsForExternalProcessTestV0(*inbound.Payload), nil
}

type staticContextBundleResolverForSpecTestV0 struct{}

func (staticContextBundleResolverForSpecTestV0) ResolveContextBundleV0(
	orquestaruntime.AgentLauncherInboundV0,
) (*orquestacontext.ContextBundleV0, error) {
	return contextBundleForExternalProcessTestV0(), nil
}

type failingAgentLauncherDependenciesResolverV0 struct {
	err error
}

func (resolver failingAgentLauncherDependenciesResolverV0) ResolveAgentLauncherDependenciesV0(
	context.Context,
	orquestaruntime.AgentLauncherInboundV0,
) (orquestaruntime.AgentLauncherResolvedDependenciesV0, error) {
	return orquestaruntime.AgentLauncherResolvedDependenciesV0{}, resolver.err
}

type staticRuntimeContextMaterializerV0 struct {
	materialized orquestacontext.ContextMaterializedBundleV0
}

func (materializer staticRuntimeContextMaterializerV0) MaterializeRuntimeContextV0(
	context.Context,
	orquestaruntime.RuntimeLaunchRequestV0,
) (orquestacontext.ContextMaterializedBundleV0, error) {
	return materializer.materialized, nil
}

type staticExternalAgentConnectorResolverV0 struct {
	profile         orquestaruntime.ExternalAgentConnectorProfileV0
	commandResolver orquestaruntime.ExternalAgentProcessCommandResolverV0
}

func (resolver staticExternalAgentConnectorResolverV0) ResolveExternalAgentConnectorV0(
	context.Context,
	orquestaruntime.RuntimeLaunchRequestV0,
) (ExternalAgentConnectorResolutionV0, error) {
	return ExternalAgentConnectorResolutionV0{
		Profile:         resolver.profile,
		CommandResolver: resolver.commandResolver,
	}, nil
}

type fakeContextRefReaderV0 struct{}

func (fakeContextRefReaderV0) ReadContextRefV0(
	sourceRef string,
	maxBytes int,
) (orquestacontext.ContextRefContentV0, []orquestacontext.ContextMaterializationIssueV0) {
	content := "contexto compacto de prueba"
	if maxBytes > 0 && len(content) > maxBytes {
		content = content[:maxBytes]
	}
	return orquestacontext.ContextRefContentV0{
		SourceRef: sourceRef,
		Content:   content,
		Bytes:     len(content),
	}, nil
}

func containsStringPrefixForTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
