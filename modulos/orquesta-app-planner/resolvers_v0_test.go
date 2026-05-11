package orquestaappplanner

import (
	"context"
	"path/filepath"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAppPlanResolversConstruyenRuntimeLaunchValidoV0(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "api")
	inbound := appPlanAgentLauncherInboundForTestV0(unit)
	capacities := appPlanCapacityDecisionResolverForTestV0{plan: plan}

	resolved, err := orquestacionnucleoapp.ComposedAgentLauncherDependenciesResolverV0{
		Ports: orquestacionnucleoapp.AgentLauncherDependencyPortsV0{
			FunctionContracts: AppPlanFunctionContractResolverV0{Plan: plan},
			CapacityDecisions: capacities,
			RuntimeBindings:   appPlanRuntimeBindingResolverForTestV0{},
			EvidenceRefs:      AppPlanLaunchEvidenceResolverV0{Plan: plan},
			ContextBundles:    AppPlanContextBundleResolverV0{Plan: plan, CapacityDecisions: capacities},
		},
	}.ResolveAgentLauncherDependenciesV0(nil, inbound)
	if err != nil {
		t.Fatalf("ResolveAgentLauncherDependenciesV0: %v", err)
	}

	launch, issues := orquestaruntime.LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		inbound,
		resolved,
		orquestaruntime.AgentLauncherRuntimeLaunchOptionsV0{
			AdapterRef:              "adapter-ref-app-planner-test",
			RequestedAt:             "2026-05-09T20:10:00Z",
			ReadinessTimeoutSeconds: 120,
			MaxStartupSeconds:       180,
		},
	)
	if len(issues) != 0 {
		t.Fatalf("launch issues: %+v", issues)
	}

	store, materializationIssues := orquestacontext.NewFileContextRefStoreV0(filepath.Join("..", ".."))
	if len(materializationIssues) != 0 {
		t.Fatalf("context store issues: %+v", materializationIssues)
	}
	materialized := orquestacontext.MaterializeContextBundleV0(*launch.ContextBundle, store)
	packet := orquestaruntime.BuildAgentStartPacketV0(launch, materialized)
	if !packet.Valid() {
		t.Fatalf("packet invalido: packet=%+v materialized=%+v", packet, materialized)
	}
	if packet.Task.TaskRef != unit.TaskRef || packet.DeliveryRefs.AckRef != unit.DeliveryRef {
		t.Fatalf("packet no conserva contrato de unidad: %+v", packet)
	}
}

func TestAppPlanContextBundleResolverUsaCapacidadDecididaV0(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "bootstrap")
	inbound := appPlanAgentLauncherInboundForTestV0(unit)
	capacities := appPlanCapacityDecisionResolverForTestV0{
		plan:            plan,
		forcedCapacity:  "xhigh",
		forcedReasoning: "xhigh",
	}

	bundle, err := AppPlanContextBundleResolverV0{
		Plan:              plan,
		CapacityDecisions: capacities,
	}.ResolveContextBundleV0(inbound)
	if err != nil {
		t.Fatalf("ResolveContextBundleV0: %v", err)
	}
	if !bundle.Valid() {
		t.Fatalf("bundle invalido: %+v", bundle.Issues)
	}
	if bundle.CapacityLevel != "xhigh" {
		t.Fatalf("capacity=%s", bundle.CapacityLevel)
	}
}

func TestAppPlanResolversConstruyenExternalAgentLaunchSpecV0(t *testing.T) {
	plan := mustAppPlanForTestV0(t)
	unit := appUnitByKeyForTestV0(t, plan, "web")
	inbound := appPlanAgentLauncherInboundForTestV0(unit)
	capacities := appPlanCapacityDecisionResolverForTestV0{plan: plan}
	store, materializationIssues := orquestacontext.NewFileContextRefStoreV0(filepath.Join("..", ".."))
	if len(materializationIssues) != 0 {
		t.Fatalf("context store issues: %+v", materializationIssues)
	}

	resolution, err := orquestacionnucleoapp.ExternalAgentLaunchSpecResolverV0{
		Dependencies: orquestacionnucleoapp.ComposedAgentLauncherDependenciesResolverV0{
			Ports: orquestacionnucleoapp.AgentLauncherDependencyPortsV0{
				FunctionContracts: AppPlanFunctionContractResolverV0{Plan: plan},
				CapacityDecisions: capacities,
				RuntimeBindings:   appPlanRuntimeBindingResolverForTestV0{},
				EvidenceRefs:      AppPlanLaunchEvidenceResolverV0{Plan: plan},
				ContextBundles:    AppPlanContextBundleResolverV0{Plan: plan, CapacityDecisions: capacities},
			},
		},
		ContextMaterializer: orquestacionnucleoapp.ContextBundleRuntimeMaterializerV0{Reader: store},
		Connector:           appPlanExternalConnectorResolverForTestV0{},
		Options: orquestaruntime.AgentLauncherRuntimeLaunchOptionsV0{
			AdapterRef:              "adapter-ref-app-plan-spec",
			RequestedAt:             "2026-05-09T20:15:00Z",
			ReadinessTimeoutSeconds: 120,
			MaxStartupSeconds:       180,
		},
	}.ResolveExternalAgentLaunchSpecV0(nil, inbound)
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	if !resolution.Spec.Valid() || !resolution.Spec.AgentPacket.Valid() {
		t.Fatalf("spec invalida: %+v packet=%+v", resolution.Spec.Issues, resolution.Spec.AgentPacket.Issues)
	}
	if resolution.Spec.AgentPacket.Task.TaskRef != unit.TaskRef {
		t.Fatalf("task ref=%s", resolution.Spec.AgentPacket.Task.TaskRef)
	}
}

func appPlanAgentLauncherInboundForTestV0(unit AppWorkUnitV0) orquestaruntime.AgentLauncherInboundV0 {
	return orquestaruntime.AgentLauncherInboundV0{
		TargetPort:     orquestaruntime.AgentLauncherTargetPortV0,
		MessageType:    orquestaruntime.AgentLauncherMessageTypeV0,
		CorrelationID:  "corr-ref-" + unit.AgentRequestID,
		IdempotencyKey: "idem-ref-" + unit.AgentRequestID,
		Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
			AgentRequestID:     unit.AgentRequestID,
			RunID:              "run-ref-app-001",
			PhaseID:            string(unit.PhaseID),
			TaskRef:            unit.TaskRef,
			CapacityRequestRef: capacityRefV0(unit),
			Role:               unit.Role,
			Summary:            unit.Summary,
			EvidenceRefs:       unit.EvidenceRefs,
		},
	}
}

type appPlanCapacityDecisionResolverForTestV0 struct {
	plan            AppMicrotaskPlanV0
	forcedCapacity  string
	forcedReasoning string
}

func (resolver appPlanCapacityDecisionResolverForTestV0) ResolveCapacityDecisionV0(
	capacityRequestRef string,
) (*orquestaruntime.RuntimeCapacityDecisionV0, error) {
	unit, ok := FindAppPlanUnitByCapacityRequestRefV0(resolver.plan, capacityRequestRef)
	if !ok {
		return nil, AppPlannerIssueV0{Field: "capacity_request_ref"}
	}
	capacity := string(unit.Capacity)
	if resolver.forcedCapacity != "" {
		capacity = resolver.forcedCapacity
	}
	reasoning := capacity
	if resolver.forcedReasoning != "" {
		reasoning = resolver.forcedReasoning
	}
	return &orquestaruntime.RuntimeCapacityDecisionV0{
		DecisionRef:     "decision-" + capacityRequestRef,
		ContractVersion: orquestaruntime.CapacityDecisionVersionV0,
		NivelCapacidad:  capacity,
		ReasoningEffort: reasoning,
		PoolRef:         "pool-ref-" + capacity,
		ModelRef:        "modelo-ref-" + capacity,
		QuotaRef:        "quota-ref-" + capacity,
	}, nil
}

type appPlanRuntimeBindingResolverForTestV0 struct{}

func (resolver appPlanRuntimeBindingResolverForTestV0) ResolveRuntimeBindingV0(
	request orquestaruntime.LaunchRuntimeAgentRequestV0,
	decision orquestaruntime.RuntimeCapacityDecisionV0,
) (*orquestaruntime.RuntimeBindingV0, error) {
	return &orquestaruntime.RuntimeBindingV0{
		LogicalAgentRef: "logical-" + request.AgentRequestID,
		RuntimeKind:     "cli",
		ConnectorRef:    "connector-ref-app-planner-test",
		ProviderRef:     "supplier-ref-app-planner-test",
		ModelRef:        decision.ModelRef,
		HomeRef:         "agenthome-ref-" + request.AgentRequestID,
		CredentialKind:  "none",
		CredentialRef:   "credref-" + request.AgentRequestID,
	}, nil
}

type appPlanExternalConnectorResolverForTestV0 struct{}

func (resolver appPlanExternalConnectorResolverForTestV0) ResolveExternalAgentConnectorV0(
	context.Context,
	orquestaruntime.RuntimeLaunchRequestV0,
) (orquestacionnucleoapp.ExternalAgentConnectorResolutionV0, error) {
	return orquestacionnucleoapp.ExternalAgentConnectorResolutionV0{
		Profile:         appPlanExternalConnectorProfileForTestV0(),
		CommandResolver: appPlanExternalCommandResolverForTestV0{},
	}, nil
}

func appPlanExternalConnectorProfileForTestV0() orquestaruntime.ExternalAgentConnectorProfileV0 {
	return orquestaruntime.ExternalAgentConnectorProfileV0{
		SchemaVersion: orquestaruntime.ExternalAgentConnectorProfileSchemaVersionV0,
		ProfileRef:    "external-profile-app-plan-test",
		ConnectorRef:  "connector-ref-app-plan-test",
		RuntimeKind:   "cli",
		LaunchMode:    orquestaruntime.RuntimeLaunchModeNewSessionV0,
		Command: orquestaruntime.ExternalAgentCommandV0{
			CommandRef:    "external-command-app-plan-test",
			ExecutableRef: "external-executable-app-plan-test",
			ArgRefs:       []string{"external-arg-app-plan-test"},
			EnvRefs:       []string{"external-env-app-plan-test"},
			WorkingDirRef: "external-workdir-app-plan-test",
		},
		Security: orquestaruntime.ExternalAgentSecurityPolicyV0{
			OptIn:                 true,
			ShellPolicy:           orquestaruntime.ExternalAgentShellForbiddenV0,
			PathInheritancePolicy: orquestaruntime.ExternalAgentPathInheritanceForbiddenV0,
			EnvironmentPolicy:     orquestaruntime.ExternalAgentEnvExplicitRefsOnlyV0,
			SecretsPolicy:         orquestaruntime.ExternalAgentSecretsReferencesOnlyV0,
			HomePathsPolicy:       orquestaruntime.ExternalAgentHomeOpaqueRefsOnlyV0,
			NetworkPolicy:         orquestaruntime.ExternalAgentNetworkClosedV0,
			TranscriptsPolicy:     orquestaruntime.ExternalAgentTranscriptsForbiddenV0,
		},
	}
}

type appPlanExternalCommandResolverForTestV0 struct{}

func (resolver appPlanExternalCommandResolverForTestV0) ResolveExternalAgentProcessCommandV0(
	context.Context,
	orquestaruntime.ExternalAgentLaunchSpecV0,
) (orquestaruntime.ProcessRuntimeLaunchRequestV0, []orquestaruntime.ExternalAgentConnectorErrorV0) {
	return orquestaruntime.ProcessRuntimeLaunchRequestV0{}, nil
}
